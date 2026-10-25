package task

import (
	"context"
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
