package notification_test

// Backlog replay against the real `events` table.
//
// Backlog item "fix-notification-backlog-replay-repeats-delivered-events"
// suspected that a restart redelivers the last 1000 events forever, because the
// live path writes rows with processed=false and (it was assumed) never closes
// them. Reading the code refutes that: ProcessEvent marks its own row at the end
// (service.go), and ProcessBacklog marks the replayed one. These tests pin both
// halves against Postgres so a refactor that drops either MarkEventProcessed
// call turns the replay self-sustaining and fails here.
//
// The catch-up read spans tenants by design and the table is shared with the
// parallel isolation test, so the repository handed to ProcessBacklog is
// narrowed to this test's tenant. Replaying (and marking) another test's rows
// would break its assertions.

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/notification/notification"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// tenantScopedEvents delegates to the real repository but only lists one
// tenant's unprocessed rows.
type tenantScopedEvents struct {
	repo     *notification.PostgresRepository
	tenantID uuid.UUID
}

func (s tenantScopedEvents) CreateEvent(ctx context.Context, e *models.Event) error {
	return s.repo.CreateEvent(ctx, e)
}

func (s tenantScopedEvents) MarkProcessed(ctx context.Context, id string) error {
	return s.repo.MarkEventProcessed(ctx, id)
}

func (s tenantScopedEvents) ListUnprocessed(ctx context.Context, limit int) ([]models.Event, error) {
	all, err := s.repo.ListUnprocessedEvents(ctx, limit)
	if err != nil {
		return nil, err
	}
	var own []models.Event
	for _, e := range all {
		if e.TenantID == s.tenantID {
			own = append(own, e)
		}
	}
	return own, nil
}

func TestEventBacklog_LivePathClosesItsRow_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "backlog live path")
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	repo := notification.NewPostgresRepository(pool)
	svc := notification.NewService(repo, nil, event.NewEventTypeRegistry(), nil, nil)
	scoped := tenantScopedEvents{repo: repo, tenantID: tenantID}

	err := svc.ProcessEvent(ctx, models.EventPayload{
		Type:      "test.backlog.live",
		TenantID:  tenantID,
		Priority:  models.PriorityNormal,
		ModuleID:  "test",
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("ProcessEvent: %v", err)
	}

	open, err := scoped.ListUnprocessed(ctx, 1000)
	if err != nil {
		t.Fatalf("list unprocessed: %v", err)
	}
	if len(open) != 0 {
		t.Fatalf("live path left %d event row(s) unprocessed; every restart would replay them", len(open))
	}
}

func TestEventBacklog_ReplayTerminates_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "backlog replay")
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	repo := notification.NewPostgresRepository(pool)
	svc := notification.NewService(repo, nil, event.NewEventTypeRegistry(), nil, nil)
	scoped := tenantScopedEvents{repo: repo, tenantID: tenantID}

	// A row whose processing never finished, as left behind by a crash.
	if err := repo.CreateEvent(ctx, newProbeEvent(tenantID, "test.backlog.crashed")); err != nil {
		t.Fatalf("seed crashed event: %v", err)
	}

	var delivered atomic.Int32
	bus := event.NewEventBus("")
	bus.RegisterHandler("*", func(ctx context.Context, p models.EventPayload) error {
		delivered.Add(1)
		return svc.ProcessEvent(ctx, p)
	})

	if err := bus.ProcessBacklog(context.Background(), scoped); err != nil {
		t.Fatalf("first ProcessBacklog: %v", err)
	}
	if got := delivered.Load(); got != 1 {
		t.Fatalf("first replay delivered %d events, want 1", got)
	}

	// The replay re-enters ProcessEvent, which writes a fresh row. That row has
	// to be closed too, or the backlog feeds itself.
	if err := bus.ProcessBacklog(context.Background(), scoped); err != nil {
		t.Fatalf("second ProcessBacklog: %v", err)
	}
	if got := delivered.Load(); got != 1 {
		t.Fatalf("second replay delivered again (total %d); the replay is self-sustaining", got)
	}
	open, err := scoped.ListUnprocessed(ctx, 1000)
	if err != nil {
		t.Fatalf("list unprocessed: %v", err)
	}
	if len(open) != 0 {
		t.Fatalf("%d event row(s) still unprocessed after replay", len(open))
	}
}
