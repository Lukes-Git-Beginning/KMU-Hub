package gateway

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
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
	"github.com/kmuhub/kmuhub/internal/helpdesk"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	helpdeskv1 "github.com/kmuhub/kmuhub/proto/helpdesk/v1"
)

// A member works the tickets assigned to them (decision 2026-10-25): edit,
// close, reopen and CSAT answer 200 on an assigned ticket and 404 on one they
// only raised or do not know; assigning and merging stay 403, and internal
// notes stay hidden. The chain is the production one: the real seed through
// ResolveTokenPermissions, the real Auth middleware and route registration, a
// real helpdesk gRPC server behind TenantInboundUnaryInterceptor and the
// Postgres repository.

var assignedEditTenant = uuid.MustParse("e4fec700-0000-4000-8000-0000000000e7")

func TestHelpdeskAssignedTicketEdit_MemberAgainstRealSeed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	testutil.EnsureTenant(t, pool, assignedEditTenant, "AssignedEditTenant")
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	svc := helpdesk.NewService(helpdesk.NewPostgresRepository(pool), slog.New(slog.DiscardHandler))
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	helpdeskv1.RegisterHelpdeskServiceServer(srv, server.NewHelpdeskGRPCServer(svc, nil, nil))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("helpdesk", lis.Addr().String())
	t.Cleanup(reg.Close)

	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_HELPDESK_ENABLED" {
			return "true"
		}
		return ""
	})
	router := chi.NewRouter()
	NewHelpdeskRoutes(reg, flags).RegisterRoutes(router, middleware.Auth(authSvc))

	tokenFor := func(role string) (string, string) {
		t.Helper()
		userID := testutil.SeedRow(t, pool, "users", map[string]any{
			"id":            uuid.New(),
			"tenant_id":     assignedEditTenant,
			"email":         "assigned-edit-" + uuid.NewString() + "@test.local",
			"password_hash": "x",
			"first_name":    "Ticket",
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
			userID, roleID, assignedEditTenant); err != nil {
			t.Fatalf("assign role: %v", err)
		}
		perms, err := authSvc.ResolveTokenPermissions(sysCtx, assignedEditTenant, userID)
		if err != nil {
			t.Fatalf("resolve permissions: %v", err)
		}
		token, err := tokenMaker.CreateAccessToken(userID, assignedEditTenant.String(), []string{role}, perms)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}
		return userID.String(), token
	}

	do := func(method, token, path, body string) (int, map[string]any) {
		t.Helper()
		var rd io.Reader
		if body != "" {
			rd = strings.NewReader(body)
		}
		req := httptest.NewRequest(method, "/api/v1/helpdesk"+path, rd)
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

	_, managerToken := tokenFor("manager")
	memberID, memberToken := tokenFor("member")
	colleagueID, colleagueToken := tokenFor("member")

	createTicket := func(token, body string) string {
		t.Helper()
		code, tk := do(http.MethodPost, token, "/tickets", body)
		if code != http.StatusCreated {
			t.Fatalf("create ticket: %d %v", code, tk)
		}
		id, _ := tk["id"].(string)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "tickets", uuid.MustParse(id)) })
		return id
	}
	assigned := createTicket(managerToken, `{"subject":"Drucker","priority":"high","assignee_id":"`+memberID+`"}`)
	raised := createTicket(memberToken, `{"subject":"Eigenes Anliegen"}`)
	foreign := createTicket(managerToken, `{"subject":"Fremd","assignee_id":"`+colleagueID+`"}`)

	t.Run("member edits, closes, reopens and rates the assigned ticket", func(t *testing.T) {
		if code, tk := do(http.MethodPut, memberToken, "/tickets/"+assigned, `{"priority":"low","status":"pending"}`); code != http.StatusOK ||
			tk["priority"] != "low" || tk["status"] != "pending" {
			t.Fatalf("PUT assigned: %d %v", code, tk)
		}
		if code, tk := do(http.MethodPost, memberToken, "/tickets/"+assigned+"/close", ""); code != http.StatusOK || tk["status"] != "closed" {
			t.Fatalf("close assigned: %d %v", code, tk)
		}
		if code, tk := do(http.MethodPost, memberToken, "/tickets/"+assigned+"/csat", `{"rating":5}`); code != http.StatusOK {
			t.Fatalf("csat assigned: %d %v", code, tk)
		}
		if code, tk := do(http.MethodPost, memberToken, "/tickets/"+assigned+"/reopen", ""); code != http.StatusOK || tk["status"] != "open" {
			t.Fatalf("reopen assigned: %d %v", code, tk)
		}
	})

	t.Run("a ticket the member only raised or does not know answers 404", func(t *testing.T) {
		for name, id := range map[string]string{"raised": raised, "foreign": foreign, "unknown": uuid.NewString()} {
			if code, _ := do(http.MethodPut, memberToken, "/tickets/"+id, `{"priority":"urgent"}`); code != http.StatusNotFound {
				t.Errorf("%s PUT: want 404, got %d", name, code)
			}
			for _, action := range []string{"/close", "/reopen"} {
				if code, _ := do(http.MethodPost, memberToken, "/tickets/"+id+action, ""); code != http.StatusNotFound {
					t.Errorf("%s %s: want 404, got %d", name, action, code)
				}
			}
			if code, _ := do(http.MethodPost, memberToken, "/tickets/"+id+"/csat", `{"rating":1}`); code != http.StatusNotFound {
				t.Errorf("%s csat: want 404, got %d", name, code)
			}
		}
		if _, tk := do(http.MethodGet, managerToken, "/tickets/"+foreign, ""); tk["priority"] == "urgent" || tk["status"] != "open" {
			t.Errorf("refused writes changed the colleague's ticket: %v", tk)
		}
		// The colleague still works their own.
		if code, _ := do(http.MethodPut, colleagueToken, "/tickets/"+foreign, `{"priority":"low"}`); code != http.StatusOK {
			t.Errorf("colleague PUT on their assigned ticket: want 200, got %d", code)
		}
	})

	t.Run("handing the ticket on stays with tenant-wide agents", func(t *testing.T) {
		if code, _ := do(http.MethodPost, memberToken, "/tickets/"+assigned+"/assign", `{"assignee_id":"`+colleagueID+`"}`); code != http.StatusForbidden {
			t.Errorf("member assign: want 403, got %d", code)
		}
		if code, _ := do(http.MethodPut, memberToken, "/tickets/"+assigned, `{"assignee_id":"`+colleagueID+`"}`); code != http.StatusForbidden {
			t.Errorf("member PUT assignee_id: want 403, got %d", code)
		}
		if code, _ := do(http.MethodPost, memberToken, "/tickets/"+raised+"/merge", `{"target_ticket_id":"`+assigned+`"}`); code != http.StatusForbidden {
			t.Errorf("member merge: want 403, got %d", code)
		}
		if _, tk := do(http.MethodGet, managerToken, "/tickets/"+assigned, ""); tk["assignee_id"] != memberID {
			t.Fatalf("refused reassignment changed the assignee: %v", tk["assignee_id"])
		}
		if code, tk := do(http.MethodPost, managerToken, "/tickets/"+raised+"/assign", `{"assignee_id":"`+colleagueID+`"}`); code != http.StatusOK || tk["assignee_id"] != colleagueID {
			t.Fatalf("manager assign: %d %v", code, tk)
		}
	})

	t.Run("internal notes stay with agents beyond own scope", func(t *testing.T) {
		if code, _ := do(http.MethodPost, memberToken, "/tickets/"+assigned+"/messages", `{"body":"intern","internal":true}`); code != http.StatusForbidden {
			t.Errorf("member internal note: want 403, got %d", code)
		}
		if code, _ := do(http.MethodPost, managerToken, "/tickets/"+assigned+"/messages", `{"body":"Agentennotiz","internal":true}`); code != http.StatusCreated {
			t.Fatalf("manager internal note: want 201, got %d", code)
		}
		_, list := do(http.MethodGet, memberToken, "/tickets/"+assigned+"/messages", "")
		msgs, _ := list["messages"].([]any)
		for _, m := range msgs {
			if mm, _ := m.(map[string]any); mm["internal"] == true {
				t.Errorf("member sees an internal note: %v", mm)
			}
		}
	})
}
