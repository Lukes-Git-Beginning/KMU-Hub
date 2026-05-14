package helpdesk

// Welle 4.2 Phase 3 — cross-tenant isolation for ticket_messages after
// migration 000126 adds tenant_id + RLS.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_TicketMessages_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	ticketA := testutil.SeedRow(t, pool, "tickets", map[string]any{
		"tenant_id":    testutil.TenantA,
		"subject":      "Ticket A",
		"requester_id": uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "tickets", ticketA)

	ticketB := testutil.SeedRow(t, pool, "tickets", map[string]any{
		"tenant_id":    testutil.TenantB,
		"subject":      "Ticket B",
		"requester_id": uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "tickets", ticketB)

	msgA := testutil.SeedRow(t, pool, "ticket_messages", map[string]any{
		"tenant_id": testutil.TenantA,
		"ticket_id": ticketA,
		"author_id": uuid.New(),
		"body":      "hello A",
	})
	defer testutil.CleanupRow(t, pool, "ticket_messages", msgA)

	msgB := testutil.SeedRow(t, pool, "ticket_messages", map[string]any{
		"tenant_id": testutil.TenantB,
		"ticket_id": ticketB,
		"author_id": uuid.New(),
		"body":      "hello B",
	})
	defer testutil.CleanupRow(t, pool, "ticket_messages", msgB)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	t.Run("ticket_messages", func(t *testing.T) {
		t.Parallel()
		testutil.AssertRowCount(t, pool, ctxA, "ticket_messages", msgA, 1)
		testutil.AssertRowCount(t, pool, ctxA, "ticket_messages", msgB, 0)
		testutil.AssertRowCount(t, pool, ctxB, "ticket_messages", msgB, 1)
		testutil.AssertRowCount(t, pool, ctxB, "ticket_messages", msgA, 0)
	})
}
