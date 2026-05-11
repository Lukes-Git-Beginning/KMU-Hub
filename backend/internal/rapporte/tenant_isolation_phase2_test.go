package rapporte

// Cross-tenant isolation tests for rapporte tables (Migration 000122).
// work_reports, report_lines, report_attachments all have tenant_id NOT NULL
// and RLS was enabled in mig 000122.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Rapporte(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	reportID := testutil.SeedRow(t, pool, "work_reports", map[string]any{
		"tenant_id": testutil.TenantA,
		"title":     "Daily Report " + uuid.New().String()[:8],
		"author_id": uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "work_reports", reportID)

	lineID := testutil.SeedRow(t, pool, "report_lines", map[string]any{
		"tenant_id":   testutil.TenantA,
		"report_id":   reportID,
		"description": "Service work",
	})
	defer testutil.CleanupRow(t, pool, "report_lines", lineID)

	attachID := testutil.SeedRow(t, pool, "report_attachments", map[string]any{
		"tenant_id":   testutil.TenantA,
		"report_id":   reportID,
		"filename":    "photo.jpg",
		"object_key":  "rapporte/" + uuid.New().String(),
		"uploaded_by": uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "report_attachments", attachID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"work_reports", "work_reports", reportID},
		{"report_lines", "report_lines", lineID},
		{"report_attachments", "report_attachments", attachID},
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
