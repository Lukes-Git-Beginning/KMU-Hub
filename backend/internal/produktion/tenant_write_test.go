package produktion

// Exercises the real write path (PostgresRepository, not testutil.SeedRow) so a
// forgotten tenant_id column would show up as a genuine INSERT/constraint failure
// instead of being silently bypassed by a system-context fixture. Complements
// tenant_isolation_phase2_test.go, which only proves RLS SELECT filtering on
// hand-seeded rows.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestProduktionWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantWrite, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantWrite, "Produktion Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Produktion Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantWrite)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	order := &ProductionOrder{
		ID:           uuid.New(),
		TenantID:     tenantWrite,
		OrderNumber:  "ORD-" + uuid.New().String()[:8],
		ProductName:  "Widget X",
		Quantity:     5,
		Status:       OrderStatusPlanned,
		PlannedStart: now,
		PlannedEnd:   now.Add(24 * time.Hour),
		Priority:     3,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.CreateOrder(ctxA, order); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "production_orders", order.ID)

	booking := &MachineBooking{
		ID:                uuid.New(),
		TenantID:          tenantWrite,
		MachineID:         "MACHINE-01",
		ProductionOrderID: order.ID,
		StartsAt:          now,
		EndsAt:            now.Add(2 * time.Hour),
		Status:            BookingStatusBooked,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := repo.CreateBooking(ctxA, booking); err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "machine_bookings", booking.ID)

	plan := &ProductionPlan{
		ID:                   uuid.New(),
		TenantID:             tenantWrite,
		Name:                 "Plan KW01",
		WeekNumber:           1,
		Year:                 2026,
		TotalCapacityHours:   40,
		PlannedCapacityHours: 32,
		Status:               PlanStatusDraft,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := repo.CreatePlan(ctxA, plan); err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "production_plans", plan.ID)

	rows := []struct {
		table string
		id    uuid.UUID
	}{
		{"production_orders", order.ID},
		{"machine_bookings", booking.ID},
		{"production_plans", plan.ID},
	}

	for _, row := range rows {
		t.Run(row.table, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, row.table, row.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, row.table, row.id, 0)
		})
	}
}
