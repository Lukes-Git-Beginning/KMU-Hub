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

// Since migration 360 a member holds fuhrpark:damage:read at scope all: the
// known damages of a pool vehicle are visible before taking it, whoever
// reported them. Update and resolve stay with damage:write (admin), and the
// reporter of a new damage stays the caller even though the read scope no
// longer narrows. Same production chain as the availability test: real seed
// through ResolveTokenPermissions, real Auth middleware and route registration,
// a real fuhrpark gRPC server and the Postgres repository.

var damageReadTenant = uuid.MustParse("e4fec700-0000-4000-8000-0000000000f4")

func TestFuhrparkDamageRead_MemberAgainstRealSeed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	testutil.EnsureTenant(t, pool, damageReadTenant, "DamageReadTenant")
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
			"tenant_id":     damageReadTenant,
			"email":         "damage-read-" + uuid.NewString() + "@test.local",
			"password_hash": "x",
			"first_name":    "Damage",
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
			userID, roleID, damageReadTenant); err != nil {
			t.Fatalf("assign role: %v", err)
		}
		perms, err := authSvc.ResolveTokenPermissions(sysCtx, damageReadTenant, userID)
		if err != nil {
			t.Fatalf("resolve permissions: %v", err)
		}
		token, err := tokenMaker.CreateAccessToken(userID, damageReadTenant.String(), []string{role}, perms)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}
		return userID, token
	}

	memberID, memberToken := tokenFor("member")
	colleagueID, _ := tokenFor("member")
	_, managerToken := tokenFor("manager")
	_, adminToken := tokenFor("admin")

	vehicleID := testutil.SeedRow(t, pool, "vehicles", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     damageReadTenant,
		"license_plate": "DR-" + uuid.NewString()[:6],
		"make":          "VW",
		"model":         "Caddy",
		"year":          2024,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicles", vehicleID) })

	const description = "Delle hinten links"
	damageID := testutil.SeedRow(t, pool, "vehicle_damages", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   damageReadTenant,
		"vehicle_id":  vehicleID,
		"description": description,
		"severity":    "moderate",
		"reported_by": colleagueID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicle_damages", damageID) })

	do := func(t *testing.T, method, path, body, token string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	listIDs := func(t *testing.T, path, token string) []string {
		t.Helper()
		rec := do(t, http.MethodGet, path, "", token)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, body %s", path, rec.Code, rec.Body.String())
		}
		var resp struct {
			Damages []struct {
				ID string `json:"id"`
			} `json:"damages"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		ids := make([]string, 0, len(resp.Damages))
		for _, d := range resp.Damages {
			ids = append(ids, d.ID)
		}
		return ids
	}
	contains := func(ids []string, id string) bool {
		for _, x := range ids {
			if x == id {
				return true
			}
		}
		return false
	}
	lists := map[string]string{
		"top-level": "/api/v1/fuhrpark/damages?vehicle_id=" + vehicleID.String(),
		"vehicle":   "/api/v1/fuhrpark/vehicles/" + vehicleID.String() + "/damages",
	}

	t.Run("member reads a colleague's damage in both lists", func(t *testing.T) {
		for name, path := range lists {
			if !contains(listIDs(t, path, memberToken), damageID.String()) {
				t.Errorf("%s list: member does not see the colleague's damage", name)
			}
		}
	})

	t.Run("manager keeps the own scope", func(t *testing.T) {
		for name, path := range lists {
			if contains(listIDs(t, path, managerToken), damageID.String()) {
				t.Errorf("%s list: manager sees the member's damage, scope own expected", name)
			}
		}
	})

	t.Run("member cannot update or resolve the damage", func(t *testing.T) {
		base := "/api/v1/fuhrpark/damages/" + damageID.String()
		if rec := do(t, http.MethodPatch, base, `{"status":"in_repair"}`, memberToken); rec.Code != http.StatusForbidden {
			t.Errorf("PATCH = %d, want 403 (body %s)", rec.Code, rec.Body.String())
		}
		if rec := do(t, http.MethodPost, base+"/resolve", `{"notes":"x"}`, memberToken); rec.Code != http.StatusForbidden {
			t.Errorf("resolve = %d, want 403 (body %s)", rec.Code, rec.Body.String())
		}
		var status string
		if err := pool.QueryRow(sysCtx, `SELECT status FROM vehicle_damages WHERE id = $1`, damageID).Scan(&status); err != nil {
			t.Fatalf("read status: %v", err)
		}
		if status != "reported" {
			t.Errorf("status = %q after refused writes, want reported", status)
		}
	})

	report := func(t *testing.T, token string) string {
		t.Helper()
		rec := do(t, http.MethodPost, "/api/v1/fuhrpark/vehicles/"+vehicleID.String()+"/damages",
			`{"description":"Kratzer","severity":"minor","reported_by":"`+colleagueID.String()+`"}`, token)
		if rec.Code != http.StatusCreated {
			t.Fatalf("report = %d, body %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Damage struct {
				ID         string `json:"id"`
				ReportedBy string `json:"reported_by"`
			} `json:"damage"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode report: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicle_damages", uuid.MustParse(resp.Damage.ID)) })
		return resp.Damage.ReportedBy
	}

	t.Run("member files a damage under themselves whatever the body names", func(t *testing.T) {
		if got := report(t, memberToken); got != memberID.String() {
			t.Errorf("reported_by = %q, want the caller %s, not the colleague named in the body", got, memberID)
		}
	})

	t.Run("admin files a damage under the named reporter", func(t *testing.T) {
		if got := report(t, adminToken); got != colleagueID.String() {
			t.Errorf("reported_by = %q, want the colleague %s named in the body", got, colleagueID)
		}
	})
}
