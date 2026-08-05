package helpdesk

// Proves the requester_name precedence introduced with migration 000290
// against a real database as kmuhub_app. The rule is only worth writing down
// if it is also enforced: the users JOIN wins wherever it resolves, and
// tickets.requester_name carries only for requesters who have no user row.
//
// Getting this backwards is silent -- both branches return *a* name, and the
// wrong one only shows up as a stale display name months after a rename.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTicketRead_RequesterNamePrecedence(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk Requester Name Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("hd-req-%s@precedence.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Ines",
		"last_name":     "Intern",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// Internal requester: has a user row, and the column is deliberately given
	// a conflicting value so the assertion can only pass if the JOIN wins.
	internal := &Ticket{
		ID: uuid.New(), TenantID: tenantID, Subject: "Interne Anfrage",
		Status: TicketStatusOpen, Priority: TicketPriorityNormal,
		RequesterID: userID, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateTicket(ctx, internal); err != nil {
		t.Fatalf("CreateTicket internal: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", internal.ID)

	// External requester: requester_id points at no user row, so the LEFT JOIN
	// yields NULL and the column is the only name there is. (Until B5 relaxes
	// requester_id to NULL, a dangling id is how an account-less requester
	// looks to the read path -- the JOIN behaves identically either way.)
	external := &Ticket{
		ID: uuid.New(), TenantID: tenantID, Subject: "Externe Anfrage",
		Status: TicketStatusOpen, Priority: TicketPriorityNormal,
		RequesterID: uuid.New(), CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateTicket(ctx, external); err != nil {
		t.Fatalf("CreateTicket external: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", external.ID)

	// CreateTicket does not write the column yet (that is B2); set it directly
	// so this test exercises the read rule and nothing else.
	if _, err := pool.Exec(ctx,
		`UPDATE tickets SET requester_name = $2 WHERE id = $1`,
		internal.ID, "Falscher Name",
	); err != nil {
		t.Fatalf("seed internal requester_name: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE tickets SET requester_name = $2 WHERE id = $1`,
		external.ID, "Erik Extern",
	); err != nil {
		t.Fatalf("seed external requester_name: %v", err)
	}

	got, err := repo.GetTicketByID(ctx, internal.ID, tenantID)
	if err != nil {
		t.Fatalf("GetTicketByID internal: %v", err)
	}
	if got.RequesterName != "Ines Intern" {
		t.Errorf("internal requester name = %q, want %q (the users JOIN must win over the column)",
			got.RequesterName, "Ines Intern")
	}

	got, err = repo.GetTicketByID(ctx, external.ID, tenantID)
	if err != nil {
		t.Fatalf("GetTicketByID external: %v", err)
	}
	if got.RequesterName != "Erik Extern" {
		t.Errorf("external requester name = %q, want %q (the column must carry when the JOIN misses)",
			got.RequesterName, "Erik Extern")
	}
}

// Existing rows must come out of 000290 with usable intake defaults rather
// than NULLs in NOT NULL columns -- a ticket with no channel is a ticket the
// module's Herkunft tabs cannot place.
func TestTicketRead_IntakeColumnDefaults(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk Intake Defaults Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	ticket := &Ticket{
		ID: uuid.New(), TenantID: tenantID, Subject: "Default-Ticket",
		Status: TicketStatusOpen, Priority: TicketPriorityNormal,
		RequesterID: uuid.New(), CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateTicket(ctx, ticket); err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", ticket.ID)

	var channel string
	var isExternal bool
	var customFields string
	if err := pool.QueryRow(ctx,
		`SELECT channel, requester_is_external, custom_fields::text
		   FROM tickets WHERE id = $1`, ticket.ID,
	).Scan(&channel, &isExternal, &customFields); err != nil {
		t.Fatalf("read intake columns: %v", err)
	}

	if channel != "agent" {
		t.Errorf("channel = %q, want %q", channel, "agent")
	}
	if isExternal {
		t.Error("requester_is_external = true, want false for a ticket created without intake fields")
	}
	if customFields != "{}" {
		t.Errorf("custom_fields = %q, want %q", customFields, "{}")
	}
}
