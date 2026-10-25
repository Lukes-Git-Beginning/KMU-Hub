package gateway

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/email/account"
	"github.com/kmuhub/kmuhub/internal/email/contactlink"
	"github.com/kmuhub/kmuhub/internal/email/message"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// A contact's mail history shows a member the linked mail of their own
// mailboxes only, and a manager or admin the linked mail of every mailbox
// (decision 2026-10-25). The chain is the production one: the real seed
// through ResolveTokenPermissions, the real Auth middleware and route
// registration, a real email gRPC server behind TenantInboundUnaryInterceptor
// and the Postgres contact-link repository.
func TestEmailContactHistory_MemberSeesOwnMailManagerSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Email Contact History")
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
		nil, nil, nil, nil, contactlink.NewPostgresRepository(pool), nil, nil, nil, nil, nil,
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
			"id": uuid.New(), "tenant_id": tenantID, "email": "email-history-" + uuid.NewString() + "@test.local",
			"password_hash": "x", "first_name": "History", "last_name": role,
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
	_, managerToken := userFor("manager")
	aliceID, aliceToken := userFor("member")
	bobID, _ := userFor("member")
	_, noRoleToken := userFor("")

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"first_name": "Kontakt", "last_name": "Verlauf-" + uuid.NewString()[:8],
		"created_by": aliceID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", contactID) })

	linkedMailOf := func(userID uuid.UUID, name string) uuid.UUID {
		t.Helper()
		accountID := testutil.SeedRow(t, pool, "email_accounts", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "user_id": userID,
			"email_address": "history-" + name + "-" + uuid.NewString() + "@test.local", "display_name": name,
			"imap_host": "imap.test.local", "smtp_host": "smtp.test.local",
			"username": "history-" + name, "password_encrypted": "x",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_accounts", accountID) })
		folderID := testutil.SeedRow(t, pool, "email_folders", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "account_id": accountID,
			"name": "Posteingang", "imap_name": "inbox", "folder_type": "inbox",
		})
		// The text columns are nullable, but the sync engine always writes
		// them; the message scan reads them as plain strings.
		msgID := testutil.SeedRow(t, pool, "email_messages", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "account_id": accountID, "folder_id": folderID,
			"uid": 1, "message_id": "<" + name + "@example.de>", "in_reply_to": "",
			"from_name": "Kontakt", "from_email": "kontakt@example.de", "subject": "Angebot " + name,
			"preview": "", "body_text": "", "body_html": "", "raw_headers": "", "date": time.Now().UTC(),
		})
		testutil.SeedRow(t, pool, "email_contact_links", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "message_id": msgID, "contact_id": contactID, "link_type": "auto",
		})
		return msgID
	}
	aliceMsg := linkedMailOf(aliceID, "alice")
	bobMsg := linkedMailOf(bobID, "bob")

	type history struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
		Total int `json:"total"`
	}
	get := func(t *testing.T, token, query string) (int, history) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/email/links/contact/"+contactID.String()+query, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var h history
		if rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &h); err != nil {
				t.Fatalf("decode history: %v (body %s)", err, rec.Body.String())
			}
		}
		return rec.Code, h
	}
	ids := func(h history) map[string]bool {
		out := map[string]bool{}
		for _, m := range h.Messages {
			out[m.ID] = true
		}
		return out
	}

	t.Run("member sees the linked mail of their own mailbox only", func(t *testing.T) {
		code, h := get(t, aliceToken, "")
		if code != http.StatusOK {
			t.Fatalf("member GET history = %d, want 200", code)
		}
		got := ids(h)
		if !got[aliceMsg.String()] || got[bobMsg.String()] || len(got) != 1 {
			t.Fatalf("member history = %v, want only own message %s", got, aliceMsg)
		}
		if h.Total != 1 {
			t.Errorf("member total = %d, want 1 (a colleague's mail counted)", h.Total)
		}
	})

	for _, tc := range []struct{ role, token string }{{"manager", managerToken}, {"admin", adminToken}} {
		t.Run(tc.role+" sees the linked mail of every mailbox", func(t *testing.T) {
			code, h := get(t, tc.token, "")
			if code != http.StatusOK {
				t.Fatalf("%s GET history = %d, want 200", tc.role, code)
			}
			got := ids(h)
			if !got[aliceMsg.String()] || !got[bobMsg.String()] {
				t.Fatalf("%s history = %v, want both %s and %s", tc.role, got, aliceMsg, bobMsg)
			}
			if h.Total != 2 {
				t.Errorf("%s total = %d, want 2", tc.role, h.Total)
			}
		})
	}

	t.Run("the member's total stays narrowed across pages", func(t *testing.T) {
		code, h := get(t, aliceToken, "?page=2&per_page=1")
		if code != http.StatusOK {
			t.Fatalf("member GET page 2 = %d, want 200", code)
		}
		if len(h.Messages) != 0 || h.Total != 1 {
			t.Errorf("member page 2 = %d messages, total %d; want 0 and 1", len(h.Messages), h.Total)
		}
	})

	t.Run("a user without the key stays out", func(t *testing.T) {
		if code, _ := get(t, noRoleToken, ""); code != http.StatusForbidden {
			t.Errorf("role-less GET history = %d, want 403", code)
		}
	})
}
