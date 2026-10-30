package task

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/project"
)

// TestStandaloneTask_RealSQL: "Meine Aufgaben -> Neue Aufgabe" creates a task with no project and
// no status. It must come back from the real repository with a NULL project/status and show up
// under the assignee filter - for its own tenant only.
func TestStandaloneTask_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Standalone Task Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Standalone Task Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	me := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", me)
	other := seedWorkUser(t, pool, tenantOther)
	defer testutil.CleanupRow(t, pool, "users", other)

	svc := NewService(NewPostgresRepository(pool), project.NewPostgresRepository(pool))
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	created, err := svc.Create(ctxOwn, CreateInput{TenantID: tenantOwn, Title: "standalone", Priority: "high", AssigneeID: &me, CreatedBy: me})
	if err != nil {
		t.Fatalf("create without project: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tasks", created.ID)
	if created.ProjectID != nil || created.StatusID != nil {
		t.Fatalf("want NULL project and status, got project=%v status=%v", created.ProjectID, created.StatusID)
	}

	tasks, total, err := svc.List(ctxOwn, tenantOwn, TaskFilters{AssigneeID: &me, Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("list own: %v", err)
	}
	if total != 1 || len(tasks) != 1 || tasks[0].ID != created.ID {
		t.Fatalf("assignee filter want exactly the standalone task, got total=%d tasks=%d", total, len(tasks))
	}

	foreign, foreignTotal, err := svc.List(ctxOther, tenantOther, TaskFilters{AssigneeID: &me, Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("list other tenant: %v", err)
	}
	if foreignTotal != 0 || len(foreign) != 0 {
		t.Fatalf("another tenant must not see the task, got total=%d", foreignTotal)
	}
}

// TestStandaloneTask_Complete_RealSQL: is_closed on a project-less task sets and clears
// completed_at through the real repository (no status exists to move to), and a foreign
// tenant cannot complete it.
func TestStandaloneTask_Complete_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Standalone Complete Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Standalone Complete Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	me := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", me)
	other := seedWorkUser(t, pool, tenantOther)
	defer testutil.CleanupRow(t, pool, "users", other)

	svc := NewService(NewPostgresRepository(pool), project.NewPostgresRepository(pool))
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	created, err := svc.Create(ctxOwn, CreateInput{TenantID: tenantOwn, Title: "tick me", AssigneeID: &me, CreatedBy: me})
	if err != nil {
		t.Fatalf("create without project: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tasks", created.ID)

	yes, no := true, false
	if _, err := svc.Update(ctxOther, tenantOther, created.ID, UpdateInput{IsClosed: &yes}, other); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign tenant want ErrNotFound, got %v", err)
	}

	got, err := svc.Update(ctxOwn, tenantOwn, created.ID, UpdateInput{IsClosed: &yes}, me)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if got.CompletedAt == nil || got.StatusID != nil {
		t.Fatalf("want completed_at and still no status, got completed=%v status=%v", got.CompletedAt, got.StatusID)
	}
	stored, err := svc.GetByID(ctxOwn, tenantOwn, created.ID)
	if err != nil || stored.CompletedAt == nil {
		t.Fatalf("completed_at must be persisted: %v / %+v", err, stored)
	}

	got, err = svc.Update(ctxOwn, tenantOwn, created.ID, UpdateInput{IsClosed: &no}, me)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if got.CompletedAt != nil {
		t.Fatalf("reopen must clear completed_at, got %v", got.CompletedAt)
	}
}

// TestStandaloneTask_DependencyRealSQL: a dependency between two tasks without a project used to
// fail with a NULL tenant_id (the INSERT read it from the project join), so POST answered 500 for
// every caller. The row must carry the caller's tenant, and a task of another tenant at either end
// must be refused.
func TestStandaloneTask_DependencyRealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Standalone Dependency Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Standalone Dependency Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	me := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", me)
	other := seedWorkUser(t, pool, tenantOther)
	defer testutil.CleanupRow(t, pool, "users", other)

	svc := NewService(NewPostgresRepository(pool), project.NewPostgresRepository(pool))
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	newTask := func(ctx context.Context, tenantID, by uuid.UUID, title string) uuid.UUID {
		t.Helper()
		created, err := svc.Create(ctx, CreateInput{TenantID: tenantID, Title: title, Priority: "high", CreatedBy: by})
		if err != nil {
			t.Fatalf("create %q: %v", title, err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "tasks", created.ID) })
		return created.ID
	}
	a := newTask(ctxOwn, tenantOwn, me, "standalone A")
	b := newTask(ctxOwn, tenantOwn, me, "standalone B")
	foreign := newTask(ctxOther, tenantOther, other, "standalone foreign")

	dep, err := svc.CreateDependency(ctxOwn, DepInput{TenantID: tenantOwn, SourceTaskID: a, TargetTaskID: b, DependencyType: "blocks", CreatedBy: me})
	if err != nil {
		t.Fatalf("dependency between two project-less tasks: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "task_dependencies", dep.ID) })
	var rowTenant uuid.UUID
	if err := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT tenant_id FROM task_dependencies WHERE id = $1`, dep.ID).Scan(&rowTenant); err != nil {
		t.Fatalf("read back dependency: %v", err)
	}
	if rowTenant != tenantOwn {
		t.Fatalf("dependency tenant_id = %s, want the caller's %s", rowTenant, tenantOwn)
	}

	for name, in := range map[string]DepInput{
		"foreign target": {TenantID: tenantOwn, SourceTaskID: a, TargetTaskID: foreign, DependencyType: "relates_to", CreatedBy: me},
		"foreign source": {TenantID: tenantOwn, SourceTaskID: foreign, TargetTaskID: b, DependencyType: "relates_to", CreatedBy: me},
	} {
		// System context: RLS would hide the foreign task anyway, the explicit tenant filter must hold without it.
		if _, err := svc.CreateDependency(testutil.WithSystemCtx(context.Background()), in); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: err = %v, want ErrNotFound", name, err)
		}
	}
}
