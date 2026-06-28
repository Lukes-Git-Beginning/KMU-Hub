package helpdesk

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Mock repository
// ---------------------------------------------------------------------------

type mockRepo struct {
	tickets         map[uuid.UUID]*Ticket
	messages        map[uuid.UUID][]*TicketMessage
	queues          map[uuid.UUID]*TicketQueue
	cannedResponses map[uuid.UUID]*CannedResponse
	slaPolicies     map[uuid.UUID]*SLAPolicy

	// Controls for FindOpenTicketsByRequester
	openTickets []*Ticket
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		tickets:         make(map[uuid.UUID]*Ticket),
		messages:        make(map[uuid.UUID][]*TicketMessage),
		queues:          make(map[uuid.UUID]*TicketQueue),
		cannedResponses: make(map[uuid.UUID]*CannedResponse),
		slaPolicies:     make(map[uuid.UUID]*SLAPolicy),
	}
}

func (r *mockRepo) CreateTicket(_ context.Context, t *Ticket) error {
	r.tickets[t.ID] = t
	return nil
}
func (r *mockRepo) GetTicketByID(_ context.Context, id uuid.UUID) (*Ticket, error) {
	if t, ok := r.tickets[id]; ok {
		return t, nil
	}
	return nil, ErrTicketNotFound
}
func (r *mockRepo) ListTickets(_ context.Context, _ uuid.UUID, _ *string, _, _ int) ([]*Ticket, int, error) {
	var list []*Ticket
	for _, t := range r.tickets {
		list = append(list, t)
	}
	return list, len(list), nil
}
func (r *mockRepo) UpdateTicket(_ context.Context, t *Ticket) error {
	r.tickets[t.ID] = t
	return nil
}
func (r *mockRepo) DeleteTicket(_ context.Context, id uuid.UUID) error {
	delete(r.tickets, id)
	return nil
}
func (r *mockRepo) FindOpenTicketsByRequester(_ context.Context, _, _ uuid.UUID, _ string) ([]*Ticket, error) {
	return r.openTickets, nil
}

func (r *mockRepo) CreateMessage(_ context.Context, m *TicketMessage) error {
	r.messages[m.TicketID] = append(r.messages[m.TicketID], m)
	return nil
}
func (r *mockRepo) ListMessagesByTicket(_ context.Context, ticketID uuid.UUID) ([]*TicketMessage, error) {
	return r.messages[ticketID], nil
}
func (r *mockRepo) ReassignMessages(_ context.Context, sourceID, targetID uuid.UUID) error {
	msgs := r.messages[sourceID]
	for _, m := range msgs {
		m.TicketID = targetID
	}
	r.messages[targetID] = append(r.messages[targetID], msgs...)
	delete(r.messages, sourceID)
	return nil
}

// MergeTicketTx simulates the atomic TX: reassign messages then update ticket.
// In the mock both operations are in-memory and therefore trivially atomic.
func (r *mockRepo) MergeTicketTx(_ context.Context, source *Ticket, targetID uuid.UUID) error {
	// Reassign messages.
	msgs := r.messages[source.ID]
	for _, m := range msgs {
		m.TicketID = targetID
	}
	r.messages[targetID] = append(r.messages[targetID], msgs...)
	delete(r.messages, source.ID)

	// Update source ticket.
	r.tickets[source.ID] = source
	return nil
}

func (r *mockRepo) CreateQueue(_ context.Context, q *TicketQueue) error {
	r.queues[q.ID] = q
	return nil
}
func (r *mockRepo) GetQueueByID(_ context.Context, id uuid.UUID) (*TicketQueue, error) {
	if q, ok := r.queues[id]; ok {
		return q, nil
	}
	return nil, ErrQueueNotFound
}
func (r *mockRepo) ListQueues(_ context.Context, _ uuid.UUID) ([]*TicketQueue, error) {
	var list []*TicketQueue
	for _, q := range r.queues {
		list = append(list, q)
	}
	return list, nil
}
func (r *mockRepo) UpdateQueue(_ context.Context, q *TicketQueue) error {
	r.queues[q.ID] = q
	return nil
}
func (r *mockRepo) DeleteQueue(_ context.Context, id uuid.UUID) error {
	delete(r.queues, id)
	return nil
}

