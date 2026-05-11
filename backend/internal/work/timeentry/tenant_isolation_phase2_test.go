package timeentry

// Cross-tenant isolation DB tests for time_entries (Migration 000122).
// time_entries.tenant_id is NOT NULL and RLS was enabled in mig 000122.
// task_id FK → tasks(id) requires a project + task chain; user_id FK → users.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_TimeEntries(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA.
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("te-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// Seed a project for TenantA (tasks require project_id FK).
	projID := testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id":  testutil.TenantA,
		"name":       "Test Project " + uuid.New().String()[:6],
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "projects", projID)

	// Seed a task for TenantA.
	taskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":  testutil.TenantA,
		"project_id": projID,
		"title":      "Test Task",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	// time_entries.
	now := time.Now().UTC()
	teID := testutil.SeedRow(t, pool, "time_entries", map[string]any{
		"tenant_id":  testutil.TenantA,
		"task_id":    taskID,
		"user_id":    userID,
		"started_at": now,
	})
	defer testutil.CleanupRow(t, pool, "time_entries", teID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	testutil.AssertRowCount(t, pool, ctxA, "time_entries", teID, 1)
	testutil.AssertRowCount(t, pool, ctxB, "time_entries", teID, 0)
}
