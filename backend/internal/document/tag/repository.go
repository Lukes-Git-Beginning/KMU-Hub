package tag

import (
	"context"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for document tag operations.
type Repository interface {
	// Create inserts a new tag into the database.
	Create(ctx context.Context, tag *models.DocumentTag) error

	// List returns all tags for the tenant, ordered by name, with file_count
	// populated.
	List(ctx context.Context, tenantID uuid.UUID) ([]*models.DocumentTag, error)

	// Delete removes a tag scoped to the tenant. File-tag associations are
	// cascade-deleted via FK.
	Delete(ctx context.Context, tenantID, id uuid.UUID) error

	// TagFile creates a file-tag association. Idempotent (ON CONFLICT DO NOTHING).
	TagFile(ctx context.Context, tenantID, fileID, tagID uuid.UUID) error

	// UntagFile removes a file-tag association.
	UntagFile(ctx context.Context, tenantID, fileID, tagID uuid.UUID) error
}
