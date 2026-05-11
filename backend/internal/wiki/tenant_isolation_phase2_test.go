package wiki

// Cross-tenant isolation tests for wiki_categories and wiki_articles (Migration 000122).
// Both tables have tenant_id NOT NULL in CREATE TABLE (mig 000076) and RLS was enabled
// in mig 000122. Tests verify TenantA-seed → TenantB-read = 0 rows.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Wiki(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	tests := []struct {
		name  string
		table string
		seed  map[string]any
	}{
		{
			"wiki_categories",
			"wiki_categories",
			map[string]any{
				"tenant_id": testutil.TenantA,
				"name":      "Test Category " + uuid.New().String(),
			},
		},
		{
			"wiki_articles",
			"wiki_articles",
			map[string]any{
				"tenant_id": testutil.TenantA,
				"title":     "Test Article",
				"slug":      "test-article-" + uuid.New().String(),
				"author_id": uuid.New(),
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			id := testutil.SeedRow(t, pool, tc.table, tc.seed)
			defer testutil.CleanupRow(t, pool, tc.table, id)

			ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
			ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

			testutil.AssertRowCount(t, pool, ctxA, tc.table, id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, id, 0)
		})
	}
}
