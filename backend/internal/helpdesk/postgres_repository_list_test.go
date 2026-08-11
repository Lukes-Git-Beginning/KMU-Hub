package helpdesk

// Covers the postgres_repository.go List*/Find*/BusinessHours methods that
// carried 0% coverage: FindOpenTicketsByRequester, ListQueues,
// ListCannedResponses, ListSLAPolicies, ListKBArticles, ListRoutingRules,
// GetBusinessHours and UpsertBusinessHours. Run against the real schema
// (Muster tenant_write_test.go), not a mock -- the scan*FromRows helpers
// backing every List* method are exercised only through a real query.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestFindOpenTicketsByRequester_FiltersStatusPrefixAndRequester(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk FindOpenTickets Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	requesterID := uuid.New()
	otherRequesterID := uuid.New()
	now := time.Now().UTC()

	mk := func(subject, status string, requester uuid.UUID) *Ticket {
		return &Ticket{
			ID: uuid.New(), TenantID: tenantID, Subject: subject, Status: status,
			Priority: TicketPriorityNormal, RequesterID: &requester,
			CreatedAt: now, UpdatedAt: now,
		}
	}

	matching := mk("Support Anfrage Login", TicketStatusOpen, requesterID)
	closedButMatchingPrefix := mk("Support Anfrage Alt", TicketStatusClosed, requesterID)
	wrongPrefix := mk("Andere Anfrage", TicketStatusOpen, requesterID)
	wrongRequester := mk("Support Anfrage Fremd", TicketStatusOpen, otherRequesterID)

	for _, tk := range []*Ticket{matching, closedButMatchingPrefix, wrongPrefix, wrongRequester} {
		if err := repo.CreateTicket(ctx, tk); err != nil {
			t.Fatalf("CreateTicket(%s): %v", tk.Subject, err)
		}
		defer testutil.CleanupRow(t, pool, "tickets", tk.ID)
	}

	found, err := repo.FindOpenTicketsByRequester(ctx, tenantID, requesterID, "Support")
	if err != nil {
		t.Fatalf("FindOpenTicketsByRequester: %v", err)
	}
	if len(found) != 1 || found[0].ID != matching.ID {
		ids := make([]uuid.UUID, len(found))
		for i, tk := range found {
			ids[i] = tk.ID
		}
		t.Fatalf("expected exactly [%v], got %v", matching.ID, ids)
	}

	none, err := repo.FindOpenTicketsByRequester(ctx, tenantID, requesterID, "Rechnung")
	if err != nil {
		t.Fatalf("FindOpenTicketsByRequester (no match): %v", err)
	}
	if len(none) != 0 {
		t.Errorf("expected no matches for an unrelated prefix, got %d", len(none))
	}
}

