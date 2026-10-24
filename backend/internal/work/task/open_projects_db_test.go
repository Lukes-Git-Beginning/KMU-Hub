package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/project"
)

// TestOpenProjects_RealSQL runs the open-project switch through the real
// project and task repositories: the membership lookup answers
// project.ErrNotProjectMember (not a mock's error), the project lookup is
// tenant-scoped, and a task of another tenant stays NotFound.
func TestOpenProjects_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Open Projects Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Open Projects Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	owner := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", owner)
	nonMember := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", nonMember)
	otherUser := seedWorkUser(t, pool, tenantOther)
	defer testutil.CleanupRow(t, pool, "users", otherUser)

	projectOwn := seedWorkProject(t, pool, tenantOwn, owner)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)
	projectOther := seedWorkProject(t, pool, tenantOther, otherUser)
	defer testutil.CleanupRow(t, pool, "projects", projectOther)
	taskOther := seedWorkTask(t, pool, tenantOther, projectOther, otherUser, "foreign")
	defer testutil.CleanupRow(t, pool, "tasks", taskOther)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	repo := NewPostgresRepository(pool)
	projRepo := project.NewPostgresRepository(pool)

	closed := NewService(repo, projRepo)
	open := NewService(repo, projRepo)
	open.SetOpenProjects(true)

	if _, err := closed.Create(ctxOwn, CreateInput{TenantID: tenantOwn, ProjectID: &projectOwn, Title: "t", CreatedBy: nonMember}); !errors.Is(err, ErrNotProjectMember) {
		t.Fatalf("closed: non-member create want ErrNotProjectMember, got %v", err)
	}

	created, err := open.Create(ctxOwn, CreateInput{TenantID: tenantOwn, ProjectID: &projectOwn, Title: "t", CreatedBy: nonMember})
	if err != nil {
		t.Fatalf("open: non-member create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tasks", created.ID)

	title := "renamed"
	if _, err := open.Update(ctxOwn, tenantOwn, created.ID, UpdateInput{Title: &title}, nonMember); err != nil {
		t.Fatalf("open: non-member update: %v", err)
	}

	// The foreign project id no longer hits a membership wall, so the tenant-scoped
	// project lookup is what keeps it out.
	if _, err := open.Create(ctxOwn, CreateInput{TenantID: tenantOwn, ProjectID: &projectOther, Title: "t", CreatedBy: nonMember}); !errors.Is(err, project.ErrNotFound) {
		t.Fatalf("open: create in another tenant's project want project.ErrNotFound, got %v", err)
	}
	if err := open.Delete(ctxOwn, tenantOwn, taskOther, nonMember); !errors.Is(err, ErrNotFound) {
		t.Fatalf("open: delete of another tenant's task want ErrNotFound, got %v", err)
	}

	if err := open.Delete(ctxOwn, tenantOwn, created.ID, nonMember); err != nil {
		t.Fatalf("open: non-member delete: %v", err)
	}
	if err := closed.Delete(ctxOwn, tenantOwn, created.ID, nonMember); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted task must be gone, got %v", err)
	}
}

// TestUpdate_ClearDueDate_RealSQL: ClearDueDate writes NULL through the real repository;
// an update without it keeps the stored date.
func TestUpdate_ClearDueDate_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Clear Due Date Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	owner := seedWorkUser(t, pool, tenantID)
	defer testutil.CleanupRow(t, pool, "users", owner)
	projectID := seedWorkProject(t, pool, tenantID, owner)
	defer testutil.CleanupRow(t, pool, "projects", projectID)
	taskID := seedWorkTask(t, pool, tenantID, projectID, owner, "due")
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	svc := NewService(NewPostgresRepository(pool), project.NewPostgresRepository(pool))
	svc.SetOpenProjects(true) // membership is not under test here

	due := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	got, err := svc.Update(ctx, tenantID, taskID, UpdateInput{DueDate: &due}, owner)
	if err != nil || got.DueDate == nil {
		t.Fatalf("set due date: %v / %v", err, got)
	}

	title := "renamed"
	got, err = svc.Update(ctx, tenantID, taskID, UpdateInput{Title: &title}, owner)
	if err != nil || got.DueDate == nil {
		t.Fatalf("an update without ClearDueDate must keep the date: %v / %v", err, got)
	}

	got, err = svc.Update(ctx, tenantID, taskID, UpdateInput{ClearDueDate: true}, owner)
	if err != nil {
		t.Fatalf("clear due date: %v", err)
	}
	if got.DueDate != nil {
		t.Fatalf("due date must be gone, got %v", got.DueDate)
	}

	// Clearing a date that is already gone is a no-op, not an error.
	if _, err := svc.Update(ctx, tenantID, taskID, UpdateInput{ClearDueDate: true}, owner); err != nil {
		t.Fatalf("clearing twice: %v", err)
	}
}
