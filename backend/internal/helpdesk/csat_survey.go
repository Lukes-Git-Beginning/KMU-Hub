package helpdesk

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CsatSurvey is a pending survey invitation: the one-click rating link handed
// out when a ticket closes. It carries no rating -- the rating arrives later,
// through the public redeem endpoint, and lands on the same
// ticket_csat_responses row.
type CsatSurvey struct {
	TicketID uuid.UUID
	Token    string
	// SendAfter is when the invitation mail becomes due: close time plus the
	// tenant's configured delay. The dispatcher polls against it.
	SendAfter time.Time
	ExpiresAt time.Time
}

// CsatSurveyToken is the row a survey link resolves to. It is the only thing
// the public redeem endpoint learns before it knows a tenant, so it carries
// exactly what deciding "may this token be redeemed, and for whom" needs and
// nothing else.
type CsatSurveyToken struct {
	ResponseID uuid.UUID
	TenantID   uuid.UUID
	TicketID   uuid.UUID
	ExpiresAt  *time.Time
	// SubmittedAt is non-nil once the ticket has been rated. Redemption clears
	// the token, so this only differs from "row not found" in the window where
	// a rating was submitted through the authenticated route while the link
	// was still out.
	SubmittedAt *time.Time
}

// Usable reports whether the link may still be redeemed. Every negative
// verdict here becomes the same ErrCsatSurveyNotFound the public endpoint
// gives an unknown token: expired, already rated and invented must not be
// distinguishable from the outside.
func (t *CsatSurveyToken) Usable(now time.Time) bool {
	if t.SubmittedAt != nil {
		return false
	}
	// A pending survey without an expiry would be a link that never dies.
	return t.ExpiresAt != nil && now.Before(*t.ExpiresAt)
}

// CsatRedemption is what a redeemed survey link reports back to the customer:
// enough to acknowledge their own submission, nothing about the ticket beyond
// the number they already have from the invitation mail.
type CsatRedemption struct {
	TicketNumber int
	Rating       int16
}

const (
	// csatCommentMaxLength bounds the free-text comment on the public route.
	// The authenticated route inherits the same limit -- an agent typing an
	// essay into the rating field is no more welcome than a script posting one.
	csatCommentMaxLength = 2000

	// csatSurveyTokenMaxLength is the longest token string worth a database
	// round trip. Real tokens are 43 characters (32 bytes base64url); anything
	// past this is a probe, and answering it costs a query.
	csatSurveyTokenMaxLength = 128

	// csatSurveyTokenBytes matches the report share links (000252): 32 bytes of
	// crypto/rand, base64url-encoded to 43 URL-safe characters. The token IS
	// the credential for the public endpoint, so it is drawn the same way.
	csatSurveyTokenBytes = 32

	// CsatSurveyTokenTTL is how long a survey link stays valid AFTER it would
	// be sent out (send time = close + CsatConfig.SurveyDelayMinutes). Expiry
	// is enforced server-side against token_expires_at, never derived from
	// anything encoded in the link itself.
	CsatSurveyTokenTTL = 30 * 24 * time.Hour
)

// newCsatSurveyToken draws a survey link secret from crypto/rand. base64url
// keeps it usable as a URL path segment without escaping.
func newCsatSurveyToken() (string, error) {
	buf := make([]byte, csatSurveyTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate csat survey token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// IssueCsatSurveyToken mints a survey link for a ticket that was just closed.
//
// It returns (nil, nil) -- not an error -- for every legitimate reason not to
// survey: the tenant has surveys switched off, or the ticket already carries a
// rating (a closed-reopened-closed ticket must not ask twice). Callers treat a
// nil survey as "nothing to send", so the ticket close itself is never
// affected by this decision.
//
// An existing *pending* token on the same ticket is replaced: the fresh close
// restarts the delay, and a link from a previous close would expire against an
// obsolete schedule. Replacement invalidates the older link, which is intended.
func (s *Service) IssueCsatSurveyToken(ctx context.Context, t *Ticket, cfg CsatConfig) (*CsatSurvey, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if t.CsatRating != nil {
		return nil, nil
	}
	if err := ValidateCsatConfig(cfg); err != nil {
		return nil, err
	}

	token, err := newCsatSurveyToken()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	sendAfter := now.Add(time.Duration(cfg.SurveyDelayMinutes) * time.Minute)
	expiresAt := sendAfter.Add(CsatSurveyTokenTTL)

	// The repository re-checks "not yet rated" in the same statement; the check
	// above only saves the write in the common case. A rating submitted between
	// the two is the reason the authoritative check has to sit in SQL.
	issued, err := s.repo.IssueCsatSurveyTokenTx(ctx, t.TenantID, t.ID, token, sendAfter, expiresAt, now)
	if err != nil {
		return nil, fmt.Errorf("issue csat survey token: %w", err)
	}
	if !issued {
		return nil, nil
	}

	s.log.InfoContext(ctx, "helpdesk: csat survey token issued",
		"ticket_id", t.ID,
		"expires_at", expiresAt,
		"delay_minutes", cfg.SurveyDelayMinutes,
	)
	return &CsatSurvey{TicketID: t.ID, Token: token, SendAfter: sendAfter, ExpiresAt: expiresAt}, nil
}

// SubmitCsatByToken redeems a survey link. This is the one path in the service
// that begins without a tenant, so the order of steps is the security
// property, not a matter of taste (same shape as
// berichte.Service.GetSharedDocument):
//
//  1. resolve the token in the system context -- the single read that must
//     escape RLS, because which tenant the caller may touch is precisely what
//     it answers;
//  2. refuse expired and already-rated links as the same "not found" an
//     invented token gets;
//  3. re-enter tenant scope with the tenant the token itself named, and only
//     then write. Everything after step 1 is ordinary tenant-scoped work; the
//     system context is never carried past the lookup.
//
// The write consumes the token, so redemption is single-use.
func (s *Service) SubmitCsatByToken(
	ctx context.Context,
	token string,
	rating int16,
	comment *string,
) (*CsatRedemption, error) {
	// Rating is validated before the lookup: a bad rating is the caller's own
	// error and deserves a 400 whether or not the token happens to be real.
	// It leaks nothing -- an invalid rating never gets as far as the token.
	if rating < 1 || rating > 5 {
		return nil, ErrInvalidCsatRating
	}
	if comment != nil && len(*comment) > csatCommentMaxLength {
		return nil, ErrCsatCommentTooLong
	}
	if token == "" || len(token) > csatSurveyTokenMaxLength {
		return nil, ErrCsatSurveyNotFound
	}

	survey, err := s.repo.GetCsatSurveyByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if !survey.Usable(now) {
		s.log.InfoContext(ctx, "helpdesk: csat survey link refused",
			"response_id", survey.ResponseID,
			"already_rated", survey.SubmittedAt != nil,
			"expired", survey.ExpiresAt != nil && !now.Before(*survey.ExpiresAt),
		)
		return nil, ErrCsatSurveyNotFound
	}

	ctx = withTenant(ctx, survey.TenantID)

	// The repository repeats "still pending, still tokened" inside the write.
	// A second redemption arriving between the check above and the write is
	// exactly the race that check cannot cover, and it must not double-rate.
	ticketNumber, err := s.repo.RedeemCsatSurveyTx(
		ctx, survey.TenantID, survey.TicketID, survey.ResponseID, rating, comment, now,
	)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "helpdesk: csat submitted via survey link",
		"ticket_id", survey.TicketID,
		"tenant_id", survey.TenantID,
		"rating", rating,
	)
	return &CsatRedemption{TicketNumber: ticketNumber, Rating: rating}, nil
}