func (r *mockRepo) CreateCannedResponse(_ context.Context, cr *CannedResponse) error {
	r.cannedResponses[cr.ID] = cr
	return nil
}
func (r *mockRepo) GetCannedResponseByID(_ context.Context, id uuid.UUID) (*CannedResponse, error) {
	if cr, ok := r.cannedResponses[id]; ok {
		return cr, nil
	}
	return nil, ErrCannedResponseNotFound
}
func (r *mockRepo) ListCannedResponses(_ context.Context, _ uuid.UUID) ([]*CannedResponse, error) {
	var list []*CannedResponse
	for _, cr := range r.cannedResponses {
		list = append(list, cr)
	}
	return list, nil
}
func (r *mockRepo) UpdateCannedResponse(_ context.Context, cr *CannedResponse) error {
	r.cannedResponses[cr.ID] = cr
	return nil
}
func (r *mockRepo) DeleteCannedResponse(_ context.Context, id uuid.UUID) error {
	delete(r.cannedResponses, id)
	return nil
}

func (r *mockRepo) CreateSLAPolicy(_ context.Context, p *SLAPolicy) error {
	r.slaPolicies[p.ID] = p
	return nil
}
func (r *mockRepo) GetSLAPolicyByID(_ context.Context, id uuid.UUID) (*SLAPolicy, error) {
	if p, ok := r.slaPolicies[id]; ok {
		return p, nil
	}
	return nil, ErrSLAPolicyNotFound
}
func (r *mockRepo) ListSLAPolicies(_ context.Context, _ uuid.UUID) ([]*SLAPolicy, error) {
	var list []*SLAPolicy
	for _, p := range r.slaPolicies {
		list = append(list, p)
	}
	return list, nil
}
func (r *mockRepo) UpdateSLAPolicy(_ context.Context, p *SLAPolicy) error {
	r.slaPolicies[p.ID] = p
	return nil
}

func (r *mockRepo) DeleteSLAPolicy(_ context.Context, id uuid.UUID) error {
	delete(r.slaPolicies, id)
	return nil
}

func (r *mockRepo) CreateKBArticle(_ context.Context, a *KBArticle) error {
	return nil
}

func (r *mockRepo) GetKBArticleByID(_ context.Context, id uuid.UUID) (*KBArticle, error) {
	return nil, ErrKBArticleNotFound
}

func (r *mockRepo) ListKBArticles(_ context.Context, _ uuid.UUID) ([]*KBArticle, error) {
	return nil, nil
}

func (r *mockRepo) UpdateKBArticle(_ context.Context, _ *KBArticle) error {
	return nil
}

