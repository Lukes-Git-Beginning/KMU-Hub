package fuhrpark

// trip_logs.driver_user_id: the service guards the tenant (the FK cannot), the
// repository carries the link through Create/Update/List and keeps it when an
// update does not mention it.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func tripLogWithDriver(tenantID uuid.UUID, driverUserID *uuid.UUID, driverName string) TripLog {
	return TripLog{
		TenantID: tenantID, VehicleID: uuid.New(), StartLocation: "A", EndLocation: "B",
		StartKm: 10, EndKm: 20, DriverName: driverName, DriverUserID: driverUserID,
	}
}

func TestService_CreateTripLog_DriverUserMustBelongToTheTenant(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenant, foreign := uuid.New(), uuid.New()
	user := uuid.New()
	repo.driverUsers = map[string]string{foreign.String() + ":" + user.String(): "Fremde Fahrerin"}

	_, err := svc.CreateTripLog(context.Background(), tripLogWithDriver(tenant, &user, "Max"))
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateTripLog_DriverNameFilledFromTheAccountOnlyWhenEmpty(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenant, user := uuid.New(), uuid.New()
	repo.driverUsers = map[string]string{tenant.String() + ":" + user.String(): "Erika Musterfrau"}

	got, err := svc.CreateTripLog(context.Background(), tripLogWithDriver(tenant, &user, "  "))
	require.NoError(t, err)
	assert.Equal(t, "Erika Musterfrau", got.DriverName)

	got, err = svc.CreateTripLog(context.Background(), tripLogWithDriver(tenant, &user, "E. Muster"))
	require.NoError(t, err)
	assert.Equal(t, "E. Muster", got.DriverName)
}

func TestService_CreateTripLog_WithoutDriverUserDoesNotLookUp(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	got, err := svc.CreateTripLog(context.Background(), tripLogWithDriver(uuid.New(), nil, "Externer Fahrer"))
	require.NoError(t, err)
	assert.Equal(t, "Externer Fahrer", got.DriverName)
	assert.Zero(t, repo.driverLookups)
}

func TestService_UpdateTripLog_OnlyChecksTheDriverWhenItIsPartOfTheUpdate(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenant, user := uuid.New(), uuid.New()

	keep := tripLogWithDriver(tenant, nil, "Max")
	keep.ID = uuid.New()
	_, err := svc.UpdateTripLog(context.Background(), keep)
	require.NoError(t, err)
	assert.Zero(t, repo.driverLookups, "an update that does not carry the driver must not ask")

	clearIt := keep
	clearIt.Fields = TripLogDriverUser
	_, err = svc.UpdateTripLog(context.Background(), clearIt)
	require.NoError(t, err)
	assert.Zero(t, repo.driverLookups, "clearing the link needs no account")

	foreign := keep
	foreign.DriverUserID, foreign.Fields = &user, TripLogDriverUser
	_, err = svc.UpdateTripLog(context.Background(), foreign)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestTripLogs_DriverUserLinkAgainstRealSchema(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Trip Driver Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Trip Driver Tenant B")

	sysCtx := testutil.WithSystemCtx(context.Background())
	seedUser := func(tenantID uuid.UUID, first, last string) uuid.UUID {
		id := uuid.New()
		_, err := pool.Exec(sysCtx,
			`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name)
			 VALUES ($1, $2, $3, 'x', $4, $5)`,
			id, tenantID, id.String()+"@trip-driver.test", first, last)
		require.NoError(t, err)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	userA := seedUser(tenantA, "Erika", "Musterfrau")
	userA2 := seedUser(tenantA, "Max", "Mustermann")
	userB := seedUser(tenantB, "Frieda", "Fremd")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	vehicle := seedVehicleWithTuev(t, repo, ctxA, pool, tenantA, nil)
	today := time.Now().UTC().Truncate(24 * time.Hour)

	newLog := func(driver *uuid.UUID, name string) TripLog {
		return TripLog{
			TenantID: tenantA, VehicleID: vehicle.ID, Date: today, StartLocation: "A", EndLocation: "B",
			Purpose: "Termin", StartKm: 100, EndKm: 130, DriverName: name, DriverUserID: driver,
		}
	}

	t.Run("a user of another tenant is refused and nothing is written", func(t *testing.T) {
		_, err := svc.CreateTripLog(ctxA, newLog(&userB, "Frieda Fremd"))
		require.ErrorIs(t, err, ErrInvalidInput)
		_, total, err := repo.ListTripLogs(ctxA, ListTripLogsParams{TenantID: tenantA, VehicleID: vehicle.ID})
		require.NoError(t, err)
		assert.Zero(t, total)

		_, err = repo.DriverUserName(ctxA, tenantA, userB)
		assert.ErrorIs(t, err, ErrDriverNotFound)
	})

	created, err := svc.CreateTripLog(ctxA, newLog(&userA, ""))
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "trip_logs", created.ID) })

	t.Run("create stores the link and prints the account name", func(t *testing.T) {
		require.NotNil(t, created.DriverUserID)
		assert.Equal(t, userA, *created.DriverUserID)
		assert.Equal(t, "Erika Musterfrau", created.DriverName)

		list, _, err := repo.ListTripLogs(ctxA, ListTripLogsParams{TenantID: tenantA, VehicleID: vehicle.ID})
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.NotNil(t, list[0].DriverUserID)
		assert.Equal(t, userA, *list[0].DriverUserID)
	})

	t.Run("an update without the driver keeps the link", func(t *testing.T) {
		upd := created
		upd.Notes = "geaendert"
		upd.DriverUserID, upd.Fields = nil, TripLogNotes
		got, err := svc.UpdateTripLog(ctxA, upd)
		require.NoError(t, err)
		require.NotNil(t, got.DriverUserID)
		assert.Equal(t, userA, *got.DriverUserID)
	})

	t.Run("an update replaces the link, a foreign user is refused", func(t *testing.T) {
		upd := created
		upd.DriverUserID, upd.Fields = &userA2, TripLogDriverUser
		got, err := svc.UpdateTripLog(ctxA, upd)
		require.NoError(t, err)
		require.NotNil(t, got.DriverUserID)
		assert.Equal(t, userA2, *got.DriverUserID)

		upd.DriverUserID = &userB
		_, err = svc.UpdateTripLog(ctxA, upd)
		require.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("an update with an explicit nil clears the link", func(t *testing.T) {
		upd := created
		upd.DriverUserID, upd.Fields = nil, TripLogDriverUser
		got, err := svc.UpdateTripLog(ctxA, upd)
		require.NoError(t, err)
		assert.Nil(t, got.DriverUserID)
	})

	t.Run("deleting the account unlinks the entry and keeps the printed name", func(t *testing.T) {
		upd := created
		upd.DriverUserID, upd.Fields = &userA2, TripLogDriverUser
		_, err := svc.UpdateTripLog(ctxA, upd)
		require.NoError(t, err)

		_, err = pool.Exec(sysCtx, `DELETE FROM users WHERE id=$1`, userA2)
		require.NoError(t, err)

		list, _, err := repo.ListTripLogs(ctxA, ListTripLogsParams{TenantID: tenantA, VehicleID: vehicle.ID})
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Nil(t, list[0].DriverUserID)
		assert.Equal(t, "Erika Musterfrau", list[0].DriverName)
	})
}
