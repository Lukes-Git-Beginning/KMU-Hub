package helpdesk

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// enabledCsatConfig is the default config with surveys switched on. The
// shipped default is opt-in (DefaultCsatConfig sets Enabled=false), so every
// test that wants a survey to happen has to say so -- otherwise it would
// silently exercise the "tenant has surveys off" branch and pass for the
// wrong reason.
func enabledCsatConfig() CsatConfig {
	cfg := DefaultCsatConfig()
	cfg.Enabled = true
	return cfg
}

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
	cfg := enabledCsatConfig()

	before := time.Now().UTC()
	survey, err := h.svc.IssueCsatSurveyToken(context.Background(), ticket, cfg)
	if err != nil || survey == nil {
		t.Fatalf("expected a survey, got survey=%v err=%v", survey, err)
	}
	if survey.TicketID != ticket.ID {
		t.Errorf("survey belongs to ticket %s, expected %s", survey.TicketID, ticket.ID)
	}

	// Expiry is close + configured delay + link lifetime, enforced server-side.
	wantAtLeast := before.Add(time.Duration(cfg.SurveyDelayMinutes) * time.Minute).Add(CsatSurveyTokenTTL)
	if survey.ExpiresAt.Before(wantAtLeast) {
		t.Errorf("expiry %s is earlier than delay+TTL %s", survey.ExpiresAt, wantAtLeast)
	}

	stored, ok := h.repo.csatTokens[ticket.ID]
	if !ok {
		t.Fatal("no survey was scheduled")
	}
	if !stored.expiresAt.Equal(survey.ExpiresAt) || !stored.sendAfter.Equal(survey.SendAfter) {
		t.Errorf("persisted schedule %+v does not match returned survey %+v", stored, survey)
	}
	// Decision 2026-10-26: the link secret is minted at dispatch, so closing a
	// ticket stores nothing redeemable.
	if stored.tokenHash != "" {
		t.Errorf("closing the ticket persisted a link hash %q; it belongs to the dispatch claim", stored.tokenHash)
	}
}

func TestIssueCsatSurveyToken_DisabledForTenant(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()
	ticket := seedClosedTicket(h, tenantID)
	cfg := enabledCsatConfig()
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

	survey, err := h.svc.IssueCsatSurveyToken(context.Background(), ticket, enabledCsatConfig())
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

	survey, err := h.svc.IssueCsatSurveyToken(context.Background(), &foreign, enabledCsatConfig())
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
	cfg := enabledCsatConfig()
	cfg.SurveyDelayMinutes = 999_999 // beyond the 14-day cap

	if _, err := h.svc.IssueCsatSurveyToken(context.Background(), ticket, cfg); err == nil {
		t.Fatal("expected an error for an out-of-range delay")
	}
	if len(h.repo.csatTokens) != 0 {
		t.Errorf("a rejected config must not persist a token, got %d", len(h.repo.csatTokens))
	}
}
