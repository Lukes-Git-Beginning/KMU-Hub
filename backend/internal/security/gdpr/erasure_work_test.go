package gdpr

// Integration coverage for WorkErasureHandler.ExecuteErasure — only
// PreviewErasure had a test before (handlers_test.go). This pins the fix for
// the double-count bug found alongside fix-crm-erasure-double-count (Lauf 9,
// Iteration 6): a task both assigned to and created by the subject must be
// counted exactly once, not twice. See JOURNAL Iteration 25.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedProjectMember inserts into project_members, which has a composite
// primary key (project_id, user_id) and no id column — testutil.SeedRow
// cannot be used because it relies on RETURNING id.
func seedProjectMember(t *testing.T, pool *pgxpool.Pool, tenantID, projectID, userID uuid.UUID, role string) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(ctx,
		`INSERT INTO project_members (project_id, user_id, role, tenant_id)
		 VALUES ($1, $2, $3, $4)`,
		projectID, userID, role, tenantID,
	)
	if err != nil {
		t.Fatalf("seed project_members: %v", err)
	}
}

func TestWorkErasureHandler_ExecuteErasure_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Work Erasure Execute Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userID := seedExportUser(t, pool, tenantOwn, "work-erasure-exec")
	defer testutil.CleanupRow(t, pool, "users", userID)
	colleagueID := seedExportUser(t, pool, tenantOwn, "work-erasure-colleague")
	defer testutil.CleanupRow(t, pool, "users", colleagueID)

	// Assigned to and created by the subject — must be counted exactly once,
	// not once by the UPDATE's RowsAffected and once more by a separate count.
	bothTaskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"task_number": 1,
		"title":       "Angebot nachfassen",
		"assignee_id": userID,
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", bothTaskID)

	// Created by the subject but assigned to somebody else — must survive with
	// its assignment untouched, and still be counted once.
	createdOnlyTaskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"task_number": 2,
		"title":       "Uebergabe",
		"assignee_id": colleagueID,
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", createdOnlyTaskID)

	// Assigned to the subject but created by somebody else — assignment must
	// be cleared, created_by is untouched.
	assignedOnlyTaskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"task_number": 3,
		"title":       "Review",
		"assignee_id": userID,
		"created_by":  colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", assignedOnlyTaskID)

	// Neither created nor assigned by the subject — must survive untouched.
	foreignTaskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"task_number": 4,
		"title":       "Fremde Aufgabe",
		"assignee_id": colleagueID,
		"created_by":  colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", foreignTaskID)

	// Older than the two-year ArbZG retention window (relative to "now" at
	// authoring time, 2026-08-22) -- must be hard-deleted.
	oldTimeEntryID := testutil.SeedRow(t, pool, "time_entries", map[string]any{
		"tenant_id":  tenantOwn,
		"task_id":    bothTaskID,
		"user_id":    userID,
		"started_at": "2020-01-01T09:00:00Z",
		"ended_at":   "2020-01-01T10:00:00Z",
	})

	// Inside the retention window -- must survive; the retention duty does not
	// lapse just because erasure was requested.
	recentTimeEntryID := testutil.SeedRow(t, pool, "time_entries", map[string]any{
		"tenant_id":  tenantOwn,
		"task_id":    bothTaskID,
		"user_id":    userID,
		"started_at": "2026-08-11T09:00:00Z",
		"ended_at":   "2026-08-11T10:00:00Z",
	})
	defer testutil.CleanupRow(t, pool, "time_entries", recentTimeEntryID)

	commentID := testutil.SeedRow(t, pool, "task_comments", map[string]any{
		"tenant_id": tenantOwn,
		"task_id":   bothTaskID,
		"author_id": userID,
		"content":   "Persoenliche Notiz",
	})
	defer testutil.CleanupRow(t, pool, "task_comments", commentID)

	// project_members.user_id is CASCADE on users(id), but AuthErasureHandler
	// anonymizes the user row instead of deleting it, so the CASCADE never
	// fires — the membership must be deleted explicitly.
	projectID := testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id":   tenantOwn,
		"name":        "Projekt Loeschtest",
		"project_key": fmt.Sprintf("PL%d", uuid.New().ID()%1000),
		"created_by":  colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "projects", projectID)
	seedProjectMember(t, pool, tenantOwn, projectID, userID, "member")
	seedProjectMember(t, pool, tenantOwn, projectID, colleagueID, "owner")

	h := NewWorkErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	affected, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureAnonymize)
	require.NoError(t, err)
	// 2 tasks unassigned (both-task and assigned-only; the created-only task keeps its
	// assignee and is therefore not changed at all, and the foreign task is untouched)
	// + 1 time entry (the one past the retention window) deleted + 1 comment anonymized
	// + 1 project membership deleted = 5.
	// recentTimeEntryID is inside the window and must not be counted.
	assert.Equal(t, 5, affected, "only rows this run actually changed are counted")

	// The both-task loses its assignment.
	var assignee *uuid.UUID
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT assignee_id FROM tasks WHERE id = $1`, bothTaskID,
	).Scan(&assignee))
	assert.Nil(t, assignee, "assignee_id must be cleared for a task assigned to the subject")

	// The created-only task keeps its assignee.
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT assignee_id FROM tasks WHERE id = $1`, createdOnlyTaskID,
	).Scan(&assignee))
	require.NotNil(t, assignee, "a task assigned to somebody else must keep its assignee")
	assert.Equal(t, colleagueID, *assignee)

	// The assigned-only task also loses its assignment.
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT assignee_id FROM tasks WHERE id = $1`, assignedOnlyTaskID,
	).Scan(&assignee))
	assert.Nil(t, assignee, "assignee_id must be cleared even when the subject did not create the task")

	// The foreign task is not touched at all.
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT assignee_id FROM tasks WHERE id = $1`, foreignTaskID,
	).Scan(&assignee))
	require.NotNil(t, assignee)
	assert.Equal(t, colleagueID, *assignee)

	// Time entry past the ArbZG retention window is deleted outright.
	var oldTimeEntryRows int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM time_entries WHERE id = $1`, oldTimeEntryID).Scan(&oldTimeEntryRows))
	assert.Equal(t, 0, oldTimeEntryRows, "time entries past the retention window must be deleted")

	// Time entry still inside the retention window survives the erasure run.
	var recentTimeEntryRows int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM time_entries WHERE id = $1`, recentTimeEntryID).Scan(&recentTimeEntryRows))
	assert.Equal(t, 1, recentTimeEntryRows, "time entries inside the retention window must survive erasure")

	// Task comments are anonymized, not deleted.
	var commentContent string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT content FROM task_comments WHERE id = $1`, commentID).Scan(&commentContent))
	assert.Equal(t, "["+erasureLabel+"]", commentContent, "comment content must be replaced by the bracketed anonymized label")

	// Only the subject's project membership is gone.
	var ownMemberships, colleagueMemberships int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM project_members WHERE user_id = $1`, userID).Scan(&ownMemberships))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM project_members WHERE user_id = $1`, colleagueID).Scan(&colleagueMemberships))
	assert.Equal(t, 0, ownMemberships, "the subject's project membership must be deleted")
	assert.Equal(t, 1, colleagueMemberships, "another project member must keep their membership")
}

func TestWorkErasureHandler_DeadPool(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	pool.Close()

	h := NewWorkErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	userID := uuid.New()

	affected, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureAnonymize)
	require.Error(t, err, "a dead pool must surface as an error, not as a silent no-op erasure")
	assert.Equal(t, 0, affected)
}
