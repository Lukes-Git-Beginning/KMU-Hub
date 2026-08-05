package helpdesk

// External requesters (000291): a ticket that names nobody in `users`.
//
// The three things that can go wrong here, and what pins each of them:
//   - the CHECK lets a half-empty identity through (external, but no way back
//     to the person) -> TestCreateTicket_RequesterIdentityCheck
//   - the users LEFT JOIN swallows the row on the read side, which would be a
//     phantom 404 for exactly the tickets the public intake creates
//     -> TestExternalRequester_ReadableThroughEveryTicketRead
//   - scope=own quietly starts matching on the submitted mail address
//     -> TestExternalRequester_InvisibleToOwnScope
//
// These run against a real database as kmuhub_app, so the CHECK and the RLS
// policy are the ones production has, not a Go-side re-implementation.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// uuidPtr is the shorthand the ticket fixtures use since Ticket.RequesterID
// became nullable.
func uuidPtr(id uuid.UUID) *uuid.UUID { return &id }

// externalTicket builds a ticket whose requester has no user account.
func externalTicket(tenantID uuid.UUID, subject, email, name string) *Ticket {
	now := time.Now().UTC()
	return &Ticket{
		ID: uuid.New(), TenantID: tenantID, Subject: subject,
		Status: TicketStatusOpen, Priority: TicketPriorityNormal,
		RequesterID:         nil,
		Channel:             TicketChannelExternal,
		RequesterEmail:      strPtr(email),
		RequesterName:       name,
		RequesterIsExternal: true,
		CustomFields:        map[string]any{},
		CreatedAt:           now, UpdatedAt: now,
	}
}

func TestExternalRequester_ReadableThroughEveryTicketRead(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk External Read Tenant")
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	repo := NewPostgresRepository(pool)

	ticket := externalTicket(tenantID, "Kaputte Rechnung", "kundin@example.org", "Petra Kunz")
	if err := repo.CreateTicket(ctx, ticket); err != nil {
		t.Fatalf("CreateTicket with nil requester: %v", err)
	}

	got, err := repo.GetTicketByID(ctx, ticket.ID, tenantID)
	if err != nil {
		t.Fatalf("GetTicketByID: %v", err)
	}
	if got.RequesterID != nil {
		t.Fatalf("expected nil requester_id, got %s", got.RequesterID)
	}
	if !got.RequesterIsExternal {
		t.Fatal("expected requester_is_external=true")
	}
	if got.RequesterEmail == nil || *got.RequesterEmail != "kundin@example.org" {
		t.Fatalf("requester_email lost: %v", got.RequesterEmail)
	}
	// The users JOIN resolves to nothing for this requester, so the persisted
	// column has to carry the display name.
	if got.RequesterName != "Petra Kunz" {
		t.Fatalf("expected requester_name from the column, got %q", got.RequesterName)
	}

	// The list read goes through a different query with the same column list --
	// a JOIN that dropped the row would show up here and nowhere else.
	list, total, err := repo.ListTickets(ctx, tenantID, nil, nil, nil, nil, 1, 50)
	if err != nil {
		t.Fatalf("ListTickets: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("expected the external ticket in the list, got total=%d rows=%d", total, len(list))
	}
	if list[0].RequesterName != "Petra Kunz" {
		t.Fatalf("list read lost requester_name: %q", list[0].RequesterName)
	}
}

func TestExternalRequester_InvisibleToOwnScope(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk External Scope Tenant")
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	repo := NewPostgresRepository(pool)

	agent := uuid.New()

	unassigned := externalTicket(tenantID, "Extern ohne Zuweisung", "a@example.org", "A")
	if err := repo.CreateTicket(ctx, unassigned); err != nil {
		t.Fatalf("CreateTicket unassigned: %v", err)
	}
	assigned := externalTicket(tenantID, "Extern mit Zuweisung", "b@example.org", "B")
	assigned.AssigneeID = &agent
	if err := repo.CreateTicket(ctx, assigned); err != nil {
		t.Fatalf("CreateTicket assigned: %v", err)
	}

	rows, total, err := repo.ListTickets(ctx, tenantID, nil, &agent, nil, nil, 1, 50)
	if err != nil {
		t.Fatalf("ListTickets own scope: %v", err)
	}
	// Decided behaviour, not an accident: external tickets carry no user id, and
	// requester_email is an unverified value from the intake body, so own-scope
	// reaches them only through assignment.
	if total != 1 || len(rows) != 1 {
		t.Fatalf("expected only the assigned external ticket in own scope, got total=%d rows=%d", total, len(rows))
	}
	if rows[0].ID != assigned.ID {
		t.Fatalf("own scope returned the wrong ticket: %s", rows[0].ID)
	}
}

func TestCreateTicket_RequesterIdentityCheck(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk External Check Tenant")
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	repo := NewPostgresRepository(pool)

	// Neither a user nor a reply address: the DB has to refuse this outright,
	// because nothing downstream could ever answer such a ticket.
	orphan := externalTicket(tenantID, "Niemandes Ticket", "", "")
	orphan.RequesterEmail = nil
	if err := repo.CreateTicket(ctx, orphan); err == nil {
		t.Fatal("expected chk_tickets_requester_identity to reject a ticket with neither requester_id nor email")
	}

	// Same for "external" without the flag actually being set.
	noFlag := externalTicket(tenantID, "Halb extern", "c@example.org", "C")
	noFlag.RequesterIsExternal = false
	if err := repo.CreateTicket(ctx, noFlag); err == nil {
		t.Fatal("expected chk_tickets_requester_identity to reject nil requester_id without requester_is_external")
	}
}

func TestServiceCreateTicket_RejectsRequesterlessTicket(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	_, err := h.svc.CreateTicket(context.Background(), uuid.New(), nil,
		"Kein Absender", TicketPriorityNormal, nil, nil, "", "", nil, nil,
		TicketIntake{Channel: TicketChannelExternal})
	if !errors.Is(err, ErrMissingRequester) {
		t.Fatalf("expected ErrMissingRequester, got %v", err)
	}

	// With the reply address the same call is legal.
	ticket, err := h.svc.CreateTicket(context.Background(), uuid.New(), nil,
		"Mit Absender", TicketPriorityNormal, nil, nil, "", "", nil, nil,
		TicketIntake{
			Channel:             TicketChannelExternal,
			RequesterIsExternal: true,
			RequesterEmail:      strPtr("kundin@example.org"),
			RequesterName:       strPtr("Petra Kunz"),
		})
	if err != nil {
		t.Fatalf("CreateTicket external: %v", err)
	}
	if ticket.RequesterID != nil {
		t.Fatalf("expected nil requester_id, got %s", ticket.RequesterID)
	}
	if ticket.RequesterName != "Petra Kunz" {
		t.Fatalf("expected the external name to be persisted, got %q", ticket.RequesterName)
	}
}
