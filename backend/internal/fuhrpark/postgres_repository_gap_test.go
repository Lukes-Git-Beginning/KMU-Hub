package fuhrpark

// Closes the read-method gap identified in cov-fuhrpark-postgres-repository-real-sql
// (BACKLOG.yml): services, damages, vehicle history, fuel logs, trip logs, vehicle
// documents and GPS positions/routes had no DB-backed test anywhere in this package.
// Vehicle core, booking, driver license and trip-log-export paths are already covered
// by postgres_repository_core_test.go, booking_conflict_test.go, driver_license_test.go
// and triplog_export_test.go respectively and are not repeated here.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestVehicleServiceCore_CRUDAndTenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Vehicle Service Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Vehicle Service Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	vehicle := seedVehicleWithTuev(t, repo, ctxA, pool, tenantA, nil)

	now := time.Now().UTC().Truncate(24 * time.Hour)
	svc := &VehicleService{
		ID: uuid.New(), TenantID: tenantA, VehicleID: vehicle.ID,
		ServiceType: "oil_change", ScheduledAt: now, Status: ServiceStatusScheduled,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateService(ctxA, svc); err != nil {
		t.Fatalf("CreateService: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicle_services", svc.ID) })

	t.Run("GetService is tenant scoped", func(t *testing.T) {
		got, err := repo.GetService(ctxA, tenantA, svc.ID)
		if err != nil {
			t.Fatalf("GetService: %v", err)
		}
		if got.ServiceType != "oil_change" {
			t.Fatalf("unexpected service: %+v", got)
		}
		if _, err := repo.GetService(ctxB, tenantB, svc.ID); !errors.Is(err, ErrServiceNotFound) {
			t.Fatalf("expected ErrServiceNotFound across tenants, got %v", err)
		}
	})

	t.Run("ListServices filters by vehicle, status and scheduled_before", func(t *testing.T) {
		vehicleID := vehicle.ID
		byVehicle, total, err := repo.ListServices(ctxA, tenantA, ListServicesFilter{VehicleID: &vehicleID}, 0, 50)
		if err != nil {
			t.Fatalf("ListServices (vehicle filter): %v", err)
		}
		if total < 1 || !containsServiceID(byVehicle, svc.ID) {
			t.Fatalf("expected service %s in vehicle-filtered list, got total=%d", svc.ID, total)
		}

		wrongStatus := ServiceStatusCompleted
		byStatus, _, err := repo.ListServices(ctxA, tenantA, ListServicesFilter{VehicleID: &vehicleID, Status: &wrongStatus}, 0, 50)
		if err != nil {
			t.Fatalf("ListServices (status filter): %v", err)
		}
		if containsServiceID(byStatus, svc.ID) {
			t.Fatalf("scheduled service must not match a completed-status filter")
		}

		beforeToday := now.AddDate(0, 0, -1)
		byScheduled, _, err := repo.ListServices(ctxA, tenantA, ListServicesFilter{VehicleID: &vehicleID, ScheduledBefore: &beforeToday}, 0, 50)
		if err != nil {
			t.Fatalf("ListServices (scheduled_before filter): %v", err)
		}
		if containsServiceID(byScheduled, svc.ID) {
			t.Fatalf("service scheduled today must not match a scheduled_before-yesterday filter")
		}

		testutil.AssertRowCount(t, pool, ctxB, "vehicle_services", svc.ID, 0)
	})

	t.Run("UpdateService is tenant scoped", func(t *testing.T) {
		updated := *svc
		updated.Status = ServiceStatusCompleted
		updated.UpdatedAt = time.Now()
		if err := repo.UpdateService(ctxA, &updated); err != nil {
			t.Fatalf("UpdateService: %v", err)
		}
		got, err := repo.GetService(ctxA, tenantA, svc.ID)
		if err != nil {
			t.Fatalf("GetService after update: %v", err)
		}
		if got.Status != ServiceStatusCompleted {
			t.Fatalf("update did not persist, got %+v", got)
		}

		wrongTenant := *svc
		wrongTenant.TenantID = tenantB
		wrongTenant.Status = ServiceStatusCancelled
		if err := repo.UpdateService(ctxB, &wrongTenant); !errors.Is(err, ErrServiceNotFound) {
			t.Fatalf("expected ErrServiceNotFound updating under the wrong tenant, got %v", err)
		}
		unchanged, err := repo.GetService(ctxA, tenantA, svc.ID)
		if err != nil {
			t.Fatalf("GetService after failed cross-tenant update: %v", err)
		}
		if unchanged.Status != ServiceStatusCompleted {
			t.Fatalf("cross-tenant update must not have modified the row, got status %q", unchanged.Status)
		}
	})

	t.Run("DeleteService is tenant scoped and idempotent-not-found", func(t *testing.T) {
		if err := repo.DeleteService(ctxB, tenantB, svc.ID); !errors.Is(err, ErrServiceNotFound) {
			t.Fatalf("expected ErrServiceNotFound deleting under the wrong tenant, got %v", err)
		}
		testutil.AssertRowCount(t, pool, ctxA, "vehicle_services", svc.ID, 1)

		if err := repo.DeleteService(ctxA, tenantA, svc.ID); err != nil {
			t.Fatalf("DeleteService: %v", err)
		}
		testutil.AssertRowCount(t, pool, ctxA, "vehicle_services", svc.ID, 0)

		if err := repo.DeleteService(ctxA, tenantA, svc.ID); !errors.Is(err, ErrServiceNotFound) {
			t.Fatalf("second DeleteService: expected ErrServiceNotFound, got %v", err)
		}
	})
}

