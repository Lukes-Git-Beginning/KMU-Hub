package consent

// feat-scrub-dependent-pii-inbox-rentals-contracts: ScrubDependentPII never
// touched inbox_messages.sender_name/.sender_email/.preview or
// rentals.renter_name/.notes -- both carry a contact's name in plain text,
// keyed off the contact through a differently-named column
// (inbox_messages.crm_contact_id) or a direct one (rentals.contact_id).
// contract_parties.external_name was researched too and deliberately left
// out -- see the godoc on ScrubDependentPII for why a contact_id match can
// never reach a populated external_name.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestAnonymizeContact_ScrubsInboxMessageAndRentalIdentity(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Consent Inbox Rentals Scrub Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Consent Inbox Rentals Scrub Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", otherTenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("inbox-rentals-scrub-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	otherUserID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     otherTenantID,
		"email":         fmt.Sprintf("inbox-rentals-scrub-other-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "Inbox",
		"last_name":  "ScrubTest",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	otherContactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  otherTenantID,
		"first_name": "Foreign",
		"last_name":  "ScrubTest",
		"created_by": otherUserID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", otherContactID)

	messageID := testutil.SeedRow(t, pool, "inbox_messages", map[string]any{
		"tenant_id":      tenantID,
		"user_id":        userID,
		"channel":        "email",
		"source_id":      "src-" + uuid.NewString(),
		"crm_contact_id": contactID,
		"sender_name":    "Inbox ScrubTest",
		"sender_email":   "inbox-scrubtest@example.com",
		"preview":        "Hallo Inbox ScrubTest, anbei die Unterlagen...",
		"received_at":    time.Now().UTC(),
	})
	defer testutil.CleanupRow(t, pool, "inbox_messages", messageID)

	otherMessageID := testutil.SeedRow(t, pool, "inbox_messages", map[string]any{
		"tenant_id":      otherTenantID,
		"user_id":        otherUserID,
		"channel":        "email",
		"source_id":      "src-" + uuid.NewString(),
		"crm_contact_id": otherContactID,
		"sender_name":    "Foreign ScrubTest",
		"sender_email":   "foreign-scrubtest@example.com",
		"preview":        "unrelated foreign tenant preview",
		"received_at":    time.Now().UTC(),
	})
	defer testutil.CleanupRow(t, pool, "inbox_messages", otherMessageID)

	objectID := testutil.SeedRow(t, pool, "rental_objects", map[string]any{
		"tenant_id": tenantID,
		"name":      "Scrub Test Bagger",
	})
	defer testutil.CleanupRow(t, pool, "rental_objects", objectID)

	otherObjectID := testutil.SeedRow(t, pool, "rental_objects", map[string]any{
		"tenant_id": otherTenantID,
		"name":      "Foreign Scrub Test Bagger",
	})
	defer testutil.CleanupRow(t, pool, "rental_objects", otherObjectID)

	now := time.Now().UTC()

	rentalID := testutil.SeedRow(t, pool, "rentals", map[string]any{
		"tenant_id":   tenantID,
		"object_id":   objectID,
		"contact_id":  contactID,
		"renter_name": "Inbox ScrubTest",
		"notes":       "picks up personally on Fridays",
		"start_date":  now,
		"end_date":    now.Add(7 * 24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "rentals", rentalID)

	otherRentalID := testutil.SeedRow(t, pool, "rentals", map[string]any{
		"tenant_id":   otherTenantID,
		"object_id":   otherObjectID,
		"contact_id":  otherContactID,
		"renter_name": "Foreign ScrubTest",
		"notes":       "unrelated foreign tenant rental note",
		"start_date":  now,
		"end_date":    now.Add(7 * 24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "rentals", otherRentalID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantID)

	if err := repo.AnonymizeContact(ctxOwn, contactID, tenantID); err != nil {
		t.Fatalf("AnonymizeContact: %v", err)
	}

	sysCtx := testutil.WithSystemCtx(context.Background())

	var senderName string
	var senderEmail, preview *string
	if err := pool.QueryRow(sysCtx, "SELECT sender_name, sender_email, preview FROM inbox_messages WHERE id = $1", messageID).Scan(&senderName, &senderEmail, &preview); err != nil {
		t.Fatalf("read back inbox message: %v", err)
	}
	if senderName != "" || senderEmail != nil || (preview != nil && *preview != "") {
		t.Fatalf("expected inbox_messages sender identity to be scrubbed, got sender_name=%q sender_email=%v preview=%v", senderName, senderEmail, preview)
	}

	var renterName string
	var rentalNotes *string
	if err := pool.QueryRow(sysCtx, "SELECT renter_name, notes FROM rentals WHERE id = $1", rentalID).Scan(&renterName, &rentalNotes); err != nil {
		t.Fatalf("read back rental: %v", err)
	}
	if renterName != "" || rentalNotes != nil {
		t.Fatalf("expected rentals renter identity to be scrubbed, got renter_name=%q notes=%v", renterName, rentalNotes)
	}

	// Negative: own tenant scrubbed above (>0 rows changed), the foreign
	// tenant's rows for a different contact must survive untouched -- zero
	// rows changed there, not "both zero".
	var otherSenderName string
	var otherSenderEmail, otherPreview *string
	if err := pool.QueryRow(sysCtx, "SELECT sender_name, sender_email, preview FROM inbox_messages WHERE id = $1", otherMessageID).Scan(&otherSenderName, &otherSenderEmail, &otherPreview); err != nil {
		t.Fatalf("read back foreign inbox message: %v", err)
	}
	if otherSenderName != "Foreign ScrubTest" || otherSenderEmail == nil || *otherSenderEmail != "foreign-scrubtest@example.com" || otherPreview == nil || *otherPreview != "unrelated foreign tenant preview" {
		t.Fatalf("expected foreign tenant's inbox_messages identity to survive untouched, got sender_name=%q sender_email=%v preview=%v", otherSenderName, otherSenderEmail, otherPreview)
	}

	var otherRenterName string
	var otherRentalNotes *string
	if err := pool.QueryRow(sysCtx, "SELECT renter_name, notes FROM rentals WHERE id = $1", otherRentalID).Scan(&otherRenterName, &otherRentalNotes); err != nil {
		t.Fatalf("read back foreign rental: %v", err)
	}
	if otherRenterName != "Foreign ScrubTest" || otherRentalNotes == nil || *otherRentalNotes != "unrelated foreign tenant rental note" {
		t.Fatalf("expected foreign tenant's rentals identity to survive untouched, got renter_name=%q notes=%v", otherRenterName, otherRentalNotes)
	}
}
