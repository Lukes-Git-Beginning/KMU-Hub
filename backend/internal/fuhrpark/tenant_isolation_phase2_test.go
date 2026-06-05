package fuhrpark

// Cross-tenant isolation tests for fuhrpark tables (Migration 000122).
// vehicles, vehicle_services, vehicle_damages all have tenant_id NOT NULL
// and RLS was enabled in mig 000122.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Fuhrpark(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	vehicleID := testutil.SeedRow(t, pool, "vehicles", map[string]any{
		"tenant_id":     testutil.TenantA,
		"license_plate": "ZH-" + uuid.New().String()[:6],
		"make":          "VW",
		"model":         "Transporter",
		"year":          2022,
	})
	defer testutil.CleanupRow(t, pool, "vehicles", vehicleID)

	serviceID := testutil.SeedRow(t, pool, "vehicle_services", map[string]any{
		"tenant_id":    testutil.TenantA,
		"vehicle_id":   vehicleID,
		"service_type": "oil_change",
		"scheduled_at": time.Now().Format("2006-01-02"),
	})
	defer testutil.CleanupRow(t, pool, "vehicle_services", serviceID)

	damageID := testutil.SeedRow(t, pool, "vehicle_damages", map[string]any{
		"tenant_id":   testutil.TenantA,
		"vehicle_id":  vehicleID,
		"description": "Minor scratch on bumper",
	})
	defer testutil.CleanupRow(t, pool, "vehicle_damages", damageID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"vehicles", "vehicles", vehicleID},
		{"vehicle_services", "vehicle_services", serviceID},
		{"vehicle_damages", "vehicle_damages", damageID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
