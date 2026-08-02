package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ChatClient is a local interface for the chat gRPC service to avoid import cycles.
type ChatClient interface {
	// ListDMsAndMentions fetches DMs and @mentions for a user since a given time.
	ListDMsAndMentions(ctx context.Context, userID uuid.UUID, since time.Time) ([]ChatMessageData, error)
	// CreateMessage sends a reply message in a chat channel.
	CreateMessage(ctx context.Context, channelID string, userID uuid.UUID, body string) error
	// MarkRead marks a chat channel as read for a user.
	MarkRead(ctx context.Context, channelID string, userID uuid.UUID) error
}

// ChatMessageData represents the raw data fetched from the chat service.
type ChatMessageData struct {
	MessageID   string
	ChannelID   string
	ChannelName string
	IsDM        bool
	AuthorID    uuid.UUID
	AuthorName  string
	Body        string
	ReceivedAt  time.Time
}

// ChatAdapter implements ChannelAdapter for Chat DMs and @mentions only.
// Regular channel messages are not routed to the inbox (per research and context decisions).
type ChatAdapter struct {
	client ChatClient
}

// Compile-time assertion that ChatAdapter implements ChannelAdapter.
var _ ChannelAdapter = (*ChatAdapter)(nil)

// NewChatAdapter creates a new ChatAdapter.
// If client is nil, the adapter degrades gracefully (returns empty results).
func NewChatAdapter(client ChatClient) *ChatAdapter {
	return &ChatAdapter{client: client}
}

// Channel returns "chat".
func (a *ChatAdapter) Channel() string {
	return "chat"
}

// FetchNewMessages creates InboxMessages from chat DM/mention data.
// If the chat client is nil, returns an empty slice (graceful degradation).
func (a *ChatAdapter) FetchNewMessages(ctx context.Context, userID uuid.UUID, since time.Time) ([]models.InboxMessage, error) {
	if a.client == nil {
		slog.Debug("chat adapter: client is nil, returning empty results")
		return nil, nil
	}

	messages, err := a.client.ListDMsAndMentions(ctx, userID, since)
	if err != nil {
		return nil, fmt.Errorf("chat adapter fetch: %w", err)
	}

	result := make([]models.InboxMessage, 0, len(messages))
	for _, msg := range messages {
		preview := msg.Body
		if len(preview) > 200 {
			preview = preview[:200]
		}

		subject := msg.ChannelName
		if msg.IsDM {
			subject = "Direktnachricht"
		}

		senderID := msg.AuthorID
		inboxMsg := models.InboxMessage{
			ID:         uuid.New(),
			UserID:     userID,
			Channel:    "chat",
			SourceID:   msg.MessageID,
			SenderName: msg.AuthorName,
			SenderID:   &senderID,
			Subject:    subject,
			Preview:    preview,
			DeepLink:   fmt.Sprintf("/chat/%s", msg.ChannelID),
			ReceivedAt: msg.ReceivedAt,
		}
		result = append(result, inboxMsg)
	}

	return result, nil
}

// HandleReply sends a reply via the chat service.
func (a *ChatAdapter) HandleReply(ctx context.Context, sourceID string, userID uuid.UUID, body string) error {
	if a.client == nil {
		return fmt.Errorf("chat adapter: client not configured")
	}
	return a.client.CreateMessage(ctx, sourceID, userID, body)
}

// HandleForward is not supported for chat DMs/mentions — there is no concept
// of forwarding to an arbitrary external recipient in chat today.
func (a *ChatAdapter) HandleForward(_ context.Context, _ string, _ uuid.UUID, _ string, _ string) error {
	return ErrForwardNotSupported
}

// MarkReadOnSource marks a chat channel as read on the chat service.
func (a *ChatAdapter) MarkReadOnSource(ctx context.Context, sourceID string, userID uuid.UUID) error {
	if a.client == nil {
		slog.Debug("chat adapter: client is nil, skipping mark read on source")
		return nil
	}
	return a.client.MarkRead(ctx, sourceID, userID)
}
