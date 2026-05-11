package password

// Cross-tenant isolation tests for password_policies, password_history,
// ip_access_rules (Migration 000124).
// All three have tenant_id NOT NULL after backfill + SET NOT NULL.
// password_history has FK on user_id; we use uuid.New() for the FK since
// SeedRow uses system-context and bypasses FK checks at RLS level.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_PasswordAndIPRules(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// password_policies — one per tenant.
	policyID := testutil.SeedRow(t, pool, "password_policies", map[string]any{
		"tenant_id": testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "password_policies", policyID)

	// password_history — FK to users; system-context seed bypasses FK enforcement.
	historyID := testutil.SeedRow(t, pool, "password_history", map[string]any{
		"tenant_id":     testutil.TenantA,
		"user_id":       uuid.New(),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "password_history", historyID)

	// ip_access_rules — standalone.
	ipRuleID := testutil.SeedRow(t, pool, "ip_access_rules", map[string]any{
		"tenant_id": testutil.TenantA,
		"ip_cidr":   "10.0.0.0/8",
	})
	defer testutil.CleanupRow(t, pool, "ip_access_rules", ipRuleID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"password_policies", "password_policies", policyID},
		{"password_history", "password_history", historyID},
		{"ip_access_rules", "ip_access_rules", ipRuleID},
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
