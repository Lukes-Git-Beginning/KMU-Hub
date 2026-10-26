package consent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ObjectDeleter removes one object from the object store. The chat/file
// FileStore satisfies it; MinIO's RemoveObject is a no-op for a missing key,
// which is what makes a retried erasure idempotent.
type ObjectDeleter interface {
	Delete(ctx context.Context, key string) error
}

// inspectionPhotoScope is the presign scope the desktop uploads rental
// inspection photos under (POST /api/v1/files/presign-upload, scope=vermietung);
// the object key is {tenant_id}/vermietung/{uuid}{ext}.
const inspectionPhotoScope = "vermietung"

// PurgeInspectionPhotos deletes the stored photo objects behind the
// rental_inspections.photo_urls of every rental the contact is the renter of.
// It must run BEFORE ScrubDependentPII, which empties that column and with it
// the only reference to the objects.
//
// Called inside the erasure transaction and before its commit: when a delete
// fails the whole erasure rolls back, the keys stay in the column and the next
// attempt retries them. The reverse order (commit, then delete) would orphan
// the objects for good.
//
// photo_urls is client-supplied text, so a value is deleted only when it is an
// object key under this tenant's own vermietung prefix; anything else (an
// external URL, another tenant's key, a path with "..") is skipped, otherwise
// an inspection could be used to delete foreign objects. With no deleter
// configured the keys are logged and left in place.
func PurgeInspectionPhotos(ctx context.Context, tx pgx.Tx, deleter ObjectDeleter, contactID, tenantID uuid.UUID) error {
	rows, err := tx.Query(ctx,
		`SELECT DISTINCT k
		 FROM rental_inspections i, unnest(i.photo_urls) AS k
		 WHERE i.tenant_id = $2
		   AND i.rental_id IN (SELECT id FROM rentals WHERE contact_id = $1 AND tenant_id = $2)`,
		contactID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("purge inspection photos: list keys: %w", err)
	}
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return fmt.Errorf("purge inspection photos: read keys: %w", err)
	}

	prefix := tenantID.String() + "/" + inspectionPhotoScope + "/"
	for _, key := range keys {
		if !strings.HasPrefix(key, prefix) || strings.Contains(key, "..") {
			slog.Warn("purge inspection photos: skipping value that is not a tenant object key",
				"contact_id", contactID, "tenant_id", tenantID)
			continue
		}
		if deleter == nil {
			slog.Warn("purge inspection photos: no object store configured, object left in storage",
				"contact_id", contactID, "tenant_id", tenantID, "object_key", key)
			continue
		}
		if err := deleter.Delete(ctx, key); err != nil {
			return fmt.Errorf("purge inspection photos: delete %q: %w", key, err)
		}
	}
	return nil
}
