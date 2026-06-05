package schichten

// Cross-tenant isolation tests for schichten tables (Migration 000122).
// shifts, shift_assignments, shift_templates all have tenant_id NOT NULL
// and RLS was enabled in mig 000122.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Schichten(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	now := time.Now().UTC()

	shiftID := testutil.SeedRow(t, pool, "shifts", map[string]any{
		"tenant_id":  testutil.TenantA,
		"title":      "Morning Shift",
		"start_time": now,
		"end_time":   now.Add(8 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "shifts", shiftID)

	assignID := testutil.SeedRow(t, pool, "shift_assignments", map[string]any{
		"tenant_id":   testutil.TenantA,
		"shift_id":    shiftID,
		"employee_id": uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "shift_assignments", assignID)

	templateID := testutil.SeedRow(t, pool, "shift_templates", map[string]any{
		"tenant_id":        testutil.TenantA,
		"name":             "Standard Morning",
		"day_of_week":      1,
		"start_hour":       8,
		"start_minute":     0,
		"duration_minutes": 480,
	})
	defer testutil.CleanupRow(t, pool, "shift_templates", templateID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"shifts", "shifts", shiftID},
		{"shift_assignments", "shift_assignments", assignID},
		{"shift_templates", "shift_templates", templateID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
