package plugin

// Cross-tenant isolation tests for plugin_installations (Migration 000122).
// plugin_installations.tenant_id is NOT NULL and RLS was enabled in mig 000122.
// Requires a parent plugin_manifests row (no tenant_id — system table).

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_PluginInstallations(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a plugin_manifest (no tenant_id — global table, no RLS).
	manifestID := testutil.SeedRow(t, pool, "plugin_manifests", map[string]any{
		"slug": "test-plugin-" + uuid.New().String()[:8],
		"name": "Test Plugin",
	})
	defer testutil.CleanupRow(t, pool, "plugin_manifests", manifestID)

	// Seed a plugin_installation for tenant A.
	installID := testutil.SeedRow(t, pool, "plugin_installations", map[string]any{
		"tenant_id":    testutil.TenantA,
		"manifest_id":  manifestID,
		"installed_by": uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "plugin_installations", installID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	testutil.AssertRowCount(t, pool, ctxA, "plugin_installations", installID, 1)
	testutil.AssertRowCount(t, pool, ctxB, "plugin_installations", installID, 0)
}
