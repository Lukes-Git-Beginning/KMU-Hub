package helpdesk

// The "own" data scope has to narrow the list in SQL, not in a post-filter over
// an already-paged response: a caller who may see two of three tickets must be
// told "2", or the pager offers them a second page of rows that do not exist.
// These tests pin both halves — the rows and the count — against a real
// database, plus the combination with the status filter, whose placeholder
// numbering now depends on how many filters came before it.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestListTickets_OwnScopeNarrowsRowsAndTotal(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	// A private tenant, not the shared TenantA/TenantB constants: this test
	// counts every ticket in the tenant, so a neighbour's fixture would make it
	// flap under -parallel.
	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Helpdesk Own Scope Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	now := time.Now().UTC()

	me, colleague := uuid.New(), uuid.New()

	newTicket := func(subject string, requester uuid.UUID, assignee *uuid.UUID, status string) *Ticket {
		ticket := &Ticket{
			ID: uuid.New(), TenantID: tenant, Subject: subject,
			Status: status, Priority: TicketPriorityNormal,
			RequesterID: requester, AssigneeID: assignee,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateTicket(ctx, ticket); err != nil {
			t.Fatalf("CreateTicket(%s): %v", subject, err)
		}
		return ticket
	}

	// Cleanup via defer, not t.Cleanup: the latter runs after the deferred
	// pool.Close() above and would leave every row behind on a dead pool.
	raised := newTicket("Raised by me", me, nil, TicketStatusOpen)
	defer testutil.CleanupRow(t, pool, "tickets", raised.ID)
	assigned := newTicket("Assigned to me", colleague, &me, TicketStatusOpen)
	defer testutil.CleanupRow(t, pool, "tickets", assigned.ID)
	foreign := newTicket("Neither", colleague, nil, TicketStatusOpen)
	defer testutil.CleanupRow(t, pool, "tickets", foreign.ID)
	solved := newTicket("Raised by me, solved", me, nil, TicketStatusSolved)
	defer testutil.CleanupRow(t, pool, "tickets", solved.ID)

	all, allTotal, err := repo.ListTickets(ctx, tenant, nil, nil, 1, 50)
	if err != nil {
		t.Fatalf("ListTickets (unscoped): %v", err)
	}
	if allTotal != 4 || len(all) != 4 {
		t.Fatalf("unscoped list: got %d rows / total %d, want 4 / 4", len(all), allTotal)
	}

	own, ownTotal, err := repo.ListTickets(ctx, tenant, nil, &me, 1, 50)
	if err != nil {
		t.Fatalf("ListTickets (own scope): %v", err)
	}
	if ownTotal != 3 {
		t.Fatalf("own scope total: got %d, want 3 — the count must match the filtered rows, not the tenant", ownTotal)
	}
	if len(own) != 3 {
		t.Fatalf("own scope rows: got %d, want 3", len(own))
	}
	seen := make(map[uuid.UUID]bool, len(own))
	for _, ticket := range own {
		seen[ticket.ID] = true
	}
	if seen[foreign.ID] {
		t.Fatalf("a ticket the caller neither raised nor was assigned leaked into the own-scoped list")
	}
	if !seen[raised.ID] || !seen[assigned.ID] {
		t.Fatalf("own-scoped list dropped a ticket the caller owns: raised=%v assigned=%v", seen[raised.ID], seen[assigned.ID])
	}

	// Status and participant filter combined — the participant placeholder sits
	// at $3 here and at $2 above, which is exactly the kind of off-by-one that
	// silently filters on the wrong value.
	open := TicketStatusOpen
	combined, combinedTotal, err := repo.ListTickets(ctx, tenant, &open, &me, 1, 50)
	if err != nil {
		t.Fatalf("ListTickets (own scope + status): %v", err)
	}
	if combinedTotal != 2 || len(combined) != 2 {
		t.Fatalf("own scope + status=open: got %d rows / total %d, want 2 / 2", len(combined), combinedTotal)
	}
	for _, ticket := range combined {
		if ticket.Status != TicketStatusOpen {
			t.Fatalf("status filter lost its value when combined with the own scope: got %q", ticket.Status)
		}
	}
}
