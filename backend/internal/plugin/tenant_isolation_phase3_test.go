package plugin

// Welle 4.2 Phase 3 — cross-tenant isolation for plugin_kv_store and
// plugin_execution_log after migration 000126 adds tenant_id + RLS.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_PluginKVStore_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	manifestID := testutil.SeedRow(t, pool, "plugin_manifests", map[string]any{
		"slug": "kv-test-" + uuid.New().String()[:8],
		"name": "KV Test Plugin",
	})
	defer testutil.CleanupRow(t, pool, "plugin_manifests", manifestID)

	installA := testutil.SeedRow(t, pool, "plugin_installations", map[string]any{
		"tenant_id":    testutil.TenantA,
		"manifest_id":  manifestID,
		"installed_by": uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "plugin_installations", installA)

	installB := testutil.SeedRow(t, pool, "plugin_installations", map[string]any{
		"tenant_id":    testutil.TenantB,
		"manifest_id":  manifestID,
		"installed_by": uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "plugin_installations", installB)

	kvA := testutil.SeedRow(t, pool, "plugin_kv_store", map[string]any{
		"tenant_id":       testutil.TenantA,
		"installation_id": installA,
		"key":             "foo",
		"value":           []byte(`"valueA"`),
	})
	defer testutil.CleanupRow(t, pool, "plugin_kv_store", kvA)

	kvB := testutil.SeedRow(t, pool, "plugin_kv_store", map[string]any{
		"tenant_id":       testutil.TenantB,
		"installation_id": installB,
		"key":             "foo",
		"value":           []byte(`"valueB"`),
	})
	defer testutil.CleanupRow(t, pool, "plugin_kv_store", kvB)

	logA := testutil.SeedRow(t, pool, "plugin_execution_log", map[string]any{
		"tenant_id":       testutil.TenantA,
		"installation_id": installA,
		"hook_type":       "before_save",
		"module":          "crm",
		"entity_type":     "contact",
		"status":          "success",
	})
	defer testutil.CleanupRow(t, pool, "plugin_execution_log", logA)

	logB := testutil.SeedRow(t, pool, "plugin_execution_log", map[string]any{
		"tenant_id":       testutil.TenantB,
		"installation_id": installB,
		"hook_type":       "before_save",
		"module":          "crm",
		"entity_type":     "contact",
		"status":          "success",
	})
	defer testutil.CleanupRow(t, pool, "plugin_execution_log", logB)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		idA   uuid.UUID
		idB   uuid.UUID
	}{
		{"plugin_kv_store", "plugin_kv_store", kvA, kvB},
		{"plugin_execution_log", "plugin_execution_log", logA, logB},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.idA, 1)
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.idB, 0)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.idB, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.idA, 0)
		})
	}
}