func TestListQueues_ScopedToTenantAndOrderedByName(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk ListQueues Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk ListQueues Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	qB := &TicketQueue{ID: uuid.New(), TenantID: tenantOwn, Name: "B Queue", CreatedAt: now, UpdatedAt: now}
	qA := &TicketQueue{ID: uuid.New(), TenantID: tenantOwn, Name: "A Queue", CreatedAt: now, UpdatedAt: now}
	qOther := &TicketQueue{ID: uuid.New(), TenantID: tenantOther, Name: "Other Tenant Queue", CreatedAt: now, UpdatedAt: now}

	if err := repo.CreateQueue(ctxOwn, qB); err != nil {
		t.Fatalf("CreateQueue(B): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "ticket_queues", qB.ID)
	if err := repo.CreateQueue(ctxOwn, qA); err != nil {
		t.Fatalf("CreateQueue(A): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "ticket_queues", qA.ID)
	if err := repo.CreateQueue(ctxOther, qOther); err != nil {
		t.Fatalf("CreateQueue(other tenant): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "ticket_queues", qOther.ID)

	list, err := repo.ListQueues(ctxOwn, tenantOwn)
	if err != nil {
		t.Fatalf("ListQueues: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 queues for tenantOwn, got %d", len(list))
	}
	if list[0].Name != "A Queue" || list[1].Name != "B Queue" {
		t.Errorf("expected alphabetical order [A Queue, B Queue], got [%s, %s]", list[0].Name, list[1].Name)
	}
}

func TestListCannedResponses_ScopedToTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk ListCanned Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk ListCanned Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	own := &CannedResponse{ID: uuid.New(), TenantID: tenantOwn, Name: "Greeting", Body: "Hello", CreatedAt: now, UpdatedAt: now}
	foreign := &CannedResponse{ID: uuid.New(), TenantID: tenantOther, Name: "Foreign", Body: "Hi", CreatedAt: now, UpdatedAt: now}

	if err := repo.CreateCannedResponse(ctxOwn, own); err != nil {
		t.Fatalf("CreateCannedResponse(own): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "canned_responses", own.ID)
	if err := repo.CreateCannedResponse(ctxOther, foreign); err != nil {
		t.Fatalf("CreateCannedResponse(foreign): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "canned_responses", foreign.ID)

	list, err := repo.ListCannedResponses(ctxOwn, tenantOwn)
	if err != nil {
		t.Fatalf("ListCannedResponses: %v", err)
	}
	if len(list) != 1 || list[0].ID != own.ID || list[0].Body != "Hello" {
		t.Fatalf("expected exactly the own-tenant response, got %+v", list)
	}
}

func TestListSLAPolicies_ScopedToTenantAndBusinessHoursRoundtrip(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk ListSLA Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk ListSLA Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	own := &SLAPolicy{
		ID: uuid.New(), TenantID: tenantOwn, Name: "Standard",
		FirstResponseMins: 30, ResolutionMins: 480,
		BusinessHours: map[string]any{"monday": "08:00-17:00"},
		CreatedAt:     now, UpdatedAt: now,
	}
	foreign := &SLAPolicy{
		ID: uuid.New(), TenantID: tenantOther, Name: "Foreign",
		FirstResponseMins: 15, ResolutionMins: 120,
		CreatedAt: now, UpdatedAt: now,
	}

	if err := repo.CreateSLAPolicy(ctxOwn, own); err != nil {
		t.Fatalf("CreateSLAPolicy(own): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "sla_policies", own.ID)
	if err := repo.CreateSLAPolicy(ctxOther, foreign); err != nil {
		t.Fatalf("CreateSLAPolicy(foreign): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "sla_policies", foreign.ID)

	list, err := repo.ListSLAPolicies(ctxOwn, tenantOwn)
	if err != nil {
		t.Fatalf("ListSLAPolicies: %v", err)
	}
	if len(list) != 1 || list[0].ID != own.ID {
		t.Fatalf("expected exactly the own-tenant policy, got %+v", list)
	}
	if list[0].BusinessHours["monday"] != "08:00-17:00" {
		t.Errorf("expected business_hours JSONB to round-trip, got %+v", list[0].BusinessHours)
	}
}

func TestListKBArticles_ScopedToTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk ListKB Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk ListKB Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	own := &KBArticle{
		ID: uuid.New(), TenantID: tenantOwn, Title: "Password Reset", Content: "steps",
		Category: "auth", Status: string(KBArticleStatusPublished), AuthorID: uuid.New(),
		CreatedAt: now, UpdatedAt: now,
	}
	foreign := &KBArticle{
		ID: uuid.New(), TenantID: tenantOther, Title: "Foreign Article", Content: "x",
		Status: string(KBArticleStatusDraft), AuthorID: uuid.New(),
		CreatedAt: now, UpdatedAt: now,
	}

	if err := repo.CreateKBArticle(ctxOwn, own); err != nil {
		t.Fatalf("CreateKBArticle(own): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "helpdesk_kb_articles", own.ID)
	if err := repo.CreateKBArticle(ctxOther, foreign); err != nil {
		t.Fatalf("CreateKBArticle(foreign): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "helpdesk_kb_articles", foreign.ID)

	list, err := repo.ListKBArticles(ctxOwn, tenantOwn)
	if err != nil {
		t.Fatalf("ListKBArticles: %v", err)
	}
	if len(list) != 1 || list[0].ID != own.ID || list[0].Title != "Password Reset" {
		t.Fatalf("expected exactly the own-tenant article, got %+v", list)
	}
}

func TestListRoutingRules_ScopedToTenantAndOrderedByPriority(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk ListRouting Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk ListRouting Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	low := &RoutingRule{
		ID: uuid.New(), TenantID: tenantOwn, Name: "Low Priority", Conditions: map[string]any{"category": "billing"},
		Priority: 5, Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	high := &RoutingRule{
		ID: uuid.New(), TenantID: tenantOwn, Name: "High Priority", Conditions: map[string]any{"category": "outage"},
		Priority: 1, Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	foreign := &RoutingRule{
		ID: uuid.New(), TenantID: tenantOther, Name: "Foreign Rule", Conditions: map[string]any{},
		Priority: 0, Enabled: true, CreatedAt: now, UpdatedAt: now,
	}

	if err := repo.CreateRoutingRule(ctxOwn, low); err != nil {
		t.Fatalf("CreateRoutingRule(low): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "helpdesk_routing_rules", low.ID)
	if err := repo.CreateRoutingRule(ctxOwn, high); err != nil {
		t.Fatalf("CreateRoutingRule(high): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "helpdesk_routing_rules", high.ID)
	if err := repo.CreateRoutingRule(ctxOther, foreign); err != nil {
		t.Fatalf("CreateRoutingRule(foreign): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "helpdesk_routing_rules", foreign.ID)

	list, err := repo.ListRoutingRules(ctxOwn, tenantOwn)
	if err != nil {
		t.Fatalf("ListRoutingRules: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 rules for tenantOwn, got %d", len(list))
	}
	if list[0].ID != high.ID || list[1].ID != low.ID {
		t.Errorf("expected priority-ascending order [high, low], got [%s, %s]", list[0].Name, list[1].Name)
	}
	if list[0].Conditions["category"] != "outage" {
		t.Errorf("expected conditions JSONB to round-trip, got %+v", list[0].Conditions)
	}
}

func TestBusinessHours_DefaultThenUpsertRoundtrips(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk BusinessHours Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	defaults, err := repo.GetBusinessHours(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetBusinessHours (no row yet): %v", err)
	}
	if defaults.Timezone != "Europe/Berlin" || len(defaults.Schedule) != 0 || len(defaults.Holidays) != 0 {
		t.Fatalf("expected empty defaults before any upsert, got %+v", defaults)
	}

	bh := &BusinessHours{
		TenantID: tenantID,
		Schedule: map[string]DaySchedule{
			"monday": {Active: true, Start: "08:00", End: "17:00"},
		},
		Holidays: []HolidayEntry{{ID: "h-1", Date: "2026-12-25", Name: "Weihnachten"}},
		Timezone: "Europe/Vienna",
	}
	if err := repo.UpsertBusinessHours(ctx, bh); err != nil {
		t.Fatalf("UpsertBusinessHours (insert): %v", err)
	}
	defer func() {
		sysCtx := testutil.WithSystemCtx(context.Background())
		if _, err := pool.Exec(sysCtx, `DELETE FROM helpdesk_business_hours WHERE tenant_id = $1`, tenantID); err != nil {
			t.Logf("cleanup helpdesk_business_hours tenant_id=%s: %v", tenantID, err)
		}
	}()

	got, err := repo.GetBusinessHours(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetBusinessHours (after insert): %v", err)
	}
	if got.Timezone != "Europe/Vienna" || len(got.Holidays) != 1 || got.Holidays[0].Name != "Weihnachten" {
		t.Fatalf("insert did not round-trip: %+v", got)
	}
	if sched, ok := got.Schedule["monday"]; !ok || sched.Start != "08:00" {
		t.Errorf("expected monday schedule to round-trip, got %+v", got.Schedule)
	}

	// Second call exercises the ON CONFLICT DO UPDATE branch.
	bh.Timezone = "Europe/Berlin"
	bh.Holidays = []HolidayEntry{}
	if err := repo.UpsertBusinessHours(ctx, bh); err != nil {
		t.Fatalf("UpsertBusinessHours (update): %v", err)
	}
	updated, err := repo.GetBusinessHours(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetBusinessHours (after update): %v", err)
	}
	if updated.Timezone != "Europe/Berlin" || len(updated.Holidays) != 0 {
		t.Fatalf("update did not overwrite the existing row: %+v", updated)
	}
}
