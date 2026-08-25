package contact

// feat-scrub-dependent-pii-dialer-tables: contact.PostgresRepository.Delete
// does not call IsInUse itself -- that gate lives in the service layer (see
// IsInUse godoc). Calling Delete directly on a contact with call campaign
// history hits the ON DELETE RESTRICT FK on dialer_campaign_contacts.contact_id
// (migration 000130) at the `DELETE FROM contacts` statement, and the whole
// transaction -- including the ScrubDependentPII updates that ran first --
// rolls back. That is why the dialer scrub itself can only be pinned through
// AnonymizeContact (consent/scrub_dialer_test.go): a contact with dialer rows
// never reaches a committed hard delete to observe the scrub through. This
// test pins the boundary instead -- Delete fails clean, no partial scrub.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestDelete_DialerCampaignContactRestrictsHardDeleteWithoutPartialScrub(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Contact Delete Dialer Restrict Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("contact-delete-dialer-restrict-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "Dialer",
		"last_name":  "DeleteRestrict",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	campaignID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"tenant_id":  tenantID,
		"name":       "Delete Restrict Campaign",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campaignID)

	campaignContactID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"tenant_id":   tenantID,
		"campaign_id": campaignID,
		"contact_id":  contactID,
		"position":    1,
		"notes":       "still on the call list",
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", campaignContactID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantID)

	if err := repo.Delete(ctxOwn, contactID, tenantID); err == nil {
		t.Fatal("expected Delete to fail with a foreign key violation for a contact with call campaign history, got nil error")
	}

	sysCtx := testutil.WithSystemCtx(context.Background())
	testutil.AssertRowCount(t, pool, sysCtx, "contacts", contactID, 1)

	var notes *string
	if err := pool.QueryRow(sysCtx, "SELECT notes FROM dialer_campaign_contacts WHERE id = $1", campaignContactID).Scan(&notes); err != nil {
		t.Fatalf("read back campaign contact: %v", err)
	}
	if notes == nil || *notes != "still on the call list" {
		t.Fatalf("expected the rolled-back transaction to leave dialer_campaign_contacts.notes untouched, got %v", notes)
	}
}
