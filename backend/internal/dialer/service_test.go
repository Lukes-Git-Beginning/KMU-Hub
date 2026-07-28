package dialer

import (
	"context"
	"errors"
	"sync"
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

	// nextContact, when non-nil, is returned by GetNextPendingContact.
	// When nextContact is nil and nextContactQueueEmpty is true, returns nil,nil
	// (triggering the service's auto-complete path).
	// When both are their zero values, returns nil, ErrNoContactsAvailable.
	nextContact            *CampaignContact
	nextContactQueueEmpty  bool

	// addedContacts records the contacts slice passed to AddContacts for assertions.
	addedContacts []CampaignContact
	// addedSkipped is the skipped count returned from AddContacts.
	addedSkipped int
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
func (r *mockCampaignRepo) GetByIDForTenant(_ context.Context, id, _ uuid.UUID) (*Campaign, error) {
	if c, ok := r.campaigns[id]; ok {
		return c, nil
	}
	return nil, ErrCampaignNotFound
}
func (r *mockCampaignRepo) List(_ context.Context, _ uuid.UUID, _ *string, _, _ int) ([]*Campaign, int, error) {
	return nil, 0, nil
}
func (r *mockCampaignRepo) Update(_ context.Context, _ *Campaign, _ uuid.UUID) error { return nil }
func (r *mockCampaignRepo) UpdateStatus(_ context.Context, id, _ uuid.UUID, status string) error {
	r.updateStatusCalls = append(r.updateStatusCalls, struct{ ID uuid.UUID; Status string }{id, status})
	return nil
}
func (r *mockCampaignRepo) Delete(_ context.Context, _, _ uuid.UUID) error { return nil }
func (r *mockCampaignRepo) AddContacts(_ context.Context, _ uuid.UUID, contacts []CampaignContact) (int, int, error) {
	r.addedContacts = append(r.addedContacts, contacts...)
	return len(contacts), r.addedSkipped, nil
}
func (r *mockCampaignRepo) GetNextPendingContact(_ context.Context, _ uuid.UUID) (*CampaignContact, error) {
	if r.nextContact != nil {
		return r.nextContact, nil
	}
	if r.nextContactQueueEmpty {
		// Simulate exhausted queue: nil contact, no error → triggers auto-complete.
		return nil, nil
	}
	return nil, ErrNoContactsAvailable
}
func (r *mockCampaignRepo) ListContacts(_ context.Context, _ uuid.UUID, _ *string, _, _ int) ([]*CampaignContact, int, error) {
	return nil, 0, nil
}
func (r *mockCampaignRepo) UpdateContactStatus(_ context.Context, _, _ uuid.UUID, _ string, _ *uuid.UUID) error {
	return nil
}
func (r *mockCampaignRepo) SetContactCallback(_ context.Context, _, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (r *mockCampaignRepo) SkipContact(_ context.Context, _, _ uuid.UUID) error    { return nil }
func (r *mockCampaignRepo) RequeueContact(_ context.Context, _, _ uuid.UUID) error { return nil }
func (r *mockCampaignRepo) IncrementContactCallCount(_ context.Context, _, _ uuid.UUID) error {
	return nil
}
func (r *mockCampaignRepo) GetCampaignContactByID(_ context.Context, id, _ uuid.UUID) (*CampaignContact, error) {
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
	mu       sync.Mutex
	sessions map[uuid.UUID]*CallSession
	events   []*CallEvent

	// Captured params of the most recent UpdateSessionWithEventAndContact call,
	// for assertions on the campaign-contact write.
	lastContactID         uuid.UUID
	lastContactStatus     string
	lastContactCallbackAt *time.Time
	// failContactUpdate simulates the contact-status write failing inside the
	// transaction: nothing is persisted (rollback), and an error is returned.
	failContactUpdate bool
}

func newMockCallRepo() *mockCallRepo {
	return &mockCallRepo{sessions: make(map[uuid.UUID]*CallSession)}
}

func (r *mockCallRepo) CreateSession(_ context.Context, s *CallSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[s.ID] = s
	return nil
}
func (r *mockCallRepo) GetSessionByID(_ context.Context, id uuid.UUID, tenantID uuid.UUID) (*CallSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[id]
	if !ok {
		return nil, ErrCallSessionNotFound
	}
	// Enforce tenant isolation when a real tenantID is supplied.
	if tenantID != uuid.Nil && s.TenantID != uuid.Nil && s.TenantID != tenantID {
		return nil, ErrCallSessionNotFound
	}
	// Return a copy to avoid data races on the caller side.
	copy := *s
	return &copy, nil
}
func (r *mockCallRepo) UpdateSession(_ context.Context, s *CallSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[s.ID] = s
	return nil
}
func (r *mockCallRepo) AppendEvent(_ context.Context, e *CallEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
	return nil
}

// UpdateSessionWithEventAndContact atomically (under the mock's mutex) updates
// the session, appends the event, and records the campaign-contact write. This
// simulates the transactional behaviour of the real Postgres implementation
// without requiring a live database: when failContactUpdate is set, nothing is
// persisted (rollback) and an error is returned.
func (r *mockCallRepo) UpdateSessionWithEventAndContact(_ context.Context, s *CallSession, e *CallEvent, contactID uuid.UUID, contactStatus string, _ *uuid.UUID, callbackAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sessions[s.ID]; !ok {
		return ErrCallSessionNotFound
	}
	// Tenant isolation check.
	existing := r.sessions[s.ID]
	if s.TenantID != uuid.Nil && existing.TenantID != uuid.Nil && existing.TenantID != s.TenantID {
		return ErrCallSessionNotFound
	}
	if r.failContactUpdate {
		// Simulate the contact write failing inside the TX — the session and
		// event are rolled back (left unmodified) just like a real ROLLBACK.
		return ErrCampaignContactNotFound
	}
	r.sessions[s.ID] = s
	r.events = append(r.events, e)
	r.lastContactID = contactID
	r.lastContactStatus = contactStatus
	r.lastContactCallbackAt = callbackAt
	return nil
}

func (r *mockCallRepo) ListEventsBySession(_ context.Context, sessionID uuid.UUID) ([]*CallEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*CallEvent
	for _, e := range r.events {
		if e.DialerCallSessionID == sessionID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (r *mockCallRepo) GetRecentCallsForTenant(_ context.Context, _ uuid.UUID, _ int) ([]RecentCallRow, error) {
	return nil, nil
}

func (r *mockCallRepo) ListCallsByContact(_ context.Context, _ uuid.UUID) ([]ContactCallRow, error) {
	return nil, nil
}

func (r *mockCallRepo) GetTenantCallsTodayCount(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (r *mockCallRepo) GetTenantAppointmentsTodayCount(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (r *mockCallRepo) GetAgentCallsTodayCount(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (r *mockCallRepo) GetAgentAvgDurationToday(_ context.Context, _ uuid.UUID) (float64, error) {
	return 0, nil
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
func (r *mockOutcomeRepo) GetByID(_ context.Context, id, _ uuid.UUID) (*CallOutcome, error) {
	if o, ok := r.outcomes[id]; ok {
		return o, nil
	}
	return nil, ErrOutcomeNotFound
}
func (r *mockOutcomeRepo) List(_ context.Context, _ uuid.UUID, _ bool) ([]*CallOutcome, error) {
	return nil, nil
}
func (r *mockOutcomeRepo) Update(_ context.Context, _ *CallOutcome) error { return nil }
func (r *mockOutcomeRepo) Delete(_ context.Context, _, _ uuid.UUID) error { return nil }
func (r *mockOutcomeRepo) EnsureDefaults(_ context.Context, _ uuid.UUID) error {
	return nil
}

type mockAgentStatusRepo struct{}

func (r *mockAgentStatusRepo) LogStatusChange(_ context.Context, _ *AgentStatusLogEntry) error {
	return nil
}

func (r *mockAgentStatusRepo) GetActiveAgentIDsForTenant(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (r *mockAgentStatusRepo) GetUserDisplayNames(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][2]string, error) {
	return nil, nil
}

type mockCRMBridge struct {
	mu               sync.Mutex
	lastCallActivity *CallActivityInput
	contactDetails   map[uuid.UUID]*ContactDetails
}

func newMockCRMBridge() *mockCRMBridge {
	return &mockCRMBridge{contactDetails: make(map[uuid.UUID]*ContactDetails)}
}

func (b *mockCRMBridge) CreateCallActivity(_ context.Context, input CallActivityInput) error {
	b.mu.Lock()
	b.lastCallActivity = &input
	b.mu.Unlock()
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
func (b *mockCRMBridge) LastCallActivity() *CallActivityInput {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastCallActivity
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
	_, err := h.svc.LogCallOutcome(context.Background(), uuid.Nil, uuid.New(), uuid.New(), nil, nil, nil, nil)
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

	_, err := h.svc.LogCallOutcome(context.Background(), uuid.Nil, sessionID, uuid.New(), nil, nil, nil, nil)
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

	_, err := h.svc.LogCallOutcome(context.Background(), uuid.Nil, sessionID, outcomeID, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that the CRM bridge received the real contact ID, not the ccID.
	if h.bridge.LastCallActivity() == nil {
		t.Fatal("expected CRM bridge to be called")
	}
	if h.bridge.LastCallActivity().ContactID != realContactID {
		t.Errorf("CRM bridge got ContactID=%s, want %s", h.bridge.LastCallActivity().ContactID, realContactID)
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
	_, err := h.svc.LogCallOutcome(context.Background(), uuid.Nil, sessionID, outcomeID, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if h.bridge.LastCallActivity() == nil {
		t.Fatal("expected CRM bridge to be called")
	}
	// Falls back to CampaignContactID when lookup fails.
	if h.bridge.LastCallActivity().ContactID != ccID {
		t.Errorf("CRM bridge got ContactID=%s, want fallback %s", h.bridge.LastCallActivity().ContactID, ccID)
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
	_, err := h.svc.LogCallOutcome(context.Background(), uuid.Nil, sessionID, outcomeID, nil, nil, &cbTime, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify outcome was set on session.
	if h.calls.sessions[sessionID].OutcomeID == nil {
		t.Error("expected OutcomeID to be set")
	}

	// Verify the campaign-contact status was updated atomically in the same call.
	if h.calls.lastContactID != ccID {
		t.Errorf("expected contact update for %s, got %s", ccID, h.calls.lastContactID)
	}
	if h.calls.lastContactStatus != ContactStatusCallback {
		t.Errorf("expected contact status %q, got %q", ContactStatusCallback, h.calls.lastContactStatus)
	}
	if h.calls.lastContactCallbackAt == nil {
		t.Error("expected callback_at to be set for a callback outcome")
	}
}

// TestLogCallOutcome_ContactUpdateFailureRollsBack verifies that when the
// campaign-contact write fails inside the transaction, the session update and
// outcome event are rolled back together (atomicity) and the error surfaces to
// the caller — rather than the prior behaviour of silently warning and leaving
// the contact stranded in an inconsistent status.
func TestLogCallOutcome_ContactUpdateFailureRollsBack(t *testing.T) {
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
	h.outcomes.outcomes[outcomeID] = &CallOutcome{
		ID:         outcomeID,
		Label:      "Erreicht",
		IsCallback: false,
	}
	h.campaigns.contacts[ccID] = &CampaignContact{
		ID:         ccID,
		CampaignID: uuid.New(),
		ContactID:  uuid.New(),
	}

	// Force the contact write to fail inside the transaction.
	h.calls.failContactUpdate = true

	_, err := h.svc.LogCallOutcome(context.Background(), uuid.Nil, sessionID, outcomeID, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected an error when the contact update fails inside the transaction")
	}

	// The session must NOT have been persisted (rolled back): OutcomeID stays nil.
	if h.calls.sessions[sessionID].OutcomeID != nil {
		t.Error("expected session OutcomeID to remain unset after rollback")
	}
	// No outcome event must have been appended.
	if len(h.calls.events) != 0 {
		t.Errorf("expected no events appended after rollback, got %d", len(h.calls.events))
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

	err := h.svc.checkCampaignCompleteByContact(context.Background(), ccID, uuid.New())
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

// ============================================================================
// Cross-tenant isolation tests (dialer_call_sessions)
// ============================================================================

// TestLogCallOutcome_CrossTenant verifies that LogCallOutcome returns
// ErrCallSessionNotFound when the tenantID doesn't match the session's tenant.
func TestLogCallOutcome_CrossTenant(t *testing.T) {
	h := newTestHarness()
	tenantA := uuid.New()
	tenantB := uuid.New()
	sessionID := uuid.New()

	// Seed a session belonging to tenant A.
	h.calls.sessions[sessionID] = &CallSession{
		ID:                sessionID,
		TenantID:          tenantA,
		CampaignContactID: uuid.New(),
		AgentID:           uuid.New(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Request with tenant B must fail to find the session.
	_, err := h.svc.LogCallOutcome(context.Background(), tenantB, sessionID, uuid.New(), nil, nil, nil, nil)

	if !errors.Is(err, ErrCallSessionNotFound) {
		t.Errorf("expected ErrCallSessionNotFound for cross-tenant access, got %v", err)
	}
}

// TestInitiateDialerCall_TenantIDTagged verifies that a successfully initiated
// call session is tagged with the caller's tenantID.
func TestInitiateDialerCall_TenantIDTagged(t *testing.T) {
	h := newTestHarnessWithConsent(true) // active consent so the call proceeds
	tenantA := uuid.New()
	ccID := uuid.New()

	h.campaigns.contacts[ccID] = &CampaignContact{
		ID:         ccID,
		CampaignID: uuid.New(),
		ContactID:  uuid.New(),
	}

	session, err := h.svc.InitiateDialerCall(context.Background(), tenantA, ccID, uuid.New(), nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.TenantID != tenantA {
		t.Errorf("expected TenantID=%s on new session, got %s", tenantA, session.TenantID)
	}
}

// ============================================================================
// Transaction / Concurrent-call tests
// ============================================================================

// TestLogCallOutcome_Concurrent_SameSession verifies that two concurrent calls
// to LogCallOutcome for the same session do not cause data races. Both calls
// must succeed (the mock is not idempotent at this layer — idempotency is
// enforced by the HTTP middleware). The test uses the race detector (-race) to
// detect any unsynchronised map/slice access inside mockCallRepo.
func TestLogCallOutcome_Concurrent_SameSession(t *testing.T) {
	t.Parallel()

	h := newTestHarness()
	tenantID := uuid.New()
	sessionID := uuid.New()
	ccID := uuid.New()
	realContactID := uuid.New()
	outcomeID := uuid.New()

	h.calls.sessions[sessionID] = &CallSession{
		ID:                sessionID,
		TenantID:          tenantID,
		CampaignContactID: ccID,
		AgentID:           uuid.New(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	h.calls.events = append(h.calls.events, &CallEvent{
		ID:                  uuid.New(),
		DialerCallSessionID: sessionID,
		EventType:           CallEventInitiated,
		OccurredAt:          time.Now().Add(-20 * time.Second),
	})
	h.outcomes.outcomes[outcomeID] = &CallOutcome{
		ID:         outcomeID,
		Label:      "Erreicht",
		IsPositive: true,
	}
	h.campaigns.contacts[ccID] = &CampaignContact{
		ID:         ccID,
		CampaignID: uuid.New(),
		ContactID:  realContactID,
	}

	const workers = 5
	errs := make([]error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			_, errs[i] = h.svc.LogCallOutcome(
				context.Background(),
				tenantID,
				sessionID,
				outcomeID,
				nil, nil, nil, nil,
			)
		}()
	}
	wg.Wait()

	// All goroutines must complete without a panic or nil-pointer dereference.
	// Errors are acceptable (e.g. concurrent UpdateSession on same ID), but the
	// mock must not corrupt its internal state. Count successes for diagnostics.
	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
		}
	}
	if successes == 0 {
		t.Errorf("expected at least one successful LogCallOutcome, all %d goroutines failed: %v", workers, errs)
	}

	// Verify the session was updated at least once (outcome set).
	h.calls.mu.Lock()
	finalSession := h.calls.sessions[sessionID]
	h.calls.mu.Unlock()
	if finalSession == nil || finalSession.OutcomeID == nil {
		t.Error("expected OutcomeID to be set on session after concurrent LogCallOutcome")
	}

	// Verify at least one OUTCOME_LOGGED event was appended.
	h.calls.mu.Lock()
	outcomeEvents := 0
	for _, e := range h.calls.events {
		if e.EventType == CallEventOutcomeLogged {
			outcomeEvents++
		}
	}
	h.calls.mu.Unlock()
	if outcomeEvents == 0 {
		t.Error("expected at least one OUTCOME_LOGGED event to be appended")
	}
}

// ============================================================================
// Campaign Lifecycle — Start / Pause / Archive
// ============================================================================

func TestStartCampaign_HappyPath(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	h.campaigns.campaigns[campaignID] = &Campaign{
		ID:           campaignID,
		Status:       CampaignStatusDraft,
		ContactCount: 3,
	}

	c, err := h.svc.StartCampaign(context.Background(), campaignID, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Status != CampaignStatusActive {
		t.Errorf("expected status=active, got %s", c.Status)
	}
	if len(h.campaigns.updateStatusCalls) != 1 || h.campaigns.updateStatusCalls[0].Status != CampaignStatusActive {
		t.Error("expected UpdateStatus(active) to be called once")
	}
}

func TestStartCampaign_AlreadyActive(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	h.campaigns.campaigns[campaignID] = &Campaign{
		ID:           campaignID,
		Status:       CampaignStatusActive,
		ContactCount: 1,
	}

	_, err := h.svc.StartCampaign(context.Background(), campaignID, uuid.New())
	if !errors.Is(err, ErrCampaignNotDraft) {
		t.Errorf("expected ErrCampaignNotDraft, got %v", err)
	}
}

func TestStartCampaign_NotFound(t *testing.T) {
	h := newTestHarness()
	_, err := h.svc.StartCampaign(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrCampaignNotFound) {
		t.Errorf("expected ErrCampaignNotFound, got %v", err)
	}
}

func TestPauseCampaign_ActiveToPaused(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	h.campaigns.campaigns[campaignID] = &Campaign{ID: campaignID, Status: CampaignStatusActive}

	c, err := h.svc.PauseCampaign(context.Background(), campaignID, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Status != CampaignStatusPaused {
		t.Errorf("expected status=paused, got %s", c.Status)
	}
}

func TestPauseCampaign_PausedToActive(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	h.campaigns.campaigns[campaignID] = &Campaign{ID: campaignID, Status: CampaignStatusPaused}

	c, err := h.svc.PauseCampaign(context.Background(), campaignID, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Status != CampaignStatusActive {
		t.Errorf("expected status=active, got %s", c.Status)
	}
}

func TestPauseCampaign_InvalidTransition(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	h.campaigns.campaigns[campaignID] = &Campaign{ID: campaignID, Status: CampaignStatusDraft}

	_, err := h.svc.PauseCampaign(context.Background(), campaignID, uuid.New())
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Errorf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

func TestArchiveCampaign_CompletedToArchived(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	h.campaigns.campaigns[campaignID] = &Campaign{ID: campaignID, Status: CampaignStatusCompleted}

	err := h.svc.ArchiveCampaign(context.Background(), campaignID, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(h.campaigns.updateStatusCalls) != 1 || h.campaigns.updateStatusCalls[0].Status != CampaignStatusArchived {
		t.Error("expected UpdateStatus(archived) to be called once")
	}
}

func TestArchiveCampaign_PausedToArchived(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	h.campaigns.campaigns[campaignID] = &Campaign{ID: campaignID, Status: CampaignStatusPaused}

	if err := h.svc.ArchiveCampaign(context.Background(), campaignID, uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArchiveCampaign_NotFound(t *testing.T) {
	h := newTestHarness()
	err := h.svc.ArchiveCampaign(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrCampaignNotFound) {
		t.Errorf("expected ErrCampaignNotFound, got %v", err)
	}
}

func TestArchiveCampaign_InvalidFromActive(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	h.campaigns.campaigns[campaignID] = &Campaign{ID: campaignID, Status: CampaignStatusActive}

	err := h.svc.ArchiveCampaign(context.Background(), campaignID, uuid.New())
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Errorf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

// ============================================================================
// CompleteWrapUp + SaveCallNotes
// ============================================================================

func TestCompleteWrapUp_Happy(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()
	sessionID := uuid.New()
	ccID := uuid.New()
	campaignID := uuid.New()

	h.calls.sessions[sessionID] = &CallSession{
		ID:                sessionID,
		TenantID:          tenantID,
		CampaignContactID: ccID,
		AgentID:           uuid.New(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	h.campaigns.contacts[ccID] = &CampaignContact{
		ID:         ccID,
		CampaignID: campaignID,
		ContactID:  uuid.New(),
	}
	// All contacts done — campaign should auto-complete.
	h.campaigns.stats = &CampaignStats{PendingContacts: 0, CallbackContacts: 0}
	h.campaigns.campaigns[campaignID] = &Campaign{ID: campaignID, Status: CampaignStatusActive}

	err := h.svc.CompleteWrapUp(context.Background(), tenantID, sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// WrapUpCompletedAt must be set on the stored session.
	h.calls.mu.Lock()
	stored := h.calls.sessions[sessionID]
	h.calls.mu.Unlock()
	if stored.WrapUpCompletedAt == nil {
		t.Error("expected WrapUpCompletedAt to be set")
	}

	// Campaign auto-complete: UpdateStatus should have been called with "completed".
	hasCompleted := false
	for _, call := range h.campaigns.updateStatusCalls {
		if call.Status == CampaignStatusCompleted {
			hasCompleted = true
		}
	}
	if !hasCompleted {
		t.Error("expected campaign to be auto-completed")
	}
}

func TestCompleteWrapUp_SessionNotFound(t *testing.T) {
	h := newTestHarness()
	err := h.svc.CompleteWrapUp(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrCallSessionNotFound) {
		t.Errorf("expected ErrCallSessionNotFound, got %v", err)
	}
}

func TestSaveCallNotes_Happy(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()
	sessionID := uuid.New()

	h.calls.sessions[sessionID] = &CallSession{
		ID:       sessionID,
		TenantID: tenantID,
		AgentID:  uuid.New(),
	}

	err := h.svc.SaveCallNotes(context.Background(), tenantID, sessionID, "Test note")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	h.calls.mu.Lock()
	stored := h.calls.sessions[sessionID]
	h.calls.mu.Unlock()
	if stored.Notes == nil || *stored.Notes != "Test note" {
		t.Errorf("expected notes=%q, got %v", "Test note", stored.Notes)
	}
}

func TestSaveCallNotes_CrossTenantBlocked(t *testing.T) {
	h := newTestHarness()
	tenantA := uuid.New()
	tenantB := uuid.New()
	sessionID := uuid.New()

	h.calls.sessions[sessionID] = &CallSession{
		ID:       sessionID,
		TenantID: tenantA,
		AgentID:  uuid.New(),
	}

	err := h.svc.SaveCallNotes(context.Background(), tenantB, sessionID, "Should fail")
	if !errors.Is(err, ErrCallSessionNotFound) {
		t.Errorf("expected ErrCallSessionNotFound for cross-tenant access, got %v", err)
	}
}

// ============================================================================
// GetNextContact + AddContactsToCampaign
// ============================================================================

func TestGetNextContact_Happy(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	contactID := uuid.New()
	ccID := uuid.New()

	h.campaigns.campaigns[campaignID] = &Campaign{ID: campaignID, Status: CampaignStatusActive}
	h.campaigns.nextContact = &CampaignContact{
		ID:        ccID,
		ContactID: contactID,
		Status:    ContactStatusPending,
	}
	h.bridge.contactDetails[contactID] = &ContactDetails{
		ID:    contactID,
		Name:  "Hans Müller",
		Phone: "+4915112345678",
	}

	cc, err := h.svc.GetNextContact(context.Background(), campaignID, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cc.ContactName != "Hans Müller" {
		t.Errorf("expected enriched name, got %q", cc.ContactName)
	}
	if cc.ContactPhone != "+4915112345678" {
		t.Errorf("expected enriched phone, got %q", cc.ContactPhone)
	}
}

func TestGetNextContact_QueueExhausted(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	h.campaigns.campaigns[campaignID] = &Campaign{ID: campaignID, Status: CampaignStatusActive}
	// Signal exhausted queue: repo returns nil,nil to trigger auto-complete path.
	h.campaigns.nextContactQueueEmpty = true

	_, err := h.svc.GetNextContact(context.Background(), campaignID, uuid.New())
	if !errors.Is(err, ErrNoContactsAvailable) {
		t.Errorf("expected ErrNoContactsAvailable, got %v", err)
	}
	// Campaign must have been transitioned to completed.
	hasCompleted := false
	for _, call := range h.campaigns.updateStatusCalls {
		if call.Status == CampaignStatusCompleted {
			hasCompleted = true
		}
	}
	if !hasCompleted {
		t.Error("expected campaign to be auto-completed when queue is exhausted")
	}
}

func TestGetNextContact_CampaignNotActive(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	h.campaigns.campaigns[campaignID] = &Campaign{ID: campaignID, Status: CampaignStatusPaused}

	_, err := h.svc.GetNextContact(context.Background(), campaignID, uuid.New())
	if !errors.Is(err, ErrCampaignNotActive) {
		t.Errorf("expected ErrCampaignNotActive, got %v", err)
	}
}

func TestAddContactsToCampaign_PhoneNormalization(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	h.campaigns.campaigns[campaignID] = &Campaign{ID: campaignID, Status: CampaignStatusDraft}

	contactWithPhone := uuid.New()
	contactNoPhone := uuid.New()
	contactBadPhone := uuid.New()

	h.bridge.contactDetails[contactWithPhone] = &ContactDetails{
		ID:    contactWithPhone,
		Name:  "Good Contact",
		Phone: "0151 12345678",
	}
	h.bridge.contactDetails[contactNoPhone] = &ContactDetails{
		ID:    contactNoPhone,
		Name:  "No Phone",
		Phone: "",
	}
	h.bridge.contactDetails[contactBadPhone] = &ContactDetails{
		ID:    contactBadPhone,
		Name:  "Bad Phone",
		Phone: "abc-invalid",
	}

	added, skipped, err := h.svc.AddContactsToCampaign(
		context.Background(),
		campaignID,
		[]uuid.UUID{contactWithPhone, contactNoPhone, contactBadPhone},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All three contacts were fetched successfully — none are skipped at fetch time.
	// Phone normalization warns but does not skip, so added=3, skipped=0.
	if added != 3 {
		t.Errorf("expected added=3, got %d", added)
	}
	if skipped != 0 {
		t.Errorf("expected skipped=0, got %d", skipped)
	}
	// Verify the normalized phone is stored for the contact with a valid number.
	found := false
	for _, cc := range h.campaigns.addedContacts {
		if cc.ContactID == contactWithPhone {
			found = true
			if cc.ContactPhone != "+4915112345678" {
				t.Errorf("expected normalized phone +4915112345678, got %q", cc.ContactPhone)
			}
		}
	}
	if !found {
		t.Error("contact with valid phone not found in added contacts")
	}
}

func TestAddContactsToCampaign_BridgeFetchFailSkips(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	h.campaigns.campaigns[campaignID] = &Campaign{ID: campaignID, Status: CampaignStatusDraft}

	// Only one contact registered in bridge; two others will fail → skipped.
	knownID := uuid.New()
	h.bridge.contactDetails[knownID] = &ContactDetails{ID: knownID, Name: "Known", Phone: "+4915112345678"}

	added, skipped, err := h.svc.AddContactsToCampaign(
		context.Background(),
		campaignID,
		[]uuid.UUID{knownID, uuid.New(), uuid.New()},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if added != 1 {
		t.Errorf("expected added=1, got %d", added)
	}
	if skipped != 2 {
		t.Errorf("expected skipped=2, got %d", skipped)
	}
}

func TestAddContactsToCampaign_CampaignNotDraft(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	h.campaigns.campaigns[campaignID] = &Campaign{ID: campaignID, Status: CampaignStatusActive}

	_, _, err := h.svc.AddContactsToCampaign(context.Background(), campaignID, []uuid.UUID{uuid.New()}, nil)
	if !errors.Is(err, ErrCampaignNotDraft) {
		t.Errorf("expected ErrCampaignNotDraft, got %v", err)
	}
}

// ============================================================================
// CreateCallOutcome + UpdateCallOutcome nil-pointer guards
// ============================================================================

func TestCreateCallOutcome_Happy(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	o, err := h.svc.CreateCallOutcome(context.Background(), tenantID, "Erreicht", "#00AA00", true, false, false, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.Label != "Erreicht" {
		t.Errorf("expected label=Erreicht, got %q", o.Label)
	}
	if o.TenantID != tenantID {
		t.Errorf("expected tenant_id to be set")
	}
	if !o.IsActive {
		t.Error("expected IsActive=true for new outcome")
	}
}

func TestUpdateCallOutcome_AllNilPointers(t *testing.T) {
	h := newTestHarness()
	outcomeID := uuid.New()
	h.outcomes.outcomes[outcomeID] = &CallOutcome{
		ID:            outcomeID,
		Label:         "Original",
		Color:         "#FF0000",
		IsPositive:    false,
		IsCallback:    false,
		IsAppointment: false,
		SortOrder:     5,
		IsActive:      true,
	}

	// All optional fields nil — should be a no-op (no field changes).
	o, err := h.svc.UpdateCallOutcome(context.Background(), outcomeID, uuid.New(), nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.Label != "Original" {
		t.Errorf("expected label unchanged, got %q", o.Label)
	}
}

func TestUpdateCallOutcome_PartialUpdate(t *testing.T) {
	h := newTestHarness()
	outcomeID := uuid.New()
	h.outcomes.outcomes[outcomeID] = &CallOutcome{
		ID:       outcomeID,
		Label:    "Old",
		SortOrder: 1,
		IsActive: true,
	}

	newLabel := "New"
	newSort := 99
	inactive := false

	o, err := h.svc.UpdateCallOutcome(context.Background(), outcomeID, uuid.New(), &newLabel, nil, nil, nil, nil, &newSort, &inactive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.Label != "New" {
		t.Errorf("expected label=New, got %q", o.Label)
	}
	if o.SortOrder != 99 {
		t.Errorf("expected sort_order=99, got %d", o.SortOrder)
	}
	if o.IsActive {
		t.Error("expected IsActive=false after patch")
	}
}

func TestUpdateCallOutcome_NotFound(t *testing.T) {
	h := newTestHarness()
	_, err := h.svc.UpdateCallOutcome(context.Background(), uuid.New(), uuid.New(), nil, nil, nil, nil, nil, nil, nil)
	if !errors.Is(err, ErrOutcomeNotFound) {
		t.Errorf("expected ErrOutcomeNotFound, got %v", err)
	}
}

// ============================================================================
// Simple smoke tests for uncovered service functions
// ============================================================================

func TestSkipContact_Happy(t *testing.T) {
	h := newTestHarness()
	if err := h.svc.SkipContact(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRequeueContact_Happy(t *testing.T) {
	h := newTestHarness()
	if err := h.svc.RequeueContact(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListCampaignContacts_Happy(t *testing.T) {
	h := newTestHarness()
	contacts, total, err := h.svc.ListCampaignContacts(context.Background(), uuid.New(), nil, 1, 20)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if total != 0 || contacts != nil {
		t.Errorf("expected empty result from mock, got %d/%v", total, contacts)
	}
}

func TestListCallOutcomes_Happy(t *testing.T) {
	h := newTestHarness()
	list, err := h.svc.ListCallOutcomes(context.Background(), uuid.New(), false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if list != nil {
		t.Errorf("expected nil list from mock, got %v", list)
	}
}

func TestDeleteCallOutcome_Happy(t *testing.T) {
	h := newTestHarness()
	outcomeID := uuid.New()
	h.outcomes.outcomes[outcomeID] = &CallOutcome{ID: outcomeID, Label: "To Delete"}
	if err := h.svc.DeleteCallOutcome(context.Background(), outcomeID, uuid.New()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetCampaignDashboard_Happy(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	h.campaigns.campaigns[campaignID] = &Campaign{ID: campaignID, Status: CampaignStatusActive}

	stats, err := h.svc.GetCampaignDashboard(context.Background(), campaignID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if stats == nil {
		t.Error("expected non-nil stats")
	}
}

func TestStartCampaign_NoContacts(t *testing.T) {
	h := newTestHarness()
	campaignID := uuid.New()
	h.campaigns.campaigns[campaignID] = &Campaign{
		ID:           campaignID,
		Status:       CampaignStatusDraft,
		ContactCount: 0, // no contacts
	}
	_, err := h.svc.StartCampaign(context.Background(), campaignID, uuid.New())
	if !errors.Is(err, ErrCampaignHasNoContacts) {
		t.Errorf("expected ErrCampaignHasNoContacts, got %v", err)
	}
}
