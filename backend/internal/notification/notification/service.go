package notification

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/delivery"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/notification/preference"
)

// Service handles notification business logic including processing events,
// managing notification CRUD, and coordinating with preference evaluation.
type Service struct {
	repo        Repository
	prefService *preference.Service
	registry    *event.EventTypeRegistry
	grouper     *Grouper
	dispatcher  *delivery.Dispatcher
}

// NewService creates a new notification service.
func NewService(
	repo Repository,
	prefService *preference.Service,
	registry *event.EventTypeRegistry,
	grouper *Grouper,
	dispatcher *delivery.Dispatcher,
) *Service {
	return &Service{
		repo:        repo,
		prefService: prefService,
		registry:    registry,
		grouper:     grouper,
		dispatcher:  dispatcher,
	}
}

// ProcessEvent handles an incoming event from the event bus.
// This is the main event handler that:
// 1. Stores the event in the durability table
// 2. Determines target users
// 3. Evaluates preferences for each target
// 4. Groups and stores notifications
// 5. Dispatches delivery
func (s *Service) ProcessEvent(ctx context.Context, payload models.EventPayload) error {
	// Store event for durability
	evt := &models.Event{
		ID:           uuid.New(),
		EventTypeKey: payload.Type,
		ModuleID:     payload.ModuleID,
		Priority:     payload.Priority,
		Payload:      payload.Payload,
		Processed:    false,
		CreatedAt:    payload.Timestamp,
	}

	if payload.ActorID != "" {
		actorID, err := uuid.Parse(payload.ActorID)
		if err == nil {
			evt.ActorID = &actorID
		}
	}
	if payload.ResourceID != "" {
		evt.ResourceID = &payload.ResourceID
	}

	if err := s.repo.CreateEvent(ctx, evt); err != nil {
		slog.Error("failed to persist event",
			"event_type", payload.Type,
			"error", err,
		)
		// Continue processing even if durability write fails
	}

	// Determine priority - use event type default if not specified
	priority := payload.Priority
	if priority == "" {
		if et := s.registry.Get(payload.Type); et != nil {
			priority = et.DefaultPriority
		} else {
			priority = models.PriorityNormal
		}
	}

	// Process for each target user
	for _, targetUserIDStr := range payload.TargetUserIDs {
		targetUserID, err := uuid.Parse(targetUserIDStr)
		if err != nil {
			slog.Warn("invalid target user ID",
				"user_id", targetUserIDStr,
				"error", err,
			)
			continue
		}

		// Skip self-notifications (actor notifying themselves)
		if payload.ActorID == targetUserIDStr {
			continue
		}

		// Evaluate preferences (tenant comes from the notification payload's TenantID context)
		decision, err := s.prefService.Evaluate(ctx, payload.TenantID, targetUserID, payload)
		if err != nil {
			slog.Error("preference evaluation failed",
				"user_id", targetUserID,
				"event_type", payload.Type,
				"error", err,
			)
			continue
		}

		if !decision.Deliver {
			slog.Debug("notification suppressed by preferences",
				"user_id", targetUserID,
				"event_type", payload.Type,
				"reason", decision.Reason,
			)
			continue
		}

		// Build notification
		notif := &models.Notification{
			UserID:       targetUserID,
			EventTypeKey: payload.Type,
			ModuleID:     payload.ModuleID,
			Priority:     priority,
			Title:        payload.Title,
			IsRead:       false,
			CreatedAt:    time.Now(),
		}

		if payload.ActorID != "" {
			actorID, _ := uuid.Parse(payload.ActorID)
			notif.ActorID = &actorID
		}
		if payload.ResourceID != "" {
			notif.ResourceID = &payload.ResourceID
		}
		if payload.Body != "" {
			notif.Body = &payload.Body
		}
		if payload.DeepLink != "" {
			notif.DeepLink = &payload.DeepLink
		}
		if payload.GroupKey != "" {
			notif.GroupKey = &payload.GroupKey
		}

		// Group and store
		result, err := s.grouper.ProcessNotification(ctx, notif)
		if err != nil {
			slog.Error("failed to process notification",
				"user_id", targetUserID,
				"event_type", payload.Type,
				"error", err,
			)
			continue
		}

		// Dispatch delivery
		s.dispatcher.Dispatch(ctx, result.Notification, decision)

		// Signal gateway for WebSocket push via pg_notify
		s.notifyDelivery(ctx, result.Notification)
	}

	// Mark event as processed
	if err := s.repo.MarkEventProcessed(ctx, evt.ID.String()); err != nil {
		slog.Error("failed to mark event as processed",
			"event_id", evt.ID,
			"error", err,
		)
	}

	return nil
}

// ListNotifications returns paginated notifications for a user within a tenant.
func (s *Service) ListNotifications(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, moduleID *string, isRead *bool, page, pageSize int) ([]*models.Notification, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize
	filter := ListFilter{
		TenantID: tenantID,
		UserID:   userID,
		ModuleID: moduleID,
		IsRead:   isRead,
	}

	return s.repo.List(ctx, filter, offset, pageSize)
}

// GetUnreadCount returns the number of unread notifications for a user within a tenant.
func (s *Service) GetUnreadCount(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID) (int, error) {
	return s.repo.GetUnreadCount(ctx, tenantID, userID)
}

// MarkRead marks a single notification as read. Returns the updated notification.
func (s *Service) MarkRead(ctx context.Context, tenantID uuid.UUID, notifID, userID uuid.UUID) (*models.Notification, error) {
	notif, err := s.repo.GetByID(ctx, tenantID, notifID)
	if err != nil {
		return nil, err
	}

	if notif.UserID != userID {
		return nil, ErrUnauthorized
	}

	now := time.Now()
	if err := s.repo.MarkRead(ctx, tenantID, notifID, now); err != nil {
		return nil, err
	}

	notif.IsRead = true
	notif.ReadAt = &now
	return notif, nil
}

// MarkAllRead marks all unread notifications as read for a user within a tenant.
func (s *Service) MarkAllRead(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, moduleID *string) (int, error) {
	return s.repo.MarkAllRead(ctx, tenantID, userID, moduleID, time.Now())
}

// notifyDelivery sends a lightweight payload via pg_notify on the delivery channel
// so the gateway can push the notification to the user's WebSocket connection.
func (s *Service) notifyDelivery(ctx context.Context, notif *models.Notification) {
	type deliveryPayload struct {
		NotificationID string `json:"notification_id"`
		UserID         string `json:"user_id"`
		Title          string `json:"title"`
		EventTypeKey   string `json:"event_type_key"`
		ModuleID       string `json:"module_id"`
		Priority       string `json:"priority"`
	}

	payload := deliveryPayload{
		NotificationID: notif.ID.String(),
		UserID:         notif.UserID.String(),
		Title:          notif.Title,
		EventTypeKey:   notif.EventTypeKey,
		ModuleID:       notif.ModuleID,
		Priority:       notif.Priority,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal delivery payload", "error", err)
		return
	}

	if err := s.repo.NotifyDelivery(ctx, string(data)); err != nil {
		slog.Error("failed to send delivery notification",
			"notification_id", notif.ID,
			"user_id", notif.UserID,
			"error", err,
		)
	}
}
