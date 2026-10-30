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
	chatfile "github.com/kmuhub/kmuhub/internal/chat/file"
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
// repositories. The same scope holds on the edit sub-routes of a task:
// dependencies, entity links and files, including their top-level deletes.

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
	memberID, memberToken := tokenFor("member")

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

	// Sub-routes. Each create returns the child wrapped under its key.
	create := func(token, path, key, body string) (int, string) {
		t.Helper()
		code, out := do(http.MethodPost, token, path, body)
		if code != http.StatusCreated {
			return code, ""
		}
		return code, idOf(out, key)
	}
	mustCreate := func(token, path, key, body string) string {
		t.Helper()
		code, id := create(token, path, key, body)
		if code != http.StatusCreated {
			t.Fatalf("POST %s = %d, want 201", path, code)
		}
		return id
	}
	depBody := func(target string) string {
		return `{"target_task_id":"` + target + `","dependency_type":"relates_to"}`
	}
	linkBody := func() string {
		return `{"entity_type":"contact","entity_id":"` + uuid.NewString() + `"}`
	}
	fileBody := func() string {
		return `{"filename":"plan.pdf","mime_type":"application/pdf","file_size":10,"storage_key":"` +
			chatfile.UnattachedUploadPrefix(ownTaskEditTenant) + uuid.NewString() + `"}`
	}
	rows := func(table, id string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(sysCtx, `SELECT count(*) FROM `+table+` WHERE id = $1`, id).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		return n
	}
	childCount := func(table, taskID string) int {
		t.Helper()
		col := "task_id"
		if table == "task_dependencies" {
			col = "source_task_id"
		}
		var n int
		if err := pool.QueryRow(sysCtx, `SELECT count(*) FROM `+table+` WHERE `+col+` = $1`, taskID).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		return n
	}
	other := createTask(memberToken, `{"title":"Zweite Aufgabe des Kollegen"}`)
	// The cases below that need a dependency of somebody else's seed it
	// directly; the caller's own go through the real POST.
	seedDep := func(source, target string, by uuid.UUID) string {
		t.Helper()
		id := testutil.SeedRow(t, pool, "task_dependencies", map[string]any{
			"id":              uuid.New(),
			"tenant_id":       ownTaskEditTenant,
			"source_task_id":  uuid.MustParse(source),
			"target_task_id":  uuid.MustParse(target),
			"dependency_type": "relates_to",
			"created_by":      by,
		})
		return id.String()
	}

	t.Run("hr_admin edits sub-objects of the tasks they own or are assigned to", func(t *testing.T) {
		for _, taskID := range []string{own, assigned} {
			dep := mustCreate(hrToken, "/tasks/"+taskID+"/dependencies", "dependency", depBody(other))
			if code, out := do(http.MethodDelete, hrToken, "/task-dependencies/"+dep, ""); code != http.StatusOK || rows("task_dependencies", dep) != 0 {
				t.Errorf("DELETE own dependency = %d %v, rows left %d", code, out, rows("task_dependencies", dep))
			}
			link := mustCreate(hrToken, "/tasks/"+taskID+"/links", "link", linkBody())
			if code, out := do(http.MethodDelete, hrToken, "/task-links/"+link, ""); code != http.StatusOK || rows("task_entity_links", link) != 0 {
				t.Errorf("DELETE own link = %d %v, rows left %d", code, out, rows("task_entity_links", link))
			}
			file := mustCreate(hrToken, "/tasks/"+taskID+"/files", "file", fileBody())
			if code, out := do(http.MethodDelete, hrToken, "/task-files/"+file, ""); code != http.StatusOK || rows("task_files", file) != 0 {
				t.Errorf("DELETE own file = %d %v, rows left %d", code, out, rows("task_files", file))
			}
		}
	})

	t.Run("hr_admin removes a colleague's dependency that points at their own task", func(t *testing.T) {
		dep := seedDep(members, own, memberID)
		if code, out := do(http.MethodDelete, hrToken, "/task-dependencies/"+dep, ""); code != http.StatusOK || rows("task_dependencies", dep) != 0 {
			t.Errorf("DELETE dependency onto own task = %d %v, rows left %d", code, out, rows("task_dependencies", dep))
		}
	})

	t.Run("hr_admin cannot touch sub-objects of a colleague's task", func(t *testing.T) {
		for _, c := range []struct{ path, key, table, body string }{
			{"/dependencies", "dependency", "task_dependencies", depBody(other)},
			{"/links", "link", "task_entity_links", linkBody()},
			{"/files", "file", "task_files", fileBody()},
		} {
			if code, _ := create(hrToken, "/tasks/"+members+c.path, c.key, c.body); code != http.StatusNotFound {
				t.Errorf("POST foreign task %s = %d, want 404", c.path, code)
			}
			if n := childCount(c.table, members); n != 0 {
				t.Errorf("refused POST %s left %d rows on the foreign task", c.path, n)
			}
		}
		for _, c := range []struct{ path, key, table, del, body string }{
			{"", "", "task_dependencies", "/task-dependencies/", ""},
			{"/links", "link", "task_entity_links", "/task-links/", linkBody()},
			{"/files", "file", "task_files", "/task-files/", fileBody()},
		} {
			var id string
			if c.path == "" {
				id = seedDep(members, other, memberID)
			} else {
				id = mustCreate(memberToken, "/tasks/"+members+c.path, c.key, c.body)
			}
			if code, out := do(http.MethodDelete, hrToken, c.del+id, ""); code != http.StatusNotFound {
				t.Errorf("DELETE foreign %s = %d %v, want 404", c.del, code, out)
			}
			if n := rows(c.table, id); n != 1 {
				t.Errorf("refused DELETE %s removed the row (rows %d)", c.del, n)
			}
		}
	})

	t.Run("member edits sub-objects tenant-wide through tasks:write", func(t *testing.T) {
		dep := seedDep(own, other, hrID)
		link := mustCreate(memberToken, "/tasks/"+own+"/links", "link", linkBody())
		file := mustCreate(memberToken, "/tasks/"+own+"/files", "file", fileBody())
		for _, c := range []struct{ del, table, id string }{
			{"/task-dependencies/", "task_dependencies", dep},
			{"/task-links/", "task_entity_links", link},
			{"/task-files/", "task_files", file},
		} {
			if code, out := do(http.MethodDelete, memberToken, c.del+c.id, ""); code != http.StatusOK || rows(c.table, c.id) != 0 {
				t.Errorf("member DELETE %s on hr_admin's task = %d %v", c.del, code, out)
			}
		}
	})
}
