package helpdesk

// Cross-tenant coverage for the intake surface (BACKLOG intake-cross-tenant-tests).
//
// external_requester_db_test.go already proves the CHECK constraint, the
// users-JOIN read path and the scope=own behaviour for external requesters --
// all against a single tenant. What was still open: two tenants each holding
// their own external-requester ticket at the same time, and whether the
// custom_fields merge in Service.UpdateTicket can be pointed at a foreign
// tenant's ticket by supplying the wrong tenantID. Both run as kmuhub_app
// against a real database, so the RLS policy and the tenant_id predicate in
// UpdateTicket's WHERE clause are the ones production has.

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestExternalRequester_CrossTenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Helpdesk External Cross Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Helpdesk External Cross Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	ticketA := externalTicket(tenantA, "External Cross Tenant A", "a@cross-tenant.example", "Anna A")
	if err := repo.CreateTicket(ctxA, ticketA); err != nil {
		t.Fatalf("CreateTicket (A): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", ticketA.ID)

	ticketB := externalTicket(tenantB, "External Cross Tenant B", "b@cross-tenant.example", "Bea B")
	if err := repo.CreateTicket(ctxB, ticketB); err != nil {
		t.Fatalf("CreateTicket (B): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", ticketB.ID)

	// Row-level: A's external ticket must not exist from B's side and vice
	// versa -- the same RLS proof as internal tickets, done for a row whose
	// only identity is a self-declared email rather than a user id.
	testutil.AssertRowCount(t, pool, ctxA, "tickets", ticketA.ID, 1)
	testutil.AssertRowCount(t, pool, ctxB, "tickets", ticketA.ID, 0)
	testutil.AssertRowCount(t, pool, ctxB, "tickets", ticketB.ID, 1)
	testutil.AssertRowCount(t, pool, ctxA, "tickets", ticketB.ID, 0)

	// GetTicketByID is the path the intake dispatch, csat and every gateway
	// route actually call -- a passing RLS row count with a failing direct
	// lookup would still be a leak.
	if _, err := repo.GetTicketByID(ctxA, ticketB.ID, tenantA); !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("GetTicketByID(B's ticket, tenant A): err = %v, want ErrTicketNotFound", err)
	}
	if _, err := repo.GetTicketByID(ctxB, ticketA.ID, tenantB); !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("GetTicketByID(A's ticket, tenant B): err = %v, want ErrTicketNotFound", err)
	}

	// List reads must not blend the two tenants' external tickets either.
	listA, totalA, err := repo.ListTickets(ctxA, tenantA, nil, nil, nil, nil, 1, 50)
	if err != nil {
		t.Fatalf("ListTickets (A): %v", err)
	}
	if totalA != 1 || len(listA) != 1 || listA[0].ID != ticketA.ID {
		t.Fatalf("tenant A's list leaked or lost rows: total=%d rows=%#v", totalA, listA)
	}
	listB, totalB, err := repo.ListTickets(ctxB, tenantB, nil, nil, nil, nil, 1, 50)
	if err != nil {
		t.Fatalf("ListTickets (B): %v", err)
	}
	if totalB != 1 || len(listB) != 1 || listB[0].ID != ticketB.ID {
		t.Fatalf("tenant B's list leaked or lost rows: total=%d rows=%#v", totalB, listB)
	}
}

func TestUpdateTicket_CustomFieldsMergeCannotCrossTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Helpdesk Merge Cross Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Helpdesk Merge Cross Tenant B")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC()

	// Tenant B's ticket carries a custom field that must never be readable or
	// mergeable from tenant A.
	ticketB := &Ticket{
		ID: uuid.New(), TenantID: tenantB, Subject: "Merge Cross Tenant Victim",
		Status: TicketStatusOpen, Priority: TicketPriorityNormal,
		RequesterID:  uuidPtr(uuid.New()),
		CustomFields: map[string]any{"vertragsnummer": "geheim-B"},
		CreatedAt:    now, UpdatedAt: now,
	}
	if err := repo.CreateTicket(ctxB, ticketB); err != nil {
		t.Fatalf("CreateTicket (B): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", ticketB.ID)

	// Attempt the merge as if it were tenant A's update -- same shape a stolen
	// or forged tenant claim would take: right ticket id, wrong tenantID
	// argument. Service.UpdateTicket loads the ticket through
	// GetTicketByID(id, tenantID) before it ever touches custom_fields, so
	// this must fail before any merge happens.
	_, err := svc.UpdateTicket(context.Background(), ticketB.ID, tenantA,
		nil, nil, nil, nil, nil, nil, nil,
		map[string]any{"injected_by_tenant_a": "pwned"},
	)
	if !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("UpdateTicket across tenants: err = %v, want ErrTicketNotFound", err)
	}

	// Tenant B's ticket must be exactly as it was: the original field intact,
	// the attempted key absent.
	got, err := repo.GetTicketByID(ctxB, ticketB.ID, tenantB)
	if err != nil {
		t.Fatalf("GetTicketByID (B, after attempted cross-tenant merge): %v", err)
	}
	if got.CustomFields["vertragsnummer"] != "geheim-B" {
		t.Fatalf("tenant B's own field was lost: %#v", got.CustomFields)
	}
	if _, leaked := got.CustomFields["injected_by_tenant_a"]; leaked {
		t.Fatalf("cross-tenant update wrote into tenant B's ticket: %#v", got.CustomFields)
	}

	// Positive control: the same merge, correctly scoped to tenant B, must
	// succeed -- proves the rejection above is the tenant guard and not some
	// unrelated failure in the call.
	updated, err := svc.UpdateTicket(ctxB, ticketB.ID, tenantB,
		nil, nil, nil, nil, nil, nil, nil,
		map[string]any{"legit_update": "ok"},
	)
	if err != nil {
		t.Fatalf("UpdateTicket within tenant B: %v", err)
	}
	if updated.CustomFields["vertragsnummer"] != "geheim-B" || updated.CustomFields["legit_update"] != "ok" {
		t.Fatalf("in-tenant merge produced unexpected fields: %#v", updated.CustomFields)
	}
}
