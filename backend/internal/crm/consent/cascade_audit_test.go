package consent

// fix-gdpr-deletion-request-audit-trail-cascade-loss: gdpr_deletion_requests
// used to CASCADE-delete with its contact (migration 000082). A contact with
// a "completed" deletion request could still be hard-deleted through a
// second, independent path (contact.Service.Delete or
// ContactRetentionHandler.Apply with action=delete) that never checks for
// existing gdpr_deletion_requests rows -- the CASCADE then wiped the one
// record proving an Art. 17 erasure was fulfilled. Migration 000322 changes
// contact_id to ON DELETE SET NULL and adds original_contact_id (no FK) as a
// permanent snapshot. This test proves the completed request row survives a
// hard delete of its contact, readable, with the snapshot intact.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestGetDeletionRequest_SurvivesContactHardDelete(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "GDPR Cascade Audit Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("gdpr-cascade-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "Cascade",
		"last_name":  "AuditTest",
		"created_by": userID,
	})

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	req := &GDPRDeletionRequest{
		ID:                uuid.New(),
		TenantID:          tenantID,
		ContactID:         &contactID,
		OriginalContactID: contactID,
		Reason:            "cascade audit trail test",
		Status:            "completed",
		CompletedAt:       &now,
		CreatedAt:         now,
	}
	if err := repo.CreateDeletionRequest(ctxOwn, req); err != nil {
		t.Fatalf("CreateDeletionRequest: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "gdpr_deletion_requests", req.ID)

	// Hard-delete the contact through a path that knows nothing about
	// gdpr_deletion_requests -- exactly what contact.Service.Delete and
	// ContactRetentionHandler.Apply(action=delete) do.
	if _, err := pool.Exec(ctxOwn, "DELETE FROM contacts WHERE id = $1 AND tenant_id = $2", contactID, tenantID); err != nil {
		t.Fatalf("hard-delete contact: %v", err)
	}

	got, err := repo.GetDeletionRequest(ctxOwn, req.ID, tenantID)
	if err != nil {
		t.Fatalf("GetDeletionRequest after contact hard-delete: expected the completed request to survive, got error: %v", err)
	}
	if got.Status != "completed" {
		t.Fatalf("expected status completed, got %q", got.Status)
	}
	if got.ContactID != nil {
		t.Fatalf("expected ContactID to be SET NULL after the contact was hard-deleted, got %v", *got.ContactID)
	}
	if got.OriginalContactID != contactID {
		t.Fatalf("expected OriginalContactID to retain %s, got %s", contactID, got.OriginalContactID)
	}
}
