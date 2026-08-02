package bookmark

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/chat/message"
	"github.com/kmuhub/kmuhub/internal/models"
)

// MessageReader is the minimal view of the message service bookmarks need:
// fetch a message with sender info while enforcing that the caller can see
// it. *message.Service satisfies it structurally (the same composition
// pattern email/label.Service uses for its MessageReader) -- GetByID already
// checks the message exists in the tenant AND that the caller is a member of
// its channel, which is exactly "can this user bookmark/see this message".
type MessageReader interface {
	GetByID(ctx context.Context, id, tenantID, userID uuid.UUID) (*models.MessageWithSender, error)
}

// Service handles bookmark business logic: toggling and listing.
type Service struct {
	repo     Repository
	messages MessageReader
}

// NewService creates a new bookmark service.
func NewService(repo Repository, messages MessageReader) *Service {
	return &Service{repo: repo, messages: messages}
}

// Toggle adds the bookmark if it does not exist, or removes it if it does.
// Returns the resulting state (true = now bookmarked). The message must
// exist and be visible to the caller -- reusing MessageReader.GetByID means
// a user can only bookmark a message they could otherwise read.
func (s *Service) Toggle(ctx context.Context, tenantID, userID, messageID uuid.UUID) (bool, error) {
	if _, err := s.messages.GetByID(ctx, messageID, tenantID, userID); err != nil {
		return false, err
	}

	exists, err := s.repo.Exists(ctx, tenantID, userID, messageID)
	if err != nil {
		return false, err
	}

	if exists {
		if err := s.repo.Remove(ctx, tenantID, userID, messageID); err != nil {
			return false, err
		}
		slog.Info("message bookmark removed", "message_id", messageID, "user_id", userID)
		return false, nil
	}

	if err := s.repo.Add(ctx, tenantID, userID, messageID); err != nil {
		return false, err
	}
	slog.Info("message bookmark added", "message_id", messageID, "user_id", userID)
	return true, nil
}

// List returns the bookmarked messages of a user, most recently bookmarked
// first. A bookmark whose message the user can no longer see (e.g. they left
// its channel since bookmarking) is skipped rather than failing the whole
// list -- the channel-membership check in GetByID is a live access check,
// not a one-time snapshot, so a revoked membership must also revoke the
// bookmark's visibility.
func (s *Service) List(ctx context.Context, tenantID, userID uuid.UUID) ([]*models.MessageWithSender, error) {
	ids, err := s.repo.ListMessageIDs(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}

	messages := make([]*models.MessageWithSender, 0, len(ids))
	for _, id := range ids {
		m, err := s.messages.GetByID(ctx, id, tenantID, userID)
		if err != nil {
			if errors.Is(err, message.ErrMessageNotFound) || errors.Is(err, message.ErrNotChannelMember) {
				continue
			}
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}
