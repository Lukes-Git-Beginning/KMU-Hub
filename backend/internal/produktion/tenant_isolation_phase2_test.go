package produktion

// Cross-tenant isolation tests for produktion tables (Migration 000122).
// production_orders, machine_bookings, production_plans all have tenant_id NOT NULL
// and RLS was enabled in mig 000122.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Produktion(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	now := time.Now().UTC()

	orderID := testutil.SeedRow(t, pool, "production_orders", map[string]any{
		"tenant_id":     testutil.TenantA,
		"order_number":  "ORD-" + uuid.New().String()[:8],
		"product_name":  "Widget X",
		"quantity":      5,
		"planned_start": now,
		"planned_end":   now.Add(24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "production_orders", orderID)

	bookingID := testutil.SeedRow(t, pool, "machine_bookings", map[string]any{
		"tenant_id":           testutil.TenantA,
		"machine_id":          "MACHINE-01",
		"production_order_id": orderID,
		"starts_at":           now,
		"ends_at":             now.Add(2 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "machine_bookings", bookingID)

	planID := testutil.SeedRow(t, pool, "production_plans", map[string]any{
		"tenant_id":   testutil.TenantA,
		"name":        "Plan KW01",
		"week_number": 1,
		"year":        2026,
	})
	defer testutil.CleanupRow(t, pool, "production_plans", planID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"production_orders", "production_orders", orderID},
		{"machine_bookings", "machine_bookings", bookingID},
		{"production_plans", "production_plans", planID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
