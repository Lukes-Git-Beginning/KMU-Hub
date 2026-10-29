package helpdesk

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CsatSurvey is a pending survey invitation scheduled when a ticket closes. It
// carries no link secret: the dispatcher mints the one-click rating link when
// it sends the mail and stores only its hash. The rating arrives later,
// through the public redeem endpoint, and lands on the same
// ticket_csat_responses row.
type CsatSurvey struct {
	TicketID uuid.UUID
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
	// the hash, so this only differs from "row not found" in the window where
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
	// round trip. Real tokens are 43 characters (linksecret: 32 bytes
	// base64url); anything past this is a probe, and answering it costs a query.
	csatSurveyTokenMaxLength = 128

	// CsatSurveyTokenTTL is how long a survey link stays valid AFTER it would
	// be sent out (send time = close + CsatConfig.SurveyDelayMinutes). Expiry
	// is enforced server-side against token_expires_at, never derived from
	// anything encoded in the link itself.
	CsatSurveyTokenTTL = 30 * 24 * time.Hour
)

// IssueCsatSurveyToken schedules a survey for a ticket that was just closed.
// The link secret itself is minted later, by CsatSurveyDispatcher at send
// time (decision 2026-10-26): until the mail goes out nothing redeemable
// exists, and the database only ever holds the secret's hash.
//
// It returns (nil, nil) -- not an error -- for every legitimate reason not to
// survey: the tenant has surveys switched off, or the ticket already carries a
// rating (a closed-reopened-closed ticket must not ask twice). Callers treat a
// nil survey as "nothing to send", so the ticket close itself is never
// affected by this decision.
//
// An existing *pending* survey on the same ticket is replaced: the fresh close
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

	now := time.Now().UTC()
	sendAfter := now.Add(time.Duration(cfg.SurveyDelayMinutes) * time.Minute)
	expiresAt := sendAfter.Add(CsatSurveyTokenTTL)

	// The repository re-checks "not yet rated" in the same statement; the check
	// above only saves the write in the common case. A rating submitted between
	// the two is the reason the authoritative check has to sit in SQL.
	issued, err := s.repo.IssueCsatSurveyTokenTx(ctx, t.TenantID, t.ID, sendAfter, expiresAt, now)
	if err != nil {
		return nil, fmt.Errorf("schedule csat survey: %w", err)
	}
	if !issued {
		return nil, nil
	}

	s.log.InfoContext(ctx, "helpdesk: csat survey scheduled",
		"ticket_id", t.ID,
		"expires_at", expiresAt,
		"delay_minutes", cfg.SurveyDelayMinutes,
	)
	return &CsatSurvey{TicketID: t.ID, SendAfter: sendAfter, ExpiresAt: expiresAt}, nil
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
