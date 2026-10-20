package produktion

// The machine_bookings_no_overlap exclusion constraint (000328) is the backstop for booking
// writes that bypass the advisory lock in CreateBookingWithLock. These tests write directly
// through the repository — skipping the application-side conflict check, exactly as a racing
// request would — and prove the database rejects the overlap and the repository maps it.

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestMachineBookingOverlapConstraint(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Produktion Overlap Constraint Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	order := seedCoreOrder(t, repo, ctx, pool, tenantID, "EXCL-"+uuid.New().String()[:8])

	now := time.Now().UTC().Truncate(time.Second)
	seed := func(t *testing.T, machineID string, start, end time.Time, status BookingStatus) *MachineBooking {
		t.Helper()
		b := &MachineBooking{
			ID: uuid.New(), TenantID: tenantID, MachineID: machineID,
			ProductionOrderID: order.ID, StartsAt: start, EndsAt: end,
			Status: status, CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateBooking(ctx, b); err != nil {
			t.Fatalf("seed booking: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "machine_bookings", b.ID) })
		return b
	}

	t.Run("unlocked insert onto an occupied window maps to ErrBookingConflict", func(t *testing.T) {
		seed(t, "M-EXCL-01", now.Add(10*time.Hour), now.Add(12*time.Hour), BookingStatusBooked)
		overlap := &MachineBooking{
			ID: uuid.New(), TenantID: tenantID, MachineID: "M-EXCL-01",
			ProductionOrderID: order.ID, StartsAt: now.Add(11 * time.Hour), EndsAt: now.Add(13 * time.Hour),
			Status: BookingStatusBooked, CreatedAt: now, UpdatedAt: now,
		}
		err := repo.CreateBooking(ctx, overlap)
		if !errors.Is(err, ErrBookingConflict) {
			t.Fatalf("expected ErrBookingConflict, got %v", err)
		}
		testutil.AssertRowCount(t, pool, ctx, "machine_bookings", overlap.ID, 0)
	})

	t.Run("moving a booking onto an occupied window maps to ErrBookingConflict", func(t *testing.T) {
		seed(t, "M-EXCL-02", now.Add(10*time.Hour), now.Add(12*time.Hour), BookingStatusInUse)
		mover := seed(t, "M-EXCL-02", now.Add(20*time.Hour), now.Add(22*time.Hour), BookingStatusBooked)

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
		seed(t, "M-EXCL-03", now.Add(10*time.Hour), now.Add(12*time.Hour), BookingStatusBooked)
		cancelled := seed(t, "M-EXCL-03", now.Add(10*time.Hour), now.Add(12*time.Hour), BookingStatusCancelled)

		reactivated := *cancelled
		reactivated.Status = BookingStatusBooked
		if err := repo.UpdateBooking(ctx, &reactivated); !errors.Is(err, ErrBookingConflict) {
			t.Fatalf("expected ErrBookingConflict, got %v", err)
		}
	})

	t.Run("adjacent windows and other machines stay allowed", func(t *testing.T) {
		seed(t, "M-EXCL-04", now.Add(10*time.Hour), now.Add(12*time.Hour), BookingStatusBooked)
		seed(t, "M-EXCL-04", now.Add(12*time.Hour), now.Add(14*time.Hour), BookingStatusBooked)
		seed(t, "M-EXCL-05", now.Add(10*time.Hour), now.Add(12*time.Hour), BookingStatusBooked)
	})

	// Constraints are checked regardless of RLS, so tenant_id must be part of the key:
	// another tenant's identically named machine is a different machine.
	t.Run("same machine and window in another tenant stays allowed", func(t *testing.T) {
		seed(t, "M-EXCL-07", now.Add(10*time.Hour), now.Add(12*time.Hour), BookingStatusBooked)

		otherTenant := uuid.New()
		testutil.EnsureTenant(t, pool, otherTenant, "Produktion Overlap Constraint Other Tenant")
		otherCtx := testutil.WithTenantCtx(context.Background(), otherTenant)
		otherOrder := seedCoreOrder(t, repo, otherCtx, pool, otherTenant, "EXCL-"+uuid.New().String()[:8])
		other := &MachineBooking{
			ID: uuid.New(), TenantID: otherTenant, MachineID: "M-EXCL-07",
			ProductionOrderID: otherOrder.ID, StartsAt: now.Add(10 * time.Hour), EndsAt: now.Add(12 * time.Hour),
			Status: BookingStatusBooked, CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateBooking(otherCtx, other); err != nil {
			t.Fatalf("other tenant's booking must not conflict: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "machine_bookings", other.ID) })
	})

	t.Run("two concurrent moves onto the same window: exactly one wins", func(t *testing.T) {
		a := seed(t, "M-EXCL-06", now.Add(10*time.Hour), now.Add(11*time.Hour), BookingStatusBooked)
		b := seed(t, "M-EXCL-06", now.Add(20*time.Hour), now.Add(21*time.Hour), BookingStatusBooked)

		target := now.Add(30 * time.Hour)
		errs := make([]error, 2)
		var wg sync.WaitGroup
		for i, orig := range []*MachineBooking{a, b} {
			wg.Add(1)
			go func(i int, moved MachineBooking) {
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
