package server

// A project template used to be an empty shell: SaveProjectAsTemplate copied
// only the project row, CreateProjectFromTemplate created a project without
// statuses or tasks, and the task copier (no caller) failed on the RLS policy
// because its rows carried no tenant. These tests run the real handlers on real
// repositories as kmuhub_app, so a dropped tenant, a status that still points
// into the source project or a skipped copy shows up as a wrong row.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/project"
	wstatus "github.com/kmuhub/kmuhub/internal/work/status"
	"github.com/kmuhub/kmuhub/internal/work/task"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

type templateFixture struct {
	pool      *pgxpool.Pool
	srv       *WorkGRPCServer
	tenant    uuid.UUID
	other     uuid.UUID
	user      uuid.UUID
	otherUser uuid.UUID
	// source is a project with statuses "Todo"/"Done" and a parent task in Todo,
	// its child in Done, and a blocks dependency parent -> child.
	source                uuid.UUID
	todo, done            uuid.UUID
	parent, child         uuid.UUID
	ctxOwn, ctxForeignOwn context.Context
}

func newTemplateFixture(t *testing.T) *templateFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	f := &templateFixture{pool: pool, tenant: uuid.New(), other: uuid.New()}
	testutil.EnsureTenant(t, pool, f.tenant, "Template Tenant")
	testutil.EnsureTenant(t, pool, f.other, "Template Other Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", f.tenant) })
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", f.other) })

	seedUser := func(tenant uuid.UUID) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenant,
			"email":         "template-" + uuid.NewString() + "@test.invalid",
			"password_hash": "x",
			"first_name":    "Tess",
			"last_name":     "Plate",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	f.user = seedUser(f.tenant)
	f.otherUser = seedUser(f.other)

	projRepo := project.NewPostgresRepository(pool)
	taskRepo := task.NewPostgresRepository(pool)
	taskSvc := task.NewService(taskRepo, projRepo)
	taskSvc.SetOpenProjects(projectsOpenToTenant)
	f.srv = NewWorkGRPCServer(
		project.NewService(projRepo), wstatus.NewService(wstatus.NewPostgresRepository(pool)),
		taskSvc, taskRepo, nil, nil, nil, nil,
	)
	f.ctxOwn = ctxWithActorAndTenant(f.user, f.tenant)
	f.ctxForeignOwn = ctxWithActorAndTenant(f.otherUser, f.other)

	f.source = testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id": f.tenant, "name": "Source Project",
		"project_key": "SRC" + uuid.NewString()[:6], "next_task_number": 1, "created_by": f.user,
	})
	seedStatus := func(name string, order int, closed bool) uuid.UUID {
		return testutil.SeedRow(t, pool, "project_statuses", map[string]any{
			"tenant_id": f.tenant, "project_id": f.source, "name": name,
			"sort_order": order, "is_default": order == 0, "is_closed": closed,
		})
	}
	f.todo = seedStatus("Todo", 0, false)
	f.done = seedStatus("Done", 1, true)

	assignee := f.user
	seedTask := func(num int, title string, statusID uuid.UUID, parentID *uuid.UUID, depth int) uuid.UUID {
		cols := map[string]any{
			"tenant_id": f.tenant, "project_id": f.source, "task_number": num, "title": title,
			"status_id": statusID, "created_by": f.user, "depth": depth, "sort_order": float64(num),
			"assignee_id": assignee, "due_date": "2030-01-01T00:00:00Z",
		}
		if parentID != nil {
			cols["parent_task_id"] = *parentID
		}
		return testutil.SeedRow(t, pool, "tasks", cols)
	}
	f.parent = seedTask(1, "Parent step", f.todo, nil, 0)
	f.child = seedTask(2, "Child step", f.done, &f.parent, 1)
	testutil.SeedRow(t, pool, "task_dependencies", map[string]any{
		"tenant_id": f.tenant, "source_task_id": f.parent, "target_task_id": f.child,
		"dependency_type": "blocks", "created_by": f.user,
	})
	// Registered last, so it runs first: users and tenants cannot go while projects reference them.
	t.Cleanup(func() {
		_, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
			`DELETE FROM projects WHERE tenant_id IN ($1, $2)`, f.tenant, f.other)
		if err != nil {
			t.Logf("cleanup projects: %v", err)
		}
	})
	return f
}

type copiedTask struct {
	id, projectID uuid.UUID
	tenant        uuid.UUID
	title         string
	statusProject *uuid.UUID
	statusName    *string
	parent        *uuid.UUID
	assignee      *uuid.UUID
	hasDueDate    bool
}

// readProject returns the tasks of one project, the number of its statuses and
// the number of dependencies between them, read as system so the assertions do
// not depend on the RLS session under test.
func (f *templateFixture) readProject(t *testing.T, projectID uuid.UUID) (tasks []copiedTask, statuses, deps int) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())

	rows, err := f.pool.Query(ctx,
		`SELECT t.id, t.project_id, t.tenant_id, t.title, ps.project_id, ps.name, t.parent_task_id,
		        t.assignee_id, t.due_date IS NOT NULL
		 FROM tasks t LEFT JOIN project_statuses ps ON ps.id = t.status_id
		 WHERE t.project_id = $1 ORDER BY t.task_number`, projectID)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var c copiedTask
		require.NoError(t, rows.Scan(&c.id, &c.projectID, &c.tenant, &c.title, &c.statusProject, &c.statusName,
			&c.parent, &c.assignee, &c.hasDueDate))
		tasks = append(tasks, c)
	}
	require.NoError(t, rows.Err())

	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT count(*) FROM project_statuses WHERE project_id = $1`, projectID).Scan(&statuses))
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT count(*) FROM task_dependencies d JOIN tasks t ON t.id = d.source_task_id
		 WHERE t.project_id = $1`, projectID).Scan(&deps))
	return tasks, statuses, deps
}

