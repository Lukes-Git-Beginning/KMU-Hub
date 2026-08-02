package bookmark

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for message bookmark persistence.
type Repository interface {
	// Exists reports whether the user already bookmarked the message.
	Exists(ctx context.Context, tenantID, userID, messageID uuid.UUID) (bool, error)

	// Add creates the bookmark. Idempotent: a repeat call is a no-op.
	Add(ctx context.Context, tenantID, userID, messageID uuid.UUID) error

	// Remove deletes the bookmark, if any.
	Remove(ctx context.Context, tenantID, userID, messageID uuid.UUID) error

	// ListMessageIDs returns the ids of a user's bookmarked messages,
	// most recently bookmarked first.
	ListMessageIDs(ctx context.Context, tenantID, userID uuid.UUID) ([]uuid.UUID, error)
}
