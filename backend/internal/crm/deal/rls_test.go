package deal

// RLS integration tests for deals + deal_stage_history (Migration 000120, tenant_isolation policy).
//
// deals.stage_id FK references pipeline_stages(id) — which also has RLS.
// We seed pipeline_stages under system context (bypasses RLS) before seeding deals.
// deal_stage_history.changed_by is nullable, so no extra user row needed for history entries.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestRLS_Deals_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-d-a-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"name":      fmt.Sprintf("Stage-D-A-%s", uuid.New()),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)

	rowID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Deal Alpha",
		"stage_id":   stageID,
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "deals", rowID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "deals", rowID, 0)
}

func TestRLS_Deals_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-d-own-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"name":      fmt.Sprintf("Stage-D-Own-%s", uuid.New()),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)

	rowID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Deal Beta",
		"stage_id":   stageID,
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "deals", rowID)

	ctxA := testutil.WithTenantCtx(ctx, testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "deals", rowID, 1)
}

func TestRLS_Deals_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-d-sys-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"name":      fmt.Sprintf("Stage-D-Sys-%s", uuid.New()),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)

	rowID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Deal Gamma",
		"stage_id":   stageID,
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "deals", rowID)

	ctxSys := testutil.WithSystemCtx(ctx)
	testutil.AssertRowCount(t, pool, ctxSys, "deals", rowID, 1)
}

func TestRLS_Deals_CrossTenantUpdateBlocked(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-d-upd-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"name":      fmt.Sprintf("Stage-D-Upd-%s", uuid.New()),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)

	rowID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "Deal Delta",
		"stage_id":   stageID,
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "deals", rowID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertUpdateRowsAffected(t, pool, ctxB,
		"UPDATE deals SET notes = 'hacked' WHERE id = $1", 0, rowID)
}

// --- deal_stage_history RLS tests ---
// deal_stage_history has a FK to deals(id) ON DELETE CASCADE.
// changed_by is nullable so we do not need an extra user row for history entries.

func TestRLS_DealStageHistory_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-dsh-a-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"name":      fmt.Sprintf("Stage-DSH-A-%s", uuid.New()),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)

	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "DSH Parent Deal A",
		"stage_id":   stageID,
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "deals", dealID)

	histID := testutil.SeedRow(t, pool, "deal_stage_history", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"deal_id":   dealID,
	})
	defer testutil.CleanupRow(t, pool, "deal_stage_history", histID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "deal_stage_history", histID, 0)
}

func TestRLS_DealStageHistory_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-dsh-own-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"name":      fmt.Sprintf("Stage-DSH-Own-%s", uuid.New()),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)

	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "DSH Parent Deal Own",
		"stage_id":   stageID,
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "deals", dealID)

	histID := testutil.SeedRow(t, pool, "deal_stage_history", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"deal_id":   dealID,
	})
	defer testutil.CleanupRow(t, pool, "deal_stage_history", histID)

	ctxA := testutil.WithTenantCtx(ctx, testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "deal_stage_history", histID, 1)
}

func TestRLS_DealStageHistory_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-dsh-sys-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"name":      fmt.Sprintf("Stage-DSH-Sys-%s", uuid.New()),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)

	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"name":       "DSH Parent Deal Sys",
		"stage_id":   stageID,
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "deals", dealID)

	histID := testutil.SeedRow(t, pool, "deal_stage_history", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"deal_id":   dealID,
	})
	defer testutil.CleanupRow(t, pool, "deal_stage_history", histID)

	ctxSys := testutil.WithSystemCtx(ctx)
	testutil.AssertRowCount(t, pool, ctxSys, "deal_stage_history", histID, 1)
}
