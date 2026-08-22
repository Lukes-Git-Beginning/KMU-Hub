package gdpr

// Coverage for TaskRetentionHandler (Lauf 10, sixth handler on the retention
// engine from A10). The cases that matter: only COMPLETED tasks past the
// cutoff are Due (an open task never is, no matter how old), a completed
// task with a still-active subtask is skipped instead of silently taking
// that subtask down via the parent_task_id cascade, deleting a completed
// task cascades its comments, anonymizing one clears both the task and its
// comments, the anonymize path is idempotent via updated_at, and a second
// tenant's completed task never appears in the first tenant's plan.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTaskRetentionHandler_SupportsAction(t *testing.T) {
	t.Parallel()

	handler := NewTaskRetentionHandler(nil)
	assert.True(t, handler.SupportsAction(models.RetentionActionDelete))
	assert.True(t, handler.SupportsAction(models.RetentionActionAnonymize))
	assert.False(t, handler.SupportsAction("retain"))
	assert.Equal(t, "tasks", handler.ResourceType())
	assert.Equal(t, "tasks", handler.Table())
}

func TestTaskRetentionHandler_PlanOnlyMatchesCompletedPastCutoffAndIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Task Retention Plan Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "task-retention-plan-user")
	defer testutil.CleanupRow(t, pool, "users", userID)

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Task Retention Plan Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", otherTenantID)
	otherUserID := seedExportUser(t, pool, otherTenantID, "task-retention-plan-other-user")
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	fresh := time.Now().UTC().AddDate(0, 0, -2)

	completedOld := seedTask(t, pool, tenantID, userID, &old, nil)
	defer testutil.CleanupRow(t, pool, "tasks", completedOld)

	completedFresh := seedTask(t, pool, tenantID, userID, &fresh, nil)
	defer testutil.CleanupRow(t, pool, "tasks", completedFresh)

	// Open task, old creation date but no completed_at -- must never be Due,
	// completion is the gate, not age.
	openOld := seedTask(t, pool, tenantID, userID, nil, nil)
	defer testutil.CleanupRow(t, pool, "tasks", openOld)
	_, err := pool.Exec(context.Background(), `UPDATE tasks SET created_at = $1, updated_at = $1 WHERE id = $2`, old, openOld)
	require.NoError(t, err)

	// Another tenant's completed-and-old task must never leak into this
	// tenant's plan.
	otherCompletedOld := seedTask(t, pool, otherTenantID, otherUserID, &old, nil)
	defer testutil.CleanupRow(t, pool, "tasks", otherCompletedOld)

	handler := NewTaskRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{completedOld}, plan.Due)
}

func TestTaskRetentionHandler_PlanDeleteSkipsTaskWithActiveSubtask(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Task Retention Subtask Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "task-retention-subtask-user")
	defer testutil.CleanupRow(t, pool, "users", userID)

	old := time.Now().UTC().AddDate(0, 0, -400)

	parent := seedTask(t, pool, tenantID, userID, &old, nil)
	defer testutil.CleanupRow(t, pool, "tasks", parent)
	activeChild := seedTask(t, pool, tenantID, userID, nil, &parent)
	defer testutil.CleanupRow(t, pool, "tasks", activeChild)

	handler := NewTaskRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	deletePlan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.NotContains(t, deletePlan.Due, parent, "must not delete a parent whose subtask is still active")
	require.Len(t, deletePlan.Skipped, 1)
	assert.Equal(t, parent, deletePlan.Skipped[0].RecordID)
	assert.Contains(t, deletePlan.Skipped[0].Reason, "offene Unteraufgabe")

	// Anonymize never deletes rows, so the parent-child cascade risk does not
	// apply -- the parent stays reachable.
	anonymizePlan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionAnonymize)
	require.NoError(t, err)
	assert.Contains(t, anonymizePlan.Due, parent)
	assert.Empty(t, anonymizePlan.Skipped)
}

func TestTaskRetentionHandler_ApplyDeleteCascadesComments(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Task Retention Apply Delete Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "task-retention-apply-del-user")
	defer testutil.CleanupRow(t, pool, "users", userID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	taskID := seedTask(t, pool, tenantID, userID, &old, nil)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	commentID := testutil.SeedRow(t, pool, "task_comments", map[string]any{
		"tenant_id": tenantID,
		"task_id":   taskID,
		"author_id": userID,
		"content":   "Erledigt, danke.",
	})

	handler := NewTaskRetentionHandler(pool)
	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{taskID}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var taskCount, commentCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE id = $1`, taskID).Scan(&taskCount))
	assert.Zero(t, taskCount, "deleted task must be gone")
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM task_comments WHERE id = $1`, commentID).Scan(&commentCount))
	assert.Zero(t, commentCount, "cascade must take the comment trail with it")
}

func TestTaskRetentionHandler_ApplyAnonymizeClearsTaskAndCommentsAndIsIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Task Retention Apply Anonymize Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "task-retention-apply-anon-user")
	defer testutil.CleanupRow(t, pool, "users", userID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	taskID := seedTask(t, pool, tenantID, userID, &old, nil)
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	commentID := testutil.SeedRow(t, pool, "task_comments", map[string]any{
		"tenant_id": tenantID,
		"task_id":   taskID,
		"author_id": userID,
		"content":   "Bitte bis Freitag pruefen, Michael.",
	})

	handler := NewTaskRetentionHandler(pool)
	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{taskID}, models.RetentionActionAnonymize, "Geloeschte Aufgabe 1")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var title, description string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT title, description FROM tasks WHERE id = $1`, taskID,
	).Scan(&title, &description))
	assert.Equal(t, "[Geloeschte Aufgabe 1]", title)
	assert.Equal(t, "[Geloeschte Aufgabe 1]", description)

	var commentBody string
	require.NoError(t, pool.QueryRow(ctx, `SELECT content FROM task_comments WHERE id = $1`, commentID).Scan(&commentBody))
	assert.Equal(t, "[Geloeschte Aufgabe 1]", commentBody)

	cutoff := time.Now().UTC().AddDate(0, 0, -90)
	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionAnonymize)
	require.NoError(t, err)
	assert.NotContains(t, plan.Due, taskID, "anonymize must bump updated_at, taking the row out of the next Plan")
}

func TestTaskRetentionHandler_ApplyUnsupportedAction(t *testing.T) {
	t.Parallel()

	handler := NewTaskRetentionHandler(nil)
	_, err := handler.Apply(context.Background(), uuid.New(), []uuid.UUID{uuid.New()}, "retain", "")
	require.Error(t, err)
}

// seedTask seeds a minimal task. completedAt is nil for an open task, a fixed
// timestamp for a completed one -- task.Service.MoveTask always stamps
// completed_at on the transition into a closed-category status, so a
// hand-seeded completed task without completed_at would not exist in
// production data. parentTaskID is nil for a top-level task.
func seedTask(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID, completedAt *time.Time, parentTaskID *uuid.UUID) uuid.UUID {
	t.Helper()
	cols := map[string]any{
		"tenant_id":   tenantID,
		"task_number": 1,
		"title":       "Testaufgabe " + uuid.New().String(),
		"description": "Freitext mit Namensbezug.",
		"created_by":  userID,
	}
	if completedAt != nil {
		cols["completed_at"] = *completedAt
		cols["updated_at"] = *completedAt
	}
	if parentTaskID != nil {
		cols["parent_task_id"] = *parentTaskID
	}
	return testutil.SeedRow(t, pool, "tasks", cols)
}
