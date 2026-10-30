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

// hr_admin holds work:task:edit at scope own and no legacy tasks:* key: they
// edit the tasks they created or are assigned to (the owners the desktop's
// useTaskCan checks), any other task answers 404 and stays as it was. member keeps editing tenant-wide through tasks:write=all (the alt key of
// the guard). The chain is the production one: the real seed through
// ResolveTokenPermissions, the real Auth middleware and route registration, a
// real work gRPC server behind TenantInboundUnaryInterceptor and the Postgres
// repositories.

var ownTaskEditTenant = uuid.MustParse("e4fec700-0000-4000-8000-0000000000d4")

func TestWorkOwnTaskEdit_HRAdminAgainstRealSeed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	testutil.EnsureTenant(t, pool, ownTaskEditTenant, "OwnTaskEditTenant")
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

	tokenFor := func(role string) (uuid.UUID, string) {
		t.Helper()
		userID := testutil.SeedRow(t, pool, "users", map[string]any{
			"id":            uuid.New(),
			"tenant_id":     ownTaskEditTenant,
			"email":         "own-task-edit-" + uuid.NewString() + "@test.local",
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
			userID, roleID, ownTaskEditTenant); err != nil {
			t.Fatalf("assign role: %v", err)
		}
		perms, err := authSvc.ResolveTokenPermissions(sysCtx, ownTaskEditTenant, userID)
		if err != nil {
			t.Fatalf("resolve permissions: %v", err)
		}
		token, err := tokenMaker.CreateAccessToken(userID, ownTaskEditTenant.String(), []string{role}, perms)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}
		return userID, token
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

	titleOf := func(token, id string) string {
		t.Helper()
		code, out := do(http.MethodGet, token, "/tasks/"+id, "")
		if code != http.StatusOK {
			t.Fatalf("GET /tasks/%s = %d %v", id, code, out)
		}
		inner, _ := out["task"].(map[string]any)
		title, _ := inner["title"].(string)
		return title
	}

	hrID, hrToken := tokenFor("hr_admin")
	_, memberToken := tokenFor("member")

	own := createTask(hrToken, `{"title":"Eigene Aufgabe"}`)
	members := createTask(memberToken, `{"title":"Aufgabe des Kollegen"}`)
	assigned := createTask(memberToken, `{"title":"Zugewiesen","assignee_id":"`+hrID.String()+`"}`)

	t.Run("hr_admin edits the task they created", func(t *testing.T) {
		if code, out := do(http.MethodPut, hrToken, "/tasks/"+own, `{"title":"Gepflegt"}`); code != http.StatusOK {
			t.Fatalf("PUT own task = %d %v, want 200", code, out)
		}
		if got := titleOf(hrToken, own); got != "Gepflegt" {
			t.Errorf("own task title = %q after PUT, want Gepflegt", got)
		}
	})

	t.Run("hr_admin edits and completes a colleague's task assigned to them", func(t *testing.T) {
		if code, out := do(http.MethodPut, hrToken, "/tasks/"+assigned, `{"title":"Erledigt","is_closed":true}`); code != http.StatusOK {
			t.Fatalf("PUT assigned task = %d %v, want 200", code, out)
		}
		if got := titleOf(hrToken, assigned); got != "Erledigt" {
			t.Errorf("assigned task title = %q after PUT, want Erledigt", got)
		}
	})

	t.Run("hr_admin cannot edit a colleague's task", func(t *testing.T) {
		if code, out := do(http.MethodPut, hrToken, "/tasks/"+members, `{"title":"Uebernommen"}`); code != http.StatusNotFound {
			t.Errorf("PUT foreign task = %d %v, want 404", code, out)
		}
		if got := titleOf(memberToken, members); got != "Aufgabe des Kollegen" {
			t.Errorf("foreign task renamed to %q by a refused PUT", got)
		}
	})

	t.Run("member stays tenant-wide through tasks:write", func(t *testing.T) {
		if code, out := do(http.MethodPut, memberToken, "/tasks/"+own, `{"title":"Vom Kollegen"}`); code != http.StatusOK {
			t.Errorf("member PUT hr_admin's task = %d %v, want 200", code, out)
		}
		if got := titleOf(hrToken, own); got != "Vom Kollegen" {
			t.Errorf("hr_admin's task title = %q after member PUT, want Vom Kollegen", got)
		}
	})
}
