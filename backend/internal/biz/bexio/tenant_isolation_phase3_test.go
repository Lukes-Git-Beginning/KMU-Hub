package bexio

// Welle 4 Phase 3 — cross-tenant isolation for integration_configs +
// bexio_sync_configs after the constraint swap and RLS activation in
// migration 000125.
//
// Before 000125: integration_configs had UNIQUE(platform) and no RLS, so two
// tenants could not coexist with the same platform. After 000125 the unique
// constraint is (platform, tenant_id) and RLS hides rows from other tenants.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_IntegrationConfigs_Bexio_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	// Fresh random tenants per run: integration_configs is UNIQUE on
	// (platform, tenant_id) and platform has a CHECK constraint, so the
	// shared TenantA fixture would collide with the parallel phase2 test.
	tenantA := uuid.New()
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Bexio P3 Tenant A")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	testutil.EnsureTenant(t, pool, tenantB, "Bexio P3 Tenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantA,
		"email":         "bexio-cfg-a-" + uuid.New().String()[:8] + "@a.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)

	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantB,
		"email":         "bexio-cfg-b-" + uuid.New().String()[:8] + "@b.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userB)

	// Both tenants coexist with the same platform — impossible before 000125.
	cfgA := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenantA,
		"platform":              "bexio",
		"credentials_vault_key": "bexio/" + uuid.New().String(),
		"created_by":            userA,
	})
	defer testutil.CleanupRow(t, pool, "integration_configs", cfgA)

	cfgB := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenantB,
		"platform":              "bexio",
		"credentials_vault_key": "bexio/" + uuid.New().String(),
		"created_by":            userB,
	})
	defer testutil.CleanupRow(t, pool, "integration_configs", cfgB)

	// bexio_sync_configs child rows — tenant_id derived per row.
	syncA := testutil.SeedRow(t, pool, "bexio_sync_configs", map[string]any{
		"tenant_id": tenantA,
		"config_id": cfgA,
	})
	defer testutil.CleanupRow(t, pool, "bexio_sync_configs", syncA)

	syncB := testutil.SeedRow(t, pool, "bexio_sync_configs", map[string]any{
		"tenant_id": tenantB,
		"config_id": cfgB,
	})
	defer testutil.CleanupRow(t, pool, "bexio_sync_configs", syncB)

	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	tests := []struct {
		name  string
		table string
		idA   uuid.UUID
		idB   uuid.UUID
	}{
		{"integration_configs", "integration_configs", cfgA, cfgB},
		{"bexio_sync_configs", "bexio_sync_configs", syncA, syncB},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.idA, 1)
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.idB, 0)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.idB, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.idA, 0)
		})
	}
}
