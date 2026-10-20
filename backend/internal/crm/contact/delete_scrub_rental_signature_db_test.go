package contact

// fix-contact-erasure-leaves-rental-signatures: the hard-delete path must also
// clear the renter's signature and the handover protocol. rentals.contact_id
// has no FK, so without the scrub the rows survive the contact unnoticed.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestDelete_ScrubsRentalSignatureAndInspections(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Contact Delete Rental Signature Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         "contact-delete-rental-" + uuid.NewString()[:8] + "@tenant.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantID, "first_name": "Delete", "last_name": "RentalSig", "created_by": userID,
	})

	objectID := testutil.SeedRow(t, pool, "rental_objects", map[string]any{"tenant_id": tenantID, "name": "Delete Sig Bagger"})
	defer testutil.CleanupRow(t, pool, "rental_objects", objectID)

	now := time.Now().UTC()
	rentalID := testutil.SeedRow(t, pool, "rentals", map[string]any{
		"tenant_id":      tenantID,
		"object_id":      objectID,
		"contact_id":     contactID,
		"renter_name":    "Delete RentalSig",
		"signature_data": "data:image/png;base64,AAAA",
		"signed_by":      "Delete RentalSig",
		"start_date":     now,
		"end_date":       now.Add(24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "rentals", rentalID)

	inspectionID := testutil.SeedRow(t, pool, "rental_inspections", map[string]any{
		"tenant_id":      tenantID,
		"rental_id":      rentalID,
		"kind":           "return",
		"notes":          "Rueckgabe durch Delete RentalSig",
		"signature_data": "data:image/png;base64,BBBB",
		"photo_urls":     []string{"https://files.example/return.jpg"},
		"checklist":      `[{"label":"Tank voll","checked":true}]`,
	})
	defer testutil.CleanupRow(t, pool, "rental_inspections", inspectionID)

	repo := NewPostgresRepository(pool)
	if err := repo.Delete(testutil.WithTenantCtx(context.Background(), tenantID), contactID, tenantID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	sysCtx := testutil.WithSystemCtx(context.Background())
	testutil.AssertRowCount(t, pool, sysCtx, "contacts", contactID, 0)

	var sig, signedBy *string
	if err := pool.QueryRow(sysCtx, "SELECT signature_data, signed_by FROM rentals WHERE id = $1", rentalID).Scan(&sig, &signedBy); err != nil {
		t.Fatalf("read back rental: %v", err)
	}
	if sig != nil || signedBy != nil {
		t.Fatalf("expected rental signature to be scrubbed by the hard delete, got signature_data=%v signed_by=%v", sig, signedBy)
	}

	var notes string
	var isig, checklist *string
	var photos []string
	if err := pool.QueryRow(sysCtx, "SELECT notes, signature_data, photo_urls, checklist::text FROM rental_inspections WHERE id = $1", inspectionID).Scan(&notes, &isig, &photos, &checklist); err != nil {
		t.Fatalf("read back inspection: %v", err)
	}
	if notes != "" || isig != nil || len(photos) != 0 || checklist != nil {
		t.Fatalf("expected inspection to be scrubbed by the hard delete, got notes=%q signature=%v photos=%v checklist=%v", notes, isig, photos, checklist)
	}
}
