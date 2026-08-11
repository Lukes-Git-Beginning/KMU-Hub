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
	kbArticles      map[uuid.UUID]*KBArticle
	routingRules    map[uuid.UUID]*RoutingRule
	businessHours   map[uuid.UUID]*BusinessHours

	// Controls for FindOpenTicketsByRequester
	openTickets []*Ticket

	// contactTenants/companyTenants back ContactExists/CompanyExists: id -> owning tenant.
	contactTenants map[uuid.UUID]uuid.UUID
	companyTenants map[uuid.UUID]uuid.UUID

	// csatCalls counts SubmitCsatTx invocations so tests can assert that an
	// invalid rating never reaches persistence.
	csatCalls int

	// csatTokens holds the pending survey tokens written by
	// IssueCsatSurveyTokenTx, keyed by ticket.
	csatTokens map[uuid.UUID]csatTokenRow

	// stats is returned as-is by GetHelpdeskStats when set.
	stats *HelpdeskStats

	// repoErr, when non-nil, is returned by the plain get/list/upsert wrappers
	// below that have no other natural failure branch (no input validation of
	// their own) -- lets tests exercise the service's error-wrapping path for
	// them without inventing a DB failure.
	repoErr error
}

// csatTokenRow is the mock's stand-in for the token columns on
// ticket_csat_responses.
type csatTokenRow struct {
	token     string
	sendAfter time.Time
	expiresAt time.Time
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		tickets:         make(map[uuid.UUID]*Ticket),
		messages:        make(map[uuid.UUID][]*TicketMessage),
		queues:          make(map[uuid.UUID]*TicketQueue),
		cannedResponses: make(map[uuid.UUID]*CannedResponse),
		slaPolicies:     make(map[uuid.UUID]*SLAPolicy),
		kbArticles:      make(map[uuid.UUID]*KBArticle),
		routingRules:    make(map[uuid.UUID]*RoutingRule),
		businessHours:   make(map[uuid.UUID]*BusinessHours),
		contactTenants:  make(map[uuid.UUID]uuid.UUID),
		companyTenants:  make(map[uuid.UUID]uuid.UUID),
		csatTokens:      make(map[uuid.UUID]csatTokenRow),
	}
}

func (r *mockRepo) CreateTicket(_ context.Context, t *Ticket) error {
	r.tickets[t.ID] = t
	return nil
}
func (r *mockRepo) GetTicketByID(_ context.Context, id, _ uuid.UUID) (*Ticket, error) {
	if t, ok := r.tickets[id]; ok {
		return t, nil
	}
	return nil, ErrTicketNotFound
}
func (r *mockRepo) ListTickets(_ context.Context, _ uuid.UUID, _ *string, participantID, contactID, orgID *uuid.UUID, _, _ int) ([]*Ticket, int, error) {
	var list []*Ticket
	for _, t := range r.tickets {
		// Mirrors the SQL own-scope filter, including its treatment of external
		// requesters: a nil RequesterID matches nobody (see ListTickets).
		if participantID != nil &&
			(t.RequesterID == nil || *t.RequesterID != *participantID) &&
			(t.AssigneeID == nil || *t.AssigneeID != *participantID) {
			continue
		}
		if contactID != nil && (t.ContactID == nil || *t.ContactID != *contactID) {
			continue
		}
		if orgID != nil && (t.OrgID == nil || *t.OrgID != *orgID) {
			continue
		}
		list = append(list, t)
	}
	return list, len(list), nil
}

func (r *mockRepo) ContactExists(_ context.Context, contactID, tenantID uuid.UUID) (bool, error) {
	t, ok := r.contactTenants[contactID]
	return ok && t == tenantID, nil
}

