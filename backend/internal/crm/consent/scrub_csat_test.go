package consent

// fix-contact-scrub-csat-comment: ticket_csat_responses.comment and its mirror
// tickets.csat_comment are the contact's own words about a ticket. ScrubDependentPII
// must clear both and leave the rating, other contacts' and other tenants' rows alone.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestAnonymizeContact_ScrubsCSATComments(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Consent CSAT Scrub Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Consent CSAT Scrub Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", otherTenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("csat-scrub-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	otherUserID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     otherTenantID,
		"email":         fmt.Sprintf("csat-scrub-other-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantID, "first_name": "Csat", "last_name": "ScrubTest", "created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	siblingID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantID, "first_name": "Csat", "last_name": "Sibling", "created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", siblingID)

	foreignContactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": otherTenantID, "first_name": "Csat", "last_name": "Foreign", "created_by": otherUserID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", foreignContactID)

	now := time.Now().UTC()
	seed := func(tenant, requester, contact uuid.UUID, comment string) (ticketID, csatID uuid.UUID) {
		ticketID = testutil.SeedRow(t, pool, "tickets", map[string]any{
			"tenant_id":    tenant,
			"subject":      "CSAT scrub " + comment,
			"requester_id": requester,
			"contact_id":   contact,
			"csat_rating":  4,
			"csat_comment": comment,
		})
		csatID = testutil.SeedRow(t, pool, "ticket_csat_responses", map[string]any{
			"tenant_id":    tenant,
			"ticket_id":    ticketID,
			"rating":       4,
			"comment":      comment,
			"submitted_at": now,
		})
		return ticketID, csatID
	}
	ticketID, csatID := seed(tenantID, userID, contactID, "Herr ScrubTest war sehr freundlich")
	defer testutil.CleanupRow(t, pool, "tickets", ticketID)
	siblingTicketID, siblingCsatID := seed(tenantID, userID, siblingID, "Kommentar des Geschwister-Kontakts")
	defer testutil.CleanupRow(t, pool, "tickets", siblingTicketID)
	foreignTicketID, foreignCsatID := seed(otherTenantID, otherUserID, foreignContactID, "Kommentar des fremden Mandanten")
	defer testutil.CleanupRow(t, pool, "tickets", foreignTicketID)

	repo := NewPostgresRepository(pool)
	if err := repo.AnonymizeContact(testutil.WithTenantCtx(context.Background(), tenantID), contactID, tenantID); err != nil {
		t.Fatalf("AnonymizeContact: %v", err)
	}

	sysCtx := testutil.WithSystemCtx(context.Background())
	read := func(ticket, csat uuid.UUID) (respComment, mirrorComment *string, respRating, mirrorRating *int) {
		if err := pool.QueryRow(sysCtx,
			`SELECT r.comment, r.rating, t.csat_comment, t.csat_rating
			   FROM ticket_csat_responses r JOIN tickets t ON t.id = r.ticket_id
			  WHERE r.id = $1 AND t.id = $2`, csat, ticket,
		).Scan(&respComment, &respRating, &mirrorComment, &mirrorRating); err != nil {
			t.Fatalf("read back csat: %v", err)
		}
		return respComment, mirrorComment, respRating, mirrorRating
	}

	rc, mc, rr, mr := read(ticketID, csatID)
	if rc != nil || mc != nil {
		t.Fatalf("expected csat comments scrubbed, got response=%v mirror=%v", rc, mc)
	}
	if rr == nil || *rr != 4 || mr == nil || *mr != 4 {
		t.Fatalf("expected rating 4 to survive, got response=%v mirror=%v", rr, mr)
	}

	for name, ids := range map[string][2]uuid.UUID{
		"same-tenant sibling": {siblingTicketID, siblingCsatID},
		"foreign tenant":      {foreignTicketID, foreignCsatID},
	} {
		rc, mc, _, _ := read(ids[0], ids[1])
		if rc == nil || mc == nil {
			t.Fatalf("%s: comments must survive untouched, got response=%v mirror=%v", name, rc, mc)
		}
	}
}
