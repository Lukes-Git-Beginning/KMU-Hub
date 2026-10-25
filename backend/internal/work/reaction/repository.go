package reaction

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for message reaction persistence
type Repository interface {
	// AddReaction adds a reaction to a message. Uses ON CONFLICT DO NOTHING
	// so duplicate reactions from the same user+emoji are silently ignored.
	AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error

	// RemoveReaction removes a specific reaction from a message.
	RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error

	// ReactionExists checks whether a specific user+emoji reaction exists on a message.
	ReactionExists(ctx context.Context, messageID, userID uuid.UUID, emoji string) (bool, error)

	// ListReactions returns all individual reactions for a message, ordered by created_at ASC.
	ListReactions(ctx context.Context, messageID uuid.UUID) ([]Reaction, error)

	// GetReactionSummary returns aggregated reaction summaries for a single message,
	// including per-emoji count, user list, and whether the current user reacted.
	GetReactionSummary(ctx context.Context, messageID, currentUserID uuid.UUID) ([]ReactionSummary, error)

	// ReachableMessages returns, for the given messages, the channel each one lives
	// in -- limited to the messages userID may see: the channel is public, or the
	// user is a member of it. Unknown, foreign-tenant and private-channel messages
	// are simply absent from the result (the caller reports them as not found).
	ReachableMessages(ctx context.Context, messageIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]uuid.UUID, error)

	// GetBatchReactionSummary returns aggregated reaction summaries for multiple messages
	// in a single query, keyed by message ID.
	GetBatchReactionSummary(ctx context.Context, messageIDs []uuid.UUID, currentUserID uuid.UUID) (map[uuid.UUID][]ReactionSummary, error)
}
