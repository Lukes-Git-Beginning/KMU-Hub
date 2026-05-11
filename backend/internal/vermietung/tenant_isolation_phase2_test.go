package vermietung

// Cross-tenant isolation tests for vermietung tables (Migration 000122).
// rental_objects, rentals, rental_inspections all have tenant_id NOT NULL
// and RLS was enabled in mig 000122.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Vermietung(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	now := time.Now().UTC()

	objectID := testutil.SeedRow(t, pool, "rental_objects", map[string]any{
		"tenant_id": testutil.TenantA,
		"name":      "Scaffolding Set " + uuid.New().String()[:6],
	})
	defer testutil.CleanupRow(t, pool, "rental_objects", objectID)

	rentalID := testutil.SeedRow(t, pool, "rentals", map[string]any{
		"tenant_id":  testutil.TenantA,
		"object_id":  objectID,
		"start_date": now,
		"end_date":   now.Add(7 * 24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "rentals", rentalID)

	inspID := testutil.SeedRow(t, pool, "rental_inspections", map[string]any{
		"tenant_id": testutil.TenantA,
		"rental_id": rentalID,
		"kind":      "handover",
	})
	defer testutil.CleanupRow(t, pool, "rental_inspections", inspID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"rental_objects", "rental_objects", objectID},
		{"rentals", "rentals", rentalID},
		{"rental_inspections", "rental_inspections", inspID},
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
