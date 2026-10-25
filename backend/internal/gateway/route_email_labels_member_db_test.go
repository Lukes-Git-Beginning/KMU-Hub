package gateway

import (
	"context"
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
	"github.com/kmuhub/kmuhub/internal/email/label"
	"github.com/kmuhub/kmuhub/internal/email/message"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// A member reads the shared label catalogue and labels the mail of their own
// mailboxes, not a colleague's, and leaves the catalogue itself to admin
// (decision 2026-10-25: catalogue shared and kept by admin, assigning on own
// messages for member). The chain is the production one: the real seed
// through ResolveTokenPermissions, the real Auth middleware and route
// registration, a real email gRPC server behind TenantInboundUnaryInterceptor
// and the Postgres account, message and label repositories.
func TestEmailLabels_MemberReadsCatalogueAndLabelsOwnMessagesOnly(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Email Labels Member")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	msgSvc := message.NewService(message.NewPostgresRepository(pool), message.NewPostgresFolderRepository(pool))
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	emailv1.RegisterEmailServiceServer(srv, server.NewEmailGRPCServer(
		account.NewService(account.NewPostgresRepository(pool), nil), msgSvc,
		nil, nil, nil, nil, nil, nil, nil, nil,
		label.NewService(label.NewPostgresRepository(pool), msgSvc), nil,
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
			"id": uuid.New(), "tenant_id": tenantID, "email": "email-label-" + uuid.NewString() + "@test.local",
			"password_hash": "x", "first_name": "Label", "last_name": role,
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

	mailOf := func(userID uuid.UUID, name string) uuid.UUID {
		t.Helper()
		accountID := testutil.SeedRow(t, pool, "email_accounts", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "user_id": userID,
			"email_address": "label-" + name + "-" + uuid.NewString() + "@test.local", "display_name": name,
			"imap_host": "imap.test.local", "smtp_host": "smtp.test.local",
			"username": "label-" + name, "password_encrypted": "x",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_accounts", accountID) })
		folderID := testutil.SeedRow(t, pool, "email_folders", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "account_id": accountID,
			"name": "Posteingang", "imap_name": "inbox", "folder_type": "inbox",
		})
		// The text columns are nullable, but the sync engine always writes
		// them; the message scan reads them as plain strings.
		return testutil.SeedRow(t, pool, "email_messages", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "account_id": accountID, "folder_id": folderID,
			"uid": 1, "message_id": "<" + name + "@example.de>", "in_reply_to": "",
			"from_name": "Sender", "from_email": "sender@example.de", "subject": "Hallo " + name,
			"preview": "", "body_text": "", "body_html": "", "raw_headers": "", "date": time.Now().UTC(),
		})
	}
	aliceMsg := mailOf(aliceID, "alice")
	bobMsg := mailOf(bobID, "bob")

	labelID := testutil.SeedRow(t, pool, "email_labels", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "name": "Rechnungen", "color": "#336699",
	})
	t.Cleanup(func() {
		if _, err := pool.Exec(sysCtx, `DELETE FROM email_labels WHERE tenant_id = $1`, tenantID); err != nil {
			t.Errorf("cleanup email_labels: %v", err)
		}
	})

	do := func(token, method, path, body string) (int, string) {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}
	labelsOf := func(t *testing.T, msgID uuid.UUID) []uuid.UUID {
		t.Helper()
		var ids []uuid.UUID
		if err := pool.QueryRow(sysCtx, `SELECT label_ids FROM email_messages WHERE id = $1`, msgID).Scan(&ids); err != nil {
			t.Fatalf("read label_ids: %v", err)
		}
		return ids
	}
	assign := `{"label_ids":["` + labelID.String() + `"]}`

	t.Run("member reads the catalogue", func(t *testing.T) {
		code, body := do(aliceToken, http.MethodGet, "/api/v1/email/labels/", "")
		if code != http.StatusOK {
			t.Fatalf("GET labels = %d, want 200 (body %s)", code, body)
		}
		if !strings.Contains(body, labelID.String()) {
			t.Errorf("catalogue misses the tenant label: %s", body)
		}
	})

	t.Run("member labels their own message and removes the label again", func(t *testing.T) {
		if code, body := do(aliceToken, http.MethodPost, "/api/v1/email/messages/"+aliceMsg.String()+"/labels", assign); code != http.StatusOK {
			t.Fatalf("assign on own message = %d, want 200 (body %s)", code, body)
		}
		if got := labelsOf(t, aliceMsg); len(got) != 1 || got[0] != labelID {
			t.Fatalf("own message label_ids = %v, want [%s]", got, labelID)
		}
		if code, body := do(aliceToken, http.MethodPost, "/api/v1/email/messages/"+aliceMsg.String()+"/labels", `{"label_ids":[]}`); code != http.StatusOK {
			t.Fatalf("clear on own message = %d, want 200 (body %s)", code, body)
		}
		if got := labelsOf(t, aliceMsg); len(got) != 0 {
			t.Fatalf("own message label_ids after clear = %v, want empty", got)
		}
	})

	t.Run("member cannot label a colleague's message", func(t *testing.T) {
		if code, body := do(aliceToken, http.MethodPost, "/api/v1/email/messages/"+bobMsg.String()+"/labels", assign); code != http.StatusNotFound {
			t.Fatalf("assign on colleague's message = %d, want 404 (body %s)", code, body)
		}
		if got := labelsOf(t, bobMsg); len(got) != 0 {
			t.Fatalf("colleague's message label_ids = %v, want untouched", got)
		}
	})

	t.Run("member cannot change the catalogue", func(t *testing.T) {
		if code, body := do(aliceToken, http.MethodPost, "/api/v1/email/labels/", `{"name":"Privat","color":"#112233"}`); code != http.StatusForbidden {
			t.Errorf("POST label = %d, want 403 (body %s)", code, body)
		}
		if code, body := do(aliceToken, http.MethodPatch, "/api/v1/email/labels/"+labelID.String(), `{"name":"Gekapert","color":"#112233"}`); code != http.StatusForbidden {
			t.Errorf("PATCH label = %d, want 403 (body %s)", code, body)
		}
		if code, body := do(aliceToken, http.MethodDelete, "/api/v1/email/labels/"+labelID.String(), ""); code != http.StatusForbidden {
			t.Errorf("DELETE label = %d, want 403 (body %s)", code, body)
		}
		var n int
		if err := pool.QueryRow(sysCtx, `SELECT count(*) FROM email_labels WHERE tenant_id = $1`, tenantID).Scan(&n); err != nil {
			t.Fatalf("count labels: %v", err)
		}
		if n != 1 {
			t.Errorf("catalogue size = %d, want 1 (a refused write changed it)", n)
		}
	})

	t.Run("admin keeps the catalogue", func(t *testing.T) {
		if code, body := do(adminToken, http.MethodPost, "/api/v1/email/labels/", `{"name":"Angebote","color":"#445566"}`); code != http.StatusCreated {
			t.Fatalf("admin POST label = %d, want 201 (body %s)", code, body)
		}
	})

	t.Run("a user without the key stays out", func(t *testing.T) {
		if code, _ := do(noRoleToken, http.MethodGet, "/api/v1/email/labels/", ""); code != http.StatusForbidden {
			t.Errorf("role-less GET labels = %d, want 403", code)
		}
		if code, _ := do(noRoleToken, http.MethodPost, "/api/v1/email/messages/"+bobMsg.String()+"/labels", assign); code != http.StatusForbidden {
			t.Errorf("role-less assign = %d, want 403", code)
		}
	})
}
