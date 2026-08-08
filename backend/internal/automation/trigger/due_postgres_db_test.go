package trigger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// These exercise PostgresDueResolver against a real database. The unit tests
// in poller_due_test.go stub the resolver away entirely, so without these the
// two SQL statements this feature rests on -- column names, parameter type
// inference, scan target types -- would never have been executed at all.
//
// Tenant scoping is the point of the isolation cases: the poller runs under
// system context, so RLS is open and the explicit WHERE tenant_id = $1 is the
// only thing standing between tenant A's automation and tenant B's invoices.

func newDueResolverPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	return pool
}

// seedOverdueInvoice inserts a sent invoice whose due date is daysOverdue days
// in the past and returns its id.
func seedOverdueInvoice(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, daysOverdue int) uuid.UUID {
	t.Helper()
	dueDate := time.Now().AddDate(0, 0, -daysOverdue)
	id := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"tenant_id":      tenantID,
		"invoice_number": "TEST-" + uuid.NewString()[:8],
		"status":         "sent",
		"gross_total":    1234.50,
		"invoice_date":   dueDate.AddDate(0, 0, -14),
		"due_date":       dueDate,
		"created_by":     uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", id) })
	return id
}

func TestPostgresDueResolver_OverdueInvoices_ScopesToTenantAndCarriesFields(t *testing.T) {
	pool := newDueResolverPool(t)

	tenantA := uuid.New()
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "due-resolver-a")
	testutil.EnsureTenant(t, pool, tenantB, "due-resolver-b")
	t.Cleanup(func() {
		testutil.CleanupRow(t, pool, "tenants", tenantA)
		testutil.CleanupRow(t, pool, "tenants", tenantB)
	})

	wantID := seedOverdueInvoice(t, pool, tenantA, 20)
	otherTenantID := seedOverdueInvoice(t, pool, tenantB, 20)

	resolver := NewPostgresDueResolver(pool)
	ctx := testutil.WithSystemCtx(context.Background())

	entities, err := resolver.Resolve(ctx, "biz.invoice.overdue", tenantA, time.Now())
	if err != nil {
		t.Fatalf("resolve overdue invoices: %v", err)
	}

	var found *DueEntity
	for i := range entities {
		if entities[i].ResourceID == wantID.String() {
			found = &entities[i]
		}
		if entities[i].ResourceID == otherTenantID.String() {
			t.Fatalf("tenant %s leaked into tenant %s's result", tenantB, tenantA)
		}
	}
	if found == nil {
		t.Fatalf("seeded overdue invoice %s not returned; got %d entities", wantID, len(entities))
	}

	// The key must carry today's date: an invoice stays overdue for weeks, and
	// the shipped dunning template only matches once days_overdue reaches 14.
	// A once-per-invoice key would burn the single chance on day one.
	wantKey := "invoice:" + wantID.String() + ":" + time.Now().Format("2006-01-02")
	if found.Key != wantKey {
		t.Errorf("entity key = %q, want %q", found.Key, wantKey)
	}

	invoice, ok := found.Payload["invoice"].(map[string]any)
	if !ok {
		t.Fatalf("payload is not nested under \"invoice\": %#v", found.Payload)
	}
	if got, isInt := invoice["days_overdue"].(int); !isInt || got != 20 {
		t.Errorf("days_overdue = %v (%T), want 20", invoice["days_overdue"], invoice["days_overdue"])
	}
	if got, isFloat := invoice["total"].(float64); !isFloat || got != 1234.50 {
		t.Errorf("total = %v (%T), want 1234.50", invoice["total"], invoice["total"])
	}
	if invoice["status"] != "sent" {
		t.Errorf("status = %v, want sent", invoice["status"])
	}
}

func TestPostgresDueResolver_OverdueInvoices_IgnoresNotYetDueAndSettled(t *testing.T) {
	pool := newDueResolverPool(t)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "due-resolver-filters")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	// Due in the future -- not overdue.
	future := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"tenant_id": tenantID, "invoice_number": "FUT-" + uuid.NewString()[:8],
		"status": "sent", "gross_total": 10.0,
		"invoice_date": time.Now(), "due_date": time.Now().AddDate(0, 0, 30),
		"created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", future) })

	// Past due but already paid -- settled, nothing to dun.
	paid := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"tenant_id": tenantID, "invoice_number": "PAID-" + uuid.NewString()[:8],
		"status": "paid", "gross_total": 10.0,
		"invoice_date": time.Now().AddDate(0, 0, -40), "due_date": time.Now().AddDate(0, 0, -20),
		"created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", paid) })

	// Past due but still a draft -- never sent, so never overdue.
	draft := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"tenant_id": tenantID, "invoice_number": "DRF-" + uuid.NewString()[:8],
		"status": "draft", "gross_total": 10.0,
		"invoice_date": time.Now().AddDate(0, 0, -40), "due_date": time.Now().AddDate(0, 0, -20),
		"created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", draft) })

	resolver := NewPostgresDueResolver(pool)
	entities, err := resolver.Resolve(testutil.WithSystemCtx(context.Background()), "biz.invoice.overdue", tenantID, time.Now())
	if err != nil {
		t.Fatalf("resolve overdue invoices: %v", err)
	}
	if len(entities) != 0 {
		t.Fatalf("expected no due entity for future/paid/draft invoices, got %d: %+v", len(entities), entities)
	}
}

