package workflow

// Cross-tenant isolation DB tests for automations and automation_executions (Migration 000122).
// Both tables have tenant_id NOT NULL and RLS was enabled in mig 000122.
// automations.owner_id FK → users(id).

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Automations(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA.
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("auto-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// automations.
	autoID := testutil.SeedRow(t, pool, "automations", map[string]any{
		"tenant_id":    testutil.TenantA,
		"name":         "New Contact Alert",
		"owner_id":     userID,
		"trigger_type": "contact.created",
	})
	defer testutil.CleanupRow(t, pool, "automations", autoID)

	// automation_executions.
	execID := testutil.SeedRow(t, pool, "automation_executions", map[string]any{
		"tenant_id":       testutil.TenantA,
		"automation_id":   autoID,
		"chain_id":        uuid.New(),
		"trigger_event":   `{"type":"contact.created"}`,
		"condition_result": true,
	})
	defer testutil.CleanupRow(t, pool, "automation_executions", execID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"automations", "automations", autoID},
		{"automation_executions", "automation_executions", execID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
