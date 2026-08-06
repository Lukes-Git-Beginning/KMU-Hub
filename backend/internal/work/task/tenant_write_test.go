package task

// wp-work: closes the write-surface gap for tasks the same way wp-crm-core
// did for CRM entities — the existing rls_test.go/tenant_isolation_test.go
// files seed exclusively via testutil.SeedRow and never call the real
// Create/Update/Delete methods.
//
// Building this test also surfaced five real dead-write bugs: MoveTask,
// GetNextTaskNumber, DeleteDependency, UnlinkEntity and RemoveFile all took a
// bare row ID with NO tenant_id predicate anywhere in their call chain (two of
// them — UnlinkEntity, RemoveFile — are called directly from the gRPC handler
// with nothing upstream validating the tenant at all). RLS backstops every one
// of them (enable_tenant_rls is active on tasks/projects/task_dependencies/
// task_entity_links/task_files), so this was never exploitable in production
// as long as the session's app.tenant_id GUC is set correctly — but an
// explicit predicate is the difference between "found nothing" and "hit a
// foreign row" the moment a caller or a background job forgets to set it.
// Fixed in the same commit; regression-tested below by calling from the row's
// own tenant RLS session (ctxOwn) while varying only the tenantID parameter
// passed to the repo method — isolating the explicit-predicate fix from RLS,
// which is already covered by rls_test.go.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedWorkUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("wp-work-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
}

func seedWorkProject(t *testing.T, pool *pgxpool.Pool, tenantID, createdBy uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id":        tenantID,
		"name":             "WP Work Project",
		"project_key":      "WP" + uuid.New().String()[:6],
		"next_task_number": 1,
		"created_by":       createdBy,
	})
}

func seedWorkTask(t *testing.T, pool *pgxpool.Pool, tenantID, projectID, createdBy uuid.UUID, title string) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantID,
		"project_id":  projectID,
		"title":       title,
		"task_number": taskNumber(),
		"created_by":  createdBy,
	})
}

