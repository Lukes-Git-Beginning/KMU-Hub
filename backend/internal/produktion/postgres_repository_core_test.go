package produktion

// Covers the repository-side gaps in the order/booking/plan core: the advisory-lock
// booking path, the standalone conflict probe, real-filter listing, and capacity
// overview — all against the real PostgresRepository, since the overlap rule and the
// tenant scoping only exist in SQL (see fuhrpark/booking_conflict_test.go for the
// pattern this mirrors).

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedCoreOrder(t *testing.T, repo *PostgresRepository, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, orderNumber string) *ProductionOrder {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	order := &ProductionOrder{
		ID:           uuid.New(),
		TenantID:     tenantID,
		OrderNumber:  orderNumber,
		ProductName:  "Widget",
		Quantity:     5,
		Status:       OrderStatusPlanned,
		PlannedStart: now,
		PlannedEnd:   now.Add(24 * time.Hour),
		Priority:     3,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.CreateOrder(ctx, order); err != nil {
		t.Fatalf("seed order: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "production_orders", order.ID) })
	return order
}

func TestCreateBookingWithLock_HalfOpenAndConflict(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Produktion Core Lock Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	order := seedCoreOrder(t, repo, ctx, pool, tenantID, "LOCK-"+uuid.New().String()[:8])

	now := time.Now().UTC().Truncate(time.Second)
	newBooking := func(start, end time.Time) *MachineBooking {
		return &MachineBooking{
			ID: uuid.New(), TenantID: tenantID, MachineID: "M-LOCK-01",
			ProductionOrderID: order.ID, StartsAt: start, EndsAt: end,
			Status: BookingStatusBooked, CreatedAt: now, UpdatedAt: now,
		}
	}

	base := newBooking(now.Add(10*time.Hour), now.Add(12*time.Hour))
	conflict, err := repo.CreateBookingWithLock(ctx, base)
	if err != nil {
		t.Fatalf("CreateBookingWithLock (base): %v", err)
	}
	if conflict != nil {
		t.Fatalf("first booking must not conflict, got %s", conflict)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "machine_bookings", base.ID) })
	testutil.AssertRowCount(t, pool, ctx, "machine_bookings", base.ID, 1)

	t.Run("overlapping is rejected and rolled back", func(t *testing.T) {
		overlap := newBooking(now.Add(11*time.Hour), now.Add(13*time.Hour))
		got, insErr := repo.CreateBookingWithLock(ctx, overlap)
		if insErr != nil {
			t.Fatalf("CreateBookingWithLock: %v", insErr)
		}
		if got == nil || *got != base.ID {
			t.Fatalf("expected conflict with %s, got %v", base.ID, got)
		}
		// Rollback proof: the conflicting attempt must not have landed.
		testutil.AssertRowCount(t, pool, ctx, "machine_bookings", overlap.ID, 0)
	})

	t.Run("adjacent (half-open) is accepted", func(t *testing.T) {
		adjacent := newBooking(now.Add(12*time.Hour), now.Add(14*time.Hour))
		got, insErr := repo.CreateBookingWithLock(ctx, adjacent)
		if insErr != nil {
			t.Fatalf("CreateBookingWithLock: %v", insErr)
		}
		if got != nil {
			t.Fatalf("touching at the boundary must not conflict, got %s", got)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "machine_bookings", adjacent.ID) })
		testutil.AssertRowCount(t, pool, ctx, "machine_bookings", adjacent.ID, 1)
	})
}

