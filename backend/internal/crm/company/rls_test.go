package company

// RLS integration tests for the companies table (Migration 000120, tenant_isolation policy).

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestRLS_Companies_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-co-a-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "companies", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Acme RLS-Test GmbH",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "companies", rowID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "companies", rowID, 0)
}

func TestRLS_Companies_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-co-own-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "companies", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Beta Corp RLS-Test",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "companies", rowID)

	ctxA := testutil.WithTenantCtx(ctx, testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "companies", rowID, 1)
}

func TestRLS_Companies_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-co-sys-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "companies", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Gamma Systems RLS-Test",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "companies", rowID)

	ctxSys := testutil.WithSystemCtx(ctx)
	testutil.AssertRowCount(t, pool, ctxSys, "companies", rowID, 1)
}

func TestRLS_Companies_CrossTenantUpdateBlocked(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-co-upd-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "companies", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Delta RLS-Test AG",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "companies", rowID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertUpdateRowsAffected(t, pool, ctxB,
		"UPDATE companies SET notes = 'hacked' WHERE id = $1", 0, rowID)
}
