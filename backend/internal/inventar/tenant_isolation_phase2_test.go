package inventar

// Cross-tenant isolation tests for inventar tables (Migration 000122).
// inventory_items, inventory_movements, stock_warnings all have tenant_id NOT NULL
// and RLS was enabled in mig 000122.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Inventar(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed an inventory_item for tenant A — basis for movements and warnings.
	itemID := testutil.SeedRow(t, pool, "inventory_items", map[string]any{
		"tenant_id": testutil.TenantA,
		"name":      "Widget A",
		"sku":       "SKU-" + uuid.New().String()[:8],
	})
	defer testutil.CleanupRow(t, pool, "inventory_items", itemID)

	// Seed an inventory_movement for tenant A.
	movID := testutil.SeedRow(t, pool, "inventory_movements", map[string]any{
		"tenant_id": testutil.TenantA,
		"item_id":   itemID,
		"quantity":  10,
	})
	defer testutil.CleanupRow(t, pool, "inventory_movements", movID)

	// Seed a stock_warning for tenant A.
	warnID := testutil.SeedRow(t, pool, "stock_warnings", map[string]any{
		"tenant_id": testutil.TenantA,
		"item_id":   itemID,
	})
	defer testutil.CleanupRow(t, pool, "stock_warnings", warnID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"inventory_items", "inventory_items", itemID},
		{"inventory_movements", "inventory_movements", movID},
		{"stock_warnings", "stock_warnings", warnID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