func containsServiceID(services []*VehicleService, id uuid.UUID) bool {
	for _, s := range services {
		if s.ID == id {
			return true
		}
	}
	return false
}

func TestVehicleDamageCore_CRUDAndTenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Vehicle Damage Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Vehicle Damage Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	vehicle := seedVehicleWithTuev(t, repo, ctxA, pool, tenantA, nil)

	now := time.Now().UTC()
	dmg := &VehicleDamage{
		ID: uuid.New(), TenantID: tenantA, VehicleID: vehicle.ID,
		Description: "Bumper scratch", Severity: "minor", Status: DamageStatusReported,
		PhotoKeys: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateDamage(ctxA, dmg); err != nil {
		t.Fatalf("CreateDamage: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicle_damages", dmg.ID) })

	t.Run("GetDamage is tenant scoped", func(t *testing.T) {
		got, err := repo.GetDamage(ctxA, tenantA, dmg.ID)
		if err != nil {
			t.Fatalf("GetDamage: %v", err)
		}
		if got.Description != "Bumper scratch" {
			t.Fatalf("unexpected damage: %+v", got)
		}
		if _, err := repo.GetDamage(ctxB, tenantB, dmg.ID); !errors.Is(err, ErrDamageNotFound) {
			t.Fatalf("expected ErrDamageNotFound across tenants, got %v", err)
		}
	})

	t.Run("ListDamages filters by vehicle and status", func(t *testing.T) {
		vehicleID := vehicle.ID
		byVehicle, total, err := repo.ListDamages(ctxA, tenantA, ListDamagesFilter{VehicleID: &vehicleID}, 0, 50)
		if err != nil {
			t.Fatalf("ListDamages (vehicle filter): %v", err)
		}
		if total < 1 || !containsDamageID(byVehicle, dmg.ID) {
			t.Fatalf("expected damage %s in vehicle-filtered list, got total=%d", dmg.ID, total)
		}

		wrongStatus := DamageStatusResolved
		byStatus, _, err := repo.ListDamages(ctxA, tenantA, ListDamagesFilter{VehicleID: &vehicleID, Status: &wrongStatus}, 0, 50)
		if err != nil {
			t.Fatalf("ListDamages (status filter): %v", err)
		}
		if containsDamageID(byStatus, dmg.ID) {
			t.Fatalf("reported damage must not match a resolved-status filter")
		}

		testutil.AssertRowCount(t, pool, ctxB, "vehicle_damages", dmg.ID, 0)
	})

	t.Run("UpdateDamage is tenant scoped", func(t *testing.T) {
		updated := *dmg
		updated.Status = DamageStatusResolved
		updated.UpdatedAt = time.Now()
		if err := repo.UpdateDamage(ctxA, &updated); err != nil {
			t.Fatalf("UpdateDamage: %v", err)
		}
		got, err := repo.GetDamage(ctxA, tenantA, dmg.ID)
		if err != nil {
			t.Fatalf("GetDamage after update: %v", err)
		}
		if got.Status != DamageStatusResolved {
			t.Fatalf("update did not persist, got %+v", got)
		}

		wrongTenant := *dmg
		wrongTenant.TenantID = tenantB
		wrongTenant.Status = DamageStatusInRepair
		if err := repo.UpdateDamage(ctxB, &wrongTenant); !errors.Is(err, ErrDamageNotFound) {
			t.Fatalf("expected ErrDamageNotFound updating under the wrong tenant, got %v", err)
		}
		unchanged, err := repo.GetDamage(ctxA, tenantA, dmg.ID)
		if err != nil {
			t.Fatalf("GetDamage after failed cross-tenant update: %v", err)
		}
		if unchanged.Status != DamageStatusResolved {
			t.Fatalf("cross-tenant update must not have modified the row, got status %q", unchanged.Status)
		}
	})
}

