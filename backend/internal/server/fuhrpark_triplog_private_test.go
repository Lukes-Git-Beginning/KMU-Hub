package server

// Private trips are masked for everyone but their driver (decision
// 2026-10-25). The masking itself lives in the fuhrpark service; what only
// the server can get wrong is the viewer: it must come from the authenticated
// x-user-id on both the list and the export, or the driver sees their own trip
// masked (viewer dropped) or a colleague sees it in full (viewer taken from
// somewhere else).

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/fuhrpark"
	fuhrparkv1 "github.com/kmuhub/kmuhub/proto/fuhrpark/v1"
)

func TestFuhrparkGRPC_PrivateTripsUnmaskedOnlyForTheirDriver(t *testing.T) {
	tenantID, driver, colleague := uuid.New(), uuid.New(), uuid.New()
	repo := newStubFuhrparkRepo()
	repo.tripLogs = []fuhrpark.TripLog{{
		ID: uuid.New(), TenantID: tenantID, VehicleID: uuid.New(),
		Date:          time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC),
		StartLocation: "Zuhause", EndLocation: "Klinik Sued", Purpose: "Arzttermin",
		StartKm: 1000, EndKm: 1042, Km: 42, IsPrivate: true, DriverName: "Erika Fahrer",
		DriverUserID: &driver,
	}}
	s := newFuhrparkTestServer(repo)

	list := func(t *testing.T, viewer uuid.UUID) *fuhrparkv1.TripLog {
		t.Helper()
		resp, err := s.ListTripLogs(fuhrparkCtxWithTenantAndUser(tenantID, viewer), &fuhrparkv1.ListTripLogsRequest{})
		require.NoError(t, err)
		require.Len(t, resp.TripLogs, 1)
		return resp.TripLogs[0]
	}
	export := func(t *testing.T, viewer uuid.UUID) string {
		t.Helper()
		resp, err := s.ExportTripLogs(fuhrparkCtxWithTenantAndUser(tenantID, viewer),
			&fuhrparkv1.ExportTripLogsRequest{TenantId: tenantID.String(), Format: "csv"})
		require.NoError(t, err)
		return string(resp.Payload)
	}

	t.Run("the driver lists the full trip", func(t *testing.T) {
		got := list(t, driver)
		require.Equal(t, "Klinik Sued", got.EndLocation)
		require.Equal(t, "Arzttermin", got.Purpose)
	})

	t.Run("a colleague lists the trip masked, km intact", func(t *testing.T) {
		got := list(t, colleague)
		require.Empty(t, got.StartLocation)
		require.Empty(t, got.EndLocation)
		require.Equal(t, "Privatfahrt", got.Purpose)
		require.EqualValues(t, 1000, got.StartKm)
		require.EqualValues(t, 1042, got.EndKm)
		require.True(t, got.IsPrivate)
	})

	t.Run("without a caller the trip is masked", func(t *testing.T) {
		resp, err := s.ListTripLogs(fuhrparkCtxWithTenant(tenantID), &fuhrparkv1.ListTripLogsRequest{})
		require.NoError(t, err)
		require.Len(t, resp.TripLogs, 1)
		require.Empty(t, resp.TripLogs[0].EndLocation)
	})

	t.Run("the driver exports the full trip", func(t *testing.T) {
		require.Contains(t, export(t, driver), "Klinik Sued")
	})

	t.Run("a colleague exports the trip masked, not omitted", func(t *testing.T) {
		csv := export(t, colleague)
		require.NotContains(t, csv, "Klinik Sued")
		require.NotContains(t, csv, "Arzttermin")
		require.Contains(t, csv, "Privatfahrt")
		require.Contains(t, csv, "1042")
		require.Equal(t, 2, strings.Count(strings.TrimSpace(csv), "\n")+1, "header plus the private trip")
	})
}
