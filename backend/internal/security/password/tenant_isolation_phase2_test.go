package password

// Cross-tenant isolation tests for password_policies, password_history,
// ip_access_rules (Migration 000124).
// All three have tenant_id NOT NULL after backfill + SET NOT NULL.
// password_history has FK on user_id (NOT NULL, ON DELETE CASCADE) → real user row required.

import (
	"context"
	"fmt"
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

	// Seed a user for TenantA — required by password_history.user_id FK (ON DELETE CASCADE, not nullable).
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("pw-hist-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// password_policies — one per tenant.
	policyID := testutil.SeedRow(t, pool, "password_policies", map[string]any{
		"tenant_id": testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "password_policies", policyID)

	// password_history — FK to users(id) NOT NULL, must be a real user row.
	historyID := testutil.SeedRow(t, pool, "password_history", map[string]any{
		"tenant_id":     testutil.TenantA,
		"user_id":       userID,
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "password_history", historyID)

	// ip_access_rules — standalone.
	ipRuleID := testutil.SeedRow(t, pool, "ip_access_rules", map[string]any{
		"tenant_id": testutil.TenantA,
		"ip_cidr":   "10.0.0.0/8",
		"rule_type": "allow",
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
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
