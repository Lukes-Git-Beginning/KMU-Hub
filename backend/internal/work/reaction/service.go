package reaction

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"
)

// ToggleResult is the response from a toggle operation, indicating whether
// the reaction was added or removed and the updated summary for the message.
type ToggleResult struct {
	Reactions []ReactionSummary `json:"reactions"`
	Added     bool              `json:"added"`
}

// Service handles reaction business logic
type Service struct {
	repo Repository
}

// NewService creates a new reaction service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ToggleReaction adds a reaction if it does not exist, or removes it if it does.
// Returns the updated reaction summary for the message and whether the reaction was added.
func (s *Service) ToggleReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (*ToggleResult, error) {
	if err := validateEmoji(emoji); err != nil {
		return nil, err
	}

	exists, err := s.repo.ReactionExists(ctx, messageID, userID, emoji)
	if err != nil {
		return nil, err
	}

	if exists {
		if removeErr := s.repo.RemoveReaction(ctx, messageID, userID, emoji); removeErr != nil {
			return nil, removeErr
		}
		slog.Info("reaction removed",
			"message_id", messageID,
			"user_id", userID,
			"emoji", emoji,
		)
	} else {
		if addErr := s.repo.AddReaction(ctx, messageID, userID, emoji); addErr != nil {
			return nil, addErr
		}
		slog.Info("reaction added",
			"message_id", messageID,
			"user_id", userID,
			"emoji", emoji,
		)
	}

	summaries, summaryErr := s.repo.GetReactionSummary(ctx, messageID, userID)
	if summaryErr != nil {
		return nil, summaryErr
	}
	if summaries == nil {
		summaries = []ReactionSummary{}
	}

	return &ToggleResult{
		Reactions: summaries,
		Added:     !exists,
	}, nil
}

// ListReactions returns all individual reactions for a message.
func (s *Service) ListReactions(ctx context.Context, messageID uuid.UUID) ([]Reaction, error) {
	return s.repo.ListReactions(ctx, messageID)
}

// GetReactionSummary returns aggregated reaction summaries for a single message.
func (s *Service) GetReactionSummary(ctx context.Context, messageID, currentUserID uuid.UUID) ([]ReactionSummary, error) {
	return s.repo.GetReactionSummary(ctx, messageID, currentUserID)
}

// GetBatchReactionSummary returns aggregated reaction summaries for multiple messages.
func (s *Service) GetBatchReactionSummary(ctx context.Context, messageIDs []uuid.UUID, currentUserID uuid.UUID) (map[uuid.UUID][]ReactionSummary, error) {
	if len(messageIDs) == 0 {
		return make(map[uuid.UUID][]ReactionSummary), nil
	}
	return s.repo.GetBatchReactionSummary(ctx, messageIDs, currentUserID)
}

// validateEmoji checks that the emoji string is non-empty and within the 32-byte limit.
func validateEmoji(emoji string) error {
	if strings.TrimSpace(emoji) == "" {
		return ErrEmojiRequired
	}
	if len(emoji) > 32 {
		return ErrEmojiTooLong
	}
	return nil
}
