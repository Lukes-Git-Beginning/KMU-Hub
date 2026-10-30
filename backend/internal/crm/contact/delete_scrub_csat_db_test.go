package contact

// fix-contact-scrub-csat-comment: the hard-delete path runs consent.ScrubDependentPII
// before the DELETE (tickets.contact_id is ON DELETE SET NULL, the ticket survives),
// so the CSAT free text of the contact's tickets must be gone after Delete.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestDelete_ScrubsCSATComments(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Contact Delete CSAT Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("contact-delete-csat-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantID, "first_name": "Delete", "last_name": "CsatScrub", "created_by": userID,
	})

	ticketID := testutil.SeedRow(t, pool, "tickets", map[string]any{
		"tenant_id": tenantID, "subject": "CSAT delete", "requester_id": userID,
		"contact_id": contactID, "csat_rating": 5, "csat_comment": "Delete CsatScrub lobt den Support",
	})
	defer testutil.CleanupRow(t, pool, "tickets", ticketID)
	csatID := testutil.SeedRow(t, pool, "ticket_csat_responses", map[string]any{
		"tenant_id": tenantID, "ticket_id": ticketID, "rating": 5,
		"comment": "Delete CsatScrub lobt den Support", "submitted_at": time.Now().UTC(),
	})

	repo := NewPostgresRepository(pool)
	if err := repo.Delete(testutil.WithTenantCtx(context.Background(), tenantID), contactID, tenantID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	sysCtx := testutil.WithSystemCtx(context.Background())
	var respComment, mirrorComment *string
	var respRating, mirrorRating *int
	if err := pool.QueryRow(sysCtx,
		`SELECT r.comment, r.rating, t.csat_comment, t.csat_rating
		   FROM ticket_csat_responses r JOIN tickets t ON t.id = r.ticket_id
		  WHERE r.id = $1 AND t.id = $2`, csatID, ticketID,
	).Scan(&respComment, &respRating, &mirrorComment, &mirrorRating); err != nil {
		t.Fatalf("read back csat after Delete: %v", err)
	}
	if respComment != nil || mirrorComment != nil {
		t.Fatalf("expected csat comments scrubbed after Delete, got response=%v mirror=%v", respComment, mirrorComment)
	}
	if respRating == nil || *respRating != 5 || mirrorRating == nil || *mirrorRating != 5 {
		t.Fatalf("expected rating 5 to survive, got response=%v mirror=%v", respRating, mirrorRating)
	}
}
