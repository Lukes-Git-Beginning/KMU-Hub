package fuhrpark

// Fuel logs, trip logs and documents of a vehicle against the real schema: the FK on
// vehicles(id) catches neither a soft-deleted vehicle nor one of another tenant, so the
// service has to look the vehicle up first.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestVehicleChildren_RejectDeletedAndUnknownVehicleAgainstRealSchema(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vehicle Children Parent Tenant")
	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	live := seedVehicleWithTuev(t, repo, ctx, pool, tenantID, nil)
	gone := seedVehicleWithTuev(t, repo, ctx, pool, tenantID, nil)
	require.NoError(t, repo.SoftDeleteVehicle(ctx, tenantID, gone.ID))

	create := map[string]func(vehicleID uuid.UUID) (uuid.UUID, error){
		"fuel_logs": func(id uuid.UUID) (uuid.UUID, error) {
			l, err := svc.CreateFuelLog(ctx, FuelLog{TenantID: tenantID, VehicleID: id, Liters: 40, CostCents: 7000, MileageKm: 1000, FuelType: "diesel"})
			return l.ID, err
		},
		"trip_logs": func(id uuid.UUID) (uuid.UUID, error) {
			l, err := svc.CreateTripLog(ctx, TripLog{TenantID: tenantID, VehicleID: id, StartLocation: "A", EndLocation: "B", StartKm: 1, EndKm: 9})
			return l.ID, err
		},
		"vehicle_documents": func(id uuid.UUID) (uuid.UUID, error) {
			d, err := svc.CreateVehicleDocument(ctx, VehicleDocument{TenantID: tenantID, VehicleID: id, DocType: "registration", ObjectKey: "k/" + uuid.NewString(), Name: "tuev.pdf"})
			return d.ID, err
		},
	}
	for table, fn := range create {
		t.Run(table, func(t *testing.T) {
			_, err := fn(gone.ID)
			require.ErrorIs(t, err, ErrVehicleNotFound, "soft-deleted vehicle")
			_, err = fn(uuid.New())
			require.ErrorIs(t, err, ErrVehicleNotFound, "unknown vehicle")

			var n int
			require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
				`SELECT count(*) FROM `+table+` WHERE tenant_id = $1 AND vehicle_id = $2`, tenantID, gone.ID).Scan(&n))
			require.Zero(t, n, "no row for the deleted vehicle")

			id, err := fn(live.ID)
			require.NoError(t, err)
			t.Cleanup(func() { testutil.CleanupRow(t, pool, table, id) })
		})
	}
}
