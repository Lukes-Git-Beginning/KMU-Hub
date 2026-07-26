package message

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ListFilter contains filtering options for listing inbox messages.
type ListFilter struct {
	TenantID    uuid.UUID // Required: always filters by tenant
	UserID      uuid.UUID
	Channel     *string
	IsRead      *bool
	IsStarred   *bool
	IsArchived  *bool
	Status      *string
	TeamInboxID *uuid.UUID
	Search      *string
	PageSize    int
	// Cursor-based pagination using (received_at, id).
	CursorReceivedAt *time.Time
	CursorID         *uuid.UUID
}

// Repository defines the interface for inbox message persistence.
type Repository interface {
	// Create persists a new inbox message.
	Create(ctx context.Context, msg *models.InboxMessage) error

	// GetByID retrieves an inbox message by its ID, scoped to the tenant.
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.InboxMessage, error)

	// List retrieves inbox messages with filtering and cursor-based pagination.
	// Returns messages and total count.
	List(ctx context.Context, filter ListFilter) ([]*models.InboxMessage, int, error)

	// Update updates an existing inbox message.
	Update(ctx context.Context, msg *models.InboxMessage) error

	// MarkRead marks an inbox message as read.
	MarkRead(ctx context.Context, id uuid.UUID) error

	// MarkUnread marks an inbox message as unread.
	MarkUnread(ctx context.Context, id uuid.UUID) error

	// ToggleStar toggles the starred status of an inbox message.
	ToggleStar(ctx context.Context, id uuid.UUID) error

	// SetStatus sets the conversation-level status of an inbox message.
	SetStatus(ctx context.Context, id uuid.UUID, status string) error

	// AddTag adds a tag to an inbox message. Idempotent: adding an existing tag is a no-op.
	AddTag(ctx context.Context, id uuid.UUID, tag string) error

	// RemoveTag removes a tag from an inbox message. Idempotent: removing an absent tag is a no-op.
	RemoveTag(ctx context.Context, id uuid.UUID, tag string) error

	// Archive archives an inbox message.
	Archive(ctx context.Context, id uuid.UUID) error

	// Unarchive unarchives an inbox message.
	Unarchive(ctx context.Context, id uuid.UUID) error

	// Snooze snoozes an inbox message until the given time.
	Snooze(ctx context.Context, id uuid.UUID, until time.Time) error

	// Unsnooze removes the snooze on an inbox message and marks it unread.
	Unsnooze(ctx context.Context, id uuid.UUID) error

	// UnsnoozeExpired unsnoozes all messages with expired snooze times.
	// Returns the number of messages unsnoozed.
	UnsnoozeExpired(ctx context.Context) (int, error)

	// AssignMessage atomically assigns a message to a user.
	// Returns false if the message is already assigned (WHERE assigned_to IS NULL).
	AssignMessage(ctx context.Context, id uuid.UUID, assigneeID uuid.UUID) (bool, error)

	// GetUnreadCounts returns unread counts grouped by channel for a user.
	GetUnreadCounts(ctx context.Context, userID uuid.UUID) ([]models.UnreadCount, error)

	// BulkMarkRead marks multiple inbox messages as read.
	BulkMarkRead(ctx context.Context, ids []uuid.UUID) error

	// BulkArchive archives multiple inbox messages.
	BulkArchive(ctx context.Context, ids []uuid.UUID) error

	// GetBySourceID retrieves an inbox message by user, channel, and source ID.
	// Used for deduplication checks.
	GetBySourceID(ctx context.Context, userID uuid.UUID, channel, sourceID string) (*models.InboxMessage, error)

	// NotifyDelivery emits a pg_notify on the notification_delivery channel
	// for WebSocket push to the user's client.
	NotifyDelivery(ctx context.Context, payload string) error
}
