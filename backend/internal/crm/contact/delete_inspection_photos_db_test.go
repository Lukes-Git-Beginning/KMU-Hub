package contact

// feat-erasure-deletes-rental-inspection-photos: the hard-delete path must delete
// the photo objects behind rental_inspections.photo_urls, not just drop the reference.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

type recordingDeleter struct {
	deleted []string
	fail    bool
}

func (d *recordingDeleter) Delete(_ context.Context, key string) error {
	if d.fail {
		return errors.New("object store down")
	}
	d.deleted = append(d.deleted, key)
	return nil
}

func TestDelete_DeletesRentalInspectionPhotoObjects(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Contact Delete Inspection Photos Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         "contact-delete-photos-" + uuid.NewString()[:8] + "@tenant.local",
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantID, "first_name": "Delete", "last_name": "Photos", "created_by": userID,
	})

	objectID := testutil.SeedRow(t, pool, "rental_objects", map[string]any{"tenant_id": tenantID, "name": "Delete Photo Bagger"})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "rental_objects", objectID) })

	now := time.Now().UTC()
	rentalID := testutil.SeedRow(t, pool, "rentals", map[string]any{
		"tenant_id": tenantID, "object_id": objectID, "contact_id": contactID,
		"renter_name": "Delete Photos", "start_date": now, "end_date": now.Add(24 * time.Hour),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "rentals", rentalID) })

	key := tenantID.String() + "/vermietung/" + uuid.NewString() + ".jpg"
	inspectionID := testutil.SeedRow(t, pool, "rental_inspections", map[string]any{
		"tenant_id": tenantID, "rental_id": rentalID, "kind": "return", "photo_urls": []string{key},
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "rental_inspections", inspectionID) })

	deleter := &recordingDeleter{fail: true}
	repo := NewPostgresRepository(pool)
	repo.SetObjectDeleter(deleter)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	sysCtx := testutil.WithSystemCtx(context.Background())

	if err := repo.Delete(ctx, contactID, tenantID); err == nil {
		t.Fatal("expected Delete to fail while the object store is down")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "contacts", contactID, 1)

	deleter.fail = false
	if err := repo.Delete(ctx, contactID, tenantID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "contacts", contactID, 0)
	if len(deleter.deleted) != 1 || deleter.deleted[0] != key {
		t.Fatalf("deleted objects = %v, want [%s]", deleter.deleted, key)
	}

	var photos []string
	if err := pool.QueryRow(sysCtx, "SELECT photo_urls FROM rental_inspections WHERE id = $1", inspectionID).Scan(&photos); err != nil {
		t.Fatalf("read back inspection: %v", err)
	}
	if len(photos) != 0 {
		t.Fatalf("expected photo_urls cleared, got %v", photos)
	}
}
