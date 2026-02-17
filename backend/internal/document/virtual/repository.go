package virtual

import (
	"context"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for virtual folder operations.
// Virtual folders surface files from other subsystems (chat, email, task)
// without physical copy. All queries are read-only.
type Repository interface {
	// ListChatFiles returns files from chat channels the user is a member of.
	ListChatFiles(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.VirtualFile, int, error)

	// ListEmailAttachments returns email attachments from the user's email accounts.
	ListEmailAttachments(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.VirtualFile, int, error)

	// ListTaskFiles returns files from tasks the user has access to
	// (via project membership or task assignment).
	ListTaskFiles(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.VirtualFile, int, error)

	// ListAll returns files from all sources combined, optionally filtered by sourceType.
	// Results are ordered by created_at DESC and paginated over the union.
	ListAll(ctx context.Context, userID uuid.UUID, sourceType string, limit, offset int) ([]*models.VirtualFile, int, error)
}
