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

	// GetByID retrieves a tag by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*models.DocumentTag, error)

	// List returns all tags ordered by name, with file_count populated.
	List(ctx context.Context) ([]*models.DocumentTag, error)

	// Delete removes a tag. File-tag associations are cascade-deleted via FK.
	Delete(ctx context.Context, id uuid.UUID) error

	// TagFile creates a file-tag association. Idempotent (ON CONFLICT DO NOTHING).
	TagFile(ctx context.Context, fileID, tagID uuid.UUID) error

	// UntagFile removes a file-tag association.
	UntagFile(ctx context.Context, fileID, tagID uuid.UUID) error

	// ListFileTags returns all tags associated with a given file.
	ListFileTags(ctx context.Context, fileID uuid.UUID) ([]*models.DocumentTag, error)

	// ListFilesByTag returns files with a given tag, with pagination.
	ListFilesByTag(ctx context.Context, tagID uuid.UUID, limit, offset int) ([]*models.DocumentFile, int, error)
}
