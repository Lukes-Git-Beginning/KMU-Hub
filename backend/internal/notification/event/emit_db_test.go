package event

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// These tests exercise the whole tenant path of an event: EmitEvent takes the
// tenant from the request context, pg_notify carries it, EventBus.dispatch
// stamps it back onto the handler context, and the handler's insert into the
// RLS-forced `events` table is admitted. The negative case proves the emit is
// refused rather than broadcast tenant-less, which used to end as a silent
// RLS rejection one layer down.

// listenUntilDelivered starts the bus and re-emits payload until the handler
// signals on delivered. The bus has no readiness signal — it issues LISTEN
// somewhere inside listenLoop — so a single emit races the subscription.
func listenUntilDelivered(t *testing.T, bus *EventBus, pool *pgxpool.Pool, emitCtx context.Context,
	payload models.EventPayload, delivered <-chan models.EventPayload) models.EventPayload {
	t.Helper()

	listenCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = bus.Listen(listenCtx) }()

	for i := 0; i < 60; i++ {
		require.NoError(t, EmitEvent(emitCtx, pool, payload))
		select {
		case got := <-delivered:
			return got
		case <-time.After(100 * time.Millisecond):
		}
	}
	t.Fatal("event never reached the handler within 6s")
	return models.EventPayload{}
}

func TestEmitEvent_TenantFromContextSurvivesToRLSGuardedInsert(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	foreignTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "emit-tenant-e2e")
	testutil.EnsureTenant(t, pool, foreignTenantID, "emit-tenant-e2e-foreign")

	eventType := "test.emit.e2e." + uuid.NewString()
	rowID := uuid.New()

	t.Cleanup(func() {
		sysCtx := testutil.WithSystemCtx(context.Background())
		_, _ = pool.Exec(sysCtx, `DELETE FROM events WHERE event_type_key = $1`, eventType)
	})

	delivered := make(chan models.EventPayload, 8)
	var once sync.Once
	var insertErr error

	bus := NewEventBus(os.Getenv("DATABASE_URL"), WithReconnectWait(100*time.Millisecond))
	bus.RegisterHandler(eventType, func(ctx context.Context, evt models.EventPayload) error {
		once.Do(func() {
			// Deliberately no tenant column value of our own: the insert must
			// be admitted by the tenant_isolation policy, which compares
			// against current_tenant_id() — i.e. the GUC the pool stamps from
			// the context dispatch handed us.
			_, insertErr = pool.Exec(ctx,
				`INSERT INTO events (id, tenant_id, event_type_key, module_id, priority, created_at)
				 VALUES ($1, $2, $3, $4, 'normal', now())`,
				rowID, evt.TenantID, evt.Type, evt.ModuleID)
			delivered <- evt
		})
		return nil
	})

	// The emitter leaves TenantID zero, exactly like every request-driven
	// service emitter does — the tenant must come from the context.
	payload := models.EventPayload{
		Type:      eventType,
		ModuleID:  "test",
		Priority:  "normal",
		Timestamp: time.Now().UTC(),
	}
	emitCtx := testutil.WithTenantCtx(context.Background(), tenantID)

	got := listenUntilDelivered(t, bus, pool, emitCtx, payload, delivered)

	assert.Equal(t, tenantID, got.TenantID, "handler must see the emitting request's tenant")
	require.NoError(t, insertErr, "RLS-guarded insert on the handler context must be admitted")

	testutil.AssertRowCount(t, pool, testutil.WithTenantCtx(context.Background(), tenantID),
		"events", rowID, 1)
	testutil.AssertRowCount(t, pool, testutil.WithTenantCtx(context.Background(), foreignTenantID),
		"events", rowID, 0)
}

func TestEmitEvent_WithoutTenantIsRefusedNotBroadcast(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "emit-tenant-negative")

	controlType := "test.emit.control." + uuid.NewString()
	orphanType := "test.emit.orphan." + uuid.NewString()

	control := make(chan models.EventPayload, 8)
	orphan := make(chan models.EventPayload, 8)

	bus := NewEventBus(os.Getenv("DATABASE_URL"), WithReconnectWait(100*time.Millisecond))
	bus.RegisterHandler(controlType, func(_ context.Context, evt models.EventPayload) error {
		control <- evt
		return nil
	})
	bus.RegisterHandler(orphanType, func(_ context.Context, evt models.EventPayload) error {
		orphan <- evt
		return nil
	})

	// Prove the listener is live first — otherwise "nothing arrived" would be
	// true for a bus that never subscribed at all.
	listenUntilDelivered(t, bus, pool,
		testutil.WithTenantCtx(context.Background(), tenantID),
		models.EventPayload{Type: controlType, ModuleID: "test", Priority: "normal", Timestamp: time.Now().UTC()},
		control)

	err := EmitEvent(context.Background(), pool, models.EventPayload{
		Type:      orphanType,
		ModuleID:  "test",
		Priority:  "normal",
		Timestamp: time.Now().UTC(),
	})
	require.ErrorIs(t, err, ErrMissingTenant)

	select {
	case evt := <-orphan:
		t.Fatalf("tenant-less event reached the bus: %+v", evt)
	case <-time.After(500 * time.Millisecond):
	}
}
