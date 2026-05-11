package einkauf

// Cross-tenant isolation tests for einkauf tables (Migration 000122).
// suppliers, purchase_orders, po_lines all have tenant_id NOT NULL and RLS
// was enabled in mig 000122.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Einkauf(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	supplierID := testutil.SeedRow(t, pool, "suppliers", map[string]any{
		"tenant_id": testutil.TenantA,
		"name":      "Acme Supplies " + uuid.New().String()[:8],
	})
	defer testutil.CleanupRow(t, pool, "suppliers", supplierID)

	poID := testutil.SeedRow(t, pool, "purchase_orders", map[string]any{
		"tenant_id":   testutil.TenantA,
		"supplier_id": supplierID,
		"po_number":   "PO-" + uuid.New().String()[:8],
	})
	defer testutil.CleanupRow(t, pool, "purchase_orders", poID)

	poLineID := testutil.SeedRow(t, pool, "po_lines", map[string]any{
		"tenant_id":    testutil.TenantA,
		"po_id":        poID,
		"product_name": "Screw M6",
		"quantity":     "100.0000",
		"unit_price":   "0.1500",
	})
	defer testutil.CleanupRow(t, pool, "po_lines", poLineID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"suppliers", "suppliers", supplierID},
		{"purchase_orders", "purchase_orders", poID},
		{"po_lines", "po_lines", poLineID},
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
