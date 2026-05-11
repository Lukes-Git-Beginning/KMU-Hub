package task

// RLS integration tests for the tasks table (Migration 000120, tenant_isolation policy).
//
// tasks.created_by FK references users(id) NOT NULL.
// tasks.task_number NOT NULL — we use a unique value per test via uuid-derived int.
// tasks.project_id is nullable — we omit it to avoid FK chain complexity.
// tasks.priority has a CHECK constraint: urgent|high|medium|low (default 'medium').
// This file coexists with tenant_isolation_test.go (mock-based) — no conflicts.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// taskNumber generates a stable positive int from a UUID to satisfy the NOT NULL constraint.
func taskNumber() int {
	return int(uuid.New().ID())%900000 + 100000
}

func TestRLS_Tasks_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-task-a-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   testutil.TenantA,
		"title":       "RLS Task Alpha",
		"task_number": taskNumber(),
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", rowID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "tasks", rowID, 0)
}

func TestRLS_Tasks_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-task-own-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   testutil.TenantA,
		"title":       "RLS Task Beta",
		"task_number": taskNumber(),
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", rowID)

	ctxA := testutil.WithTenantCtx(ctx, testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "tasks", rowID, 1)
}

func TestRLS_Tasks_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-task-sys-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   testutil.TenantA,
		"title":       "RLS Task Gamma",
		"task_number": taskNumber(),
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", rowID)

	ctxSys := testutil.WithSystemCtx(ctx)
	testutil.AssertRowCount(t, pool, ctxSys, "tasks", rowID, 1)
}

func TestRLS_Tasks_CrossTenantUpdateBlocked(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-task-upd-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   testutil.TenantA,
		"title":       "RLS Task Delta",
		"task_number": taskNumber(),
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", rowID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertUpdateRowsAffected(t, pool, ctxB,
		"UPDATE tasks SET description = 'hacked' WHERE id = $1", 0, rowID)
}
