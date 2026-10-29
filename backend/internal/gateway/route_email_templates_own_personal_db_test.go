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
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/email/template"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// A member changes and removes their own personal templates, not a shared one
// and not a colleague's, and cannot switch a visibility (decision 2026-10-25:
// own personal templates free, shared ones admin only). The chain is the
// production one: the real seed through ResolveTokenPermissions, the real Auth
// middleware and route registration, a real email gRPC server behind
// TenantInboundUnaryInterceptor and the Postgres template repository.
func TestEmailTemplates_MemberMaintainsOwnPersonalTemplatesOnly(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Email Templates Own Personal")
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
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		template.NewService(template.NewPostgresRepository(pool)),
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
			"id": uuid.New(), "tenant_id": tenantID, "email": "email-template-" + uuid.NewString() + "@test.local",
			"password_hash": "x", "first_name": "Template", "last_name": role,
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
	bobID, _ := userFor("member")
	_, noRoleToken := userFor("")

	seed := func(owner *uuid.UUID, visibility, name string) string {
		t.Helper()
		now := time.Now().UTC()
		id := testutil.SeedRow(t, pool, "email_templates", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "owner_id": owner, "visibility": visibility,
			"name": name, "subject": "Betreff", "body_html": "<p>Hallo</p>", "body_text": "Hallo",
			"created_at": now, "updated_at": now,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_templates", id) })
		return id.String()
	}
	aliceOwn := seed(&aliceID, "personal", "Alice")
	aliceToShare := seed(&aliceID, "personal", "Alice teilen")
	bobOwn := seed(&bobID, "personal", "Bob")
	shared := seed(nil, "shared", "Team")

	do := func(token, method, path, body string) (int, string) {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}
	const base = "/api/v1/email/templates/"

	t.Run("member changes their own personal template", func(t *testing.T) {
		if code, body := do(aliceToken, http.MethodPut, base+aliceOwn, `{"name":"Alice neu","visibility":"personal"}`); code != http.StatusOK {
			t.Fatalf("PUT own personal = %d, want 200 (body %s)", code, body)
		}
	})

	t.Run("member cannot change or remove a shared template", func(t *testing.T) {
		if code, body := do(aliceToken, http.MethodPut, base+shared, `{"name":"Gekapert"}`); code != http.StatusForbidden {
			t.Errorf("PUT shared = %d, want 403 (body %s)", code, body)
		}
		if code, body := do(aliceToken, http.MethodPut, base+shared, `{"visibility":"personal"}`); code != http.StatusForbidden {
			t.Errorf("PUT shared->personal = %d, want 403 (body %s)", code, body)
		}
		if code, body := do(aliceToken, http.MethodDelete, base+shared, ""); code != http.StatusForbidden {
			t.Errorf("DELETE shared = %d, want 403 (body %s)", code, body)
		}
	})

	t.Run("member cannot share their personal template", func(t *testing.T) {
		if code, body := do(aliceToken, http.MethodPut, base+aliceToShare, `{"visibility":"shared"}`); code != http.StatusForbidden {
			t.Errorf("PUT personal->shared = %d, want 403 (body %s)", code, body)
		}
	})

	t.Run("member cannot reach a colleague's personal template", func(t *testing.T) {
		if code, body := do(aliceToken, http.MethodPut, base+bobOwn, `{"name":"Gekapert"}`); code != http.StatusNotFound {
			t.Errorf("PUT colleague's personal = %d, want 404 (body %s)", code, body)
		}
		if code, body := do(aliceToken, http.MethodDelete, base+bobOwn, ""); code != http.StatusNotFound {
			t.Errorf("DELETE colleague's personal = %d, want 404 (body %s)", code, body)
		}
	})

	t.Run("the refused writes left the rows alone", func(t *testing.T) {
		var name, visibility string
		if err := pool.QueryRow(sysCtx, `SELECT name, visibility FROM email_templates WHERE id = $1`, shared).Scan(&name, &visibility); err != nil {
			t.Fatalf("read shared: %v", err)
		}
		if name != "Team" || visibility != "shared" {
			t.Errorf("shared template = %q/%q, want Team/shared", name, visibility)
		}
		if err := pool.QueryRow(sysCtx, `SELECT visibility FROM email_templates WHERE id = $1`, aliceToShare).Scan(&visibility); err != nil {
			t.Fatalf("read personal: %v", err)
		}
		if visibility != "personal" {
			t.Errorf("personal template visibility = %q, want personal", visibility)
		}
	})

	t.Run("member removes their own personal template", func(t *testing.T) {
		if code, body := do(aliceToken, http.MethodDelete, base+aliceOwn, ""); code != http.StatusOK {
			t.Fatalf("DELETE own personal = %d, want 200 (body %s)", code, body)
		}
	})

	t.Run("admin maintains the shared template", func(t *testing.T) {
		if code, body := do(adminToken, http.MethodPut, base+shared, `{"name":"Team neu"}`); code != http.StatusOK {
			t.Fatalf("admin PUT shared = %d, want 200 (body %s)", code, body)
		}
		if code, body := do(adminToken, http.MethodDelete, base+shared, ""); code != http.StatusOK {
			t.Fatalf("admin DELETE shared = %d, want 200 (body %s)", code, body)
		}
	})

	t.Run("member creates a personal template but no shared one", func(t *testing.T) {
		code, body := do(aliceToken, http.MethodPost, "/api/v1/email/templates", `{"name":"Meine","visibility":"personal"}`)
		if code != http.StatusCreated {
			t.Fatalf("POST personal = %d, want 201 (body %s)", code, body)
		}
		forgetCreated(t, pool, body)
		if code, body := do(aliceToken, http.MethodPost, "/api/v1/email/templates", `{"name":"Fuer alle","visibility":"shared"}`); code != http.StatusForbidden {
			t.Errorf("member POST shared = %d, want 403 (body %s)", code, body)
			forgetCreated(t, pool, body)
		}
		var n int
		if err := pool.QueryRow(sysCtx, `SELECT count(*) FROM email_templates WHERE tenant_id = $1 AND name = 'Fuer alle'`, tenantID).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		if n != 0 {
			t.Errorf("refused shared create left %d rows", n)
		}
	})

	t.Run("admin creates a shared template", func(t *testing.T) {
		code, body := do(adminToken, http.MethodPost, "/api/v1/email/templates", `{"name":"Team neu","visibility":"shared"}`)
		if code != http.StatusCreated {
			t.Fatalf("admin POST shared = %d, want 201 (body %s)", code, body)
		}
		forgetCreated(t, pool, body)
	})

	t.Run("a user without the key stays out", func(t *testing.T) {
		if code, _ := do(noRoleToken, http.MethodPut, base+bobOwn, `{"name":"x"}`); code != http.StatusForbidden {
			t.Fatalf("role-less PUT = %d, want 403", code)
		}
	})
}

// forgetCreated schedules the cleanup of the template a create response names.
func forgetCreated(t *testing.T, pool *pgxpool.Pool, body string) {
	t.Helper()
	var resp struct {
		Template struct {
			ID string `json:"id"`
		} `json:"template"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Template.ID == "" {
		return
	}
	id := uuid.MustParse(resp.Template.ID)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_templates", id) })
}