func containsDamageID(damages []*VehicleDamage, id uuid.UUID) bool {
	for _, d := range damages {
		if d.ID == id {
			return true
		}
	}
	return false
}

// TestGetVehicleHistory_MergesServicesAndDamagesOrderedAndTenantScoped is one of the
// >=3 join-through-vehicle RLS-smoke methods required by the unit: vehicle_services and
// vehicle_damages both carry their own tenant_id, but the UNION ALL joins them purely by
// vehicle_id, so a missing tenant_id predicate on either half would leak across tenants.
func TestGetVehicleHistory_MergesServicesAndDamagesOrderedAndTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Vehicle History Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Vehicle History Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	vehicle := seedVehicleWithTuev(t, repo, ctxA, pool, tenantA, nil)

	older := time.Now().UTC().AddDate(0, 0, -5).Truncate(24 * time.Hour)
	newer := time.Now().UTC()

	svc := &VehicleService{
		ID: uuid.New(), TenantID: tenantA, VehicleID: vehicle.ID,
		ServiceType: "inspection", ScheduledAt: older, Status: ServiceStatusCompleted,
		CreatedAt: older, UpdatedAt: older,
	}
	if err := repo.CreateService(ctxA, svc); err != nil {
		t.Fatalf("CreateService: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicle_services", svc.ID) })

	dmg := &VehicleDamage{
		ID: uuid.New(), TenantID: tenantA, VehicleID: vehicle.ID,
		Description: "Windshield chip", Severity: "minor", Status: DamageStatusReported,
		PhotoKeys: []string{}, CreatedAt: newer, UpdatedAt: newer,
	}
	if err := repo.CreateDamage(ctxA, dmg); err != nil {
		t.Fatalf("CreateDamage: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicle_damages", dmg.ID) })

	entries, total, err := repo.GetVehicleHistory(ctxA, tenantA, vehicle.ID, 0, 50)
	if err != nil {
		t.Fatalf("GetVehicleHistory: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2 history entries, got %d", total)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(entries), entries)
	}
	if entries[0].Kind != "damage" || entries[0].ID != dmg.ID {
		t.Fatalf("expected the newer damage entry first (occurred_at DESC), got %+v", entries[0])
	}
	if entries[1].Kind != "service" || entries[1].ID != svc.ID {
		t.Fatalf("expected the older service entry second, got %+v", entries[1])
	}

	_, totalB, err := repo.GetVehicleHistory(ctxB, tenantB, vehicle.ID, 0, 50)
	if err != nil {
		t.Fatalf("GetVehicleHistory (cross-tenant): %v", err)
	}
	if totalB != 0 {
		t.Fatalf("expected cross-tenant history for tenant B querying tenant A's vehicle to be empty, got total=%d", totalB)
	}
}

