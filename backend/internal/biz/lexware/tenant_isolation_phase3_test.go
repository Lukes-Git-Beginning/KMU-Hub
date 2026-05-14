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

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "lw-cfg-a-" + uuid.New().String()[:8] + "@a.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)

	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantB,
		"email":         "lw-cfg-b-" + uuid.New().String()[:8] + "@b.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userB)

	cfgA := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             testutil.TenantA,
		"platform":              "lexware",
		"credentials_vault_key": "lexware/" + uuid.New().String(),
		"created_by":            userA,
	})
	defer testutil.CleanupRow(t, pool, "integration_configs", cfgA)

	cfgB := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             testutil.TenantB,
		"platform":              "lexware",
		"credentials_vault_key": "lexware/" + uuid.New().String(),
		"created_by":            userB,
	})
	defer testutil.CleanupRow(t, pool, "integration_configs", cfgB)

	syncA := testutil.SeedRow(t, pool, "lexware_sync_configs", map[string]any{
		"tenant_id": testutil.TenantA,
		"config_id": cfgA,
	})
	defer testutil.CleanupRow(t, pool, "lexware_sync_configs", syncA)

	syncB := testutil.SeedRow(t, pool, "lexware_sync_configs", map[string]any{
		"tenant_id": testutil.TenantB,
		"config_id": cfgB,
	})
	defer testutil.CleanupRow(t, pool, "lexware_sync_configs", syncB)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	t.Run("lexware_sync_configs", func(t *testing.T) {
		t.Parallel()
		testutil.AssertRowCount(t, pool, ctxA, "lexware_sync_configs", syncA, 1)
		testutil.AssertRowCount(t, pool, ctxA, "lexware_sync_configs", syncB, 0)
		testutil.AssertRowCount(t, pool, ctxB, "lexware_sync_configs", syncB, 1)
		testutil.AssertRowCount(t, pool, ctxB, "lexware_sync_configs", syncA, 0)
	})
}
