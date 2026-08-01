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
		"tenant_id":   testutil.TenantA,
		"name":        "Test Project " + uuid.New().String()[:6],
		"project_key": "TE" + uuid.New().String()[:6],
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "projects", projID)

	// Seed a task for TenantA.
	taskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   testutil.TenantA,
		"project_id":  projID,
		"title":       "Test Task",
		"created_by":  userID,
		"task_number": 1,
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

// TestListBillable_TenantIsolation exercises ListBillable directly (not just
// row visibility): a foreign tenant must see none of it, the owner must see
// only the completed entry, joined with the right task/project/employee
// names, and a still-running timer (no ended_at/duration_seconds yet) must
// not appear as billable.
func TestListBillable_TenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantMine := uuid.New()
	tenantOther := uuid.New()
	testutil.EnsureTenant(t, pool, tenantMine, "ListBillable Mine")
	testutil.EnsureTenant(t, pool, tenantOther, "ListBillable Other")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantMine,
		"email":         fmt.Sprintf("billable-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Anna",
		"last_name":     "Mueller",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	projID := testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id":   tenantMine,
		"name":        "Website Redesign",
		"project_key": "WR" + uuid.New().String()[:6],
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "projects", projID)

	taskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantMine,
		"project_id":  projID,
		"title":       "Frontend-Entwicklung",
		"created_by":  userID,
		"task_number": 1,
	})
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	now := time.Now().UTC()
	completedID := testutil.SeedRow(t, pool, "time_entries", map[string]any{
		"tenant_id":        tenantMine,
		"task_id":          taskID,
		"user_id":          userID,
		"started_at":       now.Add(-2 * time.Hour),
		"ended_at":         now,
		"duration_seconds": 7200,
		"description":      "React-Komponenten implementiert",
	})
	defer testutil.CleanupRow(t, pool, "time_entries", completedID)

	runningID := testutil.SeedRow(t, pool, "time_entries", map[string]any{
		"tenant_id":  tenantMine,
		"task_id":    taskID,
		"user_id":    userID,
		"started_at": now,
	})
	defer testutil.CleanupRow(t, pool, "time_entries", runningID)

	repo := NewPostgresRepository(pool)

	ctxMine := testutil.WithTenantCtx(context.Background(), tenantMine)
	mine, err := repo.ListBillable(ctxMine, tenantMine)
	if err != nil {
		t.Fatalf("ListBillable(mine): %v", err)
	}
	if len(mine) != 1 {
		t.Fatalf("expected 1 billable entry (running timer must be excluded), got %d", len(mine))
	}
	entry := mine[0]
	if entry.ID != completedID {
		t.Fatalf("expected the completed entry %s, got %s", completedID, entry.ID)
	}
	if entry.ProjectName != "Website Redesign" || entry.TaskTitle != "Frontend-Entwicklung" || entry.UserName != "Anna Mueller" {
		t.Fatalf("unexpected joined names: project=%q task=%q user=%q", entry.ProjectName, entry.TaskTitle, entry.UserName)
	}
	if entry.DurationSeconds == nil || *entry.DurationSeconds != 7200 {
		t.Fatalf("expected duration 7200s, got %v", entry.DurationSeconds)
	}

	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	other, err := repo.ListBillable(ctxOther, tenantOther)
	if err != nil {
		t.Fatalf("ListBillable(other): %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("expected no entries for a foreign tenant, got %d", len(other))
	}
}
