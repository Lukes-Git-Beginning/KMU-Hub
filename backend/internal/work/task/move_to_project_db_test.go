package task

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/project"
)

// TestUpdate_MoveToProject_RealSQL: a project-less task moves into a project with the next
// task number and the first open status; a task that already has a project, a parent or
// subtasks stays put; another tenant's project reads as not found; a viewer cannot move.
func TestUpdate_MoveToProject_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Move To Project Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Move To Project Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	owner := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", owner)
	viewer := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", viewer)
	otherUser := seedWorkUser(t, pool, tenantOther)
	defer testutil.CleanupRow(t, pool, "users", otherUser)

	target := seedWorkProject(t, pool, tenantOwn, owner)
	defer testutil.CleanupRow(t, pool, "projects", target)
	otherProject := seedWorkProject(t, pool, tenantOther, otherUser)
	defer testutil.CleanupRow(t, pool, "projects", otherProject)
	viewerProject := seedWorkProject(t, pool, tenantOwn, owner)
	defer testutil.CleanupRow(t, pool, "projects", viewerProject)

	seedStatus := func(projectID uuid.UUID, tenantID uuid.UUID, name string, sortOrder int, closed bool) uuid.UUID {
		return testutil.SeedRow(t, pool, "project_statuses", map[string]any{
			"tenant_id": tenantID, "project_id": projectID, "name": name,
			"sort_order": sortOrder, "is_default": false, "is_closed": closed,
		})
	}
	_ = seedStatus(target, tenantOwn, "Erledigt", 1, true)
	openStatus := seedStatus(target, tenantOwn, "Todo", 2, false)
	_ = seedStatus(viewerProject, tenantOwn, "Todo", 1, false)
	_ = seedStatus(otherProject, tenantOther, "Todo", 1, false)

	if _, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
		`INSERT INTO project_members (project_id, user_id, tenant_id, role) VALUES ($1, $2, $3, 'viewer')`,
		viewerProject, viewer, tenantOwn); err != nil {
		t.Fatalf("seed viewer: %v", err)
	}

	seedLoose := func(parent *uuid.UUID) uuid.UUID {
		row := map[string]any{"tenant_id": tenantOwn, "title": "loose", "task_number": 0, "created_by": owner}
		if parent != nil {
			row["parent_task_id"] = *parent
			row["depth"] = 1
		}
		id := testutil.SeedRow(t, pool, "tasks", row)
		return id
	}
	loose := seedLoose(nil)
	defer testutil.CleanupRow(t, pool, "tasks", loose)
	child := seedLoose(&loose)
	defer testutil.CleanupRow(t, pool, "tasks", child)
	other := seedLoose(nil)
	defer testutil.CleanupRow(t, pool, "tasks", other)
	withProject := seedWorkTask(t, pool, tenantOwn, target, owner, "has project")
	defer testutil.CleanupRow(t, pool, "tasks", withProject)

	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)
	svc := NewService(NewPostgresRepository(pool), project.NewPostgresRepository(pool))
	svc.SetOpenProjects(true)

	// A task with a subtask, and the subtask itself, stay where they are.
	for _, id := range []uuid.UUID{loose, child} {
		if _, err := svc.Update(ctx, tenantOwn, id, UpdateInput{ProjectID: &target}, owner); !errors.Is(err, ErrCannotMoveToProject) {
			t.Fatalf("task %s with parent/subtasks want ErrCannotMoveToProject, got %v", id, err)
		}
	}
	// A task that already belongs to a project does not move to another one...
	if _, err := svc.Update(ctx, tenantOwn, withProject, UpdateInput{ProjectID: &viewerProject}, owner); !errors.Is(err, ErrCannotMoveToProject) {
		t.Fatalf("project-to-project move want ErrCannotMoveToProject, got %v", err)
	}
	// ...but naming its own project is a no-op.
	if _, err := svc.Update(ctx, tenantOwn, withProject, UpdateInput{ProjectID: &target}, owner); err != nil {
		t.Fatalf("same project must be a no-op: %v", err)
	}

	// Another tenant's project is invisible.
	if _, err := svc.Update(ctx, tenantOwn, other, UpdateInput{ProjectID: &otherProject}, owner); !errors.Is(err, project.ErrNotFound) {
		t.Fatalf("foreign project want project.ErrNotFound, got %v", err)
	}
	var stored *uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT project_id FROM tasks WHERE id = $1`, other).Scan(&stored); err != nil || stored != nil {
		t.Fatalf("task must stay project-less after the rejected move: %v / %v", err, stored)
	}

	// A viewer of the target project cannot move a task into it (closed mode).
	closedSvc := NewService(NewPostgresRepository(pool), project.NewPostgresRepository(pool))
	if _, err := closedSvc.Update(ctx, tenantOwn, other, UpdateInput{ProjectID: &viewerProject}, viewer); !errors.Is(err, ErrViewerCannotEdit) {
		t.Fatalf("viewer want ErrViewerCannotEdit, got %v", err)
	}

	// Happy path: number and open status of the target project.
	got, err := svc.Update(ctx, tenantOwn, other, UpdateInput{ProjectID: &target}, owner)
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if got.ProjectID == nil || *got.ProjectID != target {
		t.Fatalf("project_id = %v, want %s", got.ProjectID, target)
	}
	if got.StatusID == nil || *got.StatusID != openStatus {
		t.Fatalf("status_id = %v, want the first open status %s", got.StatusID, openStatus)
	}
	if got.TaskNumber < 1 {
		t.Fatalf("task_number = %d, want the project's next number", got.TaskNumber)
	}
}
