package gateway

// Fuhrpark responses as FuhrparkPage reads them: a route always carries `driver` (the page
// calls `r.driver.toLowerCase()` while searching), and the counters of a vehicle, fuel log, trip
// and route stay in the body at 0 (`Number(undefined)` is NaN). A loopback gRPC server plays
// the fuhrpark service.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	fuhrparkv1 "github.com/kmuhub/kmuhub/proto/fuhrpark/v1"
)

type fuhrparkWireStub struct {
	fuhrparkv1.UnimplementedFuhrparkServiceServer
	empty bool
}

func (s *fuhrparkWireStub) ListVehicles(context.Context, *fuhrparkv1.ListVehiclesRequest) (*fuhrparkv1.ListVehiclesResponse, error) {
	if s.empty {
		return &fuhrparkv1.ListVehiclesResponse{}, nil
	}
	return &fuhrparkv1.ListVehiclesResponse{Vehicles: []*fuhrparkv1.Vehicle{
		{Id: "v1", LicensePlate: "M-AB 1", Make: "VW", Model: "Caddy", Status: "active"},
	}, Total: 1}, nil
}

func (s *fuhrparkWireStub) ListFuelLogs(context.Context, *fuhrparkv1.ListFuelLogsRequest) (*fuhrparkv1.ListFuelLogsResponse, error) {
	if s.empty {
		return &fuhrparkv1.ListFuelLogsResponse{}, nil
	}
	return &fuhrparkv1.ListFuelLogsResponse{FuelLogs: []*fuhrparkv1.FuelLog{
		{Id: "f1", VehicleId: "v1", Date: "2026-10-01"},
	}, Total: 1}, nil
}

func (s *fuhrparkWireStub) ListTripLogs(context.Context, *fuhrparkv1.ListTripLogsRequest) (*fuhrparkv1.ListTripLogsResponse, error) {
	if s.empty {
		return &fuhrparkv1.ListTripLogsResponse{}, nil
	}
	return &fuhrparkv1.ListTripLogsResponse{TripLogs: []*fuhrparkv1.TripLog{
		{Id: "t1", VehicleId: "v1", Date: "2026-10-01", Purpose: "Kunde"},
	}, Total: 1}, nil
}

func (s *fuhrparkWireStub) GetVehicleRoutes(context.Context, *fuhrparkv1.GetVehicleRoutesRequest) (*fuhrparkv1.GetVehicleRoutesResponse, error) {
	if s.empty {
		return &fuhrparkv1.GetVehicleRoutesResponse{}, nil
	}
	return &fuhrparkv1.GetVehicleRoutesResponse{Routes: []*fuhrparkv1.VehicleRoute{
		{VehicleId: "v1", VehicleName: "M-AB 1", Date: "2026-10-01", Status: "parked"},
	}}, nil
}

func newFuhrparkWireRoutes(t *testing.T, empty bool) *FuhrparkRoutes {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	fuhrparkv1.RegisterFuhrparkServiceServer(srv, &fuhrparkWireStub{empty: empty})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("fuhrpark", lis.Addr().String())
	t.Cleanup(reg.Close)
	return NewFuhrparkRoutes(reg, nil)
}

func fuhrparkListFirst(t *testing.T, call func(http.ResponseWriter, *http.Request), key string) map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	call(rec, withAuth(httptest.NewRequest(http.MethodGet, "/x", nil), testHRUserID, testTenantID))
	assertStatus(t, rec, http.StatusOK)
	list, ok := decodeBody(t, rec)[key].([]any)
	require.True(t, ok, rec.Body.String())
	require.Len(t, list, 1)
	return list[0].(map[string]any)
}

func TestFuhrparkWire_ZeroValuesStayInTheBody(t *testing.T) {
	routes := newFuhrparkWireRoutes(t, false)

	cases := []struct {
		name   string
		call   func(http.ResponseWriter, *http.Request)
		key    string
		fields []string
	}{
		{"vehicle", routes.HandleListVehicles, "vehicles", []string{"mileage_km", "year"}},
		{"fuel log", routes.HandleListFuelLogs, "fuel_logs", []string{"liters", "cost_cents", "mileage_km"}},
		{"trip log", routes.HandleListTripLogs, "trip_logs", []string{"start_km", "end_km", "km", "is_private"}},
		{"route", routes.HandleGetVehicleRoutes, "routes", []string{"daily_km", "driver", "positions"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			item := fuhrparkListFirst(t, c.call, c.key)
			for _, f := range c.fields {
				assert.Contains(t, item, f)
			}
		})
	}
}

func TestFuhrparkWire_RouteCarriesDriverAsString(t *testing.T) {
	routes := newFuhrparkWireRoutes(t, false)
	item := fuhrparkListFirst(t, routes.HandleGetVehicleRoutes, "routes")
	assert.Equal(t, "", item["driver"])
	assert.Equal(t, []any{}, item["positions"])
}

func TestFuhrparkWire_EmptyListsAreArrays(t *testing.T) {
	routes := newFuhrparkWireRoutes(t, true)

	cases := map[string]struct {
		call func(http.ResponseWriter, *http.Request)
		key  string
	}{
		"vehicles":  {routes.HandleListVehicles, "vehicles"},
		"fuel_logs": {routes.HandleListFuelLogs, "fuel_logs"},
		"trip_logs": {routes.HandleListTripLogs, "trip_logs"},
		"routes":    {routes.HandleGetVehicleRoutes, "routes"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c.call(rec, withAuth(httptest.NewRequest(http.MethodGet, "/x", nil), testHRUserID, testTenantID))
			assertStatus(t, rec, http.StatusOK)
			list, ok := decodeBody(t, rec)[c.key].([]any)
			require.True(t, ok, rec.Body.String())
			assert.Empty(t, list)
		})
	}
}
