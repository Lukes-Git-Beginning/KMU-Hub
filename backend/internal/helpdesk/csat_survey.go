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
	TicketID  uuid.UUID
	Token     string
	ExpiresAt time.Time
}

const (
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
	expiresAt := now.
		Add(time.Duration(cfg.SurveyDelayMinutes) * time.Minute).
		Add(CsatSurveyTokenTTL)

	// The repository re-checks "not yet rated" in the same statement; the check
	// above only saves the write in the common case. A rating submitted between
	// the two is the reason the authoritative check has to sit in SQL.
	issued, err := s.repo.IssueCsatSurveyTokenTx(ctx, t.TenantID, t.ID, token, expiresAt, now)
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
	return &CsatSurvey{TicketID: t.ID, Token: token, ExpiresAt: expiresAt}, nil
}
