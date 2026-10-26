package consent

// feat-erasure-deletes-rental-inspection-photos: clearing rental_inspections.photo_urls
// only dropped the reference, the photo objects stayed in the object store.

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

type recordingDeleter struct {
	mu      sync.Mutex
	deleted []string
	fail    bool
}

func (d *recordingDeleter) Delete(_ context.Context, key string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.fail {
		return errors.New("object store down")
	}
	d.deleted = append(d.deleted, key)
	return nil
}

func (d *recordingDeleter) keys() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := slices.Clone(d.deleted)
	slices.Sort(out)
	return out
}

func TestAnonymizeContact_DeletesRentalInspectionPhotoObjects(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Consent Inspection Photos Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Consent Inspection Photos Other Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", otherTenantID) })

	seedUser := func(tenant uuid.UUID, tag string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenant,
			"email":         "insp-photos-" + tag + "-" + uuid.NewString()[:8] + "@tenant.local",
			"password_hash": "$argon2id$v=19$test",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	userID := seedUser(tenantID, "own")
	otherUserID := seedUser(otherTenantID, "other")

	seedContact := func(tenant, user uuid.UUID, last string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "contacts", map[string]any{
			"tenant_id": tenant, "first_name": "Photo", "last_name": last, "created_by": user,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", id) })
		return id
	}
	contactID := seedContact(tenantID, userID, "Renter")
	siblingID := seedContact(tenantID, userID, "Sibling")
	foreignID := seedContact(otherTenantID, otherUserID, "Foreign")

	now := time.Now().UTC()
	seedInspection := func(tenant, contact uuid.UUID, kind string, photos []string) uuid.UUID {
		objectID := testutil.SeedRow(t, pool, "rental_objects", map[string]any{"tenant_id": tenant, "name": "Photo Bagger"})
		rentalID := testutil.SeedRow(t, pool, "rentals", map[string]any{
			"tenant_id": tenant, "object_id": objectID, "contact_id": contact,
			"renter_name": "Photo Renter", "start_date": now, "end_date": now.Add(24 * time.Hour),
		})
		inspectionID := testutil.SeedRow(t, pool, "rental_inspections", map[string]any{
			"tenant_id": tenant, "rental_id": rentalID, "kind": kind, "photo_urls": photos,
		})
		t.Cleanup(func() {
			testutil.CleanupRow(t, pool, "rental_inspections", inspectionID)
			testutil.CleanupRow(t, pool, "rentals", rentalID)
			testutil.CleanupRow(t, pool, "rental_objects", objectID)
		})
		return inspectionID
	}

	own := tenantID.String() + "/vermietung/"
	foreign := otherTenantID.String() + "/vermietung/"
	handover := seedInspection(tenantID, contactID, "handover", []string{
		own + "a.jpg",
		own + "b.jpg",
		// Client-supplied text that is no object key of this tenant's vermietung scope.
		"https://files.example/external.jpg",
		foreign + "stolen.jpg",
		own + "../documents/secret.pdf",
		tenantID.String() + "/documents/contract.pdf",
	})
	ret := seedInspection(tenantID, contactID, "return", []string{own + "a.jpg", own + "c.jpg"})
	siblingInspection := seedInspection(tenantID, siblingID, "handover", []string{own + "sibling.jpg"})
	foreignInspection := seedInspection(otherTenantID, foreignID, "handover", []string{foreign + "foreign.jpg"})

	deleter := &recordingDeleter{fail: true}
	repo := NewPostgresRepository(pool)
	repo.SetObjectDeleter(deleter)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	sysCtx := testutil.WithSystemCtx(context.Background())

	photosOf := func(id uuid.UUID) []string {
		t.Helper()
		var p []string
		if err := pool.QueryRow(sysCtx, "SELECT photo_urls FROM rental_inspections WHERE id = $1", id).Scan(&p); err != nil {
			t.Fatalf("read back inspection: %v", err)
		}
		return p
	}

	// A failing object store aborts the erasure: nothing is committed, the keys
	// stay referenced, so a retry still knows what to delete.
	if err := repo.AnonymizeContact(ctx, contactID, tenantID); err == nil {
		t.Fatal("expected AnonymizeContact to fail while the object store is down")
	}
	if got := photosOf(handover); len(got) != 6 {
		t.Fatalf("expected photo_urls untouched after the failed erasure, got %v", got)
	}
	var firstName string
	if err := pool.QueryRow(sysCtx, "SELECT first_name FROM contacts WHERE id = $1", contactID).Scan(&firstName); err != nil {
		t.Fatalf("read contact: %v", err)
	}
	if firstName != "Photo" {
		t.Fatalf("expected the contact to stay un-anonymized after the failed erasure, got %q", firstName)
	}

	deleter.mu.Lock()
	deleter.fail = false
	deleter.mu.Unlock()
	if err := repo.AnonymizeContact(ctx, contactID, tenantID); err != nil {
		t.Fatalf("AnonymizeContact: %v", err)
	}

	want := []string{own + "a.jpg", own + "b.jpg", own + "c.jpg"}
	if got := deleter.keys(); !slices.Equal(got, want) {
		t.Fatalf("deleted objects = %v, want exactly %v (each once; no foreign, external, traversal or other-scope value)", got, want)
	}
	if got := photosOf(handover); len(got) != 0 {
		t.Fatalf("expected handover photo_urls cleared, got %v", got)
	}
	if got := photosOf(ret); len(got) != 0 {
		t.Fatalf("expected return photo_urls cleared, got %v", got)
	}
	if got := photosOf(siblingInspection); len(got) != 1 {
		t.Fatalf("expected the sibling contact's photos to survive, got %v", got)
	}
	if got := photosOf(foreignInspection); len(got) != 1 {
		t.Fatalf("expected the foreign tenant's photos to survive, got %v", got)
	}

	// Idempotent: the references are gone, a second run deletes nothing more.
	if err := repo.AnonymizeContact(ctx, contactID, tenantID); err != nil {
		t.Fatalf("second AnonymizeContact: %v", err)
	}
	if got := deleter.keys(); len(got) != len(want) {
		t.Fatalf("expected no further deletes on a repeat run, got %v", got)
	}
}
