package fuhrpark

// Private trips (decision 2026-10-25): everyone but the driver sees date,
// odometer readings, distance and the label "Privatfahrt" -- no destination,
// route or purpose. The export keeps the trip (a Fahrtenbuch needs its km),
// masked the same way.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestMaskPrivateTrips(t *testing.T) {
	driver := uuid.New()
	trips := []TripLog{
		{StartLocation: "A", EndLocation: "B", Purpose: "Termin", BusinessPartner: "Muster GmbH", Notes: "n", StartKm: 1, EndKm: 5, Km: 4},
		{StartLocation: "A", EndLocation: "B", Purpose: "Arzt", BusinessPartner: "x", Notes: "n", StartKm: 5, EndKm: 9, Km: 4, IsPrivate: true, DriverUserID: &driver, DriverName: "Erika"},
		{StartLocation: "A", EndLocation: "B", Purpose: "Arzt", Notes: "n", StartKm: 9, EndKm: 12, Km: 3, IsPrivate: true},
	}

	t.Run("for the driver only the unlinked private trip is masked", func(t *testing.T) {
		got := append([]TripLog(nil), trips...)
		maskPrivateTrips(got, driver)
		assert.Equal(t, trips[0], got[0], "a business trip is never touched")
		assert.Equal(t, trips[1], got[1], "the driver sees their own private trip")
		assert.Equal(t, privateTripLabel, got[2].Purpose, "a private trip without a driver account is masked for everyone")
	})

	t.Run("for anyone else every private trip is masked, km and driver stay", func(t *testing.T) {
		got := append([]TripLog(nil), trips...)
		maskPrivateTrips(got, uuid.New())
		assert.Equal(t, trips[0], got[0])
		want := trips[1]
		want.StartLocation, want.EndLocation, want.Purpose, want.BusinessPartner, want.Notes = "", "", privateTripLabel, "", ""
		assert.Equal(t, want, got[1])
	})

	t.Run("no viewer masks every private trip", func(t *testing.T) {
		got := append([]TripLog(nil), trips...)
		maskPrivateTrips(got, uuid.Nil)
		assert.Equal(t, privateTripLabel, got[1].Purpose)
		assert.Empty(t, got[1].EndLocation)
	})
}

func TestPrivateTripsMaskedAgainstRealSchema(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Private Trip Mask Tenant")
	sysCtx := testutil.WithSystemCtx(context.Background())
	seedUser := func(first string) uuid.UUID {
		id := uuid.New()
		_, err := pool.Exec(sysCtx,
			`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name)
			 VALUES ($1, $2, $3, 'x', $4, 'Test')`,
			id, tenantID, id.String()+"@private-trip.test", first)
		require.NoError(t, err)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	driver, manager := seedUser("Erika"), seedUser("Max")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	vehicle := seedVehicleWithTuev(t, repo, ctx, pool, tenantID, nil)
	day := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)

	create := func(trip TripLog) TripLog {
		trip.TenantID, trip.VehicleID, trip.Date = tenantID, vehicle.ID, day
		created, err := svc.CreateTripLog(ctx, trip)
		require.NoError(t, err)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "trip_logs", created.ID) })
		return created
	}
	business := create(TripLog{StartLocation: "Buero", EndLocation: "Kunde Nord", Purpose: "Kundentermin",
		StartKm: 100, EndKm: 140, DriverUserID: &driver})
	private := create(TripLog{StartLocation: "Zuhause", EndLocation: "Klinik Sued", Purpose: "Arzttermin",
		BusinessPartner: "Dr. Beispiel", Notes: "Kontrolle", StartKm: 140, EndKm: 182, IsPrivate: true, DriverUserID: &driver})

	listed := func(t *testing.T, viewer uuid.UUID) map[uuid.UUID]TripLog {
		t.Helper()
		logs, total, err := svc.ListTripLogs(ctx, ListTripLogsParams{TenantID: tenantID, VehicleID: vehicle.ID, ViewerID: viewer})
		require.NoError(t, err)
		require.Equal(t, 2, total, "masking never drops a trip")
		byID := make(map[uuid.UUID]TripLog, len(logs))
		for _, l := range logs {
			byID[l.ID] = l
		}
		return byID
	}

	t.Run("the driver lists their private trip in full", func(t *testing.T) {
		got := listed(t, driver)[private.ID]
		assert.Equal(t, "Klinik Sued", got.EndLocation)
		assert.Equal(t, "Arzttermin", got.Purpose)
		assert.Equal(t, "Dr. Beispiel", got.BusinessPartner)
	})

	t.Run("a manager lists the private trip masked, the business trip in full", func(t *testing.T) {
		byID := listed(t, manager)
		got := byID[private.ID]
		assert.Empty(t, got.StartLocation)
		assert.Empty(t, got.EndLocation)
		assert.Empty(t, got.BusinessPartner)
		assert.Empty(t, got.Notes)
		assert.Equal(t, privateTripLabel, got.Purpose)
		assert.Equal(t, int64(140), got.StartKm)
		assert.Equal(t, int64(182), got.EndKm)
		assert.Equal(t, int64(42), got.Km)
		assert.True(t, got.IsPrivate)
		assert.Equal(t, day, got.Date.UTC())
		assert.Equal(t, "Kunde Nord", byID[business.ID].EndLocation)
	})

	export := func(t *testing.T, viewer uuid.UUID) string {
		t.Helper()
		payload, _, _, err := svc.ExportTripLogs(ctx, ExportTripLogsParams{TenantID: tenantID, VehicleID: vehicle.ID, ViewerID: viewer}, "csv")
		require.NoError(t, err)
		return string(payload)
	}

	t.Run("the manager's export keeps the private trip, masked", func(t *testing.T) {
		csv := export(t, manager)
		assert.NotContains(t, csv, "Klinik Sued")
		assert.NotContains(t, csv, "Arzttermin")
		assert.NotContains(t, csv, "Dr. Beispiel")
		assert.Contains(t, csv, privateTripLabel)
		assert.Contains(t, csv, "182")
		assert.Contains(t, csv, "Kunde Nord")
		assert.Len(t, strings.Split(strings.TrimSpace(csv), "\n"), 3, "header, business trip, private trip")
	})

	t.Run("the driver's export shows their private trip", func(t *testing.T) {
		assert.Contains(t, export(t, driver), "Klinik Sued")
	})

	t.Run("the pdf export renders the masked trip", func(t *testing.T) {
		payload, contentType, _, err := svc.ExportTripLogs(ctx, ExportTripLogsParams{TenantID: tenantID, VehicleID: vehicle.ID, ViewerID: manager}, "pdf")
		require.NoError(t, err)
		assert.Equal(t, "application/pdf", contentType)
		assert.NotEmpty(t, payload)
	})
}
