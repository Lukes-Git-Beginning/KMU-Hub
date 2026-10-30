package gateway

import (
	"context"
	"encoding/json"
	"io"
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
	"github.com/kmuhub/kmuhub/internal/crm/deal"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// A member writes the deals of their team (decision 2026-10-25): their own and
// those of their direct reports, never a peer's or their manager's, and
// deleting stays with manager. The chain is the production one: the real seed
// through ResolveTokenPermissions, the real Auth middleware and route
// registration, a real CRM gRPC server behind TenantInboundUnaryInterceptor and
// the Postgres deal repository.

var dealTeamWriteTenant = uuid.MustParse("e4fec700-0000-4000-8000-0000000000d3")

func TestCRMDealTeamWrite_MemberAgainstRealSeed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	testutil.EnsureTenant(t, pool, dealTeamWriteTenant, "DealTeamWriteTenant")
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	crmSrv := server.NewCRMGRPCServer(nil, nil, nil, nil, nil,
		deal.NewService(deal.NewPostgresRepository(pool)), nil, nil, nil, nil, nil)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	crmv1.RegisterCRMServiceServer(srv, crmSrv)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("crm", lis.Addr().String())
	t.Cleanup(reg.Close)

	router := chi.NewRouter()
	NewCRMRoutes(reg, nil).RegisterRoutes(router, middleware.Auth(authSvc))

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": dealTeamWriteTenant,
		"name":      "Erstkontakt",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "pipeline_stages", stageID) })

	type user struct {
		id    uuid.UUID
		token string
	}
	userFor := func(role string, manager *uuid.UUID) user {
		t.Helper()
		userID := testutil.SeedRow(t, pool, "users", map[string]any{
			"id":            uuid.New(),
			"tenant_id":     dealTeamWriteTenant,
			"email":         "deal-team-" + uuid.NewString() + "@test.local",
			"password_hash": "x",
			"first_name":    "Deal",
			"last_name":     role,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
		if manager != nil {
			profileID := testutil.SeedRow(t, pool, "hr_employee_profiles", map[string]any{
				"id":              uuid.New(),
				"tenant_id":       dealTeamWriteTenant,
				"user_id":         userID,
				"start_date":      time.Now(),
				"manager_user_id": *manager,
			})
			t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_profiles", profileID) })
		}
		var roleID uuid.UUID
		if err := pool.QueryRow(sysCtx,
			`SELECT id FROM roles WHERE name = $1 AND tenant_id IS NULL`, role).Scan(&roleID); err != nil {
			t.Fatalf("preset role %q: %v", role, err)
		}
		if _, err := pool.Exec(sysCtx,
			`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`,
			userID, roleID, dealTeamWriteTenant); err != nil {
			t.Fatalf("assign role: %v", err)
		}
		perms, err := authSvc.ResolveTokenPermissions(sysCtx, dealTeamWriteTenant, userID)
		if err != nil {
			t.Fatalf("resolve permissions: %v", err)
		}
		token, err := tokenMaker.CreateAccessToken(userID, dealTeamWriteTenant.String(), []string{role}, perms)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}
		return user{id: userID, token: token}
	}

	do := func(method, token, path, body string) (int, map[string]any) {
		t.Helper()
		var rd io.Reader
		if body != "" {
			rd = strings.NewReader(body)
		}
		req := httptest.NewRequest(method, "/api/v1"+path, rd)
		req.Header.Set("Authorization", "Bearer "+token)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", uuid.NewString())
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}
	dealBody := func(name string, owner *uuid.UUID) string {
		b := `{"name":"` + name + `","stage_id":"` + stageID.String() + `"`
		if owner != nil {
			b += `,"owner_id":"` + owner.String() + `"`
		}
		return b + `}`
	}
	createDeal := func(t *testing.T, token, body string, want int) string {
		t.Helper()
		code, out := do(http.MethodPost, token, "/deals", body)
		inner, _ := out["deal"].(map[string]any)
		id, _ := inner["id"].(string)
		if id != "" {
			t.Cleanup(func() { testutil.CleanupRow(t, pool, "deals", uuid.MustParse(id)) })
		}
		if code != want {
			t.Fatalf("POST /deals %s = %d %v, want %d", body, code, out, want)
		}
		if code == http.StatusCreated && id == "" {
			t.Fatalf("no deal.id in %v", out)
		}
		return id
	}
	nameOf := func(t *testing.T, token, id string) string {
		t.Helper()
		code, out := do(http.MethodGet, token, "/deals/"+id, "")
		if code != http.StatusOK {
			t.Fatalf("GET /deals/%s = %d %v", id, code, out)
		}
		inner, _ := out["deal"].(map[string]any)
		name, _ := inner["name"].(string)
		return name
	}

	manager := userFor("manager", nil)
	admin := userFor("admin", nil)
	lead := userFor("member", &manager.id)
	report := userFor("member", &lead.id)
	peer := userFor("member", nil)

	own := createDeal(t, lead.token, dealBody("Eigener Deal", nil), http.StatusCreated)
	forReport := createDeal(t, lead.token, dealBody("Deal fuer den Bericht", &report.id), http.StatusCreated)
	reports := createDeal(t, report.token, dealBody("Deal des Berichts", nil), http.StatusCreated)
	peers := createDeal(t, peer.token, dealBody("Deal des Kollegen", nil), http.StatusCreated)

	t.Run("member creates a deal for a peer only through a manager", func(t *testing.T) {
		createDeal(t, lead.token, dealBody("Deal fuer den Kollegen", &peer.id), http.StatusForbidden)
		createDeal(t, manager.token, dealBody("Deal vom Manager", &peer.id), http.StatusCreated)
	})

	t.Run("member edits their own and their reports' deals", func(t *testing.T) {
		for _, id := range []string{own, forReport, reports} {
			if code, out := do(http.MethodPut, lead.token, "/deals/"+id, `{"name":"Gepflegt"}`); code != http.StatusOK {
				t.Fatalf("PUT team deal %s = %d %v, want 200", id, code, out)
			}
			if code, out := do(http.MethodPost, lead.token, "/deals/"+id+"/stage", `{"stage_id":"`+stageID.String()+`"}`); code != http.StatusOK {
				t.Errorf("POST stage team deal %s = %d %v, want 200", id, code, out)
			}
			if got := nameOf(t, lead.token, id); got != "Gepflegt" {
				t.Errorf("team deal %s name = %q after PUT, want Gepflegt", id, got)
			}
		}
	})

	t.Run("a peer's deal stays read-only for the member", func(t *testing.T) {
		for _, c := range []struct{ method, path, body string }{
			{http.MethodPut, "/deals/" + peers, `{"name":"Uebernommen"}`},
			{http.MethodPost, "/deals/" + peers + "/stage", `{"stage_id":"` + stageID.String() + `"}`},
			{http.MethodPost, "/deals/" + peers + "/tags", `{"tag_ids":[]}`},
			{http.MethodDelete, "/deals/" + peers + "/tags", `{"tag_ids":[]}`},
		} {
			if code, out := do(c.method, lead.token, c.path, c.body); code != http.StatusForbidden {
				t.Errorf("%s %s = %d %v, want 403", c.method, c.path, code, out)
			}
		}
		if got := nameOf(t, lead.token, peers); got != "Deal des Kollegen" {
			t.Errorf("peer deal renamed to %q by a refused PUT", got)
		}
	})

	t.Run("deal tags stay with deals:write (no desktop caller)", func(t *testing.T) {
		if code, out := do(http.MethodPost, lead.token, "/deals/"+own+"/tags", `{"tag_ids":[]}`); code != http.StatusForbidden {
			t.Errorf("member POST tags on own deal = %d %v, want 403", code, out)
		}
	})

	t.Run("the team reaches down the reporting line, never up", func(t *testing.T) {
		if code, out := do(http.MethodPut, report.token, "/deals/"+own, `{"name":"Von unten"}`); code != http.StatusForbidden {
			t.Errorf("report PUT lead's deal = %d %v, want 403", code, out)
		}
	})

	t.Run("member cannot hand their deal out of the team", func(t *testing.T) {
		if code, out := do(http.MethodPut, lead.token, "/deals/"+own, `{"owner_id":"`+peer.id.String()+`"}`); code != http.StatusForbidden {
			t.Errorf("PUT owner_id=peer = %d %v, want 403", code, out)
		}
	})

	t.Run("deleting stays with manager, even the member's own deal", func(t *testing.T) {
		if code, out := do(http.MethodDelete, lead.token, "/deals/"+own, ""); code != http.StatusForbidden {
			t.Errorf("member DELETE own deal = %d %v, want 403", code, out)
		}
	})

	// manager holds deals:write but not deals:delete and deletes at
	// crm:deal:delete=team: their own deals and their direct reports', not a
	// peer's outside the reporting line. admin deletes tenant-wide.
	t.Run("manager deletes only deals of their team", func(t *testing.T) {
		outside := createDeal(t, peer.token, dealBody("Deal ausserhalb der Linie", nil), http.StatusCreated)
		if code, out := do(http.MethodDelete, manager.token, "/deals/"+outside, ""); code != http.StatusForbidden {
			t.Errorf("manager DELETE deal outside the line = %d %v, want 403", code, out)
		}
		if got := nameOf(t, manager.token, outside); got != "Deal ausserhalb der Linie" {
			t.Errorf("deal outside the line = %q after refused DELETE", got)
		}
		mine := createDeal(t, manager.token, dealBody("Deal des Managers", nil), http.StatusCreated)
		leads := createDeal(t, lead.token, dealBody("Deal des Berichts", nil), http.StatusCreated)
		for _, id := range []string{mine, leads} {
			if code, out := do(http.MethodDelete, manager.token, "/deals/"+id, ""); code != http.StatusOK {
				t.Errorf("manager DELETE team deal %s = %d %v, want 200", id, code, out)
			}
			if code, _ := do(http.MethodGet, manager.token, "/deals/"+id, ""); code != http.StatusNotFound {
				t.Errorf("GET deleted team deal %s = %d, want 404", id, code)
			}
		}
		if code, out := do(http.MethodDelete, admin.token, "/deals/"+outside, ""); code != http.StatusOK {
			t.Errorf("admin DELETE deal outside any line = %d %v, want 200", code, out)
		}
	})

	t.Run("manager still writes any deal tenant-wide", func(t *testing.T) {
		if code, out := do(http.MethodPut, manager.token, "/deals/"+peers, `{"name":"Vom Manager"}`); code != http.StatusOK {
			t.Errorf("manager PUT peer deal = %d %v, want 200", code, out)
		}
	})
}