func TestPostgresDueResolver_UpcomingCalendarEvents_WindowAndTenantScope(t *testing.T) {
	pool := newDueResolverPool(t)
	ctx := testutil.WithSystemCtx(context.Background())

	tenantA := uuid.New()
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "due-cal-a")
	testutil.EnsureTenant(t, pool, tenantB, "due-cal-b")
	t.Cleanup(func() {
		testutil.CleanupRow(t, pool, "tenants", tenantA)
		testutil.CleanupRow(t, pool, "tenants", tenantB)
	})

	seedEvent := func(tenantID uuid.UUID, startsIn time.Duration) uuid.UUID {
		t.Helper()
		userID := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id": tenantID, "email": uuid.NewString() + "@due-test.local",
			"password_hash": "x",
		})
		calendarID := testutil.SeedRow(t, pool, "calendars", map[string]any{
			"tenant_id": tenantID, "name": "due-test", "owner_id": userID,
		})
		start := time.Now().Add(startsIn)
		eventID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
			"tenant_id": tenantID, "calendar_id": calendarID, "title": "Standup",
			"start_time": start, "end_time": start.Add(30 * time.Minute),
			"created_by": userID,
		})
		// Reverse order: the event references the calendar, which references
		// the user.
		t.Cleanup(func() {
			testutil.CleanupRow(t, pool, "calendar_events", eventID)
			testutil.CleanupRow(t, pool, "calendars", calendarID)
			testutil.CleanupRow(t, pool, "users", userID)
		})
		return eventID
	}

	inWindow := seedEvent(tenantA, 5*time.Minute)
	beyondWindow := seedEvent(tenantA, 2*time.Hour)
	alreadyStarted := seedEvent(tenantA, -5*time.Minute)
	otherTenant := seedEvent(tenantB, 5*time.Minute)

	resolver := NewPostgresDueResolver(pool)
	entities, err := resolver.Resolve(ctx, "calendar.event.upcoming", tenantA, time.Now())
	if err != nil {
		t.Fatalf("resolve upcoming calendar events: %v", err)
	}

	got := map[string]DueEntity{}
	for _, e := range entities {
		got[e.ResourceID] = e
	}
	if _, ok := got[inWindow.String()]; !ok {
		t.Errorf("event starting in 5 minutes was not returned; got %d entities", len(entities))
	}
	if _, ok := got[beyondWindow.String()]; ok {
		t.Errorf("event starting in 2 hours must be outside the %s window", upcomingEventWindow)
	}
	if _, ok := got[alreadyStarted.String()]; ok {
		t.Errorf("event that already started must not be returned as upcoming")
	}
	if _, ok := got[otherTenant.String()]; ok {
		t.Errorf("tenant %s leaked into tenant %s's result", tenantB, tenantA)
	}

	entity := got[inWindow.String()]
	// No date in the key: an event starts once, so a reminder that fires on
	// two consecutive days is a bug.
	if entity.Key != "calendar_event:"+inWindow.String() {
		t.Errorf("entity key = %q, want calendar_event:%s", entity.Key, inWindow)
	}
	evt, ok := entity.Payload["event"].(map[string]any)
	if !ok {
		t.Fatalf("payload is not nested under \"event\": %#v", entity.Payload)
	}
	if evt["title"] != "Standup" {
		t.Errorf("title = %v, want Standup", evt["title"])
	}
	minutes, isInt := evt["minutes_until"].(int)
	if !isInt || minutes < 0 || minutes > 5 {
		t.Errorf("minutes_until = %v (%T), want 0..5", evt["minutes_until"], evt["minutes_until"])
	}
}

func TestPostgresDueResolver_UnknownTriggerType(t *testing.T) {
	pool := newDueResolverPool(t)

	resolver := NewPostgresDueResolver(pool)
	_, err := resolver.Resolve(context.Background(), "crm.contact.created", uuid.New(), time.Now())

	if !errors.Is(err, ErrUnknownTimeTrigger) {
		t.Fatalf("error = %v, want ErrUnknownTimeTrigger", err)
	}
}
