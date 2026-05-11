package activity

// RLS integration tests for the activities table (Migration 000120, tenant_isolation policy).
//
// activities.created_by FK references users(id) NOT NULL.
// activities.activity_type is an enum: {call, meeting, note, email, task}.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestRLS_Activities_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-act-a-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"activity_type": "note",
		"subject":       "RLS Test Note",
		"created_by":    userID,
	})
	defer testutil.CleanupRow(t, pool, "activities", rowID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "activities", rowID, 0)
}

func TestRLS_Activities_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-act-own-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"activity_type": "call",
		"subject":       "RLS Test Call",
		"created_by":    userID,
	})
	defer testutil.CleanupRow(t, pool, "activities", rowID)

	ctxA := testutil.WithTenantCtx(ctx, testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "activities", rowID, 1)
}

func TestRLS_Activities_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-act-sys-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"activity_type": "meeting",
		"subject":       "RLS Test Meeting",
		"created_by":    userID,
	})
	defer testutil.CleanupRow(t, pool, "activities", rowID)

	ctxSys := testutil.WithSystemCtx(ctx)
	testutil.AssertRowCount(t, pool, ctxSys, "activities", rowID, 1)
}

func TestRLS_Activities_CrossTenantUpdateBlocked(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-act-upd-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"activity_type": "email",
		"subject":       "RLS Test Email",
		"created_by":    userID,
	})
	defer testutil.CleanupRow(t, pool, "activities", rowID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertUpdateRowsAffected(t, pool, ctxB,
		"UPDATE activities SET description = 'hacked' WHERE id = $1", 0, rowID)
}
