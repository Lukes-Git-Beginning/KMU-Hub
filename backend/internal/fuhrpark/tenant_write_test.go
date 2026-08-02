package fuhrpark

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

func TestFuhrparkWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantWrite, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantWrite, "Fuhrpark Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Fuhrpark Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantWrite)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	vehicle := &Vehicle{
		ID:           uuid.New(),
		TenantID:     tenantWrite,
		LicensePlate: "B-KM " + uuid.New().String()[:6],
		Make:         "Mercedes",
		Model:        "Sprinter",
		Year:         2024,
		FuelType:     "diesel",
		Status:       VehicleStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.CreateVehicle(ctxA, vehicle); err != nil {
		t.Fatalf("CreateVehicle: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "vehicles", vehicle.ID)

	svc := &VehicleService{
		ID:          uuid.New(),
		TenantID:    tenantWrite,
		VehicleID:   vehicle.ID,
		ServiceType: "inspection",
		ScheduledAt: now,
		Status:      ServiceStatusScheduled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.CreateService(ctxA, svc); err != nil {
		t.Fatalf("CreateService: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "vehicle_services", svc.ID)

	damage := &VehicleDamage{
		ID:          uuid.New(),
		TenantID:    tenantWrite,
		VehicleID:   vehicle.ID,
		Description: "Delle vorne links",
		Severity:    "minor",
		Status:      DamageStatusReported,
		PhotoKeys:   []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.CreateDamage(ctxA, damage); err != nil {
		t.Fatalf("CreateDamage: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "vehicle_damages", damage.ID)

	fuelLog, err := repo.CreateFuelLog(ctxA, FuelLog{
		TenantID:  tenantWrite,
		VehicleID: vehicle.ID,
		Date:      now,
		Liters:    45.5,
		CostCents: 8000,
		MileageKm: 12000,
		FuelType:  "diesel",
	})
	if err != nil {
		t.Fatalf("CreateFuelLog: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "fuel_logs", fuelLog.ID)

	tripLog, err := repo.CreateTripLog(ctxA, TripLog{
		TenantID:      tenantWrite,
		VehicleID:     vehicle.ID,
		Date:          now,
		StartLocation: "Depot",
		EndLocation:   "Kunde",
		StartKm:       12000,
		EndKm:         12050,
	})
	if err != nil {
		t.Fatalf("CreateTripLog: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "trip_logs", tripLog.ID)

	doc, err := repo.CreateVehicleDocument(ctxA, VehicleDocument{
		TenantID:  tenantWrite,
		VehicleID: vehicle.ID,
		DocType:   "registration",
		Name:      "Fahrzeugschein.pdf",
		ObjectKey: "fuhrpark/" + uuid.New().String() + "/registration.pdf",
	})
	if err != nil {
		t.Fatalf("CreateVehicleDocument: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "vehicle_documents", doc.ID)

	driverID := uuid.New()
	if _, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name)
		 VALUES ($1, $2, 'fuhrpark-write-driver@test.local', 'x', 'Fuhrpark', 'Driver')`,
		driverID, tenantWrite); err != nil {
		t.Fatalf("seed driver user: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "users", driverID)

	lic, err := repo.CreateDriverLicense(ctxA, DriverLicense{
		TenantID:         tenantWrite,
		DriverID:         driverID,
		LicenseClasses:   []string{"B"},
		CheckedAt:        now,
		NextCheckDueDate: now.AddDate(1, 0, 0),
	})
	if err != nil {
		t.Fatalf("CreateDriverLicense: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "driver_licenses", lic.ID)

	rows := []struct {
		table string
		id    uuid.UUID
	}{
		{"vehicles", vehicle.ID},
		{"vehicle_services", svc.ID},
		{"vehicle_damages", damage.ID},
		{"fuel_logs", fuelLog.ID},
		{"trip_logs", tripLog.ID},
		{"vehicle_documents", doc.ID},
		{"driver_licenses", lic.ID},
	}

	for _, row := range rows {
		t.Run(row.table, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, row.table, row.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, row.table, row.id, 0)
		})
	}
}
