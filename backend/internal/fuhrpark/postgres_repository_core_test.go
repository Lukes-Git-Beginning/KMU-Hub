package fuhrpark

// Covers the largest untested slice of postgres_repository.go: TUEV cron
// queries, soft-delete, list/filter, plate uniqueness, and the booking
// read/write remainder not already exercised by booking_conflict_test.go.
// FindVehiclesDueTuev is deliberately cross-tenant (the cron scans every
// tenant), so its assertions check for presence of specific seeded IDs
// rather than an exact result count, which would be flaky next to any other
// parallel test seeding a due date in the same window.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedVehicleWithTuev(t *testing.T, repo *PostgresRepository, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, dueDate *time.Time) *Vehicle {
	t.Helper()
	now := time.Now().UTC()
	v := &Vehicle{
		ID: uuid.New(), TenantID: tenantID,
		LicensePlate: "TV-" + uuid.New().String()[:6],
		Make:         "Ford", Model: "Transit", Year: 2022,
		FuelType: "diesel", Status: VehicleStatusActive,
		TuevDueDate: dueDate,
		CreatedAt:   now, UpdatedAt: now,
	}
	if err := repo.CreateVehicle(ctx, v); err != nil {
		t.Fatalf("seed vehicle: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicles", v.ID) })
	return v
}

func TestSoftDeleteVehicle_SetsDeletedAtAndIsIdempotentNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Soft Delete Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	v := seedVehicleWithTuev(t, repo, ctx, pool, tenantID, nil)

	if _, err := repo.GetVehicle(ctx, tenantID, v.ID); err != nil {
		t.Fatalf("GetVehicle before delete: %v", err)
	}

	if err := repo.SoftDeleteVehicle(ctx, tenantID, v.ID); err != nil {
		t.Fatalf("SoftDeleteVehicle: %v", err)
	}

	if _, err := repo.GetVehicle(ctx, tenantID, v.ID); !errors.Is(err, ErrVehicleNotFound) {
		t.Fatalf("GetVehicle after delete: expected ErrVehicleNotFound, got %v", err)
	}

	if err := repo.SoftDeleteVehicle(ctx, tenantID, v.ID); !errors.Is(err, ErrVehicleNotFound) {
		t.Fatalf("second SoftDeleteVehicle: expected ErrVehicleNotFound, got %v", err)
	}
}

func TestListVehicles_FiltersByStatus(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "List Vehicles Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	active := seedVehicleWithTuev(t, repo, ctx, pool, tenantID, nil)
	inactive := seedVehicleWithTuev(t, repo, ctx, pool, tenantID, nil)
	inactive.Status = VehicleStatusInactive
	if err := repo.UpdateVehicle(ctx, inactive); err != nil {
		t.Fatalf("UpdateVehicle (set inactive): %v", err)
	}

	statusActive := VehicleStatusActive
	vehicles, total, err := repo.ListVehicles(ctx, tenantID, ListVehiclesFilter{Status: &statusActive}, 0, 50)
	if err != nil {
		t.Fatalf("ListVehicles: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1 active vehicle, got %d", total)
	}
	if len(vehicles) != 1 || vehicles[0].ID != active.ID {
		t.Fatalf("expected only the active vehicle %s, got %+v", active.ID, vehicles)
	}

	// Search filter matches the seeded license plate.
	found, total, err := repo.ListVehicles(ctx, tenantID, ListVehiclesFilter{Search: active.LicensePlate}, 0, 50)
	if err != nil {
		t.Fatalf("ListVehicles (search): %v", err)
	}
	if total != 1 || len(found) != 1 || found[0].ID != active.ID {
		t.Fatalf("expected search to isolate %s, got total=%d results=%+v", active.ID, total, found)
	}
}

func TestPlateExists_ExcludeID(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Plate Exists Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	v := seedVehicleWithTuev(t, repo, ctx, pool, tenantID, nil)

	exists, err := repo.PlateExists(ctx, tenantID, v.LicensePlate, nil)
	if err != nil {
		t.Fatalf("PlateExists: %v", err)
	}
	if !exists {
		t.Fatalf("expected plate %q to exist", v.LicensePlate)
	}

	exists, err = repo.PlateExists(ctx, tenantID, v.LicensePlate, &v.ID)
	if err != nil {
		t.Fatalf("PlateExists (excludeID self): %v", err)
	}
	if exists {
		t.Fatalf("expected plate to not exist when excluding its own vehicle ID")
	}

	exists, err = repo.PlateExists(ctx, tenantID, "NOPE-"+uuid.New().String()[:6], nil)
	if err != nil {
		t.Fatalf("PlateExists (missing): %v", err)
	}
	if exists {
		t.Fatalf("expected a never-used plate to not exist")
	}
}

func containsVehicleID(vehicles []*Vehicle, id uuid.UUID) bool {
	for _, v := range vehicles {
		if v.ID == id {
			return true
		}
	}
	return false
}

