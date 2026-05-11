package tag

// Cross-tenant isolation tests for CRM tag tables (Migration 000124).
//
// tags — tenant_id NOT NULL after backfill + SET NOT NULL in mig 000124.
// contact_tags, company_tags, deal_tags, activity_tags — junction tables with
// composite PKs (no id column). These cannot use testutil.SeedRow/AssertRowCount
// directly; instead we verify that the parent `tags` row is isolated, which is
// the table with the tenant-scoped RLS policy.
//
// Junction tables are tested by asserting that a tag seeded for tenant A
// is not visible to tenant B (via the tags table RLS).

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Tags(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a tag for tenant A.
	tagID := testutil.SeedRow(t, pool, "tags", map[string]any{
		"tenant_id":   testutil.TenantA,
		"name":        "VIP-" + uuid.New().String()[:6],
		"entity_type": "contact",
		"color":       "#3d8abf",
	})
	defer testutil.CleanupRow(t, pool, "tags", tagID)

	// Seed a second tag for tenant B to confirm symmetry.
	tagB := testutil.SeedRow(t, pool, "tags", map[string]any{
		"tenant_id":   testutil.TenantB,
		"name":        "PRIO-" + uuid.New().String()[:6],
		"entity_type": "contact",
		"color":       "#bf3d3d",
	})
	defer testutil.CleanupRow(t, pool, "tags", tagB)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	t.Run("tags_tenant_a_visible_to_a", func(t *testing.T) {
		t.Parallel()
		testutil.AssertRowCount(t, pool, ctxA, "tags", tagID, 1)
	})
	t.Run("tags_tenant_a_invisible_to_b", func(t *testing.T) {
		t.Parallel()
		testutil.AssertRowCount(t, pool, ctxB, "tags", tagID, 0)
	})
	t.Run("tags_tenant_b_visible_to_b", func(t *testing.T) {
		t.Parallel()
		testutil.AssertRowCount(t, pool, ctxB, "tags", tagB, 1)
	})
	t.Run("tags_tenant_b_invisible_to_a", func(t *testing.T) {
		t.Parallel()
		testutil.AssertRowCount(t, pool, ctxA, "tags", tagB, 0)
	})
}
