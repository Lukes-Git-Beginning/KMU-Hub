package contact

// fix-contact-erasure-incomplete-set-null-table-scrub: a regular hard delete
// (Delete, the no-RESTRICT-hit path) used to leave activities.description,
// consent_records.ip_address/notes, and external tickets' requester identity
// readable after the owning contact was gone -- only the FK columns pointing
// at the contact were cleared by ON DELETE SET NULL, the content itself was
// never touched. These two tests pin the scrub Delete now runs (shared with
// consent.AnonymizeContact via consent.ScrubDependentPII) before the row
// disappears.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/crm/consent"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestDelete_ScrubsActivityAndConsentRecordPII(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Contact Delete Scrub Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("contact-delete-scrub-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "Delete",
		"last_name":  "ScrubTest",
		"created_by": userID,
	})

	activityID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"tenant_id":     tenantID,
		"contact_id":    contactID,
		"activity_type": "note",
		"subject":       "Delete Scrub Activity",
		"description":   "calls the contact a difficult customer",
		"created_by":    userID,
	})
	defer testutil.CleanupRow(t, pool, "activities", activityID)

	consentID := testutil.SeedRow(t, pool, "consent_records", map[string]any{
		"tenant_id":    tenantID,
		"contact_id":   contactID,
		"consent_type": "marketing_email",
		"granted":      true,
		"legal_basis":  "consent",
		"ip_address":   "203.0.113.7",
		"notes":        "opted in at the trade fair booth",
	})
	defer testutil.CleanupRow(t, pool, "consent_records", consentID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantID)

	if err := repo.Delete(ctxOwn, contactID, tenantID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	sysCtx := testutil.WithSystemCtx(context.Background())
	testutil.AssertRowCount(t, pool, sysCtx, "contacts", contactID, 0)

	var description *string
	if err := pool.QueryRow(sysCtx, "SELECT description FROM activities WHERE id = $1", activityID).Scan(&description); err != nil {
		t.Fatalf("read back activity: %v", err)
	}
	if description != nil {
		t.Fatalf("expected activities.description to be scrubbed by the hard delete, got %q", *description)
	}

	var ip, notes *string
	if err := pool.QueryRow(sysCtx, "SELECT ip_address::text, notes FROM consent_records WHERE id = $1", consentID).Scan(&ip, &notes); err != nil {
		t.Fatalf("read back consent record: %v", err)
	}
	if ip != nil {
		t.Fatalf("expected consent_records.ip_address to be scrubbed by the hard delete, got %q", *ip)
	}
	if notes != nil && *notes != "" {
		t.Fatalf("expected consent_records.notes to be scrubbed by the hard delete, got %q", *notes)
	}
}

func TestDelete_ScrubsExternalTicketRequesterIdentityWithoutCheckViolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Contact Delete Ticket Scrub Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("contact-delete-ticket-scrub-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "External",
		"last_name":  "Requester",
		"created_by": userID,
	})

	// requester_id stays unset (NULL): this is the external-requester branch
	// of chk_tickets_requester_identity (migration 000291), which requires
	// requester_email non-empty precisely when requester_id is NULL.
	ticketID := testutil.SeedRow(t, pool, "tickets", map[string]any{
		"tenant_id":             tenantID,
		"subject":               "Delivery delay",
		"contact_id":            contactID,
		"requester_is_external": true,
		"requester_email":       "external.requester@example.com",
		"requester_name":        "External Requester",
	})
	defer testutil.CleanupRow(t, pool, "tickets", ticketID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantID)

	if err := repo.Delete(ctxOwn, contactID, tenantID); err != nil {
		t.Fatalf("Delete: %v (a CHECK violation here means the scrub cleared requester_email without a placeholder)", err)
	}

	sysCtx := testutil.WithSystemCtx(context.Background())
	var requesterName, requesterEmail *string
	if err := pool.QueryRow(sysCtx,
		"SELECT requester_name, requester_email FROM tickets WHERE id = $1", ticketID,
	).Scan(&requesterName, &requesterEmail); err != nil {
		t.Fatalf("read back ticket: %v", err)
	}
	if requesterEmail == nil || *requesterEmail != consent.AnonymizedRequesterEmail {
		t.Fatalf("expected requester_email to be the anonymization placeholder, got %v", requesterEmail)
	}
	if requesterName == nil || *requesterName == "External Requester" {
		t.Fatalf("expected requester_name to be replaced, got %v", requesterName)
	}
}