func (r *mockRepo) CompanyExists(_ context.Context, companyID, tenantID uuid.UUID) (bool, error) {
	t, ok := r.companyTenants[companyID]
	return ok && t == tenantID, nil
}
func (r *mockRepo) GetTicketBySourceMessage(_ context.Context, tenantID, messageID uuid.UUID) (*Ticket, error) {
	for _, t := range r.tickets {
		if t.TenantID == tenantID && t.SourceMessageID != nil && *t.SourceMessageID == messageID {
			return t, nil
		}
	}
	return nil, nil
}
func (r *mockRepo) UpdateTicket(_ context.Context, t *Ticket) error {
	r.tickets[t.ID] = t
	return nil
}
func (r *mockRepo) DeleteTicket(_ context.Context, id, _ uuid.UUID) error {
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
func (r *mockRepo) ListMessagesByTicket(_ context.Context, ticketID, _ uuid.UUID) ([]*TicketMessage, error) {
	return r.messages[ticketID], nil
}
func (r *mockRepo) ReassignMessages(_ context.Context, sourceID, targetID, _ uuid.UUID) error {
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

func (r *mockRepo) SubmitCsatTx(
	_ context.Context,
	tenantID, ticketID uuid.UUID,
	rating int16,
	comment *string,
	submittedAt time.Time,
) error {
	t, ok := r.tickets[ticketID]
	if !ok || t.TenantID != tenantID {
		return ErrTicketNotFound
	}
	r.csatCalls++
	t.CsatRating = &rating
	t.CsatComment = comment
	t.UpdatedAt = submittedAt
	return nil
}

// IssueCsatSurveyTokenTx mirrors the SQL: unknown/foreign ticket and an
// already-rated ticket both yield false without an error.
func (r *mockRepo) IssueCsatSurveyTokenTx(
	_ context.Context,
	tenantID, ticketID uuid.UUID,
	token string,
	sendAfter, expiresAt, _ time.Time,
) (bool, error) {
	t, ok := r.tickets[ticketID]
	if !ok || t.TenantID != tenantID {
		return false, nil
	}
	if t.CsatRating != nil {
		return false, nil
	}
	r.csatTokens[ticketID] = csatTokenRow{token: token, sendAfter: sendAfter, expiresAt: expiresAt}
	return true, nil
}

// GetCsatSurveyByToken mirrors the SQL: an equality match on the token column,
// nothing else, and a pending row resolves regardless of the caller's tenant.
func (r *mockRepo) GetCsatSurveyByToken(_ context.Context, token string) (*CsatSurveyToken, error) {
	for ticketID, row := range r.csatTokens {
		if row.token != token || token == "" {
			continue
		}
		t := r.tickets[ticketID]
		if t == nil {
			continue
		}
		expires := row.expiresAt
		s := &CsatSurveyToken{
			ResponseID: ticketID, TenantID: t.TenantID, TicketID: ticketID,
			ExpiresAt: &expires,
		}
		if t.CsatRating != nil {
			submitted := t.UpdatedAt
			s.SubmittedAt = &submitted
		}
		return s, nil
	}
	return nil, ErrCsatSurveyNotFound
}

// RedeemCsatSurveyTx mirrors the SQL guard: only a still-pending, still-tokened
// row may be redeemed, and redemption consumes the token.
func (r *mockRepo) RedeemCsatSurveyTx(
	_ context.Context,
	tenantID, ticketID, _ uuid.UUID,
	rating int16,
	comment *string,
	submittedAt time.Time,
) (int, error) {
	t, ok := r.tickets[ticketID]
	if !ok || t.TenantID != tenantID {
		return 0, ErrCsatSurveyNotFound
	}
	row, ok := r.csatTokens[ticketID]
	if !ok || row.token == "" || t.CsatRating != nil {
		return 0, ErrCsatSurveyNotFound
	}
	delete(r.csatTokens, ticketID)
	r.csatCalls++
	t.CsatRating = &rating
	t.CsatComment = comment
	t.UpdatedAt = submittedAt
	return t.TicketNumber, nil
}

func (r *mockRepo) CreateQueue(_ context.Context, q *TicketQueue) error {
	r.queues[q.ID] = q
	return nil
}
func (r *mockRepo) GetQueueByID(_ context.Context, id, _ uuid.UUID) (*TicketQueue, error) {
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
func (r *mockRepo) DeleteQueue(_ context.Context, id, _ uuid.UUID) error {
	delete(r.queues, id)
	return nil
}

func (r *mockRepo) CreateCannedResponse(_ context.Context, cr *CannedResponse) error {
	r.cannedResponses[cr.ID] = cr
	return nil
}
func (r *mockRepo) GetCannedResponseByID(_ context.Context, id, _ uuid.UUID) (*CannedResponse, error) {
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
func (r *mockRepo) DeleteCannedResponse(_ context.Context, id, _ uuid.UUID) error {
	delete(r.cannedResponses, id)
	return nil
}

func (r *mockRepo) CreateSLAPolicy(_ context.Context, p *SLAPolicy) error {
	r.slaPolicies[p.ID] = p
	return nil
}
func (r *mockRepo) GetSLAPolicyByID(_ context.Context, id, _ uuid.UUID) (*SLAPolicy, error) {
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

func (r *mockRepo) DeleteSLAPolicy(_ context.Context, id, _ uuid.UUID) error {
	delete(r.slaPolicies, id)
	return nil
}

func (r *mockRepo) CreateKBArticle(_ context.Context, a *KBArticle) error {
	r.kbArticles[a.ID] = a
	return nil
}

func (r *mockRepo) GetKBArticleByID(_ context.Context, id, _ uuid.UUID) (*KBArticle, error) {
	if a, ok := r.kbArticles[id]; ok {
		return a, nil
	}
	return nil, ErrKBArticleNotFound
}

func (r *mockRepo) ListKBArticles(_ context.Context, _ uuid.UUID) ([]*KBArticle, error) {
	if r.repoErr != nil {
		return nil, r.repoErr
	}
	var list []*KBArticle
	for _, a := range r.kbArticles {
		list = append(list, a)
	}
	return list, nil
}

func (r *mockRepo) UpdateKBArticle(_ context.Context, a *KBArticle) error {
	r.kbArticles[a.ID] = a
	return nil
}

func (r *mockRepo) DeleteKBArticle(_ context.Context, id, _ uuid.UUID) error {
	delete(r.kbArticles, id)
	return nil
}

func (r *mockRepo) CreateRoutingRule(_ context.Context, rr *RoutingRule) error {
	r.routingRules[rr.ID] = rr
	return nil
}

func (r *mockRepo) GetRoutingRuleByID(_ context.Context, id, _ uuid.UUID) (*RoutingRule, error) {
	if rr, ok := r.routingRules[id]; ok {
		return rr, nil
	}
	return nil, ErrRoutingRuleNotFound
}

func (r *mockRepo) ListRoutingRules(_ context.Context, _ uuid.UUID) ([]*RoutingRule, error) {
	if r.repoErr != nil {
		return nil, r.repoErr
	}
	var list []*RoutingRule
	for _, rr := range r.routingRules {
		list = append(list, rr)
	}
	return list, nil
}

func (r *mockRepo) UpdateRoutingRule(_ context.Context, rr *RoutingRule) error {
	r.routingRules[rr.ID] = rr
	return nil
}

func (r *mockRepo) DeleteRoutingRule(_ context.Context, id, _ uuid.UUID) error {
	delete(r.routingRules, id)
	return nil
}

func (r *mockRepo) GetHelpdeskStats(_ context.Context, _ uuid.UUID) (*HelpdeskStats, error) {
	if r.repoErr != nil {
		return nil, r.repoErr
	}
	if r.stats != nil {
		return r.stats, nil
	}
	return &HelpdeskStats{}, nil
}

func (r *mockRepo) GetBusinessHours(_ context.Context, tenantID uuid.UUID) (*BusinessHours, error) {
	if r.repoErr != nil {
		return nil, r.repoErr
	}
	if bh, ok := r.businessHours[tenantID]; ok {
		return bh, nil
	}
	return &BusinessHours{
		TenantID: tenantID,
		Schedule: map[string]DaySchedule{},
		Holidays: []HolidayEntry{},
		Timezone: "Europe/Berlin",
	}, nil
}

func (r *mockRepo) UpsertBusinessHours(_ context.Context, bh *BusinessHours) error {
	if r.repoErr != nil {
		return r.repoErr
	}
	stored := *bh
	stored.UpdatedAt = time.Now().UTC()
	r.businessHours[bh.TenantID] = &stored
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

	ticket, err := h.svc.CreateTicket(context.Background(), tenantID, &requesterID, "Login broken", TicketPriorityHigh, nil, nil, "", "", nil, nil, TicketIntake{})
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

	ticket, err := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Help me", "", nil, nil, "", "", nil, nil, TicketIntake{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.Priority != TicketPriorityNormal {
		t.Errorf("expected default priority normal, got %s", ticket.Priority)
	}
}

func TestCreateTicket_EmptySubject(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	if err == nil {
		t.Error("expected error for empty subject")
	}
}

func TestCreateTicket_InvalidPriority(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Subject", "supercritical", nil, nil, "", "", nil, nil, TicketIntake{})
	if !errors.Is(err, ErrInvalidPriority) {
		t.Errorf("expected ErrInvalidPriority, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// CloseTicket / ReopenTicket tests
// ---------------------------------------------------------------------------

func TestCloseTicket_SetsResolvedAt(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Billing issue", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})

	closed, err := h.svc.CloseTicket(context.Background(), ticket.ID, ticket.TenantID)
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

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Feature request", TicketPriorityLow, nil, nil, "", "", nil, nil, TicketIntake{})
	h.svc.CloseTicket(context.Background(), ticket.ID, ticket.TenantID)

	reopened, err := h.svc.ReopenTicket(context.Background(), ticket.ID, ticket.TenantID)
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

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Still open", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})

	_, err := h.svc.ReopenTicket(context.Background(), ticket.ID, ticket.TenantID)
	if err == nil {
		t.Error("expected error when reopening non-closed ticket")
	}
}

// ---------------------------------------------------------------------------
// AddMessage tests
// ---------------------------------------------------------------------------

func TestAddMessage_Success(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Can't login", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	authorID := uuid.New()

	msg, err := h.svc.AddMessage(context.Background(), ticket.ID, ticket.TenantID, authorID, "Have you tried turning it off and on?", false, nil)
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

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Password reset", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	if ticket.FirstResponseAt != nil {
		t.Fatal("ticket should have no first_response_at initially")
	}

	h.svc.AddMessage(context.Background(), ticket.ID, ticket.TenantID, uuid.New(), "We will help you.", false, nil)

	stored := h.repo.tickets[ticket.ID]
	if stored.FirstResponseAt == nil {
		t.Error("expected first_response_at to be set after first public message")
	}
}

func TestAddMessage_InternalDoesNotSetFirstResponseAt(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Internal note test", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})

	h.svc.AddMessage(context.Background(), ticket.ID, ticket.TenantID, uuid.New(), "Internal note only.", true, nil)

	stored := h.repo.tickets[ticket.ID]
	if stored.FirstResponseAt != nil {
		t.Error("internal message must not set first_response_at")
	}
}

func TestAddMessage_EmptyBodyFails(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Subject", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})

	_, err := h.svc.AddMessage(context.Background(), ticket.ID, ticket.TenantID, uuid.New(), "", false, nil)
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
	ticket, _ := h.svc.CreateTicket(context.Background(), tenantID, uuidPtr(uuid.New()), "Slow system", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	policy, _ := h.svc.CreateSLAPolicy(context.Background(), tenantID, "Standard", 60, 480, nil)

	updated, err := h.svc.ApplySLAPolicy(context.Background(), ticket.ID, tenantID, policy.ID)
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

	ticket, _ := h.svc.CreateTicket(context.Background(), tenantID, uuidPtr(uuid.New()), "Crash on export", TicketPriorityHigh, nil, nil, "", "", nil, nil, TicketIntake{})
	policy, _ := h.svc.CreateSLAPolicy(context.Background(), tenantID, "Premium", 30, 240, nil)

	// Simulate first response already set.
	now := time.Now().UTC()
	ticket.FirstResponseAt = &now
	h.repo.tickets[ticket.ID] = ticket

	updated, err := h.svc.ApplySLAPolicy(context.Background(), ticket.ID, tenantID, policy.ID)
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

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Test", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})

	_, err := h.svc.ApplySLAPolicy(context.Background(), ticket.ID, ticket.TenantID, uuid.New())
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

	source, _ := h.svc.CreateTicket(context.Background(), tenantID, uuidPtr(uuid.New()), "Duplicate issue", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	target, _ := h.svc.CreateTicket(context.Background(), tenantID, uuidPtr(uuid.New()), "Original issue", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})

	// Add a message to source.
	h.svc.AddMessage(context.Background(), source.ID, tenantID, uuid.New(), "From source ticket.", false, nil)

	err := h.svc.MergeTickets(context.Background(), tenantID, source.ID, target.ID)
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

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Self merge", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})

	err := h.svc.MergeTickets(context.Background(), ticket.TenantID, ticket.ID, ticket.ID)
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

	source, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Already merged", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	target, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Target", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})

	// Pre-mark source as merged.
	source.Status = TicketStatusMerged
	h.repo.tickets[source.ID] = source

	err := h.svc.MergeTickets(context.Background(), source.TenantID, source.ID, target.ID)
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

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Assign me", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	agentID := uuid.New()

	updated, err := h.svc.AssignTicket(context.Background(), ticket.ID, ticket.TenantID, agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.AssigneeID == nil || *updated.AssigneeID != agentID {
		t.Errorf("expected assignee_id %s, got %v", agentID, updated.AssigneeID)
	}
}

func TestAssignTicket_TicketNotFound(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.AssignTicket(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrTicketNotFound) {
		t.Errorf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestUpdateTicket_PriorityChange(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Priority test", TicketPriorityLow, nil, nil, "", "", nil, nil, TicketIntake{})
	newPriority := TicketPriorityUrgent

	updated, err := h.svc.UpdateTicket(context.Background(), ticket.ID, ticket.TenantID, nil, nil, &newPriority, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Priority != TicketPriorityUrgent {
		t.Errorf("expected priority urgent, got %s", updated.Priority)
	}
}

func TestUpdateTicket_InvalidPriority(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Priority test", TicketPriorityLow, nil, nil, "", "", nil, nil, TicketIntake{})
	bad := "supercritical"

	_, err := h.svc.UpdateTicket(context.Background(), ticket.ID, ticket.TenantID, nil, nil, &bad, nil, nil, nil, nil, nil)
	if !errors.Is(err, ErrInvalidPriority) {
		t.Errorf("expected ErrInvalidPriority, got %v", err)
	}
}

func TestUpdateTicket_StatusChange(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Status test", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	newStatus := TicketStatusPending

	updated, err := h.svc.UpdateTicket(context.Background(), ticket.ID, ticket.TenantID, nil, &newStatus, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != TicketStatusPending {
		t.Errorf("expected status pending, got %s", updated.Status)
	}
}

// TestUpdateTicket_RejectsClosedAndMergedStatus proves the generic update
// path cannot be used to sneak a ticket into "closed" or "merged" -- both
// have dedicated endpoints (/close, /merge) with side effects (resolved_at +
// CSAT token issuance; message reassignment) that setting status here would
// silently skip.
func TestUpdateTicket_RejectsClosedAndMergedStatus(t *testing.T) {
	h := newTestHarness()

	for _, blocked := range []string{TicketStatusClosed, TicketStatusMerged} {
		ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Status test", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
		status := blocked

		_, err := h.svc.UpdateTicket(context.Background(), ticket.ID, ticket.TenantID, nil, &status, nil, nil, nil, nil, nil, nil)
		if !errors.Is(err, ErrInvalidStatus) {
			t.Errorf("status=%s: expected ErrInvalidStatus, got %v", blocked, err)
		}
	}
}

func TestUpdateTicket_InvalidStatus(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Status test", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	bad := "vaporized"

	_, err := h.svc.UpdateTicket(context.Background(), ticket.ID, ticket.TenantID, nil, &bad, nil, nil, nil, nil, nil, nil)
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

// TestUpdateTicket_CustomFieldsMergeAcrossUpdates proves the patch semantics
// HelpdeskPage.tsx relies on: handleCustomFieldCommit sends one changed key
// per call, and a second call must not erase the first one's key.
func TestUpdateTicket_CustomFieldsMergeAcrossUpdates(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Custom fields test", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})

	updated, err := h.svc.UpdateTicket(context.Background(), ticket.ID, ticket.TenantID, nil, nil, nil, nil, nil, nil, nil,
		map[string]any{"standort": "Bern"})
	if err != nil {
		t.Fatalf("first update: %v", err)
	}
	if updated.CustomFields["standort"] != "Bern" {
		t.Fatalf("custom_fields[standort] = %v, want Bern", updated.CustomFields["standort"])
	}

	updated, err = h.svc.UpdateTicket(context.Background(), ticket.ID, ticket.TenantID, nil, nil, nil, nil, nil, nil, nil,
		map[string]any{"garantie": true})
	if err != nil {
		t.Fatalf("second update: %v", err)
	}
	if len(updated.CustomFields) != 2 {
		t.Fatalf("custom_fields = %v, want 2 entries after merge", updated.CustomFields)
	}
	if updated.CustomFields["standort"] != "Bern" {
		t.Errorf("custom_fields[standort] = %v, want Bern (must survive the second update)", updated.CustomFields["standort"])
	}
	if updated.CustomFields["garantie"] != true {
		t.Errorf("custom_fields[garantie] = %v, want true", updated.CustomFields["garantie"])
	}
}

func TestUpdateTicket_CustomFieldsRejectsNested(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "Custom fields test", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})

	_, err := h.svc.UpdateTicket(context.Background(), ticket.ID, ticket.TenantID, nil, nil, nil, nil, nil, nil, nil,
		map[string]any{"nested": map[string]any{"a": 1}})
	if !errors.Is(err, ErrInvalidCustomFields) {
		t.Errorf("expected ErrInvalidCustomFields, got %v", err)
	}
}

func TestListMessages_AfterAdd(t *testing.T) {
	h := newTestHarness()

	ticket, _ := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()), "List msgs", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	h.svc.AddMessage(context.Background(), ticket.ID, ticket.TenantID, uuid.New(), "First message", false, nil)
	h.svc.AddMessage(context.Background(), ticket.ID, ticket.TenantID, uuid.New(), "Second message (internal)", true, nil)

	msgs, err := h.svc.ListMessages(context.Background(), ticket.ID, ticket.TenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
}

func TestGetTicket_NotFound(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetTicket(context.Background(), uuid.New(), uuid.New())
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

	err := h.svc.DeleteQueue(context.Background(), q.ID, tenantID)
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

	updated, err := h.svc.UpdateQueue(context.Background(), q.ID, tenantID, &newName, nil, nil)
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
		RequesterID: &requesterID,
		Subject:     "Login button not working on mobile",
		Status:      TicketStatusOpen,
	}
	// A different ticket that would match the prefix.
	other := &Ticket{
		ID:          uuid.New(),
		TenantID:    tenantID,
		RequesterID: &requesterID,
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

// ---------------------------------------------------------------------------
// Knowledge-base article tests
// ---------------------------------------------------------------------------

func TestCreateKBArticle_EmptyTitleFails(t *testing.T) {
	h := newTestHarness()
	_, err := h.svc.CreateKBArticle(context.Background(), uuid.New(), uuid.New(), "", "body", "cat", "")
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestCreateKBArticle_DefaultsStatusToDraft(t *testing.T) {
	h := newTestHarness()
	authorID := uuid.New()

	a, err := h.svc.CreateKBArticle(context.Background(), uuid.New(), authorID, "How to reset a password", "content", "auth", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Status != string(KBArticleStatusDraft) {
		t.Errorf("expected default status draft, got %q", a.Status)
	}
	if a.AuthorID != authorID {
		t.Errorf("expected author_id %v, got %v", authorID, a.AuthorID)
	}
}

func TestGetKBArticle_NotFound(t *testing.T) {
	h := newTestHarness()
	_, err := h.svc.GetKBArticle(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrKBArticleNotFound) {
		t.Errorf("expected ErrKBArticleNotFound, got %v", err)
	}
}

func TestUpdateKBArticle_NotFound(t *testing.T) {
	h := newTestHarness()
	title := "New Title"
	_, err := h.svc.UpdateKBArticle(context.Background(), uuid.New(), uuid.New(), &title, nil, nil, nil)
	if !errors.Is(err, ErrKBArticleNotFound) {
		t.Errorf("expected ErrKBArticleNotFound, got %v", err)
	}
}

func TestUpdateKBArticle_EmptyTitleFails(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	a, err := h.svc.CreateKBArticle(context.Background(), tenantID, uuid.New(), "Original", "c", "cat", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	empty := ""
	_, err = h.svc.UpdateKBArticle(context.Background(), a.ID, tenantID, &empty, nil, nil, nil)
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestUpdateKBArticle_PatchesFields(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	a, err := h.svc.CreateKBArticle(context.Background(), tenantID, uuid.New(), "Original", "c", "cat", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newTitle := "Updated Title"
	published := string(KBArticleStatusPublished)
	updated, err := h.svc.UpdateKBArticle(context.Background(), a.ID, tenantID, &newTitle, nil, nil, &published)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Title != newTitle || updated.Status != published {
		t.Errorf("patch did not land: title=%q status=%q", updated.Title, updated.Status)
	}
}

func TestDeleteKBArticle_Success(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	a, _ := h.svc.CreateKBArticle(context.Background(), tenantID, uuid.New(), "Doomed", "c", "cat", "")
	if err := h.svc.DeleteKBArticle(context.Background(), a.ID, tenantID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := h.repo.kbArticles[a.ID]; ok {
		t.Error("expected article to be deleted")
	}
}

func TestListKBArticles_AfterCreate(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	h.svc.CreateKBArticle(context.Background(), tenantID, uuid.New(), "Article A", "c", "cat", "")
	h.svc.CreateKBArticle(context.Background(), tenantID, uuid.New(), "Article B", "c", "cat", "")

	list, err := h.svc.ListKBArticles(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 articles, got %d", len(list))
	}
}

func TestListKBArticles_RepoErrorIsWrapped(t *testing.T) {
	h := newTestHarness()
	h.repo.repoErr = errors.New("db unavailable")

	_, err := h.svc.ListKBArticles(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error to propagate from repository")
	}
}

// ---------------------------------------------------------------------------
// Routing rule tests
// ---------------------------------------------------------------------------

func TestCreateRoutingRule_EmptyNameFails(t *testing.T) {
	h := newTestHarness()
	_, err := h.svc.CreateRoutingRule(context.Background(), uuid.New(), "", nil, nil, 0, true)
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateRoutingRule_NilConditionsDefaultToEmptyMap(t *testing.T) {
	h := newTestHarness()
	rr, err := h.svc.CreateRoutingRule(context.Background(), uuid.New(), "Route to Sales", nil, nil, 1, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rr.Conditions == nil {
		t.Error("expected nil conditions to default to an empty map")
	}
}

func TestUpdateRoutingRule_NotFound(t *testing.T) {
	h := newTestHarness()
	name := "Renamed"
	_, err := h.svc.UpdateRoutingRule(context.Background(), uuid.New(), uuid.New(), &name, nil, nil, nil, nil)
	if !errors.Is(err, ErrRoutingRuleNotFound) {
		t.Errorf("expected ErrRoutingRuleNotFound, got %v", err)
	}
}

func TestUpdateRoutingRule_EmptyNameFails(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	rr, err := h.svc.CreateRoutingRule(context.Background(), tenantID, "Original", nil, nil, 0, true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	empty := ""
	_, err = h.svc.UpdateRoutingRule(context.Background(), rr.ID, tenantID, &empty, nil, nil, nil, nil)
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestUpdateRoutingRule_PatchesFields(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	rr, err := h.svc.CreateRoutingRule(context.Background(), tenantID, "Original", nil, nil, 0, true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newName := "Renamed"
	newPriority := 5
	disabled := false
	updated, err := h.svc.UpdateRoutingRule(context.Background(), rr.ID, tenantID, &newName, nil, nil, &newPriority, &disabled)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != newName || updated.Priority != newPriority || updated.Enabled {
		t.Errorf("patch did not land: name=%q priority=%d enabled=%v", updated.Name, updated.Priority, updated.Enabled)
	}
}

func TestDeleteRoutingRule_Success(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	rr, _ := h.svc.CreateRoutingRule(context.Background(), tenantID, "Doomed", nil, nil, 0, true)
	if err := h.svc.DeleteRoutingRule(context.Background(), rr.ID, tenantID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := h.repo.routingRules[rr.ID]; ok {
		t.Error("expected rule to be deleted")
	}
}

func TestListRoutingRules_AfterCreate(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	h.svc.CreateRoutingRule(context.Background(), tenantID, "Rule A", nil, nil, 0, true)
	h.svc.CreateRoutingRule(context.Background(), tenantID, "Rule B", nil, nil, 1, false)

	list, err := h.svc.ListRoutingRules(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 rules, got %d", len(list))
	}
}

func TestListRoutingRules_RepoErrorIsWrapped(t *testing.T) {
	h := newTestHarness()
	h.repo.repoErr = errors.New("db unavailable")

	_, err := h.svc.ListRoutingRules(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error to propagate from repository")
	}
}

// ---------------------------------------------------------------------------
// Stats + business hours tests
// ---------------------------------------------------------------------------

func TestGetHelpdeskStats_Success(t *testing.T) {
	h := newTestHarness()
	h.repo.stats = &HelpdeskStats{OpenTickets: 3, AvgResponseTime: "42 min"}

	stats, err := h.svc.GetHelpdeskStats(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.OpenTickets != 3 {
		t.Errorf("expected 3 open tickets, got %d", stats.OpenTickets)
	}
}

func TestGetHelpdeskStats_RepoErrorIsWrapped(t *testing.T) {
	h := newTestHarness()
	h.repo.repoErr = errors.New("db unavailable")

	_, err := h.svc.GetHelpdeskStats(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error to propagate from repository")
	}
}

func TestGetBusinessHours_DefaultsWhenNoneStored(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	bh, err := h.svc.GetBusinessHours(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bh.Timezone != "Europe/Berlin" || bh.Schedule == nil || bh.Holidays == nil {
		t.Errorf("expected default business hours, got %+v", bh)
	}
}

func TestGetBusinessHours_RepoErrorIsWrapped(t *testing.T) {
	h := newTestHarness()
	h.repo.repoErr = errors.New("db unavailable")

	_, err := h.svc.GetBusinessHours(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error to propagate from repository")
	}
}

func TestUpsertBusinessHours_DefaultsTimezoneAndCollectionsThenRereads(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	updated, err := h.svc.UpsertBusinessHours(context.Background(), &BusinessHours{TenantID: tenantID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Timezone != "Europe/Berlin" {
		t.Errorf("expected default timezone, got %q", updated.Timezone)
	}
	if updated.Schedule == nil || updated.Holidays == nil {
		t.Errorf("expected non-nil schedule/holidays, got %+v", updated)
	}
	if updated.UpdatedAt.IsZero() {
		t.Error("expected the re-read to carry a server-set updated_at")
	}
}

func TestUpsertBusinessHours_RepoErrorIsWrapped(t *testing.T) {
	h := newTestHarness()
	h.repo.repoErr = errors.New("db unavailable")

	_, err := h.svc.UpsertBusinessHours(context.Background(), &BusinessHours{TenantID: uuid.New()})
	if err == nil {
		t.Fatal("expected error to propagate from repository")
	}
}

// ---------------------------------------------------------------------------
// Canned response Update/Delete/List tests
// ---------------------------------------------------------------------------

func TestUpdateCannedResponse_NotFound(t *testing.T) {
	h := newTestHarness()
	name := "New Name"
	_, err := h.svc.UpdateCannedResponse(context.Background(), uuid.New(), uuid.New(), &name, nil)
	if !errors.Is(err, ErrCannedResponseNotFound) {
		t.Errorf("expected ErrCannedResponseNotFound, got %v", err)
	}
}

func TestUpdateCannedResponse_EmptyNameOrBodyFails(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	cr, err := h.svc.CreateCannedResponse(context.Background(), tenantID, "Greeting", "Hello there")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	empty := ""
	if _, err := h.svc.UpdateCannedResponse(context.Background(), cr.ID, tenantID, &empty, nil); err == nil {
		t.Error("expected error for empty name")
	}
	if _, err := h.svc.UpdateCannedResponse(context.Background(), cr.ID, tenantID, nil, &empty); err == nil {
		t.Error("expected error for empty body")
	}
}

func TestUpdateCannedResponse_PatchesFields(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	cr, err := h.svc.CreateCannedResponse(context.Background(), tenantID, "Greeting", "Hello there")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newBody := "Hello and welcome"
	updated, err := h.svc.UpdateCannedResponse(context.Background(), cr.ID, tenantID, nil, &newBody)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Body != newBody || updated.Name != "Greeting" {
		t.Errorf("patch did not land: name=%q body=%q", updated.Name, updated.Body)
	}
}

func TestDeleteCannedResponse_Success(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	cr, _ := h.svc.CreateCannedResponse(context.Background(), tenantID, "Doomed", "body")
	if err := h.svc.DeleteCannedResponse(context.Background(), cr.ID, tenantID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := h.repo.cannedResponses[cr.ID]; ok {
		t.Error("expected canned response to be deleted")
	}
}

func TestListCannedResponses_AfterCreate(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	h.svc.CreateCannedResponse(context.Background(), tenantID, "Greeting", "Hello")
	h.svc.CreateCannedResponse(context.Background(), tenantID, "Closing", "Bye")

	list, err := h.svc.ListCannedResponses(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 canned responses, got %d", len(list))
	}
}

// ---------------------------------------------------------------------------
// SLA policy Update/Delete/Status tests
// ---------------------------------------------------------------------------

func TestUpdateSLAPolicy_NotFound(t *testing.T) {
	h := newTestHarness()
	name := "New Name"
	_, err := h.svc.UpdateSLAPolicy(context.Background(), uuid.New(), uuid.New(), &name, nil, nil, nil)
	if !errors.Is(err, ErrSLAPolicyNotFound) {
		t.Errorf("expected ErrSLAPolicyNotFound, got %v", err)
	}
}

func TestUpdateSLAPolicy_InvalidMinsFails(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	p, err := h.svc.CreateSLAPolicy(context.Background(), tenantID, "Standard", 30, 120, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	zero := 0
	if _, err := h.svc.UpdateSLAPolicy(context.Background(), p.ID, tenantID, nil, &zero, nil, nil); err == nil {
		t.Error("expected error for first_response_mins=0")
	}
	negative := -5
	if _, err := h.svc.UpdateSLAPolicy(context.Background(), p.ID, tenantID, nil, nil, &negative, nil); err == nil {
		t.Error("expected error for negative resolution_mins")
	}
	empty := ""
	if _, err := h.svc.UpdateSLAPolicy(context.Background(), p.ID, tenantID, &empty, nil, nil, nil); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestUpdateSLAPolicy_PatchesFields(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	p, err := h.svc.CreateSLAPolicy(context.Background(), tenantID, "Standard", 30, 120, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newName := "Premium"
	newFirst := 15
	updated, err := h.svc.UpdateSLAPolicy(context.Background(), p.ID, tenantID, &newName, &newFirst, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != newName || updated.FirstResponseMins != newFirst || updated.ResolutionMins != 120 {
		t.Errorf("patch did not land correctly: %+v", updated)
	}
}

func TestDeleteSLAPolicy_Success(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	p, _ := h.svc.CreateSLAPolicy(context.Background(), tenantID, "Doomed", 30, 120, nil)
	if err := h.svc.DeleteSLAPolicy(context.Background(), p.ID, tenantID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := h.repo.slaPolicies[p.ID]; ok {
		t.Error("expected policy to be deleted")
	}
}

func TestGetSLAStatus_WithoutPolicyIsOnTrack(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	ticket, err := h.svc.CreateTicket(context.Background(), tenantID, uuidPtr(uuid.New()), "Test", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	status, err := h.svc.GetSLAStatus(context.Background(), ticket.ID, tenantID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != SLAStatusOnTrack {
		t.Errorf("expected on_track without a policy applied, got %q", status)
	}
}

func TestGetSLAStatus_BreachedAfterDueAt(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	ticket, err := h.svc.CreateTicket(context.Background(), tenantID, uuidPtr(uuid.New()), "Test", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	policy, err := h.svc.CreateSLAPolicy(context.Background(), tenantID, "Fast", 1, 5, nil)
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	past := time.Now().UTC().Add(-time.Hour)
	ticket.DueAt = &past
	h.repo.tickets[ticket.ID] = ticket

	status, err := h.svc.GetSLAStatus(context.Background(), ticket.ID, tenantID, &policy.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != SLAStatusBreached {
		t.Errorf("expected breached, got %q", status)
	}
}

func TestGetSLAStatus_TicketNotFound(t *testing.T) {
	h := newTestHarness()
	_, err := h.svc.GetSLAStatus(context.Background(), uuid.New(), uuid.New(), nil)
	if !errors.Is(err, ErrTicketNotFound) {
		t.Errorf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestGetSLAStatus_PolicyNotFound(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()

	ticket, err := h.svc.CreateTicket(context.Background(), tenantID, uuidPtr(uuid.New()), "Test", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	missingPolicy := uuid.New()
	_, err = h.svc.GetSLAStatus(context.Background(), ticket.ID, tenantID, &missingPolicy)
	if !errors.Is(err, ErrSLAPolicyNotFound) {
		t.Errorf("expected ErrSLAPolicyNotFound, got %v", err)
	}
}
