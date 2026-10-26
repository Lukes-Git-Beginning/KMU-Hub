package gateway

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/fuhrpark"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	fuhrparkv1 "github.com/kmuhub/kmuhub/proto/fuhrpark/v1"
)

// A member holds fuhrpark:booking:read only at scope own (000339), so the
// booking list shows nothing of a colleague's reservation and the member used
// to learn that a pool vehicle is taken from the 409 of their own booking.
// GET /vehicles/{id}/availability answers with the occupied windows of every
// booking, but without who booked or why. The chain is the production one: the
// real seed through ResolveTokenPermissions, the real Auth middleware and route
// registration, a real fuhrpark gRPC server behind TenantInboundUnaryInterceptor
// and the Postgres repository.

var availabilityTenant = uuid.MustParse("e4fec700-0000-4000-8000-0000000000f3")

func TestFuhrparkVehicleAvailability_MemberAgainstRealSeed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	testutil.EnsureTenant(t, pool, availabilityTenant, "AvailabilityTenant")
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	fuhrparkv1.RegisterFuhrparkServiceServer(srv, server.NewFuhrparkGRPCServer(fuhrpark.NewService(fuhrpark.NewPostgresRepository(pool))))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("fuhrpark", lis.Addr().String())
	t.Cleanup(reg.Close)

	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_FUHRPARK_ENABLED" {
			return "true"
		}
		return ""
	})
	router := chi.NewRouter()
	NewFuhrparkRoutes(reg, flags).RegisterRoutes(router, middleware.Auth(authSvc))

	tokenFor := func(role string) (uuid.UUID, string) {
		t.Helper()
		userID := testutil.SeedRow(t, pool, "users", map[string]any{
			"id":            uuid.New(),
			"tenant_id":     availabilityTenant,
			"email":         "availability-" + uuid.NewString() + "@test.local",
			"password_hash": "x",
			"first_name":    "Availability",
			"last_name":     role,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
		var roleID uuid.UUID
		if err := pool.QueryRow(sysCtx,
			`SELECT id FROM roles WHERE name = $1 AND tenant_id IS NULL`, role).Scan(&roleID); err != nil {
			t.Fatalf("preset role %q: %v", role, err)
		}
		if _, err := pool.Exec(sysCtx,
			`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`,
			userID, roleID, availabilityTenant); err != nil {
			t.Fatalf("assign role: %v", err)
		}
		perms, err := authSvc.ResolveTokenPermissions(sysCtx, availabilityTenant, userID)
		if err != nil {
			t.Fatalf("resolve permissions: %v", err)
		}
		token, err := tokenMaker.CreateAccessToken(userID, availabilityTenant.String(), []string{role}, perms)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}
		return userID, token
	}

	_, memberToken := tokenFor("member")
	colleagueID, _ := tokenFor("member")
	_, readonlyToken := tokenFor("readonly")

	vehicleID := testutil.SeedRow(t, pool, "vehicles", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     availabilityTenant,
		"license_plate": "AV-" + uuid.NewString()[:6],
		"make":          "VW",
		"model":         "Caddy",
		"year":          2024,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicles", vehicleID) })

	day := time.Now().UTC().Truncate(24 * time.Hour).Add(72 * time.Hour)
	const purpose = "Arzttermin Kollege"
	seedBooking := func(start, end int, status string) {
		id := testutil.SeedRow(t, pool, "vehicle_bookings", map[string]any{
			"id":         uuid.New(),
			"tenant_id":  availabilityTenant,
			"vehicle_id": vehicleID,
			"user_id":    colleagueID,
			"starts_at":  day.Add(time.Duration(start) * time.Hour),
			"ends_at":    day.Add(time.Duration(end) * time.Hour),
			"purpose":    purpose,
			"status":     status,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicle_bookings", id) })
	}
	seedBooking(9, 11, "booked")
	seedBooking(14, 15, "cancelled")

	do := func(token, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	window := "?from=" + day.Format(time.RFC3339) + "&to=" + day.Add(24*time.Hour).Format(time.RFC3339)
	path := "/api/v1/fuhrpark/vehicles/" + vehicleID.String() + "/availability"

	t.Run("member sees a colleague's booked window without user or purpose", func(t *testing.T) {
		rec := do(memberToken, path+window)
		if rec.Code != http.StatusOK {
			t.Fatalf("availability = %d, body %s", rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		for _, leak := range []string{colleagueID.String(), purpose, "user_id", "purpose"} {
			if strings.Contains(body, leak) {
				t.Errorf("availability body leaks %q: %s", leak, body)
			}
		}
		var resp struct {
			VehicleID string `json:"vehicle_id"`
			Items     []struct {
				StartsAt time.Time `json:"starts_at"`
				EndsAt   time.Time `json:"ends_at"`
			} `json:"items"`
			Total int `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.VehicleID != vehicleID.String() || resp.Total != 1 || len(resp.Items) != 1 {
			t.Fatalf("availability = %+v, want the one active booking of the vehicle", resp)
		}
		if !resp.Items[0].StartsAt.Equal(day.Add(9*time.Hour)) || !resp.Items[0].EndsAt.Equal(day.Add(11*time.Hour)) {
			t.Errorf("slot = %v-%v, want 09:00-11:00", resp.Items[0].StartsAt, resp.Items[0].EndsAt)
		}
	})

	t.Run("the booking list itself stays own-scoped for the member", func(t *testing.T) {
		rec := do(memberToken, "/api/v1/fuhrpark/bookings?vehicle_id="+vehicleID.String())
		if rec.Code != http.StatusOK {
			t.Fatalf("list bookings = %d, body %s", rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), purpose) {
			t.Errorf("member booking list shows the colleague's booking: %s", rec.Body.String())
		}
	})

	t.Run("a free window answers an empty list, not null", func(t *testing.T) {
		free := "?from=" + day.Add(-48*time.Hour).Format(time.RFC3339) + "&to=" + day.Add(-24*time.Hour).Format(time.RFC3339)
		rec := do(memberToken, path+free)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"items":[]`) || !strings.Contains(rec.Body.String(), `"total":0`) {
			t.Errorf("free window = %d %s, want 200 with items [] and total 0", rec.Code, rec.Body.String())
		}
	})

	t.Run("bad input and unknown vehicles", func(t *testing.T) {
		for name, tc := range map[string]struct {
			path string
			want int
		}{
			"missing window":     {path, http.StatusBadRequest},
			"reversed window":    {path + "?from=" + day.Add(time.Hour).Format(time.RFC3339) + "&to=" + day.Format(time.RFC3339), http.StatusBadRequest},
			"window too long":    {path + "?from=" + day.Format(time.RFC3339) + "&to=" + day.Add(100*24*time.Hour).Format(time.RFC3339), http.StatusBadRequest},
			"not a timestamp":    {path + "?from=morgen&to=" + day.Format(time.RFC3339), http.StatusBadRequest},
			"unknown vehicle":    {"/api/v1/fuhrpark/vehicles/" + uuid.NewString() + "/availability" + window, http.StatusNotFound},
			"invalid vehicle id": {"/api/v1/fuhrpark/vehicles/nope/availability" + window, http.StatusBadRequest},
		} {
			if rec := do(memberToken, tc.path); rec.Code != tc.want {
				t.Errorf("%s = %d, want %d (body %s)", name, rec.Code, tc.want, rec.Body.String())
			}
		}
	})

	t.Run("a role without any booking key gets 403", func(t *testing.T) {
		if rec := do(readonlyToken, path+window); rec.Code != http.StatusForbidden {
			t.Errorf("readonly availability = %d, want 403 (body %s)", rec.Code, rec.Body.String())
		}
	})
}