func TestFindConflictingBooking_ExcludeIDAndCancelled(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Produktion Core Probe Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	order := seedCoreOrder(t, repo, ctx, pool, tenantID, "PROBE-"+uuid.New().String()[:8])

	now := time.Now().UTC().Truncate(time.Second)
	active := &MachineBooking{
		ID: uuid.New(), TenantID: tenantID, MachineID: "M-PROBE-01",
		ProductionOrderID: order.ID, StartsAt: now.Add(10 * time.Hour), EndsAt: now.Add(12 * time.Hour),
		Status: BookingStatusBooked, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateBooking(ctx, active); err != nil {
		t.Fatalf("seed active booking: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "machine_bookings", active.ID) })

	t.Run("excludeID lets a booking be moved onto itself", func(t *testing.T) {
		got, err := repo.FindConflictingBooking(ctx, tenantID, "M-PROBE-01", active.StartsAt, active.EndsAt, &active.ID)
		if err != nil {
			t.Fatalf("FindConflictingBooking: %v", err)
		}
		if got != nil {
			t.Fatalf("a booking must not conflict with itself when excluded, got %s", got)
		}
	})

	t.Run("without excludeID the same window still conflicts", func(t *testing.T) {
		got, err := repo.FindConflictingBooking(ctx, tenantID, "M-PROBE-01", active.StartsAt, active.EndsAt, nil)
		if err != nil {
			t.Fatalf("FindConflictingBooking: %v", err)
		}
		if got == nil || *got != active.ID {
			t.Fatalf("expected conflict with %s, got %v", active.ID, got)
		}
	})

	t.Run("a cancelled booking does not conflict", func(t *testing.T) {
		cancelled := &MachineBooking{
			ID: uuid.New(), TenantID: tenantID, MachineID: "M-PROBE-02",
			ProductionOrderID: order.ID, StartsAt: now.Add(20 * time.Hour), EndsAt: now.Add(22 * time.Hour),
			Status: BookingStatusCancelled, CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateBooking(ctx, cancelled); err != nil {
			t.Fatalf("seed cancelled booking: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "machine_bookings", cancelled.ID) })

		got, err := repo.FindConflictingBooking(ctx, tenantID, "M-PROBE-02", cancelled.StartsAt, cancelled.EndsAt, nil)
		if err != nil {
			t.Fatalf("FindConflictingBooking: %v", err)
		}
		if got != nil {
			t.Fatalf("a cancelled booking must not block, got %s", got)
		}
	})
}

func TestListOrders_FilterByStatus(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Produktion Core List Orders Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	planned := seedCoreOrder(t, repo, ctx, pool, tenantID, "LIST-PLANNED-"+uuid.New().String()[:8])
	inProgress := seedCoreOrder(t, repo, ctx, pool, tenantID, "LIST-INPROG-"+uuid.New().String()[:8])
	inProgress.Status = OrderStatusInProgress
	if err := repo.UpdateOrder(ctx, inProgress); err != nil {
		t.Fatalf("transition seeded order: %v", err)
	}

	status := OrderStatusInProgress
	results, total, err := repo.ListOrders(ctx, tenantID, ListOrdersFilter{Status: &status}, 0, 20)
	if err != nil {
		t.Fatalf("ListOrders: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected exactly one in_progress order for this tenant, got total=%d", total)
	}
	if len(results) != 1 || results[0].ID != inProgress.ID {
		t.Fatalf("expected only %s in the filtered result, got %+v", inProgress.ID, results)
	}
	for _, o := range results {
		if o.ID == planned.ID {
			t.Fatalf("status filter leaked the planned order into the in_progress result")
		}
	}
}

func TestListBookings_FilterByMachineID(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Produktion Core List Bookings Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	order := seedCoreOrder(t, repo, ctx, pool, tenantID, "LISTB-"+uuid.New().String()[:8])

	now := time.Now().UTC().Truncate(time.Second)
	wanted := &MachineBooking{
		ID: uuid.New(), TenantID: tenantID, MachineID: "M-WANTED",
		ProductionOrderID: order.ID, StartsAt: now, EndsAt: now.Add(2 * time.Hour),
		Status: BookingStatusBooked, CreatedAt: now, UpdatedAt: now,
	}
	other := &MachineBooking{
		ID: uuid.New(), TenantID: tenantID, MachineID: "M-OTHER",
		ProductionOrderID: order.ID, StartsAt: now, EndsAt: now.Add(2 * time.Hour),
		Status: BookingStatusBooked, CreatedAt: now, UpdatedAt: now,
	}
	for _, b := range []*MachineBooking{wanted, other} {
		if err := repo.CreateBooking(ctx, b); err != nil {
			t.Fatalf("seed booking: %v", err)
		}
		t.Cleanup(func(id uuid.UUID) func() { return func() { testutil.CleanupRow(t, pool, "machine_bookings", id) } }(b.ID))
	}

	machineID := "M-WANTED"
	results, total, err := repo.ListBookings(ctx, tenantID, ListBookingsFilter{MachineID: &machineID}, 0, 20)
	if err != nil {
		t.Fatalf("ListBookings: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected exactly one booking for M-WANTED, got total=%d", total)
	}
	if len(results) != 1 || results[0].ID != wanted.ID {
		t.Fatalf("expected only %s in the filtered result, got %+v", wanted.ID, results)
	}
}

func TestGetCapacityOverview_RealDB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Produktion Core Capacity Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	order := seedCoreOrder(t, repo, ctx, pool, tenantID, "CAP-"+uuid.New().String()[:8])

	now := time.Now().UTC()
	plan := &ProductionPlan{
		ID: uuid.New(), TenantID: tenantID, Name: "KW22 Capacity",
		WeekNumber: 22, Year: 2026, TotalCapacityHours: 40, PlannedCapacityHours: 32,
		Status: PlanStatusDraft, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreatePlan(ctx, plan); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "production_plans", plan.ID) })

	monday := time.Date(2026, 5, 25, 8, 0, 0, 0, time.UTC)
	booking := &MachineBooking{
		ID: uuid.New(), TenantID: tenantID, MachineID: "M-CAP-01",
		ProductionOrderID: order.ID, StartsAt: monday, EndsAt: monday.Add(8 * time.Hour),
		Status: BookingStatusBooked, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateBooking(ctx, booking); err != nil {
		t.Fatalf("seed booking: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "machine_bookings", booking.ID) })

	overview, err := repo.GetCapacityOverview(ctx, tenantID, "M-CAP-01", plan.ID)
	if err != nil {
		t.Fatalf("GetCapacityOverview: %v", err)
	}
	if overview.TotalCapacityHours != 40 {
		t.Fatalf("expected total 40, got %v", overview.TotalCapacityHours)
	}
	if overview.BookedHours != 8 {
		t.Fatalf("expected booked 8, got %v", overview.BookedHours)
	}
	if overview.AvailableHours != 32 {
		t.Fatalf("expected available 32, got %v", overview.AvailableHours)
	}

	t.Run("unknown plan surfaces ErrPlanNotFound", func(t *testing.T) {
		_, err := repo.GetCapacityOverview(ctx, tenantID, "M-CAP-01", uuid.New())
		if !errors.Is(err, ErrPlanNotFound) {
			t.Fatalf("expected ErrPlanNotFound, got %v", err)
		}
	})
}