func TestFuelLogs_ListFilterUpdateDelete(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Fuel Log Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Fuel Log Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	vehicle1 := seedVehicleWithTuev(t, repo, ctxA, pool, tenantA, nil)
	vehicle2 := seedVehicleWithTuev(t, repo, ctxA, pool, tenantA, nil)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	log1, err := repo.CreateFuelLog(ctxA, FuelLog{
		TenantID: tenantA, VehicleID: vehicle1.ID, Date: today,
		Liters: 40, CostCents: 6500, MileageKm: 10000, FuelType: "diesel",
	})
	if err != nil {
		t.Fatalf("CreateFuelLog (vehicle1): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "fuel_logs", log1.ID) })

	log2, err := repo.CreateFuelLog(ctxA, FuelLog{
		TenantID: tenantA, VehicleID: vehicle2.ID, Date: today,
		Liters: 30, CostCents: 5000, MileageKm: 20000, FuelType: "petrol",
	})
	if err != nil {
		t.Fatalf("CreateFuelLog (vehicle2): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "fuel_logs", log2.ID) })

	t.Run("ListFuelLogs filters by vehicle and stays tenant scoped", func(t *testing.T) {
		byVehicle, total, err := repo.ListFuelLogs(ctxA, ListFuelLogsParams{TenantID: tenantA, VehicleID: vehicle1.ID, Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("ListFuelLogs (vehicle filter): %v", err)
		}
		if total != 1 || len(byVehicle) != 1 || byVehicle[0].ID != log1.ID {
			t.Fatalf("expected only vehicle1's log, got total=%d list=%+v", total, byVehicle)
		}

		all, totalAll, err := repo.ListFuelLogs(ctxA, ListFuelLogsParams{TenantID: tenantA, Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("ListFuelLogs (no vehicle filter): %v", err)
		}
		if totalAll < 2 || !containsFuelLogID(all, log1.ID) || !containsFuelLogID(all, log2.ID) {
			t.Fatalf("expected both fuel logs without a vehicle filter, got total=%d list=%+v", totalAll, all)
		}

		crossTenant, totalCross, err := repo.ListFuelLogs(ctxB, ListFuelLogsParams{TenantID: tenantB, Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("ListFuelLogs (cross-tenant): %v", err)
		}
		if totalCross != 0 || len(crossTenant) != 0 {
			t.Fatalf("expected no fuel logs visible to a foreign tenant, got total=%d list=%+v", totalCross, crossTenant)
		}
	})

	t.Run("UpdateFuelLog is tenant scoped", func(t *testing.T) {
		updated := log1
		updated.Liters = 42
		saved, err := repo.UpdateFuelLog(ctxA, updated)
		if err != nil {
			t.Fatalf("UpdateFuelLog: %v", err)
		}
		if saved.Liters != 42 {
			t.Fatalf("update did not persist, got %+v", saved)
		}

		wrongTenant := log1
		wrongTenant.TenantID = tenantB
		wrongTenant.Liters = 99
		if _, err := repo.UpdateFuelLog(ctxB, wrongTenant); !errors.Is(err, ErrFuelLogNotFound) {
			t.Fatalf("expected ErrFuelLogNotFound updating under the wrong tenant, got %v", err)
		}
	})

	t.Run("DeleteFuelLog is tenant scoped and idempotent-not-found", func(t *testing.T) {
		if err := repo.DeleteFuelLog(ctxB, tenantB, log1.ID); !errors.Is(err, ErrFuelLogNotFound) {
			t.Fatalf("expected ErrFuelLogNotFound deleting under the wrong tenant, got %v", err)
		}
		testutil.AssertRowCount(t, pool, ctxA, "fuel_logs", log1.ID, 1)

		if err := repo.DeleteFuelLog(ctxA, tenantA, log1.ID); err != nil {
			t.Fatalf("DeleteFuelLog: %v", err)
		}
		if err := repo.DeleteFuelLog(ctxA, tenantA, log1.ID); !errors.Is(err, ErrFuelLogNotFound) {
			t.Fatalf("second DeleteFuelLog: expected ErrFuelLogNotFound, got %v", err)
		}
	})
}

func containsFuelLogID(logs []FuelLog, id uuid.UUID) bool {
	for _, l := range logs {
		if l.ID == id {
			return true
		}
	}
	return false
}

func TestTripLogs_ListUpdateDeleteAndInvalidKmGuard(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Trip Log Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Trip Log Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	vehicle1 := seedVehicleWithTuev(t, repo, ctxA, pool, tenantA, nil)
	vehicle2 := seedVehicleWithTuev(t, repo, ctxA, pool, tenantA, nil)

	today := time.Now().UTC().Truncate(24 * time.Hour)

	t.Run("CreateTripLog rejects end_km below start_km before touching the DB", func(t *testing.T) {
		_, err := repo.CreateTripLog(ctxA, TripLog{
			TenantID: tenantA, VehicleID: vehicle1.ID, Date: today,
			StartLocation: "A", EndLocation: "B", Purpose: "Kundenbesuch",
			StartKm: 100, EndKm: 50, DriverName: "Max Mustermann",
		})
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput for end_km < start_km, got %v", err)
		}
	})

	log1, err := repo.CreateTripLog(ctxA, TripLog{
		TenantID: tenantA, VehicleID: vehicle1.ID, Date: today,
		StartLocation: "Buero", EndLocation: "Kunde", Purpose: "Termin",
		StartKm: 1000, EndKm: 1050, DriverName: "Max Mustermann",
	})
	if err != nil {
		t.Fatalf("CreateTripLog (vehicle1): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "trip_logs", log1.ID) })

	log2, err := repo.CreateTripLog(ctxA, TripLog{
		TenantID: tenantA, VehicleID: vehicle2.ID, Date: today,
		StartLocation: "Lager", EndLocation: "Baustelle", Purpose: "Lieferung",
		StartKm: 500, EndKm: 520, DriverName: "Erika Musterfrau",
	})
	if err != nil {
		t.Fatalf("CreateTripLog (vehicle2): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "trip_logs", log2.ID) })

	t.Run("ListTripLogs filters by vehicle and computes km", func(t *testing.T) {
		byVehicle, total, err := repo.ListTripLogs(ctxA, ListTripLogsParams{TenantID: tenantA, VehicleID: vehicle1.ID, Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("ListTripLogs (vehicle filter): %v", err)
		}
		if total != 1 || len(byVehicle) != 1 || byVehicle[0].ID != log1.ID {
			t.Fatalf("expected only vehicle1's trip log, got total=%d list=%+v", total, byVehicle)
		}
		if byVehicle[0].Km != 50 {
			t.Fatalf("expected the generated km column to be 50, got %d", byVehicle[0].Km)
		}

		all, totalAll, err := repo.ListTripLogs(ctxA, ListTripLogsParams{TenantID: tenantA, Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("ListTripLogs (no vehicle filter): %v", err)
		}
		if totalAll < 2 || !containsTripLogID(all, log1.ID) || !containsTripLogID(all, log2.ID) {
			t.Fatalf("expected both trip logs without a vehicle filter, got total=%d list=%+v", totalAll, all)
		}

		testutil.AssertRowCount(t, pool, ctxB, "trip_logs", log1.ID, 0)
	})

	t.Run("UpdateTripLog is tenant scoped and rejects end_km below start_km", func(t *testing.T) {
		updated := log1
		updated.Purpose = "Nachtermin"
		saved, err := repo.UpdateTripLog(ctxA, updated)
		if err != nil {
			t.Fatalf("UpdateTripLog: %v", err)
		}
		if saved.Purpose != "Nachtermin" {
			t.Fatalf("update did not persist, got %+v", saved)
		}

		invalid := log1
		invalid.EndKm = log1.StartKm - 1
		if _, err := repo.UpdateTripLog(ctxA, invalid); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput for end_km < start_km, got %v", err)
		}

		wrongTenant := log1
		wrongTenant.TenantID = tenantB
		wrongTenant.Purpose = "Should not land"
		if _, err := repo.UpdateTripLog(ctxB, wrongTenant); !errors.Is(err, ErrTripLogNotFound) {
			t.Fatalf("expected ErrTripLogNotFound updating under the wrong tenant, got %v", err)
		}
	})

	t.Run("DeleteTripLog is tenant scoped and idempotent-not-found", func(t *testing.T) {
		if err := repo.DeleteTripLog(ctxB, tenantB, log1.ID); !errors.Is(err, ErrTripLogNotFound) {
			t.Fatalf("expected ErrTripLogNotFound deleting under the wrong tenant, got %v", err)
		}
		testutil.AssertRowCount(t, pool, ctxA, "trip_logs", log1.ID, 1)

		if err := repo.DeleteTripLog(ctxA, tenantA, log1.ID); err != nil {
			t.Fatalf("DeleteTripLog: %v", err)
		}
		if err := repo.DeleteTripLog(ctxA, tenantA, log1.ID); !errors.Is(err, ErrTripLogNotFound) {
			t.Fatalf("second DeleteTripLog: expected ErrTripLogNotFound, got %v", err)
		}
	})
}

func containsTripLogID(logs []TripLog, id uuid.UUID) bool {
	for _, l := range logs {
		if l.ID == id {
			return true
		}
	}
	return false
}

func TestVehicleDocuments_ListDeleteTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Vehicle Document Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Vehicle Document Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	vehicle := seedVehicleWithTuev(t, repo, ctxA, pool, tenantA, nil)

	doc, err := repo.CreateVehicleDocument(ctxA, VehicleDocument{
		TenantID: tenantA, VehicleID: vehicle.ID, DocType: "registration",
		Name: "Fahrzeugschein", ObjectKey: "fuhrpark/" + uuid.New().String() + ".pdf",
	})
	if err != nil {
		t.Fatalf("CreateVehicleDocument: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicle_documents", doc.ID) })

	t.Run("ListVehicleDocuments is tenant and vehicle scoped", func(t *testing.T) {
		list, total, err := repo.ListVehicleDocuments(ctxA, ListVehicleDocumentsParams{TenantID: tenantA, VehicleID: vehicle.ID, Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("ListVehicleDocuments: %v", err)
		}
		if total != 1 || len(list) != 1 || list[0].ID != doc.ID {
			t.Fatalf("expected the seeded document, got total=%d list=%+v", total, list)
		}

		_, totalCross, err := repo.ListVehicleDocuments(ctxB, ListVehicleDocumentsParams{TenantID: tenantB, VehicleID: vehicle.ID, Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("ListVehicleDocuments (cross-tenant): %v", err)
		}
		if totalCross != 0 {
			t.Fatalf("expected no documents visible to a foreign tenant querying tenant A's vehicle_id, got total=%d", totalCross)
		}
	})

	t.Run("DeleteVehicleDocument is tenant scoped and idempotent-not-found", func(t *testing.T) {
		if err := repo.DeleteVehicleDocument(ctxB, tenantB, doc.ID); !errors.Is(err, ErrDocumentNotFound) {
			t.Fatalf("expected ErrDocumentNotFound deleting under the wrong tenant, got %v", err)
		}
		testutil.AssertRowCount(t, pool, ctxA, "vehicle_documents", doc.ID, 1)

		if err := repo.DeleteVehicleDocument(ctxA, tenantA, doc.ID); err != nil {
			t.Fatalf("DeleteVehicleDocument: %v", err)
		}
		if err := repo.DeleteVehicleDocument(ctxA, tenantA, doc.ID); !errors.Is(err, ErrDocumentNotFound) {
			t.Fatalf("second DeleteVehicleDocument: expected ErrDocumentNotFound, got %v", err)
		}
	})
}

// TestGpsPositions_IngestGetAndRouteAggregation also serves as the RLS-smoke for
// GetVehicleRoutes, the one genuine SQL JOIN in this repository (gps_positions to
// vehicles) -- everything else filters by an own tenant_id column instead of a join.
func TestGpsPositions_IngestGetAndRouteAggregation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "GPS Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "GPS Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	vehicle := seedVehicleWithTuev(t, repo, ctxA, pool, tenantA, nil)

	now := time.Now().UTC()
	positions := []GpsPosition{
		{Lat: 52.5, Lng: 13.4, RecordedAt: now.Add(-2 * time.Minute)},
		{Lat: 52.6, Lng: 13.5, RecordedAt: now.Add(-1 * time.Minute)},
		{Lat: 52.7, Lng: 13.6, RecordedAt: now},
	}
	count, err := repo.IngestGpsPositions(ctxA, tenantA, vehicle.ID, positions)
	if err != nil {
		t.Fatalf("IngestGpsPositions: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 positions ingested, got %d", count)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(testutil.WithSystemCtx(context.Background()),
			"DELETE FROM gps_positions WHERE tenant_id=$1 AND vehicle_id=$2", tenantA, vehicle.ID)
	})

	t.Run("GetGpsPositions returns the window ordered ascending and stays tenant scoped", func(t *testing.T) {
		got, err := repo.GetGpsPositions(ctxA, GetGpsPositionsParams{
			TenantID: tenantA, VehicleID: vehicle.ID,
			From: now.Add(-3 * time.Minute), To: now.Add(time.Minute),
		})
		if err != nil {
			t.Fatalf("GetGpsPositions: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("expected 3 positions in range, got %d", len(got))
		}
		if !got[0].RecordedAt.Before(got[1].RecordedAt) || !got[1].RecordedAt.Before(got[2].RecordedAt) {
			t.Fatalf("expected ascending order by recorded_at, got %+v", got)
		}

		crossTenant, err := repo.GetGpsPositions(ctxB, GetGpsPositionsParams{
			TenantID: tenantB, VehicleID: vehicle.ID,
			From: now.Add(-3 * time.Minute), To: now.Add(time.Minute),
		})
		if err != nil {
			t.Fatalf("GetGpsPositions (cross-tenant): %v", err)
		}
		if len(crossTenant) != 0 {
			t.Fatalf("expected no positions visible to a foreign tenant, got %+v", crossTenant)
		}
	})

	t.Run("GetVehicleRoutes aggregates by day, reports driving for a recent fix, and stays tenant scoped", func(t *testing.T) {
		routes, err := repo.GetVehicleRoutes(ctxA, GetVehicleRoutesParams{
			TenantID: tenantA, VehicleID: vehicle.ID,
			DateFrom: now.AddDate(0, 0, -1), DateTo: now.AddDate(0, 0, 1),
		})
		if err != nil {
			t.Fatalf("GetVehicleRoutes: %v", err)
		}
		if len(routes) != 1 {
			t.Fatalf("expected one route-day, got %d: %+v", len(routes), routes)
		}
		if routes[0].VehicleID != vehicle.ID || routes[0].VehicleName != vehicle.LicensePlate {
			t.Fatalf("unexpected route header: %+v", routes[0])
		}
		if routes[0].Status != "driving" {
			t.Fatalf("expected status 'driving' for a fix within the last 5 minutes, got %q", routes[0].Status)
		}
		if len(routes[0].Positions) != 3 {
			t.Fatalf("expected all 3 positions attached to the day's route, got %d", len(routes[0].Positions))
		}

		// tenantB has no vehicle and no gps_positions of its own -- the join must not
		// let tenant A's vehicle row leak into tenant B's aggregation.
		crossTenant, err := repo.GetVehicleRoutes(ctxB, GetVehicleRoutesParams{
			TenantID: tenantB, VehicleID: vehicle.ID,
			DateFrom: now.AddDate(0, 0, -1), DateTo: now.AddDate(0, 0, 1),
		})
		if err != nil {
			t.Fatalf("GetVehicleRoutes (cross-tenant): %v", err)
		}
		if len(crossTenant) != 0 {
			t.Fatalf("expected no routes visible to a foreign tenant, got %+v", crossTenant)
		}
	})
}

// TestService_IngestGpsPositions_RejectsForeignTenantVehicle is the DB-backed
// counterpart to fix-fuhrpark-gps-ingest-no-vehicle-tenant-check: a caller
// must not be able to attribute GPS positions to a vehicle_id owned by a
// different tenant, even though gps_positions.vehicle_id only has a bare FK
// on vehicles(id) with no tenant check of its own.
func TestService_IngestGpsPositions_RejectsForeignTenantVehicle(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "GPS Ingest Owner Tenant")
	testutil.EnsureTenant(t, pool, tenantB, "GPS Ingest Attacker Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	vehicle := seedVehicleWithTuev(t, repo, ctxA, pool, tenantA, nil)

	n, err := svc.IngestGpsPositions(ctxB, tenantB, vehicle.ID, []GpsPosition{{Lat: 1, Lng: 2, RecordedAt: time.Now()}})
	if !errors.Is(err, ErrVehicleNotFound) {
		t.Fatalf("expected ErrVehicleNotFound for a foreign-tenant vehicle_id, got n=%d err=%v", n, err)
	}

	var count int
	if err := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		"SELECT count(*) FROM gps_positions WHERE vehicle_id=$1", vehicle.ID).Scan(&count); err != nil {
		t.Fatalf("count gps_positions: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no gps_positions row inserted for the rejected ingest, got %d", count)
	}

	own, err := svc.IngestGpsPositions(ctxA, tenantA, vehicle.ID, []GpsPosition{{Lat: 1, Lng: 2, RecordedAt: time.Now()}})
	if err != nil {
		t.Fatalf("IngestGpsPositions for the owning tenant: %v", err)
	}
	if own != 1 {
		t.Fatalf("expected 1 position ingested for the owning tenant, got %d", own)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(testutil.WithSystemCtx(context.Background()),
			"DELETE FROM gps_positions WHERE vehicle_id=$1", vehicle.ID)
	})
}
