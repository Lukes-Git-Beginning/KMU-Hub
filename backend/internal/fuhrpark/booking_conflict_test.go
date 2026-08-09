package fuhrpark

// The overlap rule lives in SQL, so it can only be proven against a real
// database. Two properties matter and neither is visible in a mock:
//
//   1. Half-open intervals. A booking that starts exactly when another ends
//      does NOT conflict -- handing the keys over at 12:00 has to stay
//      possible. An inclusive comparison would reject it.
//   2. Tenant scoping of the conflict probe itself. RLS already hides foreign
//      rows from SELECT, but a probe written without tenant_id would still be
//      one dropped policy away from letting a foreign tenant block a vehicle
//      it cannot even see.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedBookingVehicle(t *testing.T, repo *PostgresRepository, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	now := time.Now().UTC()
	v := &Vehicle{
		ID: uuid.New(), TenantID: tenantID,
		LicensePlate: "BK-" + uuid.New().String()[:6],
		Make:         "VW", Model: "Caddy", Year: 2023,
		FuelType: "diesel", Status: VehicleStatusActive,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateVehicle(ctx, v); err != nil {
		t.Fatalf("seed vehicle: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicles", v.ID) })
	return v.ID
}

func seedBookingUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, label string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name)
		 VALUES ($1, $2, $3, 'x', 'Booking', 'User')`,
		id, tenantID, label+"-"+id.String()+"@test.local"); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
	return id
}

func TestBookingConflict_HalfOpenAndTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// t.Cleanup, not defer: the seed helpers register their row cleanups the
	// same way, and t.Cleanup runs after every defer in this function. Closing
	// the pool with defer would pull it out from under them.
	t.Cleanup(pool.Close)

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Booking Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Booking Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC().Truncate(time.Second)

	// Both tenants get their own vehicle and driver.
	vehicleA := seedBookingVehicle(t, repo, ctxA, pool, tenantA)
	vehicleB := seedBookingVehicle(t, repo, ctxB, pool, tenantB)
	userA := seedBookingUser(t, pool, tenantA, "booking-a")
	userB := seedBookingUser(t, pool, tenantB, "booking-b")

	newBooking := func(tenantID, vehicleID, userID uuid.UUID, start, end time.Time) *VehicleBooking {
		return &VehicleBooking{
			ID: uuid.New(), TenantID: tenantID, VehicleID: vehicleID, UserID: userID,
			StartsAt: start, EndsAt: end, Purpose: "test", Status: BookingStatusBooked,
			CreatedAt: now, UpdatedAt: now,
		}
	}

	// 10:00-12:00 in tenant A.
	base := newBooking(tenantA, vehicleA, userA, now.Add(10*time.Hour), now.Add(12*time.Hour))
	conflict, err := repo.CreateBookingWithLock(ctxA, base)
	if err != nil {
		t.Fatalf("CreateBookingWithLock (base): %v", err)
	}
	if conflict != nil {
		t.Fatalf("first booking must not conflict, got %s", conflict)
	}
	defer testutil.CleanupRow(t, pool, "vehicle_bookings", base.ID)

	t.Run("overlapping is rejected", func(t *testing.T) {
		overlap := newBooking(tenantA, vehicleA, userA, now.Add(11*time.Hour), now.Add(13*time.Hour))
		got, insErr := repo.CreateBookingWithLock(ctxA, overlap)
		if insErr != nil {
			t.Fatalf("CreateBookingWithLock: %v", insErr)
		}
		if got == nil || *got != base.ID {
			t.Fatalf("expected conflict with %s, got %v", base.ID, got)
		}
		testutil.AssertRowCount(t, pool, ctxA, "vehicle_bookings", overlap.ID, 0)
	})

	t.Run("adjacent is accepted", func(t *testing.T) {
		adjacent := newBooking(tenantA, vehicleA, userA, now.Add(12*time.Hour), now.Add(14*time.Hour))
		got, insErr := repo.CreateBookingWithLock(ctxA, adjacent)
		if insErr != nil {
			t.Fatalf("CreateBookingWithLock: %v", insErr)
		}
		if got != nil {
			t.Fatalf("touching at the boundary must not conflict, got %s", got)
		}
		defer testutil.CleanupRow(t, pool, "vehicle_bookings", adjacent.ID)
		testutil.AssertRowCount(t, pool, ctxA, "vehicle_bookings", adjacent.ID, 1)
	})

	t.Run("a released booking frees the slot", func(t *testing.T) {
		cancelled := newBooking(tenantA, vehicleA, userA, now.Add(20*time.Hour), now.Add(22*time.Hour))
		cancelled.Status = BookingStatusCancelled
		if _, insErr := repo.CreateBookingWithLock(ctxA, cancelled); insErr != nil {
			t.Fatalf("CreateBookingWithLock: %v", insErr)
		}
		defer testutil.CleanupRow(t, pool, "vehicle_bookings", cancelled.ID)

		got, findErr := repo.FindConflictingBooking(ctxA, tenantA, vehicleA, now.Add(20*time.Hour), now.Add(22*time.Hour), nil)
		if findErr != nil {
			t.Fatalf("FindConflictingBooking: %v", findErr)
		}
		if got != nil {
			t.Fatalf("a cancelled booking must not block, got %s", got)
		}
	})

	t.Run("conflict probe is tenant scoped", func(t *testing.T) {
		// Same wall clock, other tenant, other vehicle: nothing may be found.
		got, findErr := repo.FindConflictingBooking(ctxB, tenantB, vehicleB, now.Add(10*time.Hour), now.Add(12*time.Hour), nil)
		if findErr != nil {
			t.Fatalf("FindConflictingBooking: %v", findErr)
		}
		if got != nil {
			t.Fatalf("tenant B must not see tenant A's booking, got %s", got)
		}

		// And the write path agrees: tenant B books the same window.
		theirs := newBooking(tenantB, vehicleB, userB, now.Add(10*time.Hour), now.Add(12*time.Hour))
		conflictB, insErr := repo.CreateBookingWithLock(ctxB, theirs)
		if insErr != nil {
			t.Fatalf("CreateBookingWithLock (tenant B): %v", insErr)
		}
		if conflictB != nil {
			t.Fatalf("tenant A's booking must not block tenant B, got %s", conflictB)
		}
		defer testutil.CleanupRow(t, pool, "vehicle_bookings", theirs.ID)

		// Cross-tenant read stays empty in both directions.
		testutil.AssertRowCount(t, pool, ctxA, "vehicle_bookings", theirs.ID, 0)
		testutil.AssertRowCount(t, pool, ctxB, "vehicle_bookings", base.ID, 0)
	})

	t.Run("excludeID lets a booking be moved onto itself", func(t *testing.T) {
		got, findErr := repo.FindConflictingBooking(ctxA, tenantA, vehicleA, base.StartsAt, base.EndsAt, &base.ID)
		if findErr != nil {
			t.Fatalf("FindConflictingBooking: %v", findErr)
		}
		if got != nil {
			t.Fatalf("a booking must not conflict with itself when excluded, got %s", got)
		}
	})

	t.Run("booking a user from another tenant is refused", func(t *testing.T) {
		foreign := newBooking(tenantA, vehicleA, userB, now.Add(30*time.Hour), now.Add(31*time.Hour))
		_, insErr := repo.CreateBookingWithLock(ctxA, foreign)
		if !errors.Is(insErr, ErrDriverNotFound) {
			t.Fatalf("expected ErrDriverNotFound, got %v", insErr)
		}
		testutil.AssertRowCount(t, pool, ctxA, "vehicle_bookings", foreign.ID, 0)
	})
}
