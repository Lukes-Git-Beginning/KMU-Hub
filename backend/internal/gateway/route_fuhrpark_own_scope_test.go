package gateway

// A member holds fuhrpark:booking:read|write, fuhrpark:license:read|write and
// fuhrpark:damage:read at scope "own" (migration 339). These tests drive the real router
// against a real gRPC server and pin what the scope has to mean: the request reaches the
// service narrowed to the caller, whatever the body or query names, and a role at scope "all"
// is not narrowed.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	fuhrparkv1 "github.com/kmuhub/kmuhub/proto/fuhrpark/v1"
)

// fuhrparkScopeStub records, per RPC, the identity the gateway forwarded: the owner/user/driver
// id a request was narrowed to, or "" when it was not narrowed.
type fuhrparkScopeStub struct {
	fuhrparkv1.UnimplementedFuhrparkServiceServer
	mu   sync.Mutex
	seen map[string]string
}

func (s *fuhrparkScopeStub) record(rpc, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seen == nil {
		s.seen = map[string]string{}
	}
	s.seen[rpc] = id
}

func (s *fuhrparkScopeStub) got(rpc string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.seen[rpc]
	return id, ok
}

func (s *fuhrparkScopeStub) ListVehicleBookings(_ context.Context, r *fuhrparkv1.ListVehicleBookingsRequest) (*fuhrparkv1.ListVehicleBookingsResponse, error) {
	s.record("ListVehicleBookings", r.GetUserId())
	return &fuhrparkv1.ListVehicleBookingsResponse{}, nil
}
func (s *fuhrparkScopeStub) CreateVehicleBooking(_ context.Context, r *fuhrparkv1.CreateVehicleBookingRequest) (*fuhrparkv1.VehicleBookingResponse, error) {
	s.record("CreateVehicleBooking", r.GetUserId())
	return &fuhrparkv1.VehicleBookingResponse{}, nil
}
func (s *fuhrparkScopeStub) UpdateVehicleBooking(_ context.Context, r *fuhrparkv1.UpdateVehicleBookingRequest) (*fuhrparkv1.VehicleBookingResponse, error) {
	s.record("UpdateVehicleBooking", r.GetOwnerUserId())
	return &fuhrparkv1.VehicleBookingResponse{}, nil
}
func (s *fuhrparkScopeStub) DeleteVehicleBooking(_ context.Context, r *fuhrparkv1.DeleteVehicleBookingRequest) (*fuhrparkv1.DeleteVehicleBookingResponse, error) {
	s.record("DeleteVehicleBooking", r.GetOwnerUserId())
	return &fuhrparkv1.DeleteVehicleBookingResponse{}, nil
}
func (s *fuhrparkScopeStub) ListDriverLicenses(_ context.Context, r *fuhrparkv1.ListDriverLicensesRequest) (*fuhrparkv1.ListDriverLicensesResponse, error) {
	s.record("ListDriverLicenses", r.GetDriverId())
	return &fuhrparkv1.ListDriverLicensesResponse{}, nil
}
func (s *fuhrparkScopeStub) CreateDriverLicense(_ context.Context, r *fuhrparkv1.CreateDriverLicenseRequest) (*fuhrparkv1.CreateDriverLicenseResponse, error) {
	s.record("CreateDriverLicense", r.GetDriverId())
	return &fuhrparkv1.CreateDriverLicenseResponse{}, nil
}
func (s *fuhrparkScopeStub) UpdateDriverLicense(_ context.Context, r *fuhrparkv1.UpdateDriverLicenseRequest) (*fuhrparkv1.UpdateDriverLicenseResponse, error) {
	s.record("UpdateDriverLicense", r.GetOwnerUserId())
	return &fuhrparkv1.UpdateDriverLicenseResponse{}, nil
}
func (s *fuhrparkScopeStub) DeleteDriverLicense(_ context.Context, r *fuhrparkv1.DeleteDriverLicenseRequest) (*fuhrparkv1.DeleteDriverLicenseResponse, error) {
	s.record("DeleteDriverLicense", r.GetOwnerUserId())
	return &fuhrparkv1.DeleteDriverLicenseResponse{}, nil
}
func (s *fuhrparkScopeStub) ListDamages(_ context.Context, r *fuhrparkv1.ListDamagesRequest) (*fuhrparkv1.ListDamagesResponse, error) {
	s.record("ListDamages", r.GetReportedBy())
	return &fuhrparkv1.ListDamagesResponse{}, nil
}
func (s *fuhrparkScopeStub) ReportDamage(_ context.Context, r *fuhrparkv1.ReportDamageRequest) (*fuhrparkv1.DamageResponse, error) {
	s.record("ReportDamage", r.GetReportedBy())
	return &fuhrparkv1.DamageResponse{}, nil
}

