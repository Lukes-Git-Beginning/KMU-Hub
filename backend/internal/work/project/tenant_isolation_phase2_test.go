package project

// Cross-tenant isolation DB tests for project/task collaboration tables (Migration 000124).
// task_comments, task_activities, task_files, task_dependencies, project_members,
// project_statuses all have tenant_id NOT NULL after backfill.
//
// All child tables require task/project parent rows.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_ProjectTaskCollaboration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA.
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("proj2-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// Seed a project.
	projID := testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id":  testutil.TenantA,
		"name":       "Collab Project " + uuid.New().String()[:6],
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "projects", projID)

	// Seed a task.
	taskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":  testutil.TenantA,
		"project_id": projID,
		"title":      "Collab Task",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	// task_comments.
	commentID := testutil.SeedRow(t, pool, "task_comments", map[string]any{
		"tenant_id":  testutil.TenantA,
		"task_id":    taskID,
		"author_id":  userID,
		"body":       "Test comment",
	})
	defer testutil.CleanupRow(t, pool, "task_comments", commentID)

	// task_activities.
	actID := testutil.SeedRow(t, pool, "task_activities", map[string]any{
		"tenant_id":     testutil.TenantA,
		"task_id":       taskID,
		"actor_id":      userID,
		"activity_type": "created",
	})
	defer testutil.CleanupRow(t, pool, "task_activities", actID)

	// task_files.
	tfID := testutil.SeedRow(t, pool, "task_files", map[string]any{
		"tenant_id":   testutil.TenantA,
		"task_id":     taskID,
		"uploaded_by": userID,
		"filename":    "spec.pdf",
		"storage_key": "tasks/" + uuid.New().String(),
	})
	defer testutil.CleanupRow(t, pool, "task_files", tfID)

	// task_dependencies (source → target).
	targetTaskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":  testutil.TenantA,
		"project_id": projID,
		"title":      "Dependency Task",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", targetTaskID)

	depID := testutil.SeedRow(t, pool, "task_dependencies", map[string]any{
		"tenant_id":      testutil.TenantA,
		"source_task_id": taskID,
		"target_task_id": targetTaskID,
	})
	defer testutil.CleanupRow(t, pool, "task_dependencies", depID)

	// project_members.
	pmID := testutil.SeedRow(t, pool, "project_members", map[string]any{
		"tenant_id":  testutil.TenantA,
		"project_id": projID,
		"user_id":    userID,
	})
	defer testutil.CleanupRow(t, pool, "project_members", pmID)

	// project_statuses.
	psID := testutil.SeedRow(t, pool, "project_statuses", map[string]any{
		"tenant_id":  testutil.TenantA,
		"project_id": projID,
		"name":       "In Progress",
	})
	defer testutil.CleanupRow(t, pool, "project_statuses", psID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"task_comments", "task_comments", commentID},
		{"task_activities", "task_activities", actID},
		{"task_files", "task_files", tfID},
		{"task_dependencies", "task_dependencies", depID},
		{"project_members", "project_members", pmID},
		{"project_statuses", "project_statuses", psID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