func (r *mockRepo) DeleteKBArticle(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (r *mockRepo) CreateRoutingRule(_ context.Context, _ *RoutingRule) error {
	return nil
}

func (r *mockRepo) GetRoutingRuleByID(_ context.Context, id uuid.UUID) (*RoutingRule, error) {
	return nil, ErrRoutingRuleNotFound
}

func (r *mockRepo) ListRoutingRules(_ context.Context, _ uuid.UUID) ([]*RoutingRule, error) {
	return nil, nil
}

func (r *mockRepo) UpdateRoutingRule(_ context.Context, _ *RoutingRule) error {
	return nil
}

func (r *mockRepo) DeleteRoutingRule(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (r *mockRepo) GetHelpdeskStats(_ context.Context, _ uuid.UUID) (*HelpdeskStats, error) {
	return &HelpdeskStats{}, nil
}

func (r *mockRepo) GetBusinessHours(_ context.Context, tenantID uuid.UUID) (*BusinessHours, error) {
	return &BusinessHours{
		TenantID: tenantID,
		Schedule: map[string]DaySchedule{},
		Holidays: []HolidayEntry{},
		Timezone: "Europe/Berlin",
	}, nil
}

func (r *mockRepo) UpsertBusinessHours(_ context.Context, bh *BusinessHours) error {
	return nil
}

// ---------------------------------------------------------------------------
// Test harness
// ---------------------------------------------------------------------------

type testHarness struct {
	svc  *Service
	repo *mockRepo
}

func newTestHarness() *testHarness {
	r := newMockRepo()
	svc := NewService(r, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	return &testHarness{svc: svc, repo: r}
}

// ---------------------------------------------------------------------------
// CreateTicket tests
// ---------------------------------------------------------------------------

func TestCreateTicket_Success(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()
	requesterID := uuid.New()

	ticket, err := h.svc.CreateTicket(context.Background(), tenantID, requesterID, "Login broken", TicketPriorityHigh, nil, nil, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.ID == uuid.Nil {
		t.Error("expected a valid ticket ID")
	}
	if ticket.Status != TicketStatusOpen {
		t.Errorf("expected status open, got %s", ticket.Status)
	}
	if ticket.Priority != TicketPriorityHigh {
		t.Errorf("expected priority high, got %s", ticket.Priority)
	}
	if ticket.TenantID != tenantID {
		t.Errorf("expected tenant_id %s, got %s", tenantID, ticket.TenantID)
	}
}

func TestCreateTicket_DefaultPriority(t *testing.T) {
	h := newTestHarness()

	ticket, err := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Help me", "", nil, nil, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.Priority != TicketPriorityNormal {
		t.Errorf("expected default priority normal, got %s", ticket.Priority)
	}
}

func TestCreateTicket_EmptySubject(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "", TicketPriorityNormal, nil, nil, "", "")
	if err == nil {
		t.Error("expected error for empty subject")
	}
}

func TestCreateTicket_InvalidPriority(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Subject", "supercritical", nil, nil, "", "")
	if !errors.Is(err, ErrInvalidPriority) {
		t.Errorf("expected ErrInvalidPriority, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// CloseTicket / ReopenTicket tests
// ---------------------------------------------------------------------------

func TestCloseTicket_SetsResolvedAt(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Billing issue", TicketPriorityNormal, nil, nil, "", "")

	closed, err := h.svc.CloseTicket(context.Background(), ticket.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if closed.Status != TicketStatusClosed {
		t.Errorf("expected status closed, got %s", closed.Status)
	}
	if closed.ResolvedAt == nil {
		t.Error("expected resolved_at to be set")
	}
}

func TestReopenTicket_FromClosed(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Feature request", TicketPriorityLow, nil, nil, "", "")
	h.svc.CloseTicket(context.Background(), ticket.ID)

	reopened, err := h.svc.ReopenTicket(context.Background(), ticket.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reopened.Status != TicketStatusOpen {
		t.Errorf("expected status open, got %s", reopened.Status)
	}
	if reopened.ResolvedAt != nil {
		t.Error("expected resolved_at to be cleared")
	}
}

func TestReopenTicket_FromOpenFails(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Still open", TicketPriorityNormal, nil, nil, "", "")

	_, err := h.svc.ReopenTicket(context.Background(), ticket.ID)
	if err == nil {
		t.Error("expected error when reopening non-closed ticket")
	}
}

// ---------------------------------------------------------------------------
// AddMessage tests
// ---------------------------------------------------------------------------

func TestAddMessage_Success(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Can't login", TicketPriorityNormal, nil, nil, "", "")
	authorID := uuid.New()

	msg, err := h.svc.AddMessage(context.Background(), ticket.ID, authorID, "Have you tried turning it off and on?", false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.ID == uuid.Nil {
		t.Error("expected a valid message ID")
	}
	if msg.Body != "Have you tried turning it off and on?" {
		t.Errorf("unexpected body: %s", msg.Body)
	}
}

func TestAddMessage_SetsFirstResponseAt(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Password reset", TicketPriorityNormal, nil, nil, "", "")
	if ticket.FirstResponseAt != nil {
		t.Fatal("ticket should have no first_response_at initially")
	}

	h.svc.AddMessage(context.Background(), ticket.ID, uuid.New(), "We will help you.", false, nil)

	stored := h.repo.tickets[ticket.ID]
	if stored.FirstResponseAt == nil {
		t.Error("expected first_response_at to be set after first public message")
	}
}

func TestAddMessage_InternalDoesNotSetFirstResponseAt(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Internal note test", TicketPriorityNormal, nil, nil, "", "")

	h.svc.AddMessage(context.Background(), ticket.ID, uuid.New(), "Internal note only.", true, nil)

	stored := h.repo.tickets[ticket.ID]
	if stored.FirstResponseAt != nil {
		t.Error("internal message must not set first_response_at")
	}
}

func TestAddMessage_EmptyBodyFails(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Subject", TicketPriorityNormal, nil, nil, "", "")

	_, err := h.svc.AddMessage(context.Background(), ticket.ID, uuid.New(), "", false, nil)
	if err == nil {
		t.Error("expected error for empty body")
	}
}

// ---------------------------------------------------------------------------
// ApplySLAPolicy tests
// ---------------------------------------------------------------------------

func TestApplySLAPolicy_SetsDueAt(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	// Create ticket and SLA policy.
	ticket, _ := h.svc.CreateTicket(context.Background(), tenantID, uuid.New(), "Slow system", TicketPriorityNormal, nil, nil, "", "")
	policy, _ := h.svc.CreateSLAPolicy(context.Background(), tenantID, "Standard", 60, 480, nil)

	updated, err := h.svc.ApplySLAPolicy(context.Background(), ticket.ID, policy.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.DueAt == nil {
		t.Fatal("expected due_at to be set")
	}

	// due_at should be approximately now + 60 minutes (first_response_mins).
	expectedMin := time.Now().Add(59 * time.Minute)
	expectedMax := time.Now().Add(61 * time.Minute)
	if updated.DueAt.Before(expectedMin) || updated.DueAt.After(expectedMax) {
		t.Errorf("due_at %v not in expected range [%v, %v]", updated.DueAt, expectedMin, expectedMax)
	}
}

func TestApplySLAPolicy_UsesResolutionMinsAfterFirstResponse(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	ticket, _ := h.svc.CreateTicket(context.Background(), tenantID, uuid.New(), "Crash on export", TicketPriorityHigh, nil, nil, "", "")
	policy, _ := h.svc.CreateSLAPolicy(context.Background(), tenantID, "Premium", 30, 240, nil)

	// Simulate first response already set.
	now := time.Now().UTC()
	ticket.FirstResponseAt = &now
	h.repo.tickets[ticket.ID] = ticket

	updated, err := h.svc.ApplySLAPolicy(context.Background(), ticket.ID, policy.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.DueAt == nil {
		t.Fatal("expected due_at to be set")
	}

	// Should use resolution_mins (240) not first_response_mins (30).
	expectedMin := time.Now().Add(239 * time.Minute)
	expectedMax := time.Now().Add(241 * time.Minute)
	if updated.DueAt.Before(expectedMin) || updated.DueAt.After(expectedMax) {
		t.Errorf("due_at %v not in expected range [%v, %v]", updated.DueAt, expectedMin, expectedMax)
	}
}

func TestApplySLAPolicy_PolicyNotFound(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Test", TicketPriorityNormal, nil, nil, "", "")

	_, err := h.svc.ApplySLAPolicy(context.Background(), ticket.ID, uuid.New())
	if !errors.Is(err, ErrSLAPolicyNotFound) {
		t.Errorf("expected ErrSLAPolicyNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// MergeTickets tests
// ---------------------------------------------------------------------------

func TestMergeTickets_Success(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	source, _ := h.svc.CreateTicket(context.Background(), tenantID, uuid.New(), "Duplicate issue", TicketPriorityNormal, nil, nil, "", "")
	target, _ := h.svc.CreateTicket(context.Background(), tenantID, uuid.New(), "Original issue", TicketPriorityNormal, nil, nil, "", "")

	// Add a message to source.
	h.svc.AddMessage(context.Background(), source.ID, uuid.New(), "From source ticket.", false, nil)

	err := h.svc.MergeTickets(context.Background(), source.ID, target.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Source should be merged.
	stored := h.repo.tickets[source.ID]
	if stored.Status != TicketStatusMerged {
		t.Errorf("expected status merged, got %s", stored.Status)
	}
	if stored.MergedIntoID == nil || *stored.MergedIntoID != target.ID {
		t.Errorf("expected merged_into_id = %s, got %v", target.ID, stored.MergedIntoID)
	}

	// Messages should have moved to target.
	if len(h.repo.messages[target.ID]) != 1 {
		t.Errorf("expected 1 message on target, got %d", len(h.repo.messages[target.ID]))
	}
}

func TestMergeTickets_CannotMergeSelf(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Self merge", TicketPriorityNormal, nil, nil, "", "")

	err := h.svc.MergeTickets(context.Background(), ticket.ID, ticket.ID)
	if !errors.Is(err, ErrCannotMergeSelf) {
		t.Errorf("expected ErrCannotMergeSelf, got %v", err)
	}
}

// TestMergeTickets_AtomicTx verifies that MergeTicketTx (the transactional
// path) moves messages AND marks the source as merged in a single call.
// This guards against regressions where the two writes were separate (the
// pre-fix bug: crash between writes left tickets in an inconsistent state).
func TestMergeTickets_AtomicTx(t *testing.T) {
	repo := newMockRepo()

	source := &Ticket{
		ID:     uuid.New(),
		Status: TicketStatusOpen,
	}
	target := &Ticket{
		ID:     uuid.New(),
		Status: TicketStatusOpen,
	}
	repo.tickets[source.ID] = source
	repo.tickets[target.ID] = target

	// Seed a message on source.
	msg := &TicketMessage{ID: uuid.New(), TicketID: source.ID}
	repo.messages[source.ID] = []*TicketMessage{msg}

	// Set the merge fields as merge.go does before calling MergeTicketTx.
	source.Status = TicketStatusMerged
	source.MergedIntoID = &target.ID

	if err := repo.MergeTicketTx(context.Background(), source, target.ID); err != nil {
		t.Fatalf("MergeTicketTx returned unexpected error: %v", err)
	}

	// Message must have moved to target.
	if len(repo.messages[target.ID]) != 1 {
		t.Errorf("expected 1 message on target after MergeTicketTx, got %d", len(repo.messages[target.ID]))
	}
	if len(repo.messages[source.ID]) != 0 {
		t.Errorf("expected 0 messages on source after MergeTicketTx, got %d", len(repo.messages[source.ID]))
	}

	// Source ticket must be marked merged.
	stored := repo.tickets[source.ID]
	if stored.Status != TicketStatusMerged {
		t.Errorf("expected source status merged, got %s", stored.Status)
	}
	if stored.MergedIntoID == nil || *stored.MergedIntoID != target.ID {
		t.Errorf("expected merged_into_id=%s, got %v", target.ID, stored.MergedIntoID)
	}
}

func TestMergeTickets_AlreadyMerged(t *testing.T) {
	h := newTestHarness()

	source, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Already merged", TicketPriorityNormal, nil, nil, "", "")
	target, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Target", TicketPriorityNormal, nil, nil, "", "")

	// Pre-mark source as merged.
	source.Status = TicketStatusMerged
	h.repo.tickets[source.ID] = source

	err := h.svc.MergeTickets(context.Background(), source.ID, target.ID)
	if !errors.Is(err, ErrAlreadyMerged) {
		t.Errorf("expected ErrAlreadyMerged, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// SLA compute status tests
// ---------------------------------------------------------------------------

func TestComputeStatus_OnTrack(t *testing.T) {
	policy := &SLAPolicy{FirstResponseMins: 60, ResolutionMins: 480}
	ticket := &Ticket{}
	due := time.Now().Add(50 * time.Minute)
	ticket.DueAt = &due

	status := ComputeStatus(ticket, policy, time.Now())
	if status != SLAStatusOnTrack {
		t.Errorf("expected on_track, got %s", status)
	}
}

func TestComputeStatus_AtRisk(t *testing.T) {
	policy := &SLAPolicy{FirstResponseMins: 60, ResolutionMins: 480}
	ticket := &Ticket{}
	// Due in 5 minutes — within 20% of 60-min window = 12 min threshold
	due := time.Now().Add(5 * time.Minute)
	ticket.DueAt = &due

	status := ComputeStatus(ticket, policy, time.Now())
	if status != SLAStatusAtRisk {
		t.Errorf("expected at_risk, got %s", status)
	}
}

func TestComputeStatus_Breached(t *testing.T) {
	policy := &SLAPolicy{FirstResponseMins: 60, ResolutionMins: 480}
	ticket := &Ticket{}
	due := time.Now().Add(-1 * time.Minute)
	ticket.DueAt = &due

	status := ComputeStatus(ticket, policy, time.Now())
	if status != SLAStatusBreached {
		t.Errorf("expected breached, got %s", status)
	}
}

func TestComputeStatus_NoDueAt(t *testing.T) {
	policy := &SLAPolicy{FirstResponseMins: 60}
	ticket := &Ticket{} // no due_at

	status := ComputeStatus(ticket, policy, time.Now())
	if status != SLAStatusOnTrack {
		t.Errorf("expected on_track when no due_at, got %s", status)
	}
}

// ---------------------------------------------------------------------------
// Queue & CannedResponse smoke tests
// ---------------------------------------------------------------------------

func TestCreateQueue_Success(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	q, err := h.svc.CreateQueue(context.Background(), tenantID, "Support Tier 1", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.ID == uuid.Nil {
		t.Error("expected valid queue ID")
	}
	if q.Name != "Support Tier 1" {
		t.Errorf("unexpected name: %s", q.Name)
	}
}

func TestCreateCannedResponse_Success(t *testing.T) {
	h := newTestHarness()

	cr, err := h.svc.CreateCannedResponse(context.Background(), uuid.New(), "Thank you", "Thank you for contacting us.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cr.ID == uuid.Nil {
		t.Error("expected valid canned response ID")
	}
}

func TestAssignTicket_Success(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Assign me", TicketPriorityNormal, nil, nil, "", "")
	agentID := uuid.New()

	updated, err := h.svc.AssignTicket(context.Background(), ticket.ID, agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.AssigneeID == nil || *updated.AssigneeID != agentID {
		t.Errorf("expected assignee_id %s, got %v", agentID, updated.AssigneeID)
	}
}

func TestAssignTicket_TicketNotFound(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.AssignTicket(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrTicketNotFound) {
		t.Errorf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestUpdateTicket_PriorityChange(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Priority test", TicketPriorityLow, nil, nil, "", "")
	newPriority := TicketPriorityUrgent

	updated, err := h.svc.UpdateTicket(context.Background(), ticket.ID, nil, &newPriority, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Priority != TicketPriorityUrgent {
		t.Errorf("expected priority urgent, got %s", updated.Priority)
	}
}

func TestUpdateTicket_InvalidPriority(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "Priority test", TicketPriorityLow, nil, nil, "", "")
	bad := "supercritical"

	_, err := h.svc.UpdateTicket(context.Background(), ticket.ID, nil, &bad, nil, nil)
	if !errors.Is(err, ErrInvalidPriority) {
		t.Errorf("expected ErrInvalidPriority, got %v", err)
	}
}

func TestListMessages_AfterAdd(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "List msgs", TicketPriorityNormal, nil, nil, "", "")
	h.svc.AddMessage(context.Background(), ticket.ID, uuid.New(), "First message", false, nil)
	h.svc.AddMessage(context.Background(), ticket.ID, uuid.New(), "Second message (internal)", true, nil)

	msgs, err := h.svc.ListMessages(context.Background(), ticket.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
}

func TestGetTicket_NotFound(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetTicket(context.Background(), uuid.New())
	if !errors.Is(err, ErrTicketNotFound) {
		t.Errorf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestCreateSLAPolicy_InvalidMins(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateSLAPolicy(context.Background(), uuid.New(), "Bad", 0, 60, nil)
	if err == nil {
		t.Error("expected error for first_response_mins=0")
	}

	_, err = h.svc.CreateSLAPolicy(context.Background(), uuid.New(), "Bad2", 60, -1, nil)
	if err == nil {
		t.Error("expected error for resolution_mins negative")
	}
}

func TestListQueues_Empty(t *testing.T) {
	h := newTestHarness()

	queues, err := h.svc.ListQueues(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queues) != 0 {
		t.Errorf("expected empty list, got %d", len(queues))
	}
}

func TestDeleteQueue_Success(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	q, _ := h.svc.CreateQueue(context.Background(), tenantID, "Temp Queue", nil, nil)

	err := h.svc.DeleteQueue(context.Background(), q.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := h.repo.queues[q.ID]; ok {
		t.Error("expected queue to be deleted")
	}
}

func TestUpdateQueue_RenameSuccess(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	q, _ := h.svc.CreateQueue(context.Background(), tenantID, "Old Name", nil, nil)
	newName := "New Name"

	updated, err := h.svc.UpdateQueue(context.Background(), q.ID, &newName, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("expected name 'New Name', got %s", updated.Name)
	}
}

func TestListSLAPolicies_AfterCreate(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	h.svc.CreateSLAPolicy(context.Background(), tenantID, "Policy A", 30, 120, nil)
	h.svc.CreateSLAPolicy(context.Background(), tenantID, "Policy B", 60, 480, nil)

	policies, err := h.svc.ListSLAPolicies(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 2 {
		t.Errorf("expected 2 policies, got %d", len(policies))
	}
}

func TestApplyPolicy_NilPolicy(t *testing.T) {
	ticket := &Ticket{Subject: "Test"}
	result := ApplyPolicy(ticket, nil, time.Now())
	if result.DueAt != nil {
		t.Error("expected nil due_at when policy is nil")
	}
}

func TestSubjectPrefix_ShortSubject(t *testing.T) {
	result := subjectPrefix("Hi", 5)
	if result != "Hi" {
		t.Errorf("expected 'Hi', got %s", result)
	}
}

func TestSubjectPrefix_Empty(t *testing.T) {
	result := subjectPrefix("", 5)
	if result != "" {
		t.Errorf("expected empty string, got %s", result)
	}
}

func TestDetectDuplicates_ExcludesSelf(t *testing.T) {
	repo := newMockRepo()
	tenantID := uuid.New()
	requesterID := uuid.New()

	// The ticket being checked.
	self := &Ticket{
		ID:          uuid.New(),
		TenantID:    tenantID,
		RequesterID: requesterID,
		Subject:     "Login button not working on mobile",
		Status:      TicketStatusOpen,
	}
	// A different ticket that would match the prefix.
	other := &Ticket{
		ID:          uuid.New(),
		TenantID:    tenantID,
		RequesterID: requesterID,
		Subject:     "Login button not working on desktop",
		Status:      TicketStatusOpen,
	}
	repo.openTickets = []*Ticket{self, other}

	dupes, err := DetectDuplicates(context.Background(), repo, tenantID, self)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, d := range dupes {
		if d.ID == self.ID {
			t.Error("DetectDuplicates must not include the ticket itself")
		}
	}
	if len(dupes) != 1 || dupes[0].ID != other.ID {
		t.Errorf("expected [other], got %v", dupes)
	}
}