func TestFindVehiclesDueTuev_WindowAndReminderIdempotency(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Tuev Window Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	from := time.Now().UTC().Truncate(24 * time.Hour)
	to := from.AddDate(0, 0, 6)

	atFrom := seedVehicleWithTuev(t, repo, ctx, pool, tenantID, &from)
	atTo := seedVehicleWithTuev(t, repo, ctx, pool, tenantID, &to)
	beforeWindow := from.AddDate(0, 0, -1)
	outsideBefore := seedVehicleWithTuev(t, repo, ctx, pool, tenantID, &beforeWindow)
	afterWindow := to.AddDate(0, 0, 1)
	outsideAfter := seedVehicleWithTuev(t, repo, ctx, pool, tenantID, &afterWindow)

	t.Run("window boundaries are inclusive", func(t *testing.T) {
		due, err := repo.FindVehiclesDueTuev(ctx, from, to)
		if err != nil {
			t.Fatalf("FindVehiclesDueTuev: %v", err)
		}
		if !containsVehicleID(due, atFrom.ID) {
			t.Fatalf("expected vehicle at from-boundary %s to be included", atFrom.ID)
		}
		if !containsVehicleID(due, atTo.ID) {
			t.Fatalf("expected vehicle at to-boundary %s to be included", atTo.ID)
		}
		if containsVehicleID(due, outsideBefore.ID) {
			t.Fatalf("vehicle before the window %s must not be included", outsideBefore.ID)
		}
		if containsVehicleID(due, outsideAfter.ID) {
			t.Fatalf("vehicle after the window %s must not be included", outsideAfter.ID)
		}
	})

	t.Run("a fresh reminder suppresses re-notification", func(t *testing.T) {
		recentlyReminded := seedVehicleWithTuev(t, repo, ctx, pool, tenantID, &from)
		if err := repo.MarkTuevReminderSent(ctx, recentlyReminded.ID, tenantID); err != nil {
			t.Fatalf("MarkTuevReminderSent: %v", err)
		}

		due, err := repo.FindVehiclesDueTuev(ctx, from, to)
		if err != nil {
			t.Fatalf("FindVehiclesDueTuev: %v", err)
		}
		if containsVehicleID(due, recentlyReminded.ID) {
			t.Fatalf("vehicle reminded within the last 23h %s must not reappear", recentlyReminded.ID)
		}
	})

	t.Run("a stale reminder allows re-notification", func(t *testing.T) {
		staleReminder := seedVehicleWithTuev(t, repo, ctx, pool, tenantID, &from)
		if err := repo.MarkTuevReminderSent(ctx, staleReminder.ID, tenantID); err != nil {
			t.Fatalf("MarkTuevReminderSent: %v", err)
		}
		if _, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
			`UPDATE vehicles SET tuev_reminder_sent_at = NOW() - INTERVAL '25 hours' WHERE id = $1`,
			staleReminder.ID); err != nil {
			t.Fatalf("backdate reminder timestamp: %v", err)
		}

		due, err := repo.FindVehiclesDueTuev(ctx, from, to)
		if err != nil {
			t.Fatalf("FindVehiclesDueTuev: %v", err)
		}
		if !containsVehicleID(due, staleReminder.ID) {
			t.Fatalf("vehicle with a >23h old reminder %s must be eligible again", staleReminder.ID)
		}
	})
}

func TestMarkTuevReminderSent_CrossTenantGuard(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Tuev Guard Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Tuev Guard Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)

	v := seedVehicleWithTuev(t, repo, ctxA, pool, tenantA, nil)

	if err := repo.MarkTuevReminderSent(ctxA, v.ID, tenantB); !errors.Is(err, ErrVehicleNotFound) {
		t.Fatalf("expected ErrVehicleNotFound when stamping under the wrong tenant, got %v", err)
	}

	got, err := repo.GetVehicle(ctxA, tenantA, v.ID)
	if err != nil {
		t.Fatalf("GetVehicle: %v", err)
	}
	if got.TuevReminderSentAt != nil {
		t.Fatalf("cross-tenant MarkTuevReminderSent must not have stamped the vehicle, got %v", got.TuevReminderSentAt)
	}
}

