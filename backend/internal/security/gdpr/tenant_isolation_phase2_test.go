package gdpr

// Cross-tenant isolation tests for GDPR tables (Migration 000122 + 000124).
// gdpr_deletion_requests — NOT NULL from mig 000114, RLS in mig 000122. contact_id FK is
// nullable (ON DELETE SET NULL, mig 000322); original_contact_id (no FK) is NOT NULL instead.
// gdpr_export_requests   — NOT NULL after backfill in mig 000124, RLS in mig 000124.
// gdpr_erasure_log       — NOT NULL after backfill in mig 000124, RLS in mig 000124.
//
// gdpr_export_requests.user_id FK to users(id).
// gdpr_erasure_log.executed_by FK to users(id).
// We seed a minimal user for TenantA.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_GDPR(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA.
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("gdpr-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// Seed a contact for TenantA.
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  testutil.TenantA,
		"first_name": "GDPR",
		"last_name":  "Test",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	// gdpr_deletion_requests — original_contact_id NOT NULL (no FK), contact_id nullable,
	// requested_by nullable FK to users.
	delReqID := testutil.SeedRow(t, pool, "gdpr_deletion_requests", map[string]any{
		"tenant_id":           testutil.TenantA,
		"contact_id":          contactID,
		"original_contact_id": contactID,
		"requested_by":        userID,
	})
	defer testutil.CleanupRow(t, pool, "gdpr_deletion_requests", delReqID)

	// gdpr_export_requests — user_id FK to users.
	expReqID := testutil.SeedRow(t, pool, "gdpr_export_requests", map[string]any{
		"tenant_id": testutil.TenantA,
		"user_id":   userID,
	})
	defer testutil.CleanupRow(t, pool, "gdpr_export_requests", expReqID)

	// gdpr_erasure_log — executed_by FK to users; original_user_id, anonymized_label,
	// modules_affected, confirmation_hash are NOT NULL.
	erasureID := testutil.SeedRow(t, pool, "gdpr_erasure_log", map[string]any{
		"tenant_id":         testutil.TenantA,
		"original_user_id":  uuid.New(),
		"anonymized_label":  "user_" + uuid.New().String()[:8],
		"executed_by":       userID,
		"modules_affected":  `["crm","email"]`,
		"confirmation_hash": uuid.New().String(),
	})
	defer testutil.CleanupRow(t, pool, "gdpr_erasure_log", erasureID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"gdpr_deletion_requests", "gdpr_deletion_requests", delReqID},
		{"gdpr_export_requests", "gdpr_export_requests", expReqID},
		{"gdpr_erasure_log", "gdpr_erasure_log", erasureID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
