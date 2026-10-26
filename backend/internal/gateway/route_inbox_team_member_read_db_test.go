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
	"github.com/kmuhub/kmuhub/internal/inbox/adapter"
	"github.com/kmuhub/kmuhub/internal/inbox/message"
	"github.com/kmuhub/kmuhub/internal/inbox/routing"
	"github.com/kmuhub/kmuhub/internal/inbox/team"
	"github.com/kmuhub/kmuhub/internal/inbox/thread"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	inboxv1 "github.com/kmuhub/kmuhub/proto/inbox/v1"
)

// A member reads the team inboxes it belongs to (decision 2026-10-25) through
// kommunikation:team_inbox:read, and only those. The chain is the production one: the real seed
// through ResolveTokenPermissions, the real Auth middleware and route registration, a real inbox
// gRPC server behind TenantInboundUnaryInterceptor and the Postgres inbox repositories.
func TestInboxTeamInbox_MemberReadsOwnTeamsOnly(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Inbox Team Member Read")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	msgRepo := message.NewPostgresRepository(pool)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	inboxv1.RegisterInboxServiceServer(srv, server.NewInboxGRPCServer(
		message.NewService(msgRepo, adapter.NewAdapterRegistry()),
		team.NewService(team.NewPostgresRepository(pool), msgRepo),
		routing.NewService(routing.NewPostgresRepository(pool), msgRepo, nil),
		thread.NewService(thread.NewPostgresRepository(pool)),
	))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("notification", lis.Addr().String())
	t.Cleanup(reg.Close)

	router := chi.NewRouter()
	NewInboxRoutes(reg).RegisterRoutes(router, middleware.Auth(authSvc))

	userFor := func(role string) (uuid.UUID, string) {
		t.Helper()
		userID := testutil.SeedRow(t, pool, "users", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "email": "inbox-team-read-" + uuid.NewString() + "@test.local",
			"password_hash": "x", "first_name": "Inbox", "last_name": role,
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

	adminID, _ := userFor("admin")
	memberID, memberToken := userFor("member")
	_, noRoleToken := userFor("")

	seedTeam := func(members ...uuid.UUID) uuid.UUID {
		id := testutil.SeedRow(t, pool, "team_inboxes", map[string]any{
			"tenant_id": tenantID, "name": "Team " + uuid.NewString()[:8], "created_by": adminID,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "team_inboxes", id) })
		for _, u := range members {
			if _, err := pool.Exec(sysCtx,
				"INSERT INTO team_inbox_members (team_inbox_id, user_id, role, tenant_id) VALUES ($1, $2, 'member', $3)",
				id, u, tenantID); err != nil {
				t.Fatalf("seed team member: %v", err)
			}
		}
		return id
	}
	ownTeam, foreignTeam := seedTeam(adminID, memberID), seedTeam(adminID)

	tenantCtx := testutil.WithTenantCtx(context.Background(), tenantID)
	seedMessage := func(teamID uuid.UUID) uuid.UUID {
		msg := &models.InboxMessage{
			ID: uuid.New(), TenantID: tenantID, UserID: adminID, Channel: "email", TeamInboxID: &teamID,
			SourceID: "team-read-" + uuid.NewString()[:8], Tags: []string{}, ReceivedAt: time.Now().UTC(),
		}
		if err := msgRepo.Create(tenantCtx, msg); err != nil {
			t.Fatalf("seed message: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "inbox_messages", msg.ID) })
		return msg.ID
	}
	ownMsg, foreignMsg := seedMessage(ownTeam), seedMessage(foreignTeam)

	get := func(token, path string) (int, []byte) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code, rec.Body.Bytes()
	}
	ids := func(t *testing.T, body []byte, wrapped bool) map[string]bool {
		t.Helper()
		var items []map[string]any
		if wrapped {
			var resp struct {
				Messages []map[string]any `json:"messages"`
			}
			if err := json.Unmarshal(body, &resp); err != nil {
				t.Fatalf("decode: %v (body %s)", err, body)
			}
			items = resp.Messages
		} else if err := json.Unmarshal(body, &items); err != nil {
			t.Fatalf("decode: %v (body %s)", err, body)
		}
		out := make(map[string]bool, len(items))
		for _, it := range items {
			if id, ok := it["id"].(string); ok {
				out[id] = true
			} else if id, ok := it["user_id"].(string); ok {
				out[id] = true
			}
		}
		return out
	}
	expectOK := func(t *testing.T, path string) []byte {
		t.Helper()
		code, body := get(memberToken, path)
		if code != http.StatusOK {
			t.Fatalf("member GET %s = %d, want 200 (body %s)", path, code, body)
		}
		return body
	}

	t.Run("lists only the team inboxes it belongs to", func(t *testing.T) {
		got := ids(t, expectOK(t, "/api/v1/inbox/teams"), false)
		if !got[ownTeam.String()] || got[foreignTeam.String()] {
			t.Errorf("member team list = %v, want %s and not %s", got, ownTeam, foreignTeam)
		}
	})

	t.Run("reads the messages of its team, not of a foreign team", func(t *testing.T) {
		got := ids(t, expectOK(t, "/api/v1/inbox/messages?team_inbox_id="+ownTeam.String()), true)
		if !got[ownMsg.String()] {
			t.Errorf("member does not see the message of its team: %v", got)
		}
		got = ids(t, expectOK(t, "/api/v1/inbox/messages?team_inbox_id="+foreignTeam.String()), true)
		if len(got) != 0 {
			t.Errorf("member sees messages of a team it is not in: %v", got)
		}
	})

	t.Run("opens a single message of its team, 404 for a foreign team", func(t *testing.T) {
		expectOK(t, "/api/v1/inbox/messages/"+ownMsg.String())
		if code, body := get(memberToken, "/api/v1/inbox/messages/"+foreignMsg.String()); code != http.StatusNotFound {
			t.Errorf("member GET foreign team message = %d, want 404 (body %s)", code, body)
		}
	})

	t.Run("lists the members of its team", func(t *testing.T) {
		got := ids(t, expectOK(t, "/api/v1/inbox/teams/"+ownTeam.String()+"/members"), false)
		if !got[memberID.String()] || !got[adminID.String()] {
			t.Errorf("member list = %v, want %s and %s", got, memberID, adminID)
		}
	})

	t.Run("the key opens no admin read route", func(t *testing.T) {
		if code, _ := get(memberToken, "/api/v1/inbox/rules"); code != http.StatusForbidden {
			t.Errorf("member GET /inbox/rules = %d, want 403", code)
		}
	})

	t.Run("a user without the key stays out", func(t *testing.T) {
		if code, _ := get(noRoleToken, "/api/v1/inbox/teams"); code != http.StatusForbidden {
			t.Errorf("role-less GET /inbox/teams = %d, want 403", code)
		}
	})
}