// TestTaskWrites_LandInCallerTenant covers the core Create/Update/Delete
// write surface, following the same pattern as wp-crm-core/wp-crm-meta.
func TestTaskWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Task Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Task Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	projectOwn := seedWorkProject(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())

	now := time.Now().UTC()
	tk := &models.Task{
		ID:         uuid.New(),
		TenantID:   tenantOwn,
		ProjectID:  &projectOwn,
		TaskNumber: taskNumber(),
		Title:      "Write-Test-" + uuid.New().String()[:8],
		Priority:   models.TaskPriorityNormal,
		CreatedBy:  userOwn,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.Create(ctxOther, tk); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "tasks", tk.ID, 0)

	if err := repo.Create(ctxOwn, tk); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tasks", tk.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "tasks", tk.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "tasks", tk.ID, 0)

	// Update carries an explicit tenant_id predicate AND checks RowsAffected
	// (unlike e.g. tag.Update) — a foreign-ctx call must come back an error,
	// not a silent no-op.
	foreign := *tk
	foreign.Title = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign); err == nil {
		t.Fatalf("Update (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetByID(ctxOwn, tk.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != tk.Title {
		t.Fatalf("a foreign-tenant write reached the task: title=%q", got.Title)
	}

	foreign.Title = "Renamed-" + uuid.New().String()[:8]
	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, tk.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.Title != foreign.Title {
		t.Fatalf("own-tenant write did not land: title=%q", got.Title)
	}

	// Delete carries an explicit tenantID predicate.
	if err := repo.Delete(ctxOther, tk.ID, tenantOwn); err == nil {
		t.Fatalf("Delete (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "tasks", tk.ID, 1)

	if err := repo.Delete(ctxOwn, tk.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "tasks", tk.ID, 0)
}

// TestMoveTask_TenantScoped regression-tests the MoveTask fix: the repo call
// now takes an explicit tenantID and predicates the UPDATE on it.
func TestMoveTask_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "MoveTask Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "MoveTask Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	projectOwn := seedWorkProject(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)
	statusID := testutil.SeedRow(t, pool, "project_statuses", map[string]any{
		"tenant_id":  tenantOwn,
		"project_id": projectOwn,
		"name":       "In Progress",
	})
	defer testutil.CleanupRow(t, pool, "project_statuses", statusID)
	taskID := seedWorkTask(t, pool, tenantOwn, projectOwn, userOwn, "MoveTask target")
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	// Wrong tenantID passed to the repo call must not move the task, even
	// though the calling ctx (RLS session) is the row's real owner.
	if err := repo.MoveTask(ctxOwn, tenantOther, taskID, statusID, 42.0, nil); err != nil {
		t.Fatalf("MoveTask (foreign tenantID): unexpected error %v", err)
	}
	got, err := repo.GetByID(ctxOwn, taskID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.StatusID != nil && *got.StatusID == statusID {
		t.Fatalf("MoveTask with a foreign tenantID moved the task anyway")
	}
	if got.SortOrder == 42.0 {
		t.Fatalf("MoveTask with a foreign tenantID changed sort_order anyway")
	}

	if err := repo.MoveTask(ctxOwn, tenantOwn, taskID, statusID, 42.0, nil); err != nil {
		t.Fatalf("MoveTask (own tenantID): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, taskID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.StatusID == nil || *got.StatusID != statusID || got.SortOrder != 42.0 {
		t.Fatalf("MoveTask with the correct tenantID did not land: status=%v sort=%v", got.StatusID, got.SortOrder)
	}
}

// TestGetNextTaskNumber_TenantScoped regression-tests the GetNextTaskNumber
// fix: a foreign tenantID must not be able to advance another tenant's
// project counter.
func TestGetNextTaskNumber_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "TaskNumber Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "TaskNumber Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	projectOwn := seedWorkProject(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	if _, err := repo.GetNextTaskNumber(ctxOwn, tenantOther, projectOwn); err == nil {
		t.Fatalf("GetNextTaskNumber (foreign tenantID): expected an error, got nil")
	}

	num, err := repo.GetNextTaskNumber(ctxOwn, tenantOwn, projectOwn)
	if err != nil {
		t.Fatalf("GetNextTaskNumber (own tenantID): %v", err)
	}
	if num != 1 {
		t.Fatalf("expected the first task number to be 1, got %d — the foreign call must not have advanced the counter", num)
	}
}

// TestDeleteDependency_TenantScoped regression-tests the DeleteDependency
// fix. Reachable straight from the gRPC handler with a bare dependency ID —
// no tenant validation existed anywhere in the call chain before this fix.
func TestDeleteDependency_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Dependency Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Dependency Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	projectOwn := seedWorkProject(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)
	sourceID := seedWorkTask(t, pool, tenantOwn, projectOwn, userOwn, "Dependency source")
	defer testutil.CleanupRow(t, pool, "tasks", sourceID)
	targetID := seedWorkTask(t, pool, tenantOwn, projectOwn, userOwn, "Dependency target")
	defer testutil.CleanupRow(t, pool, "tasks", targetID)

	depID := testutil.SeedRow(t, pool, "task_dependencies", map[string]any{
		"tenant_id":       tenantOwn,
		"source_task_id":  sourceID,
		"target_task_id":  targetID,
		"dependency_type": "blocks",
		"created_by":      userOwn,
	})
	defer testutil.CleanupRow(t, pool, "task_dependencies", depID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	sysCtx := testutil.WithSystemCtx(context.Background())

	if err := repo.DeleteDependency(ctxOwn, tenantOther, depID); err != nil {
		t.Fatalf("DeleteDependency (foreign tenantID): unexpected error %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "task_dependencies", depID, 1)

	if err := repo.DeleteDependency(ctxOwn, tenantOwn, depID); err != nil {
		t.Fatalf("DeleteDependency (own tenantID): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "task_dependencies", depID, 0)
}

// TestUnlinkEntity_TenantScoped regression-tests the UnlinkEntity fix. Same
// gap as DeleteDependency: called directly from the gRPC handler with a bare
// link ID.
func TestUnlinkEntity_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "EntityLink Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "EntityLink Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	projectOwn := seedWorkProject(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)
	taskID := seedWorkTask(t, pool, tenantOwn, projectOwn, userOwn, "EntityLink task")
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	linkID := testutil.SeedRow(t, pool, "task_entity_links", map[string]any{
		"tenant_id":   tenantOwn,
		"task_id":     taskID,
		"entity_type": "contact",
		"entity_id":   uuid.New(),
		"created_by":  userOwn,
	})
	defer testutil.CleanupRow(t, pool, "task_entity_links", linkID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	sysCtx := testutil.WithSystemCtx(context.Background())

	if err := repo.UnlinkEntity(ctxOwn, tenantOther, linkID); err != nil {
		t.Fatalf("UnlinkEntity (foreign tenantID): unexpected error %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "task_entity_links", linkID, 1)

	if err := repo.UnlinkEntity(ctxOwn, tenantOwn, linkID); err != nil {
		t.Fatalf("UnlinkEntity (own tenantID): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "task_entity_links", linkID, 0)
}

// TestRemoveFile_TenantScoped regression-tests the RemoveFile fix. Same gap:
// called directly from the gRPC handler with a bare file ID.
func TestRemoveFile_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "TaskFile Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "TaskFile Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	projectOwn := seedWorkProject(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)
	taskID := seedWorkTask(t, pool, tenantOwn, projectOwn, userOwn, "TaskFile task")
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	fileID := testutil.SeedRow(t, pool, "task_files", map[string]any{
		"tenant_id":   tenantOwn,
		"task_id":     taskID,
		"uploaded_by": userOwn,
		"filename":    "spec.pdf",
		"mime_type":   "application/pdf",
		"file_size":   1024,
		"storage_key": "tasks/" + uuid.New().String(),
	})
	defer testutil.CleanupRow(t, pool, "task_files", fileID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	sysCtx := testutil.WithSystemCtx(context.Background())

	if err := repo.RemoveFile(ctxOwn, tenantOther, fileID); err != nil {
		t.Fatalf("RemoveFile (foreign tenantID): unexpected error %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "task_files", fileID, 1)

	if err := repo.RemoveFile(ctxOwn, tenantOwn, fileID); err != nil {
		t.Fatalf("RemoveFile (own tenantID): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "task_files", fileID, 0)
}
