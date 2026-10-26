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
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/customfield"
	"github.com/kmuhub/kmuhub/internal/work/label"
	"github.com/kmuhub/kmuhub/internal/work/project"
	wstatus "github.com/kmuhub/kmuhub/internal/work/status"
	"github.com/kmuhub/kmuhub/internal/work/task"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

// A member deletes the tasks they created (decision 2026-10-25), not a
// colleague's, and deleting a project stays with manager. The chain is the
// production one: the real seed through ResolveTokenPermissions, the real Auth
// middleware and route registration, a real work gRPC server behind
// TenantInboundUnaryInterceptor and the Postgres repositories.

var ownTaskDeleteTenant = uuid.MustParse("e4fec700-0000-4000-8000-0000000000d2")

func TestWorkOwnTaskDelete_MemberAgainstRealSeed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	testutil.EnsureTenant(t, pool, ownTaskDeleteTenant, "OwnTaskDeleteTenant")
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	projRepo := project.NewPostgresRepository(pool)
	taskRepo := task.NewPostgresRepository(pool)
	workSrv := server.NewWorkGRPCServer(
		project.NewService(projRepo), wstatus.NewService(wstatus.NewPostgresRepository(pool)),
		task.NewService(taskRepo, projRepo), taskRepo, nil, nil,
		label.NewService(label.NewPostgresRepository(pool)), customfield.NewService(customfield.NewPostgresRepository(pool)),
	)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	workv1.RegisterWorkServiceServer(srv, workSrv)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("work", lis.Addr().String())
	t.Cleanup(reg.Close)

	router := chi.NewRouter()
	NewWorkRoutes(reg).RegisterRoutes(router, middleware.Auth(authSvc))

	tokenFor := func(role string) string {
		t.Helper()
		userID := testutil.SeedRow(t, pool, "users", map[string]any{
			"id":            uuid.New(),
			"tenant_id":     ownTaskDeleteTenant,
			"email":         "own-task-delete-" + uuid.NewString() + "@test.local",
			"password_hash": "x",
			"first_name":    "Task",
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
			userID, roleID, ownTaskDeleteTenant); err != nil {
			t.Fatalf("assign role: %v", err)
		}
		perms, err := authSvc.ResolveTokenPermissions(sysCtx, ownTaskDeleteTenant, userID)
		if err != nil {
			t.Fatalf("resolve permissions: %v", err)
		}
		token, err := tokenMaker.CreateAccessToken(userID, ownTaskDeleteTenant.String(), []string{role}, perms)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}
		return token
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

	idOf := func(body map[string]any, key string) string {
		t.Helper()
		inner, _ := body[key].(map[string]any)
		id, _ := inner["id"].(string)
		if id == "" {
			t.Fatalf("no %s.id in %v", key, body)
		}
		return id
	}
	createTask := func(token, body string) string {
		t.Helper()
		code, out := do(http.MethodPost, token, "/tasks", body)
		if code != http.StatusCreated {
			t.Fatalf("create task: %d %v", code, out)
		}
		id := idOf(out, "task")
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "tasks", uuid.MustParse(id)) })
		return id
	}

	managerToken := tokenFor("manager")
	memberToken := tokenFor("member")
	colleagueToken := tokenFor("member")

	code, out := do(http.MethodPost, memberToken, "/projects",
		`{"name":"Eigenes Projekt","project_key":"OWN`+strings.ToUpper(uuid.NewString()[:4])+`"}`)
	if code != http.StatusCreated {
		t.Fatalf("member creates project: %d %v", code, out)
	}
	projectID := idOf(out, "project")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "projects", uuid.MustParse(projectID)) })

	ownStandalone := createTask(memberToken, `{"title":"Eigene Notiz"}`)
	ownInProject := createTask(memberToken, `{"title":"Eigener Schritt","project_id":"`+projectID+`"}`)
	colleagues := createTask(colleagueToken, `{"title":"Aufgabe des Kollegen"}`)
	managers := createTask(managerToken, `{"title":"Aufgabe des Managers"}`)

	t.Run("member deletes the tasks they created", func(t *testing.T) {
		for _, id := range []string{ownStandalone, ownInProject} {
			if code, out := do(http.MethodDelete, memberToken, "/tasks/"+id, ""); code != http.StatusOK {
				t.Fatalf("DELETE own task %s = %d %v, want 200", id, code, out)
			}
			if code, _ := do(http.MethodGet, memberToken, "/tasks/"+id, ""); code != http.StatusNotFound {
				t.Errorf("GET deleted task %s = %d, want 404", id, code)
			}
		}
	})

	t.Run("member cannot delete a colleague's or a manager's task", func(t *testing.T) {
		for _, id := range []string{colleagues, managers} {
			if code, out := do(http.MethodDelete, memberToken, "/tasks/"+id, ""); code != http.StatusNotFound {
				t.Errorf("DELETE foreign task %s = %d %v, want 404", id, code, out)
			}
			if code, _ := do(http.MethodGet, memberToken, "/tasks/"+id, ""); code != http.StatusOK {
				t.Errorf("foreign task %s is gone after a refused delete (GET %d)", id, code)
			}
		}
	})

	t.Run("deleting a project stays with manager, even the member's own", func(t *testing.T) {
		if code, out := do(http.MethodDelete, memberToken, "/projects/"+projectID, ""); code != http.StatusForbidden {
			t.Errorf("member DELETE project = %d %v, want 403", code, out)
		}
	})

	t.Run("manager still deletes any task tenant-wide", func(t *testing.T) {
		if code, out := do(http.MethodDelete, managerToken, "/tasks/"+colleagues, ""); code != http.StatusOK {
			t.Errorf("manager DELETE colleague's task = %d %v, want 200", code, out)
		}
	})
}
