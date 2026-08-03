package fuhrpark

// Driver license (Fuehrerscheinkontrolle) CRUD and tenant isolation against
// the real PostgresRepository. Complements tenant_write_test.go (which only
// proves the row lands with the caller's tenant_id) with the behaviour
// specific to this table: the users join in CreateDriverLicense must refuse
// a driver_id that belongs to another tenant, even though driver_licenses
// itself already has RLS.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestFuhrparkDriverLicense_CRUDAndTenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Driver License Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Driver License Tenant B")

	sysCtx := testutil.WithSystemCtx(context.Background())
	driverA := uuid.New()
	if _, err := pool.Exec(sysCtx,
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name)
		 VALUES ($1, $2, 'fuhrpark-driver-a@test.local', 'x', 'Fuhrpark', 'DriverA')`,
		driverA, tenantA); err != nil {
		t.Fatalf("seed driverA: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "users", driverA)

	driverB := uuid.New()
	if _, err := pool.Exec(sysCtx,
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name)
		 VALUES ($1, $2, 'fuhrpark-driver-b@test.local', 'x', 'Fuhrpark', 'DriverB')`,
		driverB, tenantB); err != nil {
		t.Fatalf("seed driverB: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "users", driverB)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC().Truncate(24 * time.Hour)
	nextDue := now.AddDate(1, 0, 0)

	t.Run("create rejects driver from another tenant", func(t *testing.T) {
		_, err := repo.CreateDriverLicense(ctxA, DriverLicense{
			TenantID:         tenantA,
			DriverID:         driverB, // belongs to tenantB, caller claims tenantA
			LicenseClasses:   []string{"B"},
			CheckedAt:        now,
			NextCheckDueDate: nextDue,
		})
		if !errors.Is(err, ErrDriverNotFound) {
			t.Fatalf("expected ErrDriverNotFound, got %v", err)
		}
	})

	lic, err := repo.CreateDriverLicense(ctxA, DriverLicense{
		TenantID:         tenantA,
		DriverID:         driverA,
		LicenseClasses:   []string{"B", "BE"},
		CheckedAt:        now,
		NextCheckDueDate: nextDue,
	})
	if err != nil {
		t.Fatalf("CreateDriverLicense: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "driver_licenses", lic.ID)

	t.Run("tenant isolation", func(t *testing.T) {
		testutil.AssertRowCount(t, pool, ctxA, "driver_licenses", lic.ID, 1)
		testutil.AssertRowCount(t, pool, ctxB, "driver_licenses", lic.ID, 0)
	})

	t.Run("list filters by driver", func(t *testing.T) {
		byDriver, total, err := repo.ListDriverLicenses(ctxA, ListDriverLicensesParams{TenantID: tenantA, DriverID: driverA})
		if err != nil {
			t.Fatalf("ListDriverLicenses: %v", err)
		}
		if total != 1 || len(byDriver) != 1 || byDriver[0].ID != lic.ID {
			t.Fatalf("expected exactly the seeded license, got %d/%d", len(byDriver), total)
		}

		byOtherDriver, total, err := repo.ListDriverLicenses(ctxA, ListDriverLicensesParams{TenantID: tenantA, DriverID: uuid.New()})
		if err != nil {
			t.Fatalf("ListDriverLicenses (no match): %v", err)
		}
		if total != 0 || len(byOtherDriver) != 0 {
			t.Fatalf("expected no rows for unrelated driver, got %d/%d", len(byOtherDriver), total)
		}
	})

	t.Run("update corrects an entry", func(t *testing.T) {
		newDue := nextDue.AddDate(0, 6, 0)
		updated, err := repo.UpdateDriverLicense(ctxA, DriverLicense{
			ID:               lic.ID,
			TenantID:         tenantA,
			LicenseClasses:   []string{"B", "C1"},
			CheckedAt:        now,
			NextCheckDueDate: newDue,
		})
		if err != nil {
			t.Fatalf("UpdateDriverLicense: %v", err)
		}
		if len(updated.LicenseClasses) != 2 || updated.LicenseClasses[1] != "C1" {
			t.Fatalf("license classes not updated: %+v", updated.LicenseClasses)
		}
		if !updated.NextCheckDueDate.Equal(newDue) {
			t.Fatalf("next_check_due_date not updated: got %v want %v", updated.NextCheckDueDate, newDue)
		}
	})

	t.Run("update across tenant boundary is a no-op", func(t *testing.T) {
		_, err := repo.UpdateDriverLicense(ctxB, DriverLicense{
			ID:               lic.ID,
			TenantID:         tenantB,
			LicenseClasses:   []string{"B"},
			CheckedAt:        now,
			NextCheckDueDate: nextDue,
		})
		if !errors.Is(err, ErrDriverLicenseNotFound) {
			t.Fatalf("expected ErrDriverLicenseNotFound, got %v", err)
		}
	})

	t.Run("delete across tenant boundary is a no-op", func(t *testing.T) {
		if err := repo.DeleteDriverLicense(ctxB, tenantB, lic.ID); !errors.Is(err, ErrDriverLicenseNotFound) {
			t.Fatalf("expected ErrDriverLicenseNotFound, got %v", err)
		}
		testutil.AssertRowCount(t, pool, ctxA, "driver_licenses", lic.ID, 1)
	})
}
