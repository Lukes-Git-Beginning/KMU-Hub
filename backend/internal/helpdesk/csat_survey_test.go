package helpdesk

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
)

// seedClosedTicket puts a closed, unrated ticket into the mock repo.
func seedClosedTicket(h *testHarness, tenantID uuid.UUID) *Ticket {
	t := &Ticket{
		ID:       uuid.New(),
		TenantID: tenantID,
		Subject:  "Monitor flickers",
		Status:   TicketStatusClosed,
		Priority: TicketPriorityNormal,
	}
	h.repo.tickets[t.ID] = t
	return t
}

func TestIssueCsatSurveyToken_Success(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()
	ticket := seedClosedTicket(h, tenantID)
	cfg := DefaultCsatConfig()

	before := time.Now().UTC()
	survey, err := h.svc.IssueCsatSurveyToken(context.Background(), ticket, cfg)
	if err != nil || survey == nil {
		t.Fatalf("expected a survey, got survey=%v err=%v", survey, err)
	}
	if survey.TicketID != ticket.ID {
		t.Errorf("survey belongs to ticket %s, expected %s", survey.TicketID, ticket.ID)
	}

	// 32 bytes of crypto/rand, base64url without padding -> 43 characters.
	raw, err := base64.RawURLEncoding.DecodeString(survey.Token)
	if err != nil {
		t.Fatalf("token is not base64url: %v", err)
	}
	if len(raw) != csatSurveyTokenBytes {
		t.Errorf("token carries %d bytes of entropy, expected %d", len(raw), csatSurveyTokenBytes)
	}

	// Expiry is close + configured delay + link lifetime, enforced server-side.
	wantAtLeast := before.Add(time.Duration(cfg.SurveyDelayMinutes) * time.Minute).Add(CsatSurveyTokenTTL)
	if survey.ExpiresAt.Before(wantAtLeast) {
		t.Errorf("expiry %s is earlier than delay+TTL %s", survey.ExpiresAt, wantAtLeast)
	}

	stored, ok := h.repo.csatTokens[ticket.ID]
	if !ok {
		t.Fatal("no token was persisted")
	}
	if stored.token != survey.Token || !stored.expiresAt.Equal(survey.ExpiresAt) {
		t.Errorf("persisted token/expiry %+v does not match returned survey %+v", stored, survey)
	}
}

func TestIssueCsatSurveyToken_TokensAreUnique(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()
	cfg := DefaultCsatConfig()

	seen := make(map[string]bool, 16)
	for i := range 16 {
		ticket := seedClosedTicket(h, tenantID)
		survey, err := h.svc.IssueCsatSurveyToken(context.Background(), ticket, cfg)
		if err != nil || survey == nil {
			t.Fatalf("issue #%d: survey=%v err=%v", i, survey, err)
		}
		if seen[survey.Token] {
			t.Fatalf("token %q was handed out twice", survey.Token)
		}
		seen[survey.Token] = true
	}
}

func TestIssueCsatSurveyToken_DisabledForTenant(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()
	ticket := seedClosedTicket(h, tenantID)
	cfg := DefaultCsatConfig()
	cfg.Enabled = false

	survey, err := h.svc.IssueCsatSurveyToken(context.Background(), ticket, cfg)
	if err != nil {
		t.Fatalf("disabled surveys must not be an error, got %v", err)
	}
	if survey != nil {
		t.Errorf("expected no survey, got %+v", survey)
	}
	if len(h.repo.csatTokens) != 0 {
		t.Errorf("expected no persistence, got %d token(s)", len(h.repo.csatTokens))
	}
}

func TestIssueCsatSurveyToken_AlreadyRated(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()
	ticket := seedClosedTicket(h, tenantID)
	rating := int16(4)
	ticket.CsatRating = &rating

	survey, err := h.svc.IssueCsatSurveyToken(context.Background(), ticket, DefaultCsatConfig())
	if err != nil {
		t.Fatalf("an already rated ticket must not be an error, got %v", err)
	}
	if survey != nil {
		t.Errorf("expected no second survey for a rated ticket, got %+v", survey)
	}
	if len(h.repo.csatTokens) != 0 {
		t.Errorf("expected no persistence, got %d token(s)", len(h.repo.csatTokens))
	}
}

func TestIssueCsatSurveyToken_ForeignTicketGetsNoToken(t *testing.T) {
	h := newTestHarness()
	ticket := seedClosedTicket(h, uuid.New())
	// A ticket whose tenant differs from the one the repository is asked for:
	// the SQL selects the ticket by (id, tenant_id) and finds nothing.
	foreign := *ticket
	foreign.TenantID = uuid.New()

	survey, err := h.svc.IssueCsatSurveyToken(context.Background(), &foreign, DefaultCsatConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if survey != nil {
		t.Errorf("expected no survey for a foreign ticket, got %+v", survey)
	}
}

func TestIssueCsatSurveyToken_InvalidConfigRejected(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()
	ticket := seedClosedTicket(h, tenantID)
	cfg := DefaultCsatConfig()
	cfg.SurveyDelayMinutes = 999_999 // beyond the 14-day cap

	if _, err := h.svc.IssueCsatSurveyToken(context.Background(), ticket, cfg); err == nil {
		t.Fatal("expected an error for an out-of-range delay")
	}
	if len(h.repo.csatTokens) != 0 {
		t.Errorf("a rejected config must not persist a token, got %d", len(h.repo.csatTokens))
	}
}
