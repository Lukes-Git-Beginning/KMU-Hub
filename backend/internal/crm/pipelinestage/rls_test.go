package pipelinestage

// RLS integration tests for the pipeline_stages table (Migration 000120, tenant_isolation policy).
//
// pipeline_stages has no created_by column — only name, color (default), sort_order (default),
// is_won (default), is_lost (default), probability (default) + tenant_id are required.
// Note: unique index on (is_won) WHERE is_won=true and (is_lost) WHERE is_lost=true —
// we keep is_won and is_lost both false (the defaults) to avoid conflicts.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestRLS_PipelineStages_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	rowID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"name":      fmt.Sprintf("Stage-RLS-A-%s", uuid.New()),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", rowID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "pipeline_stages", rowID, 0)
}

func TestRLS_PipelineStages_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	rowID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"name":      fmt.Sprintf("Stage-RLS-Own-%s", uuid.New()),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", rowID)

	ctxA := testutil.WithTenantCtx(ctx, testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "pipeline_stages", rowID, 1)
}

func TestRLS_PipelineStages_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	rowID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"name":      fmt.Sprintf("Stage-RLS-Sys-%s", uuid.New()),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", rowID)

	ctxSys := testutil.WithSystemCtx(ctx)
	testutil.AssertRowCount(t, pool, ctxSys, "pipeline_stages", rowID, 1)
}

func TestRLS_PipelineStages_CrossTenantUpdateBlocked(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	rowID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"name":      fmt.Sprintf("Stage-RLS-Upd-%s", uuid.New()),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", rowID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertUpdateRowsAffected(t, pool, ctxB,
		"UPDATE pipeline_stages SET name = 'hacked' WHERE id = $1", 0, rowID)
}
