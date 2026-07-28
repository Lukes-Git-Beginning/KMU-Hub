package berichte

// Cross-tenant isolation tests for berichte tables (Migration 000122).
// report_definitions, report_cache, report_schedules, report_runs all have tenant_id NOT NULL
// and RLS was enabled in mig 000122.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Berichte(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a report_definition for tenant A — basis for report_cache and report_schedules.
	defID := testutil.SeedRow(t, pool, "report_definitions", map[string]any{
		"tenant_id": testutil.TenantA,
		"name":      "Revenue Report " + uuid.New().String(),
		"module":    "finanzen",
		"kind":      "custom",
	})
	defer testutil.CleanupRow(t, pool, "report_definitions", defID)

	// Seed a report_cache entry (requires definition_id FK).
	cacheID := testutil.SeedRow(t, pool, "report_cache", map[string]any{
		"tenant_id":     testutil.TenantA,
		"definition_id": defID,
		"params_hash":   "abc123",
		"result":        `{"rows":[]}`,
		"expires_at":    time.Now().Add(24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "report_cache", cacheID)

	// Seed a report_schedule.
	schedID := testutil.SeedRow(t, pool, "report_schedules", map[string]any{
		"tenant_id":       testutil.TenantA,
		"definition_id":   defID,
		"name":            "Monthly Schedule",
		"cron_expression": "0 8 1 * *",
	})
	defer testutil.CleanupRow(t, pool, "report_schedules", schedID)

	// Seed a report_run.
	runID := testutil.SeedRow(t, pool, "report_runs", map[string]any{
		"tenant_id":     testutil.TenantA,
		"definition_id": defID,
		"trigger":       "manual",
		"status":        "success",
	})
	defer testutil.CleanupRow(t, pool, "report_runs", runID)

	// Seed a report_document (Migration 000245). Columns beyond tenant_id/title
	// carry defaults, so the block tree stays out of the seed.
	docID := testutil.SeedRow(t, pool, "report_documents", map[string]any{
		"tenant_id": testutil.TenantA,
		"title":     "Quartalsbericht " + uuid.New().String(),
		"module":    "finanzen",
	})
	defer testutil.CleanupRow(t, pool, "report_documents", docID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"report_definitions", "report_definitions", defID},
		{"report_cache", "report_cache", cacheID},
		{"report_schedules", "report_schedules", schedID},
		{"report_runs", "report_runs", runID},
		{"report_documents", "report_documents", docID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
