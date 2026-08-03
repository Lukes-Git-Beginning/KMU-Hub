package notification_test

// Cross-tenant isolation for the `events` durability table after migration
// 000271 gave it a tenant_id and a policy.
//
// The test drives the three real repository methods rather than raw SQL,
// because the interesting part is not whether a hand-written INSERT respects
// the policy — it is whether the worker can still do its job under it. Two of
// those methods span tenants by design (the catch-up read and the processed
// flag) and run as system; the write does not and must be rejected when the
// context carries no tenant. Both halves are asserted here, because getting
// either one wrong turns the event bus into a silent no-op.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/notification"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// assertRLSRejected fails unless err is an RLS refusal. Checking the SQLSTATE
// and not merely "some error came back" is the point: a missing column or a
// typo'd table also produces an error, and a test satisfied by any failure
// keeps passing after the policy is gone.
func assertRLSRejected(t *testing.T, err error, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: succeeded; the RLS policy is not in force", what)
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Fatalf("%s: rejected, but not by RLS (want SQLSTATE 42501): %v", what, err)
	}
}

func newProbeEvent(tenantID uuid.UUID, key string) *models.Event {
	return &models.Event{
		ID:           uuid.New(),
		TenantID:     tenantID,
		EventTypeKey: key,
		ModuleID:     "test",
		Priority:     "normal",
		Processed:    false,
		CreatedAt:    time.Now(),
	}
}

func TestTenantIsolation_Events_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "events isolation A")
	testutil.EnsureTenant(t, pool, tenantB, "events isolation B")

	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	repo := notification.NewPostgresRepository(pool)

	evtA := newProbeEvent(tenantA, "test.isolation.a")
	if err := repo.CreateEvent(ctxA, evtA); err != nil {
		t.Fatalf("create event for tenant A: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "events", evtA.ID) })

	testutil.AssertRowCount(t, pool, ctxA, "events", evtA.ID, 1)
	testutil.AssertRowCount(t, pool, ctxB, "events", evtA.ID, 0)

	// The write must not be able to claim a foreign tenant either: the policy's
	// WITH CHECK compares the row against the connection, so a row stamped for
	// B on A's context is rejected rather than stored.
	crossEvt := newProbeEvent(tenantB, "test.isolation.cross")
	err := repo.CreateEvent(ctxA, crossEvt)
	if err == nil {
		testutil.CleanupRow(t, pool, "events", crossEvt.ID)
	}
	assertRLSRejected(t, err, "event stamped for tenant B written on tenant A's context")
}

// A context without a tenant must not be able to write. This is what makes the
// stamping in EventBus.dispatch load-bearing rather than decorative: the
// listener's background context carries no tenant, and if this insert quietly
// succeeded the table would fill with rows no policy could separate.
func TestEvents_WriteWithoutTenantContext_Rejected_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "events unstamped write")

	repo := notification.NewPostgresRepository(pool)
	evt := newProbeEvent(tenant, "test.unstamped")

	err := repo.CreateEvent(context.Background(), evt)
	if err == nil {
		testutil.CleanupRow(t, pool, "events", evt.ID)
	}
	assertRLSRejected(t, err, "event written without a tenant context")
}

// The catch-up path after downtime. It reads across tenants, so it runs as
// system — and it must return each event's tenant, because ProcessBacklog
// rebuilds the payload from this row and had no other way to restore it.
func TestEvents_BacklogReadSpansTenantsAndCarriesTenantID_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "events backlog A")
	testutil.EnsureTenant(t, pool, tenantB, "events backlog B")

	repo := notification.NewPostgresRepository(pool)

	evtA := newProbeEvent(tenantA, "test.backlog.a")
	evtB := newProbeEvent(tenantB, "test.backlog.b")
	for _, e := range []*models.Event{evtA, evtB} {
		ctx := testutil.WithTenantCtx(context.Background(), e.TenantID)
		if err := repo.CreateEvent(ctx, e); err != nil {
			t.Fatalf("create event %s: %v", e.EventTypeKey, err)
		}
		id := e.ID
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "events", id) })
	}

	// Deliberately an unstamped context — this is how ProcessBacklog calls it.
	found := map[uuid.UUID]uuid.UUID{}
	events, err := repo.ListUnprocessedEvents(context.Background(), 1000)
	if err != nil {
		t.Fatalf("list unprocessed: %v", err)
	}
	for _, e := range events {
		found[e.ID] = e.TenantID
	}

	for _, e := range []*models.Event{evtA, evtB} {
		got, ok := found[e.ID]
		if !ok {
			t.Fatalf("event %s missing from the backlog read; the catch-up path sees nothing under RLS", e.EventTypeKey)
		}
		if got != e.TenantID {
			t.Fatalf("event %s: backlog returned tenant %s, want %s", e.EventTypeKey, got, e.TenantID)
		}
	}

	// Marking processed also comes from the unstamped worker context. An event
	// that cannot be marked is replayed on every restart, forever.
	if err := repo.MarkEventProcessed(context.Background(), evtA.ID.String()); err != nil {
		t.Fatalf("mark processed: %v", err)
	}

	after, err := repo.ListUnprocessedEvents(context.Background(), 1000)
	if err != nil {
		t.Fatalf("list unprocessed after mark: %v", err)
	}
	for _, e := range after {
		if e.ID == evtA.ID {
			t.Fatal("event still unprocessed after MarkEventProcessed")
		}
	}
}
