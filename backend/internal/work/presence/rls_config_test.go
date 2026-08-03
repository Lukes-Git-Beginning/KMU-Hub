package presence_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/presence"
)

// presence_config is unique on tenant_id since migration 000274, so these tests
// seed their own tenants instead of sharing testutil.TenantA/TenantB: a parallel
// test writing a config for the same tenant would collide on that index. Fixed
// ids keep repeated runs from adding a tenant row every time — EnsureTenant is
// idempotent.
var (
	pcTenantOne = uuid.MustParse("9fe50000-0000-0000-0000-000000000001")
	pcTenantTwo = uuid.MustParse("9fe50000-0000-0000-0000-000000000002")
)

// ensureThrowawayTenant creates a tenant that only this run uses. The two
// fresh-tenant cases below need an id that provably has no config row, which
// the fixed ids above cannot guarantee across runs. Callers must `defer` the
// returned cleanup rather than register it with t.Cleanup: these tests close
// the pool with a defer, and t.Cleanup runs after every defer — the removal
// would hit a closed pool and fail silently, leaking a tenant per run.
func ensureThrowawayTenant(t *testing.T, pool *pgxpool.Pool, name string) (uuid.UUID, func()) {
	t.Helper()
	id := uuid.New()
	testutil.EnsureTenant(t, pool, id, name)
	return id, func() { testutil.CleanupRow(t, pool, "tenants", id) }
}

func seedPresenceTenants(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	testutil.EnsureTenant(t, pool, pcTenantOne, "Presence Config Tenant One")
	testutil.EnsureTenant(t, pool, pcTenantTwo, "Presence Config Tenant Two")
}

// seedPresenceUser creates a user in the tenant, needed because
// presence_config.updated_by references users(id). Cleanup is a deferred
// callback for the same reason as ensureThrowawayTenant.
func seedPresenceUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) (uuid.UUID, func()) {
	t.Helper()
	id := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     tenantID,
		"email":         "presence-rls-" + uuid.NewString() + "@example.test",
		"password_hash": "x",
	})
	return id, func() { testutil.CleanupRow(t, pool, "users", id) }
}

// TestRLS_PresenceConfig_ForeignTenantSeesNothing is the standard isolation
// check for the tenant_isolation policy migration 000274 added.
func TestRLS_PresenceConfig_ForeignTenantSeesNothing(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	seedPresenceTenants(t, pool)

	configID := testutil.SeedRow(t, pool, "presence_config", map[string]any{
		"id":                   uuid.New(),
		"tenant_id":            pcTenantOne,
		"away_timeout_seconds": 111,
	})
	defer testutil.CleanupRow(t, pool, "presence_config", configID)

	ctxOne := testutil.WithTenantCtx(context.Background(), pcTenantOne)
	testutil.AssertRowCount(t, pool, ctxOne, "presence_config", configID, 1)

	ctxTwo := testutil.WithTenantCtx(context.Background(), pcTenantTwo)
	testutil.AssertRowCount(t, pool, ctxTwo, "presence_config", configID, 0)
}

// TestPresenceConfig_TenantsHoldIndependentTimeouts is the bug 000274 exists
// for. UpdateConfig ran an UPDATE with no WHERE clause against a single global
// row, so one tenant's administrator moved the away timeout for the whole
// installation. Both tenants must now keep their own value.
func TestPresenceConfig_TenantsHoldIndependentTimeouts(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	seedPresenceTenants(t, pool)
	repo := presence.NewPostgresConfigRepository(pool)

	userOne, dropUserOne := seedPresenceUser(t, pool, pcTenantOne)
	defer dropUserOne()
	userTwo, dropUserTwo := seedPresenceUser(t, pool, pcTenantTwo)
	defer dropUserTwo()

	ctxOne := testutil.WithTenantCtx(context.Background(), pcTenantOne)
	if err := repo.UpdateConfig(ctxOne, pcTenantOne, 120, userOne); err != nil {
		t.Fatalf("update for tenant one: %v", err)
	}
	defer cleanupPresenceConfig(t, pool, pcTenantOne)

	ctxTwo := testutil.WithTenantCtx(context.Background(), pcTenantTwo)
	if err := repo.UpdateConfig(ctxTwo, pcTenantTwo, 900, userTwo); err != nil {
		t.Fatalf("update for tenant two: %v", err)
	}
	defer cleanupPresenceConfig(t, pool, pcTenantTwo)

	cfgOne, err := repo.GetConfig(ctxOne, pcTenantOne)
	if err != nil {
		t.Fatalf("read back tenant one: %v", err)
	}
	if cfgOne.AwayTimeoutSeconds != 120 {
		t.Fatalf("tenant one timeout: want 120, got %d — the second tenant's write leaked", cfgOne.AwayTimeoutSeconds)
	}

	cfgTwo, err := repo.GetConfig(ctxTwo, pcTenantTwo)
	if err != nil {
		t.Fatalf("read back tenant two: %v", err)
	}
	if cfgTwo.AwayTimeoutSeconds != 900 {
		t.Fatalf("tenant two timeout: want 900, got %d", cfgTwo.AwayTimeoutSeconds)
	}
}

