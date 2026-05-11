package savedfilter

// Cross-tenant isolation tests for saved_filters (Migration 000122).
// saved_filters.tenant_id is NOT NULL and RLS was enabled in mig 000122.
// created_by FK to users(id) — we seed a minimal user first.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_SavedFilters(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA.
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("sf-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	sfID := testutil.SeedRow(t, pool, "saved_filters", map[string]any{
		"tenant_id":   testutil.TenantA,
		"name":        "Active Contacts",
		"entity_type": "contact",
		"filter_json": `{"status":"active"}`,
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "saved_filters", sfID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	testutil.AssertRowCount(t, pool, ctxA, "saved_filters", sfID, 1)
	testutil.AssertRowCount(t, pool, ctxB, "saved_filters", sfID, 0)
}
