package fuhrpark

// The export query's filters (vehicle, date range, tenant) live in SQL, so
// they can only be proven against a real database. What matters and is
// invisible in a mock: the WHERE clause actually narrows the result set
// instead of the handler filtering an over-fetched slice in Go, and the
// chronological ORDER BY is what lets the caller number rows 1..N without
// gaps -- reversing it would silently renumber the Fahrtenbuch on every
// export.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedTripLog(t *testing.T, repo *PostgresRepository, ctx context.Context, tenantID, vehicleID uuid.UUID, date time.Time) TripLog {
	t.Helper()
	created, err := repo.CreateTripLog(ctx, TripLog{
		TenantID:        tenantID,
		VehicleID:       vehicleID,
		Date:            date,
		StartLocation:   "Buero",
		EndLocation:     "Kunde",
		Purpose:         "Kundentermin",
		BusinessPartner: "Muster GmbH",
		StartKm:         100,
		EndKm:           150,
		DriverName:      "Max Mustermann",
	})
	if err != nil {
		t.Fatalf("seed trip log: %v", err)
	}
	return created
}

func TestListTripLogsForExport_FiltersRunInSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "TripExport Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "TripExport Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	vehicleA1 := seedBookingVehicle(t, repo, ctxA, pool, tenantA)
	vehicleA2 := seedBookingVehicle(t, repo, ctxA, pool, tenantA)
	vehicleB := seedBookingVehicle(t, repo, ctxB, pool, tenantB)

	day := func(offset int) time.Time {
		return time.Date(2026, 3, 1+offset, 0, 0, 0, 0, time.UTC)
	}

	// Tenant A, vehicle 1: three trips across March 1-3.
	t1 := seedTripLog(t, repo, ctxA, tenantA, vehicleA1, day(0))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "trip_logs", t1.ID) })
	t2 := seedTripLog(t, repo, ctxA, tenantA, vehicleA1, day(1))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "trip_logs", t2.ID) })
	t3 := seedTripLog(t, repo, ctxA, tenantA, vehicleA1, day(2))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "trip_logs", t3.ID) })

	// Tenant A, vehicle 2: one trip, must not appear in a vehicle-1 export.
	other := seedTripLog(t, repo, ctxA, tenantA, vehicleA2, day(1))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "trip_logs", other.ID) })

	// Tenant B: same window, must never appear in tenant A's export.
	foreign := seedTripLog(t, repo, ctxB, tenantB, vehicleB, day(1))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "trip_logs", foreign.ID) })

	t.Run("date range narrows in SQL, not in Go", func(t *testing.T) {
		from, to := day(1), day(1)
		got, err := repo.ListTripLogsForExport(ctxA, ExportTripLogsParams{
			TenantID: tenantA, VehicleID: vehicleA1, From: &from, To: &to,
		})
		if err != nil {
			t.Fatalf("ListTripLogsForExport: %v", err)
		}
		if len(got) != 1 || got[0].ID != t2.ID {
			t.Fatalf("expected only the March 2 trip, got %d rows", len(got))
		}
	})

	t.Run("vehicle filter excludes the other vehicle", func(t *testing.T) {
		got, err := repo.ListTripLogsForExport(ctxA, ExportTripLogsParams{TenantID: tenantA, VehicleID: vehicleA1})
		if err != nil {
			t.Fatalf("ListTripLogsForExport: %v", err)
		}
		for _, l := range got {
			if l.ID == other.ID {
				t.Fatalf("trip log %s belongs to a different vehicle and must not appear", l.ID)
			}
		}
		if len(got) != 3 {
			t.Fatalf("expected 3 trips for vehicle A1, got %d", len(got))
		}
	})

	t.Run("chronological order, oldest first", func(t *testing.T) {
		got, err := repo.ListTripLogsForExport(ctxA, ExportTripLogsParams{TenantID: tenantA, VehicleID: vehicleA1})
		if err != nil {
			t.Fatalf("ListTripLogsForExport: %v", err)
		}
		if len(got) != 3 || got[0].ID != t1.ID || got[1].ID != t2.ID || got[2].ID != t3.ID {
			t.Fatalf("expected ascending date order t1,t2,t3, got %+v", got)
		}
	})

	t.Run("tenant scoped, foreign trip never leaks", func(t *testing.T) {
		got, err := repo.ListTripLogsForExport(ctxA, ExportTripLogsParams{TenantID: tenantA})
		if err != nil {
			t.Fatalf("ListTripLogsForExport: %v", err)
		}
		for _, l := range got {
			if l.ID == foreign.ID {
				t.Fatalf("tenant B's trip log leaked into tenant A's export")
			}
		}

		gotB, err := repo.ListTripLogsForExport(ctxB, ExportTripLogsParams{TenantID: tenantB})
		if err != nil {
			t.Fatalf("ListTripLogsForExport (tenant B): %v", err)
		}
		if len(gotB) != 1 || gotB[0].ID != foreign.ID {
			t.Fatalf("expected tenant B to see only its own trip, got %+v", gotB)
		}
	})

	t.Run("empty range returns an empty slice, not an error", func(t *testing.T) {
		from, to := day(100), day(101)
		got, err := repo.ListTripLogsForExport(ctxA, ExportTripLogsParams{TenantID: tenantA, From: &from, To: &to})
		if err != nil {
			t.Fatalf("ListTripLogsForExport: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected no rows for an out-of-range window, got %d", len(got))
		}
	})
}
