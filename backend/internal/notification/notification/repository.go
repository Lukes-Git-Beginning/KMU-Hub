package notification

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for notification persistence.
type Repository interface {
	// Create persists a new notification.
	Create(ctx context.Context, notif *models.Notification) error

	// GetByID retrieves a notification by its ID within a tenant.
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*models.Notification, error)

	// List retrieves notifications for a user with optional filtering.
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Notification, int, error)

	// GetUnreadCount returns the number of unread notifications for a user within a tenant.
	GetUnreadCount(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID) (int, error)

	// MarkRead marks a single notification as read within a tenant.
	MarkRead(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, readAt time.Time) error

	// MarkAllRead marks all unread notifications as read for a user within a tenant, optionally filtered by module.
	MarkAllRead(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, moduleID *string, readAt time.Time) (int, error)

	// MarkDeliveredDesktop marks a notification as delivered via desktop push within a tenant.
	MarkDeliveredDesktop(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error

	// FindRecentByGroupKey finds the most recent notification with the same group key
	// within a time window and tenant. Used for smart grouping.
	FindRecentByGroupKey(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, groupKey string, since time.Time) (*models.Notification, error)

	// IncrementGroupCount increments the group count of a notification.
	IncrementGroupCount(ctx context.Context, id uuid.UUID, newTitle string) error

	// CreateEvent persists an event to the durability table.
	CreateEvent(ctx context.Context, event *models.Event) error

	// ListUnprocessedEvents retrieves unprocessed events for catch-up.
	ListUnprocessedEvents(ctx context.Context, limit int) ([]models.Event, error)

	// MarkEventProcessed marks an event as processed.
	MarkEventProcessed(ctx context.Context, eventID string) error

	// SetPinned sets the is_pinned flag on a notification within a tenant.
	SetPinned(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, pinned bool) error

	// SetDismissed sets the is_dismissed flag on a notification within a tenant.
	SetDismissed(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, dismissed bool) error

	// Snooze sets snoozed_until on a notification within a tenant and marks it read.
	Snooze(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, until time.Time) error

	// NotifyDelivery emits a pg_notify on the notification_delivery channel
	// to signal the gateway to push a real-time WebSocket notification.
	NotifyDelivery(ctx context.Context, payload string) error
}

// ListFilter contains filtering options for listing notifications.
type ListFilter struct {
	TenantID    uuid.UUID
	UserID      uuid.UUID
	ModuleID    *string
	IsRead      *bool
	IsPinned    *bool
	IsDismissed *bool
}
