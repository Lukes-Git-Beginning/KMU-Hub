package customfield

// Cross-tenant isolation tests for custom_field_definitions (Migration 000122).
// custom_field_definitions.tenant_id is NOT NULL and RLS was enabled in mig 000122.
// created_by FK to users(id) — we seed a minimal user first.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_CustomFieldDefinitions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA (required by created_by FK).
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("cfd-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// custom_field_definitions (entity_type + field_name UNIQUE constraint).
	cfID := testutil.SeedRow(t, pool, "custom_field_definitions", map[string]any{
		"tenant_id":   testutil.TenantA,
		"entity_type": "contact",
		"field_name":  "custom_score_" + uuid.New().String()[:8],
		"field_label": "Custom Score",
		"field_type":  "number",
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "custom_field_definitions", cfID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	testutil.AssertRowCount(t, pool, ctxA, "custom_field_definitions", cfID, 1)
	testutil.AssertRowCount(t, pool, ctxB, "custom_field_definitions", cfID, 0)
}
