package consent

// fix-contact-erasure-leaves-rental-signatures: ScrubDependentPII cleared
// rentals.renter_name/.notes but left the renter's signature
// (rentals.signature_data/.signed_by) and the handover/return protocols
// (rental_inspections) standing. rentals.contact_id has no FK, so the rows
// outlived the contact unnoticed.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestAnonymizeContact_ScrubsRentalSignatureAndInspections(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Consent Rental Signature Scrub Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Consent Rental Signature Scrub Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", otherTenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         "rental-sig-scrub-" + uuid.NewString()[:8] + "@tenant.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	otherUserID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     otherTenantID,
		"email":         "rental-sig-scrub-other-" + uuid.NewString()[:8] + "@tenant.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantID, "first_name": "Sig", "last_name": "ScrubTest", "created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	otherContactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": otherTenantID, "first_name": "Foreign", "last_name": "SigTest", "created_by": otherUserID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", otherContactID)

	now := time.Now().UTC()

	seedRental := func(tenant, contact uuid.UUID, name string) (objectID, rentalID, inspectionID uuid.UUID) {
		objectID = testutil.SeedRow(t, pool, "rental_objects", map[string]any{"tenant_id": tenant, "name": "Sig Bagger " + name})

		rentalID = testutil.SeedRow(t, pool, "rentals", map[string]any{
			"tenant_id":      tenant,
			"object_id":      objectID,
			"contact_id":     contact,
			"renter_name":    name,
			"signature_data": "data:image/png;base64,AAAA",
			"signed_by":      name,
			"signed_at":      now,
			"start_date":     now,
			"end_date":       now.Add(24 * time.Hour),
		})

		inspectionID = testutil.SeedRow(t, pool, "rental_inspections", map[string]any{
			"tenant_id":      tenant,
			"rental_id":      rentalID,
			"kind":           "handover",
			"notes":          "Uebergabe an " + name,
			"signature_data": "data:image/png;base64,BBBB",
			"photo_urls":     []string{"https://files.example/" + name + ".jpg"},
			"checklist":      `[{"label":"Tank voll","checked":true}]`,
		})
		return objectID, rentalID, inspectionID
	}

	objectID, rentalID, inspectionID := seedRental(tenantID, contactID, "Sig ScrubTest")
	defer testutil.CleanupRow(t, pool, "rental_inspections", inspectionID)
	defer testutil.CleanupRow(t, pool, "rentals", rentalID)
	defer testutil.CleanupRow(t, pool, "rental_objects", objectID)

	otherObjectID, otherRentalID, otherInspectionID := seedRental(otherTenantID, otherContactID, "Foreign SigTest")
	defer testutil.CleanupRow(t, pool, "rental_inspections", otherInspectionID)
	defer testutil.CleanupRow(t, pool, "rentals", otherRentalID)
	defer testutil.CleanupRow(t, pool, "rental_objects", otherObjectID)

	repo := NewPostgresRepository(pool)
	if err := repo.AnonymizeContact(testutil.WithTenantCtx(context.Background(), tenantID), contactID, tenantID); err != nil {
		t.Fatalf("AnonymizeContact: %v", err)
	}

	sysCtx := testutil.WithSystemCtx(context.Background())

	readRental := func(id uuid.UUID) (sig, signedBy *string, signedAt *time.Time) {
		t.Helper()
		if err := pool.QueryRow(sysCtx, "SELECT signature_data, signed_by, signed_at FROM rentals WHERE id = $1", id).Scan(&sig, &signedBy, &signedAt); err != nil {
			t.Fatalf("read back rental: %v", err)
		}
		return sig, signedBy, signedAt
	}
	readInspection := func(id uuid.UUID) (notes string, sig *string, photos []string, checklist *string) {
		t.Helper()
		if err := pool.QueryRow(sysCtx, "SELECT notes, signature_data, photo_urls, checklist::text FROM rental_inspections WHERE id = $1", id).Scan(&notes, &sig, &photos, &checklist); err != nil {
			t.Fatalf("read back inspection: %v", err)
		}
		return notes, sig, photos, checklist
	}

	sig, signedBy, signedAt := readRental(rentalID)
	if sig != nil || signedBy != nil {
		t.Fatalf("expected rental signature to be scrubbed, got signature_data=%v signed_by=%v", sig, signedBy)
	}
	if signedAt == nil {
		t.Fatalf("expected rentals.signed_at (not personal data) to survive")
	}

	notes, isig, photos, checklist := readInspection(inspectionID)
	if notes != "" || isig != nil || len(photos) != 0 || checklist != nil {
		t.Fatalf("expected inspection to be scrubbed, got notes=%q signature=%v photos=%v checklist=%v", notes, isig, photos, checklist)
	}

	// Foreign tenant's rows for a different contact must survive untouched.
	osig, osignedBy, _ := readRental(otherRentalID)
	if osig == nil || osignedBy == nil || *osignedBy != "Foreign SigTest" {
		t.Fatalf("expected foreign tenant's rental signature to survive, got signature_data=%v signed_by=%v", osig, osignedBy)
	}
	onotes, oisig, ophotos, ochecklist := readInspection(otherInspectionID)
	if onotes == "" || oisig == nil || len(ophotos) != 1 || ochecklist == nil {
		t.Fatalf("expected foreign tenant's inspection to survive, got notes=%q signature=%v photos=%v checklist=%v", onotes, oisig, ophotos, ochecklist)
	}
}
