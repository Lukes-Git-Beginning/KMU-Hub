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
	"github.com/kmuhub/kmuhub/internal/email/account"
	emailsync "github.com/kmuhub/kmuhub/internal/email/sync"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/security/safedial"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// plainVault stands in for the vault service: account create/update only need
// a reversible encryption.
type plainVault struct{}

func (plainVault) Encrypt(_ context.Context, p []byte) (string, error) { return "enc:" + string(p), nil }
func (plainVault) Decrypt(_ context.Context, e string) ([]byte, error) {
	return []byte(strings.TrimPrefix(e, "enc:")), nil
}

// A member sets up, changes and removes their own mail account, never someone
// else's (decision 2026-10-25); admin keeps provisioning accounts for others.
// The chain is the production one: the real seed through
// ResolveTokenPermissions, the real Auth middleware and route registration, a
// real email gRPC server behind TenantInboundUnaryInterceptor and the Postgres
// account repository.
func TestEmailAccount_MemberManagesOwnAccountOnly(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Email Account Self Setup")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	// The allowlisted loopback host keeps the test off DNS and the network.
	guard, err := safedial.New("127.0.0.1")
	if err != nil {
		t.Fatalf("dial guard: %v", err)
	}
	accountSvc := account.NewService(account.NewPostgresRepository(pool), plainVault{}).WithDialGuard(guard)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	emailv1.RegisterEmailServiceServer(srv, server.NewEmailGRPCServer(
		accountSvc, nil, nil, nil, nil, emailsync.NewEngine(accountSvc, nil, nil, nil),
		nil, nil, nil, nil, nil, nil,
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
			"id": uuid.New(), "tenant_id": tenantID, "email": "email-acct-" + uuid.NewString() + "@test.local",
			"password_hash": "x", "first_name": "Mail", "last_name": role,
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
	memberID, memberToken := userFor("member")
	otherID, _ := userFor("member")
	_, noRoleToken := userFor("")
	t.Cleanup(func() {
		if _, err := pool.Exec(sysCtx, `DELETE FROM email_accounts WHERE tenant_id = $1`, tenantID); err != nil {
			t.Errorf("cleanup email_accounts: %v", err)
		}
	})

	do := func(token, method, path, body string) (int, []byte) {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code, rec.Body.Bytes()
	}
	createBody := func(userID string) string {
		return `{"user_id":"` + userID + `","email_address":"mail-` + uuid.NewString()[:8] + `@example.com",` +
			`"imap_host":"127.0.0.1","smtp_host":"127.0.0.1","username":"u","password":"p","sync_enabled":false}`
	}
	create := func(t *testing.T, token, userID string) (int, string, string) {
		t.Helper()
		code, body := do(token, http.MethodPost, "/api/v1/email/accounts/", createBody(userID))
		var resp struct {
			Account struct {
				ID     string `json:"id"`
				UserID string `json:"user_id"`
			} `json:"account"`
		}
		if code == http.StatusCreated {
			if err := json.Unmarshal(body, &resp); err != nil {
				t.Fatalf("decode: %v (body %s)", err, body)
			}
		}
		return code, resp.Account.ID, resp.Account.UserID
	}
	accountExists := func(t *testing.T, id string) bool {
		t.Helper()
		var n int
		if err := pool.QueryRow(sysCtx, `SELECT count(*) FROM email_accounts WHERE id = $1`, id).Scan(&n); err != nil {
			t.Fatalf("count account: %v", err)
		}
		return n == 1
	}

	t.Run("member sets up their own account, user_id pinned to the caller", func(t *testing.T) {
		code, _, owner := create(t, memberToken, "")
		if code != http.StatusCreated || owner != memberID.String() {
			t.Fatalf("member create without user_id = %d owner %q, want 201 owner %s", code, owner, memberID)
		}
		code, _, owner = create(t, memberToken, memberID.String())
		if code != http.StatusCreated || owner != memberID.String() {
			t.Fatalf("member create with own user_id = %d owner %q, want 201 owner %s", code, owner, memberID)
		}
	})

	t.Run("member cannot set up an account for someone else", func(t *testing.T) {
		if code, _, _ := create(t, memberToken, otherID.String()); code != http.StatusForbidden {
			t.Fatalf("member create for another user = %d, want 403", code)
		}
	})

	// admin provisions an account for the other member: the target of the next checks.
	code, foreignID, owner := create(t, adminToken, otherID.String())
	if code != http.StatusCreated || owner != otherID.String() {
		t.Fatalf("admin create for another user = %d owner %q, want 201 owner %s", code, owner, otherID)
	}

	t.Run("member cannot change or remove someone else's account", func(t *testing.T) {
		if code, body := do(memberToken, http.MethodPut, "/api/v1/email/accounts/"+foreignID,
			`{"display_name":"hijacked"}`); code != http.StatusNotFound {
			t.Errorf("member PUT foreign account = %d, want 404 (body %s)", code, body)
		}
		if code, body := do(memberToken, http.MethodDelete, "/api/v1/email/accounts/"+foreignID, ""); code != http.StatusNotFound {
			t.Errorf("member DELETE foreign account = %d, want 404 (body %s)", code, body)
		}
		if !accountExists(t, foreignID) {
			t.Fatal("the foreign account is gone")
		}
	})

	t.Run("member changes and removes their own account", func(t *testing.T) {
		_, ownID, _ := create(t, memberToken, "")
		if code, body := do(memberToken, http.MethodPut, "/api/v1/email/accounts/"+ownID,
			`{"display_name":"Mine"}`); code != http.StatusOK {
			t.Fatalf("member PUT own account = %d, want 200 (body %s)", code, body)
		}
		if code, body := do(memberToken, http.MethodDelete, "/api/v1/email/accounts/"+ownID, ""); code != http.StatusOK {
			t.Fatalf("member DELETE own account = %d, want 200 (body %s)", code, body)
		}
		if accountExists(t, ownID) {
			t.Fatal("the member's own account was not removed")
		}
	})

	t.Run("member may test a connection", func(t *testing.T) {
		// Port 1 on loopback refuses at once: the answer is a failed test, not a 403.
		if code, body := do(memberToken, http.MethodPost, "/api/v1/email/accounts/test",
			`{"imap_host":"127.0.0.1","imap_port":1,"smtp_host":"127.0.0.1","smtp_port":1,"username":"u","password":"p"}`); code != http.StatusOK {
			t.Fatalf("member POST /test = %d, want 200 (body %s)", code, body)
		}
	})

	t.Run("admin changes and removes another user's account", func(t *testing.T) {
		if code, body := do(adminToken, http.MethodPut, "/api/v1/email/accounts/"+foreignID,
			`{"display_name":"Provisioned"}`); code != http.StatusOK {
			t.Fatalf("admin PUT foreign account = %d, want 200 (body %s)", code, body)
		}
		if code, body := do(adminToken, http.MethodDelete, "/api/v1/email/accounts/"+foreignID, ""); code != http.StatusOK {
			t.Fatalf("admin DELETE foreign account = %d, want 200 (body %s)", code, body)
		}
	})

	t.Run("a user without the key stays out", func(t *testing.T) {
		if code, _, _ := create(t, noRoleToken, ""); code != http.StatusForbidden {
			t.Fatalf("role-less create = %d, want 403", code)
		}
	})
}
