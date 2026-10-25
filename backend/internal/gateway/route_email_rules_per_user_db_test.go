package gateway

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/email/rule"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// A member keeps their own email rules and cannot touch a colleague's or the
// tenant-wide ones (decision 2026-10-25: rules per user, existing rules stay
// tenant-wide admin rules). The chain is the production one: the real seed
// through ResolveTokenPermissions, the real Auth middleware and route
// registration, a real email gRPC server behind TenantInboundUnaryInterceptor
// and the Postgres rule repository.
func TestEmailRules_MemberManagesOwnRulesOnly(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Email Rules Per User")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	emailv1.RegisterEmailServiceServer(srv, server.NewEmailGRPCServer(
		nil, nil, nil, nil, nil, nil, nil, nil, nil,
		rule.NewService(rule.NewPostgresRepository(pool)), nil, nil,
	))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("email", lis.Addr().String())
	t.Cleanup(reg.Close)

	router := chi.NewRouter()
	NewEmailRoutes(reg).RegisterRoutes(router, middleware.Auth(authSvc))

	userFor := func(role string) (uuid.UUID, string) {
		t.Helper()
		userID := testutil.SeedRow(t, pool, "users", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "email": "email-rule-" + uuid.NewString() + "@test.local",
			"password_hash": "x", "first_name": "Rule", "last_name": role,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
		if role != "" {
			var roleID uuid.UUID
			if err := pool.QueryRow(sysCtx,
				`SELECT id FROM roles WHERE name = $1 AND tenant_id IS NULL`, role).Scan(&roleID); err != nil {
				t.Fatalf("preset role %q: %v", role, err)
			}
			if _, err := pool.Exec(sysCtx,
				`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`,
				userID, roleID, tenantID); err != nil {
				t.Fatalf("assign role: %v", err)
			}
		}
		perms, err := authSvc.ResolveTokenPermissions(sysCtx, tenantID, userID)
		if err != nil {
			t.Fatalf("resolve permissions: %v", err)
		}
		token, err := tokenMaker.CreateAccessToken(userID, tenantID.String(), []string{role}, perms)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}
		return userID, token
	}

	_, adminToken := userFor("admin")
	aliceID, aliceToken := userFor("member")
	_, bobToken := userFor("member")
	_, noRoleToken := userFor("")
	t.Cleanup(func() {
		if _, err := pool.Exec(sysCtx, `DELETE FROM email_rules WHERE tenant_id = $1`, tenantID); err != nil {
			t.Errorf("cleanup email_rules: %v", err)
		}
	})

	labelID := testutil.SeedRow(t, pool, "email_labels", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "name": "Rechnungen",
	})
	// A rule from before migration 000348: no owner, tenant-wide.
	tenantWideID := testutil.SeedRow(t, pool, "email_rules", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "name": "Alt", "field": "subject", "op": "contains",
		"value": "Alt", "action_type": "label", "action_target": labelID,
	})

	do := func(token, method, path, body string) (int, []byte) {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code, rec.Body.Bytes()
	}
	create := func(t *testing.T, token string) string {
		t.Helper()
		code, body := do(token, http.MethodPost, "/api/v1/email/rules/",
			`{"name":"Rechnungen","field":"subject","op":"contains","value":"Rechnung","action_type":"label","action_target":"`+labelID.String()+`"}`)
		if code != http.StatusCreated {
			t.Fatalf("create rule = %d, want 201 (body %s)", code, body)
		}
		var resp struct {
			Rule struct {
				ID string `json:"id"`
			} `json:"rule"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("decode: %v (body %s)", err, body)
		}
		return resp.Rule.ID
	}
	owner := func(t *testing.T, id string) *uuid.UUID {
		t.Helper()
		var userID *uuid.UUID
		if err := pool.QueryRow(sysCtx, `SELECT user_id FROM email_rules WHERE id = $1`, id).Scan(&userID); err != nil {
			t.Fatalf("read rule owner: %v", err)
		}
		return userID
	}

	aliceRule := create(t, aliceToken)
	bobRule := create(t, bobToken)

	t.Run("member creates a rule owned by them", func(t *testing.T) {
		if got := owner(t, aliceRule); got == nil || *got != aliceID {
			t.Fatalf("rule owner = %v, want %s", got, aliceID)
		}
	})

	t.Run("member lists the tenant-wide and their own rules only", func(t *testing.T) {
		code, body := do(aliceToken, http.MethodGet, "/api/v1/email/rules/", "")
		if code != http.StatusOK {
			t.Fatalf("list = %d, want 200 (body %s)", code, body)
		}
		var resp struct {
			Rules []struct {
				ID string `json:"id"`
			} `json:"rules"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("decode: %v (body %s)", err, body)
		}
		ids := make([]string, 0, len(resp.Rules))
		for _, r := range resp.Rules {
			ids = append(ids, r.ID)
		}
		want := []string{tenantWideID.String(), aliceRule}
		slices.Sort(ids)
		slices.Sort(want)
		if !slices.Equal(ids, want) {
			t.Fatalf("rules = %v, want %v", ids, want)
		}
	})

	t.Run("member cannot change or remove a colleague's rule", func(t *testing.T) {
		if code, body := do(aliceToken, http.MethodPatch, "/api/v1/email/rules/"+bobRule, `{"value":"x"}`); code != http.StatusNotFound {
			t.Errorf("PATCH colleague's rule = %d, want 404 (body %s)", code, body)
		}
		if code, body := do(aliceToken, http.MethodDelete, "/api/v1/email/rules/"+bobRule, ""); code != http.StatusNotFound {
			t.Errorf("DELETE colleague's rule = %d, want 404 (body %s)", code, body)
		}
	})

	t.Run("member cannot change or remove a tenant-wide rule", func(t *testing.T) {
		path := "/api/v1/email/rules/" + tenantWideID.String()
		if code, body := do(aliceToken, http.MethodPatch, path, `{"value":"x"}`); code != http.StatusForbidden {
			t.Errorf("PATCH tenant-wide rule = %d, want 403 (body %s)", code, body)
		}
		if code, body := do(aliceToken, http.MethodDelete, path, ""); code != http.StatusForbidden {
			t.Errorf("DELETE tenant-wide rule = %d, want 403 (body %s)", code, body)
		}
	})

	t.Run("member applies the rules to their mailbox", func(t *testing.T) {
		if code, body := do(aliceToken, http.MethodPost, "/api/v1/email/rules/apply", ""); code != http.StatusOK {
			t.Fatalf("apply = %d, want 200 (body %s)", code, body)
		}
	})

	t.Run("member changes and removes their own rule", func(t *testing.T) {
		if code, body := do(aliceToken, http.MethodPatch, "/api/v1/email/rules/"+aliceRule, `{"value":"Mahnung"}`); code != http.StatusOK {
			t.Fatalf("PATCH own rule = %d, want 200 (body %s)", code, body)
		}
		if code, body := do(aliceToken, http.MethodDelete, "/api/v1/email/rules/"+aliceRule, ""); code != http.StatusOK {
			t.Fatalf("DELETE own rule = %d, want 200 (body %s)", code, body)
		}
	})

	t.Run("admin maintains the tenant-wide rule", func(t *testing.T) {
		if code, body := do(adminToken, http.MethodPatch, "/api/v1/email/rules/"+tenantWideID.String(), `{"value":"Neu"}`); code != http.StatusOK {
			t.Fatalf("admin PATCH tenant-wide rule = %d, want 200 (body %s)", code, body)
		}
	})

	t.Run("a user without the key stays out", func(t *testing.T) {
		if code, _ := do(noRoleToken, http.MethodGet, "/api/v1/email/rules/", ""); code != http.StatusForbidden {
			t.Fatalf("role-less list = %d, want 403", code)
		}
	})
}
