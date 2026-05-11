package helpdesk

// Cross-tenant isolation tests for helpdesk tables (Migration 000122).
// sla_policies, ticket_queues, tickets, canned_responses all have tenant_id NOT NULL
// and RLS was enabled in mig 000122.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Helpdesk(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed an sla_policy for tenant A first — tickets and queues may reference it.
	slaPolicyID := testutil.SeedRow(t, pool, "sla_policies", map[string]any{
		"tenant_id":            testutil.TenantA,
		"name":                 "Standard SLA " + uuid.New().String(),
		"first_response_mins":  60,
		"resolution_mins":      480,
	})
	defer testutil.CleanupRow(t, pool, "sla_policies", slaPolicyID)

	// Seed a ticket_queue for tenant A.
	queueID := testutil.SeedRow(t, pool, "ticket_queues", map[string]any{
		"tenant_id": testutil.TenantA,
		"name":      "Support Queue " + uuid.New().String(),
	})
	defer testutil.CleanupRow(t, pool, "ticket_queues", queueID)

	// Seed a ticket for tenant A — requester_id must be a UUID (no FK to users in RLS test).
	ticketID := testutil.SeedRow(t, pool, "tickets", map[string]any{
		"tenant_id":    testutil.TenantA,
		"subject":      "Test Ticket",
		"requester_id": uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "tickets", ticketID)

	// Seed a canned_response for tenant A.
	cannedID := testutil.SeedRow(t, pool, "canned_responses", map[string]any{
		"tenant_id": testutil.TenantA,
		"name":      "Standard Reply " + uuid.New().String(),
		"body":      "Thank you for contacting us.",
	})
	defer testutil.CleanupRow(t, pool, "canned_responses", cannedID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"sla_policies", "sla_policies", slaPolicyID},
		{"ticket_queues", "ticket_queues", queueID},
		{"tickets", "tickets", ticketID},
		{"canned_responses", "canned_responses", cannedID},
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