// TestPresenceConfig_UpsertCreatesRowForFreshTenant covers the reason
// UpdateConfig upserts: tenants created after 000274 hold no row, and a plain
// UPDATE would have reported success while changing nothing.
func TestPresenceConfig_UpsertCreatesRowForFreshTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	fresh, dropTenant := ensureThrowawayTenant(t, pool, "Presence Config Fresh Tenant")
	defer dropTenant()

	repo := presence.NewPostgresConfigRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), fresh)

	if _, err := repo.GetConfig(ctx, fresh); !errors.Is(err, presence.ErrConfigNotFound) {
		t.Fatalf("fresh tenant: want ErrConfigNotFound, got %v", err)
	}

	user, dropUser := seedPresenceUser(t, pool, fresh)
	defer dropUser()
	if err := repo.UpdateConfig(ctx, fresh, 240, user); err != nil {
		t.Fatalf("first write for fresh tenant: %v", err)
	}
	defer cleanupPresenceConfig(t, pool, fresh)

	cfg, err := repo.GetConfig(ctx, fresh)
	if err != nil {
		t.Fatalf("read back fresh tenant: %v", err)
	}
	if cfg.AwayTimeoutSeconds != 240 {
		t.Fatalf("fresh tenant timeout: want 240, got %d", cfg.AwayTimeoutSeconds)
	}
}

// TestPresenceService_GetConfig_FallsBackToCodeDefault documents the decision
// taken instead of seeding a provisioning row: a tenant that never wrote a
// config reads the code default rather than an error.
func TestPresenceService_GetConfig_FallsBackToCodeDefault(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	fresh, dropTenant := ensureThrowawayTenant(t, pool, "Presence Config Default Tenant")
	defer dropTenant()

	svc := presence.NewService(nil, presence.NewPostgresConfigRepository(pool))
	ctx := testutil.WithTenantCtx(context.Background(), fresh)

	cfg, err := svc.GetConfig(ctx)
	if err != nil {
		t.Fatalf("get config for fresh tenant: %v", err)
	}
	if cfg.AwayTimeoutSeconds != presence.DefaultAwayTimeoutSeconds {
		t.Fatalf("want code default %d, got %d", presence.DefaultAwayTimeoutSeconds, cfg.AwayTimeoutSeconds)
	}
	if cfg.TenantID != fresh {
		t.Fatalf("default config must carry the calling tenant, got %s", cfg.TenantID)
	}
}

// TestPresenceService_GetConfig_WithoutTenantFails guards the sysctx hazard
// 000273 uncovered: a caller with no tenant on the context must fail rather
// than read whichever row the policy happens to admit.
func TestPresenceService_GetConfig_WithoutTenantFails(t *testing.T) {
	svc := presence.NewService(nil, presence.NewPostgresConfigRepository(nil))

	if _, err := svc.GetConfig(context.Background()); !errors.Is(err, presence.ErrMissingTenant) {
		t.Fatalf("want ErrMissingTenant, got %v", err)
	}
	if err := svc.UpdateConfig(context.Background(), 120, uuid.New()); !errors.Is(err, presence.ErrMissingTenant) {
		t.Fatalf("want ErrMissingTenant, got %v", err)
	}
}

// cleanupPresenceConfig removes a tenant's config row; the table is keyed on
// tenant_id, not on an id the test knows in advance.
func cleanupPresenceConfig(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(ctx, `DELETE FROM presence_config WHERE tenant_id = $1`, tenantID); err != nil {
		t.Logf("cleanup presence_config tenant=%s: %v", tenantID, err)
	}
}
