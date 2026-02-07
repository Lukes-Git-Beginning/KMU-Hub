package notification

import (
	"context"
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

		// Evaluate preferences
		decision, err := s.prefService.Evaluate(ctx, targetUserID, payload)
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

// ListNotifications returns paginated notifications for a user.
func (s *Service) ListNotifications(ctx context.Context, userID uuid.UUID, moduleID *string, isRead *bool, page, pageSize int) ([]*models.Notification, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize
	filter := ListFilter{
		UserID:   userID,
		ModuleID: moduleID,
		IsRead:   isRead,
	}

	return s.repo.List(ctx, filter, offset, pageSize)
}

// GetUnreadCount returns the number of unread notifications for a user.
func (s *Service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.GetUnreadCount(ctx, userID)
}

// MarkRead marks a single notification as read. Returns the updated notification.
func (s *Service) MarkRead(ctx context.Context, notifID, userID uuid.UUID) (*models.Notification, error) {
	notif, err := s.repo.GetByID(ctx, notifID)
	if err != nil {
		return nil, err
	}

	if notif.UserID != userID {
		return nil, ErrUnauthorized
	}

	now := time.Now()
	if err := s.repo.MarkRead(ctx, notifID, now); err != nil {
		return nil, err
	}

	notif.IsRead = true
	notif.ReadAt = &now
	return notif, nil
}

// MarkAllRead marks all unread notifications as read for a user.
func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID, moduleID *string) (int, error) {
	return s.repo.MarkAllRead(ctx, userID, moduleID, time.Now())
}
