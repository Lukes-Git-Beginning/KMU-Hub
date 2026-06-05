package lexware

// Welle 4 Phase 3 — cross-tenant isolation for lexware_sync_configs after
// migration 000125 activates RLS and the per-tenant constraint on
// integration_configs.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_LexwareSyncConfigs_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	// Fresh random tenants per run: integration_configs is UNIQUE on
	// (platform, tenant_id) and platform has a CHECK constraint, so the
	// shared TenantA fixture would collide with the parallel sibling test.
	tenantA := uuid.New()
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Lexware P3 Tenant A")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	testutil.EnsureTenant(t, pool, tenantB, "Lexware P3 Tenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantA,
		"email":         "lw-cfg-a-" + uuid.New().String()[:8] + "@a.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)

	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantB,
		"email":         "lw-cfg-b-" + uuid.New().String()[:8] + "@b.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userB)

	cfgA := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenantA,
		"platform":              "lexware",
		"credentials_vault_key": "lexware/" + uuid.New().String(),
		"created_by":            userA,
	})
	defer testutil.CleanupRow(t, pool, "integration_configs", cfgA)

	cfgB := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenantB,
		"platform":              "lexware",
		"credentials_vault_key": "lexware/" + uuid.New().String(),
		"created_by":            userB,
	})
	defer testutil.CleanupRow(t, pool, "integration_configs", cfgB)

	syncA := testutil.SeedRow(t, pool, "lexware_sync_configs", map[string]any{
		"tenant_id": tenantA,
		"config_id": cfgA,
	})
	defer testutil.CleanupRow(t, pool, "lexware_sync_configs", syncA)

	syncB := testutil.SeedRow(t, pool, "lexware_sync_configs", map[string]any{
		"tenant_id": tenantB,
		"config_id": cfgB,
	})
	defer testutil.CleanupRow(t, pool, "lexware_sync_configs", syncB)

	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	t.Run("lexware_sync_configs", func(t *testing.T) {
		testutil.AssertRowCount(t, pool, ctxA, "lexware_sync_configs", syncA, 1)
		testutil.AssertRowCount(t, pool, ctxA, "lexware_sync_configs", syncB, 0)
		testutil.AssertRowCount(t, pool, ctxB, "lexware_sync_configs", syncB, 1)
		testutil.AssertRowCount(t, pool, ctxB, "lexware_sync_configs", syncA, 0)
	})
}
