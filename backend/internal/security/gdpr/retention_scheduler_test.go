package gdpr

// Coverage for the retention scheduler (A14): the pg_try_advisory_lock
// leader election, tenant enumeration, and per-tenant fan-out that turn the
// A10 engine into something that actually runs. Everything here is DB-backed
// on purpose -- pg_try_advisory_lock only means anything against a real
// Postgres session.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestRunScheduledRetention_SkipsWhenLockHeldElsewhere is the mutations-probe
// target: it proves two concurrent scheduler ticks cannot both run. Rather
// than racing two goroutines against wall-clock timing (flaky -- a fast,
// empty-registry run can finish before a second goroutine's lock check even
// lands), this holds the lock deterministically on a dedicated connection,
// exactly as an in-flight replica's run would, and checks the second call
// backs off instead of double-running.
func TestRunScheduledRetention_SkipsWhenLockHeldElsewhere(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	conn, err := pool.Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Release()

	var locked bool
	require.NoError(t, conn.QueryRow(context.Background(),
		`SELECT pg_try_advisory_lock($1)`, retentionScheduleLockKey).Scan(&locked))
	require.True(t, locked, "test setup must win the lock first")
	defer func() {
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, retentionScheduleLockKey)
	}()

	engine := NewRetentionEngine(pool, NewPostgresRepository(pool), NewRetentionRegistry())
	ran, err := RunScheduledRetention(context.Background(), pool, engine, RetentionModeDryRun)
	require.NoError(t, err)
	assert.False(t, ran, "a concurrent replica holding the lock must make this call skip, not double-run")
}

// There is deliberately no "RunScheduledRetention with the lock free" test
// here: the local dev Postgres this loop runs against has accumulated
// 13.8k+ rows in `tenants` from unrelated tests across many prior nightloop
// runs (see JOURNAL.md iteration 14 for the count and the reasoning why this
// is dev-environment debris, not a code bug). RunScheduledRetention's
// unlocked path calls listTenantIDs and runs the engine once per row it
// gets back -- against that table it takes over a minute and writes a
// retention_runs row for every one of those 13.8k stale tenants, which is
// both a slow gate and actively worse pollution. The two pieces that path
// composes (the lock gate, proven above; the per-tenant fan-out, proven by
// TestRunForTenants_RunsEngineOncePerTenant below against a controlled list)
// are already covered without touching the real table at that scale.

// TestRunForTenants_RunsEngineOncePerTenant proves the fan-out itself,
// against a controlled tenant list rather than the shared `tenants` table --
// other tests seed and clean up their own tenant rows there in parallel, so
// asserting an exact run count against the full table would be brittle.
func TestRunForTenants_RunsEngineOncePerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA := uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Retention Scheduler Tenant A")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Retention Scheduler Tenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	engine := NewRetentionEngine(pool, NewPostgresRepository(pool), NewRetentionRegistry())

	runForTenants(context.Background(), engine, RetentionModeDryRun, []uuid.UUID{tenantA, tenantB})

	for _, tenantID := range []uuid.UUID{tenantA, tenantB} {
		ctx := testutil.WithTenantCtx(context.Background(), tenantID)
		var count int
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM retention_runs WHERE tenant_id = $1 AND triggered_by = 'schedule'`,
			tenantID,
		).Scan(&count))
		assert.Equal(t, 1, count, "tenant %s must have exactly one scheduled run logged", tenantID)
	}

	_, _ = pool.Exec(testutil.WithSystemCtx(context.Background()),
		`DELETE FROM retention_runs WHERE tenant_id = ANY($1)`, []uuid.UUID{tenantA, tenantB})
}

// TestListTenantIDs_ContainsSeededTenant does not assert an exact count --
// the `tenants` table is shared with every other parallel test in this
// package, so only containment is safe to check.
func TestListTenantIDs_ContainsSeededTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "List Tenant IDs Fixture")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	ids, err := listTenantIDs(testutil.WithSystemCtx(context.Background()), pool)
	require.NoError(t, err)
	assert.Contains(t, ids, tenantID)
}
