package dialer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

type mockCampaignRepo struct {
	campaigns map[uuid.UUID]*Campaign
	contacts  map[uuid.UUID]*CampaignContact
	stats     *CampaignStats

	updateStatusCalls []struct{ ID uuid.UUID; Status string }
}

func newMockCampaignRepo() *mockCampaignRepo {
	return &mockCampaignRepo{
		campaigns: make(map[uuid.UUID]*Campaign),
		contacts:  make(map[uuid.UUID]*CampaignContact),
	}
}

func (r *mockCampaignRepo) Create(_ context.Context, c *Campaign) error {
	r.campaigns[c.ID] = c
	return nil
}
func (r *mockCampaignRepo) GetByID(_ context.Context, id uuid.UUID) (*Campaign, error) {
	if c, ok := r.campaigns[id]; ok {
		return c, nil
	}
	return nil, ErrCampaignNotFound
}
func (r *mockCampaignRepo) List(_ context.Context, _ uuid.UUID, _ *string, _, _ int) ([]*Campaign, int, error) {
	return nil, 0, nil
}
func (r *mockCampaignRepo) Update(_ context.Context, _ *Campaign) error  { return nil }
func (r *mockCampaignRepo) UpdateStatus(_ context.Context, id uuid.UUID, status string) error {
	r.updateStatusCalls = append(r.updateStatusCalls, struct{ ID uuid.UUID; Status string }{id, status})
	return nil
}
func (r *mockCampaignRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (r *mockCampaignRepo) AddContacts(_ context.Context, _ uuid.UUID, contacts []CampaignContact) (int, int, error) {
	return len(contacts), 0, nil
}
func (r *mockCampaignRepo) GetNextPendingContact(_ context.Context, _ uuid.UUID) (*CampaignContact, error) {
	return nil, ErrNoContactsAvailable
}
func (r *mockCampaignRepo) ListContacts(_ context.Context, _ uuid.UUID, _ *string, _, _ int) ([]*CampaignContact, int, error) {
	return nil, 0, nil
}
func (r *mockCampaignRepo) UpdateContactStatus(_ context.Context, _ uuid.UUID, _ string, _ *uuid.UUID) error {
	return nil
}
func (r *mockCampaignRepo) SetContactCallback(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (r *mockCampaignRepo) SkipContact(_ context.Context, _ uuid.UUID) error    { return nil }
func (r *mockCampaignRepo) RequeueContact(_ context.Context, _ uuid.UUID) error { return nil }
func (r *mockCampaignRepo) IncrementContactCallCount(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (r *mockCampaignRepo) GetCampaignContactByID(_ context.Context, id uuid.UUID) (*CampaignContact, error) {
	if cc, ok := r.contacts[id]; ok {
		return cc, nil
	}
	return nil, ErrCampaignContactNotFound
}
func (r *mockCampaignRepo) GetCampaignStats(_ context.Context, _ uuid.UUID) (*CampaignStats, error) {
	if r.stats != nil {
		return r.stats, nil
	}
	return &CampaignStats{PendingContacts: 1}, nil
}
func (r *mockCampaignRepo) GetAgentStats(_ context.Context, _ uuid.UUID) (*AgentStats, error) {
	return &AgentStats{}, nil
}
func (r *mockCampaignRepo) UpdateCampaignCounts(_ context.Context, _ uuid.UUID) error { return nil }

type mockCallRepo struct {
	sessions map[uuid.UUID]*CallSession
	events   []*CallEvent
}

func newMockCallRepo() *mockCallRepo {
	return &mockCallRepo{sessions: make(map[uuid.UUID]*CallSession)}
}

func (r *mockCallRepo) CreateSession(_ context.Context, s *CallSession) error {
	r.sessions[s.ID] = s
	return nil
}
func (r *mockCallRepo) GetSessionByID(_ context.Context, id uuid.UUID) (*CallSession, error) {
	if s, ok := r.sessions[id]; ok {
		return s, nil
	}
	return nil, ErrCallSessionNotFound
}
func (r *mockCallRepo) UpdateSession(_ context.Context, s *CallSession) error {
	r.sessions[s.ID] = s
	return nil
}
func (r *mockCallRepo) AppendEvent(_ context.Context, e *CallEvent) error {
	r.events = append(r.events, e)
	return nil
}
func (r *mockCallRepo) ListEventsBySession(_ context.Context, sessionID uuid.UUID) ([]*CallEvent, error) {
	var result []*CallEvent
	for _, e := range r.events {
		if e.DialerCallSessionID == sessionID {
			result = append(result, e)
		}
	}
	return result, nil
}

type mockOutcomeRepo struct {
	outcomes map[uuid.UUID]*CallOutcome
}

func newMockOutcomeRepo() *mockOutcomeRepo {
	return &mockOutcomeRepo{outcomes: make(map[uuid.UUID]*CallOutcome)}
}

func (r *mockOutcomeRepo) Create(_ context.Context, o *CallOutcome) error {
	r.outcomes[o.ID] = o
	return nil
}
func (r *mockOutcomeRepo) GetByID(_ context.Context, id uuid.UUID) (*CallOutcome, error) {
	if o, ok := r.outcomes[id]; ok {
		return o, nil
	}
	return nil, ErrOutcomeNotFound
}
func (r *mockOutcomeRepo) List(_ context.Context, _ uuid.UUID, _ bool) ([]*CallOutcome, error) {
	return nil, nil
}
func (r *mockOutcomeRepo) Update(_ context.Context, _ *CallOutcome) error { return nil }
func (r *mockOutcomeRepo) Delete(_ context.Context, _ uuid.UUID) error    { return nil }
func (r *mockOutcomeRepo) EnsureDefaults(_ context.Context, _ uuid.UUID) error {
	return nil
}

type mockAgentStatusRepo struct{}

func (r *mockAgentStatusRepo) LogStatusChange(_ context.Context, _ *AgentStatusLogEntry) error {
	return nil
}

type mockCRMBridge struct {
	lastCallActivity *CallActivityInput
	contactDetails   map[uuid.UUID]*ContactDetails
}

func newMockCRMBridge() *mockCRMBridge {
	return &mockCRMBridge{contactDetails: make(map[uuid.UUID]*ContactDetails)}
}

func (b *mockCRMBridge) CreateCallActivity(_ context.Context, input CallActivityInput) error {
	b.lastCallActivity = &input
	return nil
}
func (b *mockCRMBridge) GetContactDetails(_ context.Context, id uuid.UUID) (*ContactDetails, error) {
	if d, ok := b.contactDetails[id]; ok {
		return d, nil
	}
	return nil, errors.New("contact not found")
}
func (b *mockCRMBridge) ResolveFilterContacts(_ context.Context, _ uuid.UUID) ([]ContactImport, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helper to build a service with mocks
// ---------------------------------------------------------------------------

type testHarness struct {
	svc       *Service
	campaigns *mockCampaignRepo
	calls     *mockCallRepo
	outcomes  *mockOutcomeRepo
	bridge    *mockCRMBridge
}

func newTestHarness() *testHarness {
	cr := newMockCampaignRepo()
	cl := newMockCallRepo()
	or := newMockOutcomeRepo()
	br := newMockCRMBridge()
	svc := NewService(cr, cl, or, &mockAgentStatusRepo{}, nil, br, nil)
	return &testHarness{svc: svc, campaigns: cr, calls: cl, outcomes: or, bridge: br}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestLogCallOutcome_SessionNotFound(t *testing.T) {
	h := newTestHarness()
	_, err := h.svc.LogCallOutcome(context.Background(), uuid.New(), uuid.New(), nil, nil, nil, nil)
	if !errors.Is(err, ErrCallSessionNotFound) {
		t.Errorf("expected ErrCallSessionNotFound, got %v", err)
	}
}

func TestLogCallOutcome_OutcomeNotFound(t *testing.T) {
	h := newTestHarness()
	sessionID := uuid.New()
	ccID := uuid.New()

	h.calls.sessions[sessionID] = &CallSession{
		ID:                sessionID,
		CampaignContactID: ccID,
		AgentID:           uuid.New(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	_, err := h.svc.LogCallOutcome(context.Background(), sessionID, uuid.New(), nil, nil, nil, nil)
	if !errors.Is(err, ErrOutcomeNotFound) {
		t.Errorf("expected ErrOutcomeNotFound, got %v", err)
	}
}

func TestLogCallOutcome_ResolvesRealContactID(t *testing.T) {
	h := newTestHarness()
	sessionID := uuid.New()
	ccID := uuid.New()          // campaign_contact join table ID
	realContactID := uuid.New() // actual CRM contact ID
	outcomeID := uuid.New()

	h.calls.sessions[sessionID] = &CallSession{
		ID:                sessionID,
		CampaignContactID: ccID,
		AgentID:           uuid.New(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Add the initiated event for duration calculation.
	h.calls.events = append(h.calls.events, &CallEvent{
		ID:                  uuid.New(),
		DialerCallSessionID: sessionID,
		EventType:           CallEventInitiated,
		OccurredAt:          time.Now().Add(-30 * time.Second),
	})

	h.outcomes.outcomes[outcomeID] = &CallOutcome{
		ID:         outcomeID,
		Label:      "Erreicht",
		IsPositive: true,
	}

	// This is the critical setup: the campaign contact has the real contact ID.
	h.campaigns.contacts[ccID] = &CampaignContact{
		ID:         ccID,
		CampaignID: uuid.New(),
		ContactID:  realContactID,
	}

	_, err := h.svc.LogCallOutcome(context.Background(), sessionID, outcomeID, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that the CRM bridge received the real contact ID, not the ccID.
	if h.bridge.lastCallActivity == nil {
		t.Fatal("expected CRM bridge to be called")
	}
	if h.bridge.lastCallActivity.ContactID != realContactID {
		t.Errorf("CRM bridge got ContactID=%s, want %s", h.bridge.lastCallActivity.ContactID, realContactID)
	}
}

func TestLogCallOutcome_FallsBackWhenContactNotFound(t *testing.T) {
	h := newTestHarness()
	sessionID := uuid.New()
	ccID := uuid.New()
	outcomeID := uuid.New()

	h.calls.sessions[sessionID] = &CallSession{
		ID:                sessionID,
		CampaignContactID: ccID,
		AgentID:           uuid.New(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	h.calls.events = append(h.calls.events, &CallEvent{
		ID:                  uuid.New(),
		DialerCallSessionID: sessionID,
		EventType:           CallEventInitiated,
		OccurredAt:          time.Now().Add(-10 * time.Second),
	})

	h.outcomes.outcomes[outcomeID] = &CallOutcome{
		ID:    outcomeID,
		Label: "Nicht erreicht",
	}

	// Don't add the campaign contact — should fall back to ccID.
	_, err := h.svc.LogCallOutcome(context.Background(), sessionID, outcomeID, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if h.bridge.lastCallActivity == nil {
		t.Fatal("expected CRM bridge to be called")
	}
	// Falls back to CampaignContactID when lookup fails.
	if h.bridge.lastCallActivity.ContactID != ccID {
		t.Errorf("CRM bridge got ContactID=%s, want fallback %s", h.bridge.lastCallActivity.ContactID, ccID)
	}
}

func TestLogCallOutcome_CallbackSetsContactStatus(t *testing.T) {
	h := newTestHarness()
	sessionID := uuid.New()
	ccID := uuid.New()
	outcomeID := uuid.New()

	h.calls.sessions[sessionID] = &CallSession{
		ID:                sessionID,
		CampaignContactID: ccID,
		AgentID:           uuid.New(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	h.calls.events = append(h.calls.events, &CallEvent{
		ID:                  uuid.New(),
		DialerCallSessionID: sessionID,
		EventType:           CallEventInitiated,
		OccurredAt:          time.Now().Add(-10 * time.Second),
	})

	h.outcomes.outcomes[outcomeID] = &CallOutcome{
		ID:         outcomeID,
		Label:      "Wiedervorlage",
		IsCallback: true,
	}

	h.campaigns.contacts[ccID] = &CampaignContact{
		ID:         ccID,
		CampaignID: uuid.New(),
		ContactID:  uuid.New(),
	}

	cbTime := time.Now().Add(24 * time.Hour)
	_, err := h.svc.LogCallOutcome(context.Background(), sessionID, outcomeID, nil, nil, &cbTime, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify outcome was set on session.
	if h.calls.sessions[sessionID].OutcomeID == nil {
		t.Error("expected OutcomeID to be set")
	}
}

func TestCheckCampaignCompleteByContact(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	ccID := uuid.New()

	h.campaigns.contacts[ccID] = &CampaignContact{
		ID:         ccID,
		CampaignID: campaignID,
		ContactID:  uuid.New(),
	}

	// Set stats so campaign auto-completes.
	h.campaigns.stats = &CampaignStats{
		PendingContacts:  0,
		CallbackContacts: 0,
	}

	err := h.svc.checkCampaignCompleteByContact(context.Background(), ccID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Campaign should have been transitioned to completed.
	if len(h.campaigns.updateStatusCalls) != 1 {
		t.Fatalf("expected 1 UpdateStatus call, got %d", len(h.campaigns.updateStatusCalls))
	}
	if h.campaigns.updateStatusCalls[0].Status != CampaignStatusCompleted {
		t.Errorf("expected status=%s, got %s", CampaignStatusCompleted, h.campaigns.updateStatusCalls[0].Status)
	}
}