func (f *templateFixture) assertCopyOfSource(t *testing.T, projectID uuid.UUID) {
	t.Helper()
	tasks, statuses, deps := f.readProject(t, projectID)
	require.Len(t, tasks, 2, "both template tasks must be copied")
	assert.Equal(t, 2, statuses, "the statuses are copied with the tasks")
	assert.Equal(t, 1, deps, "the dependency between the two tasks is copied")

	parent, child := tasks[0], tasks[1]
	assert.Equal(t, "Parent step", parent.title)
	assert.Equal(t, "Child step", child.title)
	for _, c := range tasks {
		assert.Equal(t, f.tenant, c.tenant, "a copied task must land in the caller's tenant")
		assert.NotEqual(t, f.parent, c.id)
		assert.NotEqual(t, f.child, c.id)
		assert.Nil(t, c.assignee, "assignee is not part of a template")
		assert.False(t, c.hasDueDate, "dates are not part of a template")
		if assert.NotNil(t, c.statusProject, "status must be mapped, not dropped") {
			assert.Equal(t, projectID, *c.statusProject, "status must belong to the NEW project, not the source")
		}
	}
	assert.Equal(t, "Todo", *parent.statusName)
	assert.Equal(t, "Done", *child.statusName)
	if assert.NotNil(t, child.parent) {
		assert.Equal(t, parent.id, *child.parent, "parent link points at the copied parent")
	}
}

func TestWorkGRPC_TemplateRoundTripCopiesStatusesAndTasks(t *testing.T) {
	f := newTemplateFixture(t)

	saved, err := f.srv.SaveProjectAsTemplate(f.ctxOwn, &workv1.SaveProjectAsTemplateRequest{
		ProjectId: f.source.String(), TemplateName: "Onboarding Template", CreatedBy: f.user.String(),
	})
	require.NoError(t, err)
	templateID := uuid.MustParse(saved.Template.Id)
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "projects", templateID) })
	f.assertCopyOfSource(t, templateID)

	created, err := f.srv.CreateProjectFromTemplate(f.ctxOwn, &workv1.CreateProjectFromTemplateRequest{
		TemplateId: templateID.String(), Name: "Customer Onboarding", ProjectKey: "ONB" + uuid.NewString()[:5],
		CreatedBy: f.user.String(),
	})
	require.NoError(t, err)
	newID := uuid.MustParse(created.Project.Id)
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "projects", newID) })
	f.assertCopyOfSource(t, newID)
}

func TestWorkGRPC_CreateFromTemplate_ForeignTenantTemplateIsNotFound(t *testing.T) {
	f := newTemplateFixture(t)
	// The source project doubles as a template of the owning tenant.
	_, err := f.pool.Exec(testutil.WithSystemCtx(context.Background()),
		`UPDATE projects SET is_template = true WHERE id = $1`, f.source)
	require.NoError(t, err)

	key := "FRN" + uuid.NewString()[:5]
	_, err = f.srv.CreateProjectFromTemplate(f.ctxForeignOwn, &workv1.CreateProjectFromTemplateRequest{
		TemplateId: f.source.String(), Name: "Stolen Template", ProjectKey: key, CreatedBy: f.otherUser.String(),
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))

	var n int
	require.NoError(t, f.pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT count(*) FROM projects WHERE project_key = $1`, key).Scan(&n))
	assert.Zero(t, n, "no project row for the foreign caller")
	var tasksInOther int
	require.NoError(t, f.pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT count(*) FROM tasks WHERE tenant_id = $1`, f.other).Scan(&tasksInOther))
	assert.Zero(t, tasksInOther, "no task written for the foreign tenant")
}

// The task service is the last line: even if a caller hands it a project of
// another tenant, it refuses before reading or writing a row.
func TestTaskService_CreateFromTemplate_RefusesForeignProjects(t *testing.T) {
	f := newTemplateFixture(t)

	target := testutil.SeedRow(t, f.pool, "projects", map[string]any{
		"tenant_id": f.other, "name": "Foreign Target", "project_key": "TGT" + uuid.NewString()[:6],
		"next_task_number": 1, "created_by": f.otherUser,
	})
	taskSvc := task.NewService(task.NewPostgresRepository(f.pool), project.NewPostgresRepository(f.pool))

	err := taskSvc.CreateFromTemplate(f.ctxForeignOwn, f.other, f.source, target, f.otherUser, nil)
	require.Error(t, err, "the source project belongs to another tenant")
	tasks, _, _ := f.readProject(t, target)
	assert.Empty(t, tasks)
}

// A task row without its tenant is rejected by the RLS policy (the old failure),
// so the copy sets it itself; a status without a mapping is dropped, never kept.
func TestTaskService_CreateFromTemplate_UnmappedStatusLandsWithoutStatus(t *testing.T) {
	f := newTemplateFixture(t)

	target := testutil.SeedRow(t, f.pool, "projects", map[string]any{
		"tenant_id": f.tenant, "name": "Plain Target", "project_key": "PLN" + uuid.NewString()[:6],
		"next_task_number": 1, "created_by": f.user,
	})
	taskSvc := task.NewService(task.NewPostgresRepository(f.pool), project.NewPostgresRepository(f.pool))

	require.NoError(t, taskSvc.CreateFromTemplate(f.ctxOwn, f.tenant, f.source, target, f.user, nil))
	tasks, _, deps := f.readProject(t, target)
	require.Len(t, tasks, 2)
	assert.Equal(t, 1, deps)
	for _, c := range tasks {
		assert.Equal(t, f.tenant, c.tenant)
		assert.Nil(t, c.statusProject, "a status without a mapping must not point into the source project")
	}
}
