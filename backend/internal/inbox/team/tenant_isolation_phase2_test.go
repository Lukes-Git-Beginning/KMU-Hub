package team

// Cross-tenant isolation tests for inbox tables (Migration 000124).
// team_inboxes, routing_rules, inbox_messages all have tenant_id NOT NULL.
// team_inbox_members is a composite-PK junction table — covered by team_inboxes isolation.
//
// team_inboxes.created_by, routing_rules.created_by, inbox_messages.user_id
// all FK to users(id) — we seed a minimal user first.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Inbox(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA.
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("inbox-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// team_inboxes.
	inboxID := testutil.SeedRow(t, pool, "team_inboxes", map[string]any{
		"tenant_id":  testutil.TenantA,
		"name":       "Support " + uuid.New().String()[:6],
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "team_inboxes", inboxID)

	// routing_rules.
	ruleID := testutil.SeedRow(t, pool, "routing_rules", map[string]any{
		"tenant_id":  testutil.TenantA,
		"name":       "Auto-route " + uuid.New().String()[:6],
		"conditions": `{"field":"subject","op":"contains","value":"urgent"}`,
		"actions":    `{"assign_to_queue":"support"}`,
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "routing_rules", ruleID)

	// inbox_messages (channel + source_id + received_at are NOT NULL).
	msgID := testutil.SeedRow(t, pool, "inbox_messages", map[string]any{
		"tenant_id":   testutil.TenantA,
		"user_id":     userID,
		"channel":     "email",
		"source_id":   uuid.New().String(),
		"received_at": "2026-05-11T10:00:00Z",
	})
	defer testutil.CleanupRow(t, pool, "inbox_messages", msgID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"team_inboxes", "team_inboxes", inboxID},
		{"routing_rules", "routing_rules", ruleID},
		{"inbox_messages", "inbox_messages", msgID},
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