func TestBookingCore_GetUpdateDeleteList(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Booking Core Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Booking Core Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC().Truncate(time.Second)

	vehicleA := seedBookingVehicle(t, repo, ctxA, pool, tenantA)
	userA := seedBookingUser(t, pool, tenantA, "booking-core-a")

	booking := &VehicleBooking{
		ID: uuid.New(), TenantID: tenantA, VehicleID: vehicleA, UserID: userA,
		StartsAt: now.Add(48 * time.Hour), EndsAt: now.Add(50 * time.Hour),
		Purpose: "Kundentermin", Status: BookingStatusBooked,
		CreatedAt: now, UpdatedAt: now,
	}
	if conflict, err := repo.CreateBookingWithLock(ctxA, booking); err != nil || conflict != nil {
		t.Fatalf("CreateBookingWithLock: conflict=%v err=%v", conflict, err)
	}
	defer testutil.CleanupRow(t, pool, "vehicle_bookings", booking.ID)

	t.Run("GetBooking finds the row and stays tenant scoped", func(t *testing.T) {
		got, err := repo.GetBooking(ctxA, tenantA, booking.ID)
		if err != nil {
			t.Fatalf("GetBooking: %v", err)
		}
		if got.Purpose != "Kundentermin" || got.Status != BookingStatusBooked {
			t.Fatalf("unexpected booking contents: %+v", got)
		}

		if _, err := repo.GetBooking(ctxA, tenantA, uuid.New()); !errors.Is(err, ErrBookingNotFound) {
			t.Fatalf("expected ErrBookingNotFound for unknown ID, got %v", err)
		}
		if _, err := repo.GetBooking(ctxB, tenantB, booking.ID); !errors.Is(err, ErrBookingNotFound) {
			t.Fatalf("expected ErrBookingNotFound across tenants, got %v", err)
		}
	})

	t.Run("UpdateBooking changes fields and stays tenant scoped", func(t *testing.T) {
		updated := *booking
		updated.Purpose = "Werkstatt"
		updated.Status = BookingStatusInUse
		updated.UpdatedAt = now.Add(time.Minute)
		if err := repo.UpdateBooking(ctxA, &updated); err != nil {
			t.Fatalf("UpdateBooking: %v", err)
		}

		got, err := repo.GetBooking(ctxA, tenantA, booking.ID)
		if err != nil {
			t.Fatalf("GetBooking after update: %v", err)
		}
		if got.Purpose != "Werkstatt" || got.Status != BookingStatusInUse {
			t.Fatalf("update did not persist, got %+v", got)
		}

		wrongTenant := *booking
		wrongTenant.TenantID = tenantB
		wrongTenant.Purpose = "Should not land"
		if err := repo.UpdateBooking(ctxB, &wrongTenant); !errors.Is(err, ErrBookingNotFound) {
			t.Fatalf("expected ErrBookingNotFound updating under the wrong tenant, got %v", err)
		}
		unchanged, err := repo.GetBooking(ctxA, tenantA, booking.ID)
		if err != nil {
			t.Fatalf("GetBooking after failed cross-tenant update: %v", err)
		}
		if unchanged.Purpose != "Werkstatt" {
			t.Fatalf("cross-tenant update must not have modified the row, got purpose %q", unchanged.Purpose)
		}
	})

	t.Run("ListBookings filters by vehicle, status and window", func(t *testing.T) {
		byVehicle, total, err := repo.ListBookings(ctxA, ListVehicleBookingsParams{TenantID: tenantA, VehicleID: vehicleA})
		if err != nil {
			t.Fatalf("ListBookings (vehicle filter): %v", err)
		}
		if total < 1 || !containsBookingID(byVehicle, booking.ID) {
			t.Fatalf("expected booking %s in vehicle-filtered list, got total=%d list=%+v", booking.ID, total, byVehicle)
		}

		wrongStatus := BookingStatusCompleted
		byStatus, _, err := repo.ListBookings(ctxA, ListVehicleBookingsParams{TenantID: tenantA, VehicleID: vehicleA, Status: &wrongStatus})
		if err != nil {
			t.Fatalf("ListBookings (status filter): %v", err)
		}
		if containsBookingID(byStatus, booking.ID) {
			t.Fatalf("booking with status in_use must not match a completed-status filter")
		}

		outOfWindowFrom := now.Add(200 * time.Hour)
		byWindow, _, err := repo.ListBookings(ctxA, ListVehicleBookingsParams{TenantID: tenantA, VehicleID: vehicleA, From: &outOfWindowFrom})
		if err != nil {
			t.Fatalf("ListBookings (window filter): %v", err)
		}
		if containsBookingID(byWindow, booking.ID) {
			t.Fatalf("booking ending before the window's From must not match")
		}

		testutil.AssertRowCount(t, pool, ctxB, "vehicle_bookings", booking.ID, 0)
	})

	t.Run("DeleteBooking is tenant scoped and idempotent-not-found", func(t *testing.T) {
		if err := repo.DeleteBooking(ctxB, tenantB, booking.ID); !errors.Is(err, ErrBookingNotFound) {
			t.Fatalf("expected ErrBookingNotFound deleting under the wrong tenant, got %v", err)
		}
		testutil.AssertRowCount(t, pool, ctxA, "vehicle_bookings", booking.ID, 1)

		if err := repo.DeleteBooking(ctxA, tenantA, booking.ID); err != nil {
			t.Fatalf("DeleteBooking: %v", err)
		}
		testutil.AssertRowCount(t, pool, ctxA, "vehicle_bookings", booking.ID, 0)

		if err := repo.DeleteBooking(ctxA, tenantA, booking.ID); !errors.Is(err, ErrBookingNotFound) {
			t.Fatalf("second DeleteBooking: expected ErrBookingNotFound, got %v", err)
		}
	})
}

func containsBookingID(bookings []*VehicleBooking, id uuid.UUID) bool {
	for _, b := range bookings {
		if b.ID == id {
			return true
		}
	}
	return false
}
