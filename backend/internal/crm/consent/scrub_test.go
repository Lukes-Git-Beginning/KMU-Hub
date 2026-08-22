package consent

// fix-contact-erasure-incomplete-set-null-table-scrub: AnonymizeContact (the
// RESTRICT-hit path, called when a contact has call campaign history or
// advisory protocols and cannot be hard-deleted) already cleared
// activities.description and consent_records.ip_address/notes but never
// touched tickets -- an external requester's name/email survived unreadable
// on the contact and readable on every ticket they ever opened. This pins
// that AnonymizeContact now reaches tickets too, through the same
// ScrubDependentPII contact.PostgresRepository.Delete uses.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestAnonymizeContact_ScrubsExternalTicketRequesterIdentity(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Consent Anonymize Ticket Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("consent-anon-ticket-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "Anonymize",
		"last_name":  "TicketTest",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	// requester_id stays unset (NULL): the external-requester branch of
	// chk_tickets_requester_identity (migration 000291).
	ticketID := testutil.SeedRow(t, pool, "tickets", map[string]any{
		"tenant_id":             tenantID,
		"subject":               "Anonymize Ticket Test",
		"contact_id":            contactID,
		"requester_is_external": true,
		"requester_email":       "anon-ticket-requester@example.com",
		"requester_name":        "Anon Ticket Requester",
	})
	defer testutil.CleanupRow(t, pool, "tickets", ticketID)

	// Internal requester on the same contact, with a stray fallback name/email
	// left over from before requester_id precedence existed (000290): identity
	// lives on the user row, so AnonymizeContact must NULL these unused
	// columns outright, not write the external placeholder into them.
	internalTicketID := testutil.SeedRow(t, pool, "tickets", map[string]any{
		"tenant_id":       tenantID,
		"subject":         "Anonymize Internal Requester Ticket Test",
		"contact_id":      contactID,
		"requester_id":    userID,
		"requester_name":  "Stray Fallback Name",
		"requester_email": "stray-fallback@example.com",
	})
	defer testutil.CleanupRow(t, pool, "tickets", internalTicketID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantID)

	if err := repo.AnonymizeContact(ctxOwn, contactID, tenantID); err != nil {
		t.Fatalf("AnonymizeContact: %v", err)
	}

	sysCtx := testutil.WithSystemCtx(context.Background())

	var requesterName, requesterEmail *string
	if err := pool.QueryRow(sysCtx,
		"SELECT requester_name, requester_email FROM tickets WHERE id = $1", ticketID,
	).Scan(&requesterName, &requesterEmail); err != nil {
		t.Fatalf("read back external ticket: %v", err)
	}
	if requesterEmail == nil || *requesterEmail != AnonymizedRequesterEmail {
		t.Fatalf("expected requester_email to be the anonymization placeholder, got %v", requesterEmail)
	}
	if requesterName == nil || *requesterName == "Anon Ticket Requester" {
		t.Fatalf("expected requester_name to be replaced, got %v", requesterName)
	}

	var internalName, internalEmail *string
	if err := pool.QueryRow(sysCtx,
		"SELECT requester_name, requester_email FROM tickets WHERE id = $1", internalTicketID,
	).Scan(&internalName, &internalEmail); err != nil {
		t.Fatalf("read back internal-requester ticket: %v", err)
	}
	if internalName != nil {
		t.Fatalf("expected requester_name on an internal-requester ticket to be NULL, not the external placeholder, got %v", internalName)
	}
	if internalEmail != nil {
		t.Fatalf("expected requester_email on an internal-requester ticket to be NULL, not the external placeholder, got %v", internalEmail)
	}
}