func TestFuhrparkRoutes_MemberSelfServiceIsNarrowedToTheCaller(t *testing.T) {
	const (
		vehicleID = "11111111-1111-1111-1111-111111111111"
		recordID  = "22222222-2222-2222-2222-222222222222"
	)
	callerID := uuid.NewString()
	stranger := uuid.NewString()

	// What the member preset holds after migration 339 (plus damage:create from 256).
	memberPerms := []string{
		"fuhrpark:booking:read", "fuhrpark:booking:write",
		"fuhrpark:license:read", "fuhrpark:license:write",
		"fuhrpark:damage:read", "fuhrpark:damage:create",
	}
	memberScopes := map[string]string{
		"fuhrpark:booking:read": auth.ScopeOwn, "fuhrpark:booking:write": auth.ScopeOwn,
		"fuhrpark:license:read": auth.ScopeOwn, "fuhrpark:license:write": auth.ScopeOwn,
		"fuhrpark:damage:read": auth.ScopeOwn,
	}
	allScopes := map[string]string{
		"fuhrpark:booking:read": auth.ScopeAll, "fuhrpark:booking:write": auth.ScopeAll,
		"fuhrpark:license:read": auth.ScopeAll, "fuhrpark:license:write": auth.ScopeAll,
		"fuhrpark:damage:read": auth.ScopeAll,
	}

	bookingBody := `{"vehicle_id":"` + vehicleID + `","user_id":"` + stranger + `","starts_at":"2026-11-01T08:00:00Z","ends_at":"2026-11-01T12:00:00Z"}`
	licenseBody := `{"driver_id":"` + stranger + `","license_classes":["B"],"next_check_due_date":"2027-01-01"}`
	licenseUpdate := `{"license_classes":["B"],"next_check_due_date":"2027-01-01"}`
	calls := []struct {
		name, rpc, method, path, body string
		wantStatus                    int
		// forwarded is what the stub saw for a narrowed caller; unrestricted is what it saw for
		// a caller at scope "all" ("" = not narrowed).
		unrestricted string
	}{
		{"list bookings", "ListVehicleBookings", "GET", "/api/v1/fuhrpark/bookings?user_id=" + stranger, "", http.StatusOK, stranger},
		{"create booking", "CreateVehicleBooking", "POST", "/api/v1/fuhrpark/bookings", bookingBody, http.StatusCreated, stranger},
		{"update booking", "UpdateVehicleBooking", "PATCH", "/api/v1/fuhrpark/bookings/" + recordID + "/", `{"status":"cancelled"}`, http.StatusOK, ""},
		{"delete booking", "DeleteVehicleBooking", "DELETE", "/api/v1/fuhrpark/bookings/" + recordID + "/", "", http.StatusNoContent, ""},
		{"list licenses", "ListDriverLicenses", "GET", "/api/v1/fuhrpark/driver-licenses?driver_id=" + stranger, "", http.StatusOK, stranger},
		{"create license", "CreateDriverLicense", "POST", "/api/v1/fuhrpark/driver-licenses", licenseBody, http.StatusCreated, stranger},
		{"update license", "UpdateDriverLicense", "PATCH", "/api/v1/fuhrpark/driver-licenses/" + recordID + "/", licenseUpdate, http.StatusOK, ""},
		{"delete license", "DeleteDriverLicense", "DELETE", "/api/v1/fuhrpark/driver-licenses/" + recordID + "/", "", http.StatusNoContent, ""},
		{"list damages", "ListDamages", "GET", "/api/v1/fuhrpark/damages", "", http.StatusOK, ""},
		{"list vehicle damages", "ListDamages", "GET", "/api/v1/fuhrpark/vehicles/" + vehicleID + "/damages", "", http.StatusOK, ""},
		// The reporter is pinned by damage:write, not by the read scope (migration 360): without
		// damage:write even a caller at scope all files under themselves.
		{"report damage", "ReportDamage", "POST", "/api/v1/fuhrpark/vehicles/" + vehicleID + "/damages",
			`{"description":"Delle","severity":"minor","reported_by":"` + stranger + `"}`, http.StatusCreated, callerID},
	}

	run := func(t *testing.T, scopes map[string]string, want func(unrestricted string) string) {
		t.Helper()
		stub := &fuhrparkScopeStub{}
		router := chi.NewRouter()
		newFuhrparkRoutes(registryFor(t, "fuhrpark", func(s *grpc.Server) { fuhrparkv1.RegisterFuhrparkServiceServer(s, stub) })).
			RegisterRoutes(router, guardTestAuth)

		for _, c := range calls {
			req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
			req.Header.Set("Content-Type", "application/json")
			req = withScopes(withPermissions(withAuth(req, callerID, testTenantID), memberPerms...), scopes)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assert.Equal(t, c.wantStatus, rec.Code, "%s: %s", c.name, rec.Body.String())
			id, seen := stub.got(c.rpc)
			require.True(t, seen, "%s never reached the service", c.name)
			assert.Equal(t, want(c.unrestricted), id, "%s: identity forwarded to the service", c.name)
		}
	}

	t.Run("member at own scope is narrowed to the caller whatever body or query names", func(t *testing.T) {
		run(t, memberScopes, func(string) string { return callerID })
	})

	t.Run("a role at scope all is not narrowed", func(t *testing.T) {
		run(t, allScopes, func(unrestricted string) string { return unrestricted })
	})

	t.Run("damage:write files a damage under the reporter named in the body", func(t *testing.T) {
		stub := &fuhrparkScopeStub{}
		router := chi.NewRouter()
		newFuhrparkRoutes(registryFor(t, "fuhrpark", func(s *grpc.Server) { fuhrparkv1.RegisterFuhrparkServiceServer(s, stub) })).
			RegisterRoutes(router, guardTestAuth)
		for name, tc := range map[string]struct {
			scopes map[string]string
			want   string
		}{
			"write at all": {map[string]string{"fuhrpark:damage:write": auth.ScopeAll}, stranger},
			"write at own": {map[string]string{"fuhrpark:damage:write": auth.ScopeOwn}, callerID},
		} {
			req := httptest.NewRequest("POST", "/api/v1/fuhrpark/vehicles/"+vehicleID+"/damages",
				strings.NewReader(`{"description":"Delle","severity":"minor","reported_by":"`+stranger+`"}`))
			req.Header.Set("Content-Type", "application/json")
			req = withScopes(withPermissions(withAuth(req, callerID, testTenantID), "fuhrpark:damage:write", "fuhrpark:damage:read"), tc.scopes)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusCreated, rec.Code, "%s: %s", name, rec.Body.String())
			id, _ := stub.got("ReportDamage")
			assert.Equal(t, tc.want, id, "%s: reporter forwarded to the service", name)
		}
	})

	t.Run("a token without the booking, license and damage keys is refused", func(t *testing.T) {
		stub := &fuhrparkScopeStub{}
		router := chi.NewRouter()
		newFuhrparkRoutes(registryFor(t, "fuhrpark", func(s *grpc.Server) { fuhrparkv1.RegisterFuhrparkServiceServer(s, stub) })).
			RegisterRoutes(router, guardTestAuth)
		for _, c := range calls {
			if c.rpc == "ReportDamage" {
				continue // damage:create is a key the member already held
			}
			req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
			req = withPermissions(withAuth(req, callerID, testTenantID), "fuhrpark:vehicle:read", "fuhrpark:fuel:read")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusForbidden, rec.Code, c.name)
		}
		assert.Empty(t, stub.seen)
	})
}
