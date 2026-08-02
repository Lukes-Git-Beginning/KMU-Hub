package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// NotificationClient is a local interface for the notification service to avoid import cycles.
type NotificationClient interface {
	// ListNotifications fetches notifications for a user since a given time.
	ListNotifications(ctx context.Context, userID uuid.UUID, since time.Time) ([]NotificationData, error)
	// MarkRead marks a notification as read.
	MarkRead(ctx context.Context, notificationID string, userID uuid.UUID) error
}

// NotificationData represents the raw data fetched from the notification service.
type NotificationData struct {
	NotificationID string
	Title          string
	Body           string
	ModuleName     string
	DeepLink       string
	IsActionable   bool
	ReceivedAt     time.Time
}

// NotificationAdapter implements ChannelAdapter for system notifications.
type NotificationAdapter struct {
	client NotificationClient
}

// Compile-time assertion that NotificationAdapter implements ChannelAdapter.
var _ ChannelAdapter = (*NotificationAdapter)(nil)

// NewNotificationAdapter creates a new NotificationAdapter.
// If client is nil, the adapter degrades gracefully (returns empty results).
func NewNotificationAdapter(client NotificationClient) *NotificationAdapter {
	return &NotificationAdapter{client: client}
}

// Channel returns "notification".
func (a *NotificationAdapter) Channel() string {
	return "notification"
}

// FetchNewMessages creates InboxMessages from notification data.
// If the notification client is nil, returns an empty slice (graceful degradation).
func (a *NotificationAdapter) FetchNewMessages(ctx context.Context, userID uuid.UUID, since time.Time) ([]models.InboxMessage, error) {
	if a.client == nil {
		slog.Debug("notification adapter: client is nil, returning empty results")
		return nil, nil
	}

	notifications, err := a.client.ListNotifications(ctx, userID, since)
	if err != nil {
		return nil, fmt.Errorf("notification adapter fetch: %w", err)
	}

	result := make([]models.InboxMessage, 0, len(notifications))
	for _, notif := range notifications {
		preview := notif.Body
		if len(preview) > 200 {
			preview = preview[:200]
		}

		senderName := "System"
		if notif.ModuleName != "" {
			senderName = notif.ModuleName
		}

		inboxMsg := models.InboxMessage{
			ID:         uuid.New(),
			UserID:     userID,
			Channel:    "notification",
			SourceID:   notif.NotificationID,
			SenderName: senderName,
			Subject:    notif.Title,
			Preview:    preview,
			DeepLink:   notif.DeepLink,
			ReceivedAt: notif.ReceivedAt,
		}
		result = append(result, inboxMsg)
	}

	return result, nil
}

// HandleReply handles reply actions for notifications.
// Most notifications are not replyable. For actionable notifications (e.g., leave approval),
// the action is routed via the deep_link to the appropriate module.
func (a *NotificationAdapter) HandleReply(_ context.Context, _ string, _ uuid.UUID, _ string) error {
	// Notifications are generally not replyable through the inbox.
	// Actionable notifications direct users to their deep_link for action.
	return nil
}

// HandleForward is not supported for notifications — they have no external
// recipient concept; actionable notifications direct users via deep_link.
func (a *NotificationAdapter) HandleForward(_ context.Context, _ string, _ uuid.UUID, _ string, _ string) error {
	return ErrForwardNotSupported
}

// MarkReadOnSource marks a notification as read on the notification service.
func (a *NotificationAdapter) MarkReadOnSource(ctx context.Context, sourceID string, userID uuid.UUID) error {
	if a.client == nil {
		slog.Debug("notification adapter: client is nil, skipping mark read on source")
		return nil
	}
	return a.client.MarkRead(ctx, sourceID, userID)
}
