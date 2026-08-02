package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// EmailClient is a local interface for the email gRPC service to avoid import cycles.
// The actual implementation will be wired when the inbox consumer is registered.
type EmailClient interface {
	// ListMessages fetches email messages for a user since a given time.
	ListMessages(ctx context.Context, userID uuid.UUID, since time.Time) ([]EmailMessageData, error)
	// SendReply sends a reply to an email thread.
	SendReply(ctx context.Context, threadID string, userID uuid.UUID, body string) error
	// ForwardEmail forwards an email message to a new recipient with an optional note.
	ForwardEmail(ctx context.Context, messageID string, userID uuid.UUID, to string, note string) error
	// MarkRead marks an email thread as read.
	MarkRead(ctx context.Context, threadID string, userID uuid.UUID) error
}

// EmailMessageData represents the raw data fetched from the email service.
type EmailMessageData struct {
	ThreadID     string
	FromName     string
	FromEmail    string
	Subject      string
	Body         string
	ReceivedAt   time.Time
	MessageID    string
}

// EmailAdapter implements ChannelAdapter for email messages.
type EmailAdapter struct {
	client EmailClient
}

// Compile-time assertion that EmailAdapter implements ChannelAdapter.
var _ ChannelAdapter = (*EmailAdapter)(nil)

// NewEmailAdapter creates a new EmailAdapter.
// If client is nil, the adapter degrades gracefully (returns empty results).
func NewEmailAdapter(client EmailClient) *EmailAdapter {
	return &EmailAdapter{client: client}
}

// Channel returns "email".
func (a *EmailAdapter) Channel() string {
	return "email"
}

// FetchNewMessages creates InboxMessages from email data.
// If the email client is nil, returns an empty slice (graceful degradation).
func (a *EmailAdapter) FetchNewMessages(ctx context.Context, userID uuid.UUID, since time.Time) ([]models.InboxMessage, error) {
	if a.client == nil {
		slog.Debug("email adapter: client is nil, returning empty results")
		return nil, nil
	}

	messages, err := a.client.ListMessages(ctx, userID, since)
	if err != nil {
		return nil, fmt.Errorf("email adapter fetch: %w", err)
	}

	result := make([]models.InboxMessage, 0, len(messages))
	for _, msg := range messages {
		preview := msg.Body
		if len(preview) > 200 {
			preview = preview[:200]
		}

		email := msg.FromEmail
		inboxMsg := models.InboxMessage{
			ID:          uuid.New(),
			UserID:      userID,
			Channel:     "email",
			SourceID:    msg.ThreadID, // One inbox item per thread
			SenderName:  msg.FromName,
			SenderEmail: &email,
			Subject:     msg.Subject,
			Preview:     preview,
			DeepLink:    fmt.Sprintf("/mails/%s", msg.MessageID),
			ReceivedAt:  msg.ReceivedAt,
		}
		result = append(result, inboxMsg)
	}

	return result, nil
}

// HandleReply sends a reply via the email service.
func (a *EmailAdapter) HandleReply(ctx context.Context, sourceID string, userID uuid.UUID, body string) error {
	if a.client == nil {
		return fmt.Errorf("email adapter: client not configured")
	}
	return a.client.SendReply(ctx, sourceID, userID, body)
}

// HandleForward forwards an email message to a new recipient via the email service.
func (a *EmailAdapter) HandleForward(ctx context.Context, sourceID string, userID uuid.UUID, to string, note string) error {
	if a.client == nil {
		return fmt.Errorf("email adapter: client not configured")
	}
	return a.client.ForwardEmail(ctx, sourceID, userID, to, note)
}

// MarkReadOnSource marks an email thread as read on the email service.
func (a *EmailAdapter) MarkReadOnSource(ctx context.Context, sourceID string, userID uuid.UUID) error {
	if a.client == nil {
		slog.Debug("email adapter: client is nil, skipping mark read on source")
		return nil
	}
	return a.client.MarkRead(ctx, sourceID, userID)
}
