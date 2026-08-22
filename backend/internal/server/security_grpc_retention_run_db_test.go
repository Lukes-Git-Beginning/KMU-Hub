package server

// DB-backed coverage for GetLatestRetentionRun, the admin-visibility half of
// A14 (retention_scheduler.go writes retention_runs / retention_run_items;
// this RPC only reads them). Same reason ListRetentionPolicies/ListIPRules
// need their own file: the handler goes straight to s.pool with no service
// to stub, and the fields under test (skip_reasons JSONB, nullable
// finished_at/error) only reveal scan bugs against a real row.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
)

func TestSecurityGRPCServer_GetLatestRetentionRun_NoRunYet(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Retention Run Empty Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	srv := NewSecurityGRPCServer(nil, nil, nil, nil, nil, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	resp, err := srv.GetLatestRetentionRun(ctx, &securityv1.GetLatestRetentionRunRequest{})
	require.NoError(t, err)
	require.False(t, resp.HasRun, "a tenant that never ran retention must not report a fabricated run")
	require.Nil(t, resp.Run)
	require.Empty(t, resp.Items)
}

func TestSecurityGRPCServer_GetLatestRetentionRun_ReturnsRunAndItems(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Retention Run Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	skippedID := uuid.New()
	runID := testutil.SeedRow(t, pool, "retention_runs", map[string]any{
		"tenant_id":        tenantID,
		"mode":             "dry_run",
		"status":           "completed",
		"triggered_by":     "schedule",
		"policies_total":   1,
		"records_matched":  3,
		"records_affected": 0,
		"records_skipped":  1,
	})
	defer testutil.CleanupRow(t, pool, "retention_runs", runID)

	itemID := testutil.SeedRow(t, pool, "retention_run_items", map[string]any{
		"tenant_id":      tenantID,
		"run_id":         runID,
		"resource_type":  "contacts",
		"action":         models.RetentionActionDelete,
		"retention_days": 365,
		"cutoff":         time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		"status":         "dry_run",
		"matched":        3,
		"affected":       0,
		"skipped":        1,
		"skip_reasons":   `[{"record_id":"` + skippedID.String() + `","reason":"blocked by advisory_protocols"}]`,
	})
	_ = itemID // cascades away with runID, no separate cleanup needed

	srv := NewSecurityGRPCServer(nil, nil, nil, nil, nil, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	resp, err := srv.GetLatestRetentionRun(ctx, &securityv1.GetLatestRetentionRunRequest{})
	require.NoError(t, err)
	require.True(t, resp.HasRun)
	require.NotNil(t, resp.Run)
	require.Equal(t, runID.String(), resp.Run.Id)
	require.Equal(t, "dry_run", resp.Run.Mode)
	require.Equal(t, "schedule", resp.Run.TriggeredBy)
	require.Equal(t, int32(3), resp.Run.RecordsMatched)

	require.Len(t, resp.Items, 1)
	item := resp.Items[0]
	require.Equal(t, "contacts", item.ResourceType)
	require.Equal(t, int32(3), item.Matched)
	require.Len(t, item.SkipReasons, 1)
	require.Equal(t, skippedID.String(), item.SkipReasons[0].RecordId)
	require.Equal(t, "blocked by advisory_protocols", item.SkipReasons[0].Reason)
}

// TestSecurityGRPCServer_GetLatestRetentionRun_TenantIsolation proves a
// foreign tenant's run never leaks through the ORDER BY ... LIMIT 1 query --
// there is no explicit WHERE tenant_id here, RLS carries the whole guarantee.
func TestSecurityGRPCServer_GetLatestRetentionRun_TenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Retention Run Isolation Own")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	tenantForeign := uuid.New()
	testutil.EnsureTenant(t, pool, tenantForeign, "Retention Run Isolation Foreign")
	defer testutil.CleanupRow(t, pool, "tenants", tenantForeign)

	foreignRunID := testutil.SeedRow(t, pool, "retention_runs", map[string]any{
		"tenant_id":    tenantForeign,
		"mode":         "dry_run",
		"status":       "completed",
		"triggered_by": "schedule",
	})
	defer testutil.CleanupRow(t, pool, "retention_runs", foreignRunID)

	srv := NewSecurityGRPCServer(nil, nil, nil, nil, nil, pool)
	ownCtx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	resp, err := srv.GetLatestRetentionRun(ownCtx, &securityv1.GetLatestRetentionRunRequest{})
	require.NoError(t, err)
	require.False(t, resp.HasRun, "a foreign tenant's run must not surface as this tenant's latest")
}
