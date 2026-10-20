package fuhrpark

// The vehicle_bookings_no_overlap exclusion constraint (000329) is the backstop for booking
// writes that bypass the advisory lock in CreateBookingWithLock. UpdateBooking never takes
// that lock, and CreateBookingWithLock's in-transaction probe cannot see an uncommitted move
// from another transaction. These tests reproduce both races against a real database and
// prove the constraint rejects the overlap and the repository maps it to ErrBookingConflict.

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestVehicleBookingOverlapConstraint(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Fuhrpark Overlap Constraint Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	userID := seedBookingUser(t, pool, tenantID, "booking-excl")
	now := time.Now().UTC().Truncate(time.Second)

	newBooking := func(vehicleID uuid.UUID, start, end time.Time, status BookingStatus) *VehicleBooking {
		return &VehicleBooking{
			ID: uuid.New(), TenantID: tenantID, VehicleID: vehicleID, UserID: userID,
			StartsAt: start, EndsAt: end, Purpose: "test", Status: status,
			CreatedAt: now, UpdatedAt: now,
		}
	}
	// seed inserts through the locked path. Its probe ignores released bookings, so an active
	// booking can be placed onto the window of a cancelled one (not the other way round).
	seed := func(t *testing.T, vehicleID uuid.UUID, start, end time.Time, status BookingStatus) *VehicleBooking {
		t.Helper()
		b := newBooking(vehicleID, start, end, status)
		conflictID, err := repo.CreateBookingWithLock(ctx, b)
		if err != nil || conflictID != nil {
			t.Fatalf("seed booking: err=%v conflict=%v", err, conflictID)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicle_bookings", b.ID) })
		return b
	}

	t.Run("moving a booking onto an occupied window maps to ErrBookingConflict", func(t *testing.T) {
		vehicleID := seedBookingVehicle(t, repo, ctx, pool, tenantID)
		seed(t, vehicleID, now.Add(10*time.Hour), now.Add(12*time.Hour), BookingStatusInUse)
		mover := seed(t, vehicleID, now.Add(20*time.Hour), now.Add(22*time.Hour), BookingStatusBooked)

		moved := *mover
		moved.StartsAt, moved.EndsAt = now.Add(11*time.Hour), now.Add(13*time.Hour)
		if err := repo.UpdateBooking(ctx, &moved); !errors.Is(err, ErrBookingConflict) {
			t.Fatalf("expected ErrBookingConflict, got %v", err)
		}
		got, err := repo.GetBooking(ctx, tenantID, mover.ID)
		if err != nil {
			t.Fatalf("GetBooking: %v", err)
		}
		if !got.StartsAt.Equal(mover.StartsAt) {
			t.Fatalf("rejected move must leave the booking in place, starts_at now %s", got.StartsAt)
		}
	})

	t.Run("reactivating a cancelled booking onto an occupied window is rejected", func(t *testing.T) {
		vehicleID := seedBookingVehicle(t, repo, ctx, pool, tenantID)
		cancelled := seed(t, vehicleID, now.Add(10*time.Hour), now.Add(12*time.Hour), BookingStatusCancelled)
		seed(t, vehicleID, now.Add(10*time.Hour), now.Add(12*time.Hour), BookingStatusBooked)

		reactivated := *cancelled
		reactivated.Status = BookingStatusBooked
		if err := repo.UpdateBooking(ctx, &reactivated); !errors.Is(err, ErrBookingConflict) {
			t.Fatalf("expected ErrBookingConflict, got %v", err)
		}
	})

	t.Run("adjacent windows and other vehicles stay allowed", func(t *testing.T) {
		vehicleID := seedBookingVehicle(t, repo, ctx, pool, tenantID)
		otherVehicle := seedBookingVehicle(t, repo, ctx, pool, tenantID)
		seed(t, vehicleID, now.Add(10*time.Hour), now.Add(12*time.Hour), BookingStatusBooked)
		mover := seed(t, vehicleID, now.Add(20*time.Hour), now.Add(21*time.Hour), BookingStatusBooked)
		seed(t, otherVehicle, now.Add(10*time.Hour), now.Add(12*time.Hour), BookingStatusBooked)

		adjacent := *mover
		adjacent.StartsAt, adjacent.EndsAt = now.Add(12*time.Hour), now.Add(14*time.Hour)
		if err := repo.UpdateBooking(ctx, &adjacent); err != nil {
			t.Fatalf("adjacent window must stay allowed: %v", err)
		}
	})

	// The probe inside CreateBookingWithLock reads committed rows only. A move that is still
	// uncommitted when the probe runs is invisible to it; the insert then waits on the
	// exclusion constraint and must fail once the move commits.
	t.Run("insert racing an uncommitted move maps to ErrBookingConflict", func(t *testing.T) {
		vehicleID := seedBookingVehicle(t, repo, ctx, pool, tenantID)
		mover := seed(t, vehicleID, now.Add(20*time.Hour), now.Add(21*time.Hour), BookingStatusBooked)

		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck // no-op once committed
		if _, err := tx.Exec(ctx,
			`UPDATE vehicle_bookings SET starts_at = $3, ends_at = $4 WHERE id = $1 AND tenant_id = $2`,
			mover.ID, tenantID, now.Add(30*time.Hour), now.Add(32*time.Hour)); err != nil {
			t.Fatalf("uncommitted move: %v", err)
		}

		racer := newBooking(vehicleID, now.Add(31*time.Hour), now.Add(33*time.Hour), BookingStatusBooked)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicle_bookings", racer.ID) })
		type result struct {
			conflictID *uuid.UUID
			err        error
		}
		done := make(chan result, 1)
		go func() {
			conflictID, err := repo.CreateBookingWithLock(ctx, racer)
			done <- result{conflictID, err}
		}()

		waitForBlockedInsert(t, pool, ctx, done)
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit move: %v", err)
		}

		res := <-done
		if res.conflictID != nil || !errors.Is(res.err, ErrBookingConflict) {
			t.Fatalf("expected ErrBookingConflict, got conflict=%v err=%v", res.conflictID, res.err)
		}
		testutil.AssertRowCount(t, pool, ctx, "vehicle_bookings", racer.ID, 0)
	})

	t.Run("two concurrent moves onto the same window: exactly one wins", func(t *testing.T) {
		vehicleID := seedBookingVehicle(t, repo, ctx, pool, tenantID)
		a := seed(t, vehicleID, now.Add(10*time.Hour), now.Add(11*time.Hour), BookingStatusBooked)
		b := seed(t, vehicleID, now.Add(20*time.Hour), now.Add(21*time.Hour), BookingStatusBooked)

		target := now.Add(40 * time.Hour)
		errs := make([]error, 2)
		var wg sync.WaitGroup
		for i, orig := range []*VehicleBooking{a, b} {
			wg.Add(1)
			go func(i int, moved VehicleBooking) {
				defer wg.Done()
				moved.StartsAt, moved.EndsAt = target, target.Add(2*time.Hour)
				errs[i] = repo.UpdateBooking(ctx, &moved)
			}(i, *orig)
		}
		wg.Wait()

		wins, conflicts := 0, 0
		for _, err := range errs {
			switch {
			case err == nil:
				wins++
			case errors.Is(err, ErrBookingConflict):
				conflicts++
			default:
				t.Fatalf("unexpected error: %v", err)
			}
		}
		if wins != 1 || conflicts != 1 {
			t.Fatalf("expected 1 win and 1 conflict, got %d wins / %d conflicts", wins, conflicts)
		}
	})
}

// waitForBlockedInsert polls pg_stat_activity until an INSERT into vehicle_bookings waits on a
// lock, i.e. the racing insert has passed its probe and is held by the exclusion constraint.
// Fails early if the insert returned without blocking.
func waitForBlockedInsert[T any](t *testing.T, pool *pgxpool.Pool, ctx context.Context, done <-chan T) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case r := <-done:
			t.Fatalf("insert returned before the move committed: %+v", r)
		default:
		}
		var waiting int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM pg_stat_activity
			 WHERE wait_event_type = 'Lock' AND query LIKE '%INSERT INTO vehicle_bookings%'`,
		).Scan(&waiting); err != nil {
			t.Fatalf("poll pg_stat_activity: %v", err)
		}
		if waiting > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("racing insert never blocked on the exclusion constraint")
}
