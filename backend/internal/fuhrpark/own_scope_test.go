package fuhrpark

// Own-scope support for the Fuhrpark self-service keys (migration 339): the gateway narrows a
// member to their own bookings, license checks and reported damages. The service has to refuse a
// foreign record as NotFound (never Forbidden: that would confirm it exists) and the damage list
// has to filter in SQL, so total and page boundaries describe only the caller's rows.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestFuhrparkOwnScope_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Fuhrpark Own Scope Tenant")
	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	owner := seedBookingUser(t, pool, tenantID, "own-scope-owner")
	stranger := seedBookingUser(t, pool, tenantID, "own-scope-stranger")
	vehicleID := seedBookingVehicle(t, repo, ctx, pool, tenantID)
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("booking: only the owner passes EnsureBookingOwner", func(t *testing.T) {
		b := &VehicleBooking{
			ID: uuid.New(), TenantID: tenantID, VehicleID: vehicleID, UserID: owner,
			StartsAt: now.Add(48 * time.Hour), EndsAt: now.Add(50 * time.Hour),
			Status: BookingStatusBooked, CreatedAt: now, UpdatedAt: now,
		}
		if conflictID, err := repo.CreateBookingWithLock(ctx, b); err != nil || conflictID != nil {
			t.Fatalf("seed booking: err=%v conflict=%v", err, conflictID)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicle_bookings", b.ID) })

		if err := svc.EnsureBookingOwner(ctx, tenantID, b.ID, owner); err != nil {
			t.Fatalf("owner must pass: %v", err)
		}
		if err := svc.EnsureBookingOwner(ctx, tenantID, b.ID, stranger); !errors.Is(err, ErrBookingNotFound) {
			t.Fatalf("a foreign booking must be ErrBookingNotFound, got %v", err)
		}
		if err := svc.EnsureBookingOwner(ctx, tenantID, uuid.New(), owner); !errors.Is(err, ErrBookingNotFound) {
			t.Fatalf("an unknown booking must be ErrBookingNotFound, got %v", err)
		}
	})

	t.Run("license: only the driver passes EnsureDriverLicenseOwner", func(t *testing.T) {
		lic, err := repo.CreateDriverLicense(ctx, DriverLicense{
			TenantID: tenantID, DriverID: owner, LicenseClasses: []string{"B"},
			CheckedAt: now, NextCheckDueDate: now.AddDate(1, 0, 0),
		})
		if err != nil {
			t.Fatalf("seed license: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "driver_licenses", lic.ID) })

		if err := svc.EnsureDriverLicenseOwner(ctx, tenantID, lic.ID, owner); err != nil {
			t.Fatalf("driver must pass: %v", err)
		}
		if err := svc.EnsureDriverLicenseOwner(ctx, tenantID, lic.ID, stranger); !errors.Is(err, ErrDriverLicenseNotFound) {
			t.Fatalf("a foreign license must be ErrDriverLicenseNotFound, got %v", err)
		}
		if err := svc.EnsureDriverLicenseOwner(ctx, tenantID, uuid.New(), owner); !errors.Is(err, ErrDriverLicenseNotFound) {
			t.Fatalf("an unknown license must be ErrDriverLicenseNotFound, got %v", err)
		}
	})

	t.Run("damages: ReportedBy narrows list and total to the reporter", func(t *testing.T) {
		seed := func(reporter uuid.UUID, desc string) *VehicleDamage {
			d := &VehicleDamage{
				ID: uuid.New(), TenantID: tenantID, VehicleID: vehicleID,
				Description: desc, Severity: "minor", Status: DamageStatusReported,
				ReportedBy: &reporter, PhotoKeys: []string{}, CreatedAt: now, UpdatedAt: now,
			}
			if err := repo.CreateDamage(ctx, d); err != nil {
				t.Fatalf("seed damage: %v", err)
			}
			t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicle_damages", d.ID) })
			return d
		}
		mine1, mine2 := seed(owner, "mine 1"), seed(owner, "mine 2")
		theirs := seed(stranger, "theirs")

		got, total, err := svc.ListDamages(ctx, ListDamagesInput{TenantID: tenantID, ReportedBy: &owner, PageSize: 50})
		if err != nil {
			t.Fatalf("ListDamages: %v", err)
		}
		if total != 2 || len(got) != 2 || containsDamageID(got, theirs.ID) ||
			!containsDamageID(got, mine1.ID) || !containsDamageID(got, mine2.ID) {
			t.Fatalf("expected exactly the owner's two damages, got %d rows / total %d", len(got), total)
		}

		_, all, err := svc.ListDamages(ctx, ListDamagesInput{TenantID: tenantID, PageSize: 50})
		if err != nil {
			t.Fatalf("ListDamages (unrestricted): %v", err)
		}
		if all != 3 {
			t.Fatalf("unrestricted list must see all three damages, got total %d", all)
		}
	})
}
