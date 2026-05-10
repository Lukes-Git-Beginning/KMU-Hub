package message

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/inbox/adapter"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles inbox message business logic including CRUD, snooze,
// reply proxy, and the background snooze worker.
type Service struct {
	repo     Repository
	adapters *adapter.AdapterRegistry
}

// NewService creates a new inbox message service.
func NewService(repo Repository, adapters *adapter.AdapterRegistry) *Service {
	return &Service{
		repo:     repo,
		adapters: adapters,
	}
}

// Create persists a new inbox message after checking for duplicates.
// If a duplicate exists (same user, channel, source_id), the existing message's
// subject and preview are updated if the new message is newer.
func (s *Service) Create(ctx context.Context, msg *models.InboxMessage) error {
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	if msg.Tags == nil {
		msg.Tags = []string{}
	}

	// Check for duplicate via source_id
	existing, err := s.repo.GetBySourceID(ctx, msg.UserID, msg.Channel, msg.SourceID)
	if err != nil {
		return fmt.Errorf("dedup check: %w", err)
	}

	if existing != nil {
		// Update subject/preview if newer
		if msg.ReceivedAt.After(existing.ReceivedAt) {
			existing.Subject = msg.Subject
			existing.Preview = msg.Preview
			existing.ReceivedAt = msg.ReceivedAt
			if err := s.repo.Update(ctx, existing); err != nil {
				return fmt.Errorf("update existing duplicate: %w", err)
			}
		}
		return ErrDuplicateMessage
	}

	return s.repo.Create(ctx, msg)
}

// GetByID retrieves an inbox message by its ID, scoped to the tenant.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.InboxMessage, error) {
	return s.repo.GetByID(ctx, id, tenantID)
}

// List retrieves inbox messages with filtering and pagination.
func (s *Service) List(ctx context.Context, filter ListFilter) ([]*models.InboxMessage, int, error) {
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 50
	}
	return s.repo.List(ctx, filter)
}

// MarkRead marks an inbox message as read and syncs back to the originating system.
// tenantID is used for the initial GetByID isolation check.
func (s *Service) MarkRead(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	msg, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	if err := s.repo.MarkRead(ctx, id); err != nil {
		return err
	}

	// Sync read status back to the originating system
	if a, ok := s.adapters.Get(msg.Channel); ok {
		if err := a.MarkReadOnSource(ctx, msg.SourceID, msg.UserID); err != nil {
			slog.Warn("failed to sync read status to source",
				"channel", msg.Channel,
				"source_id", msg.SourceID,
				"error", err,
			)
			// Non-fatal: don't fail the local operation
		}
	}

	return nil
}

// MarkUnread marks an inbox message as unread.
func (s *Service) MarkUnread(ctx context.Context, id uuid.UUID) error {
	return s.repo.MarkUnread(ctx, id)
}

// ToggleStar toggles the starred status of an inbox message.
func (s *Service) ToggleStar(ctx context.Context, id uuid.UUID) error {
	return s.repo.ToggleStar(ctx, id)
}

// Archive archives an inbox message.
func (s *Service) Archive(ctx context.Context, id uuid.UUID) error {
	return s.repo.Archive(ctx, id)
}

// Unarchive unarchives an inbox message.
func (s *Service) Unarchive(ctx context.Context, id uuid.UUID) error {
	return s.repo.Unarchive(ctx, id)
}

// Snooze snoozes an inbox message until the given time.
// The snooze time must be in the future.
func (s *Service) Snooze(ctx context.Context, id uuid.UUID, until time.Time) error {
	if until.Before(time.Now()) {
		return ErrInvalidSnoozeTime
	}
	return s.repo.Snooze(ctx, id, until)
}

// Reply routes a reply back through the originating channel's adapter.
// tenantID is used for the initial GetByID isolation check.
func (s *Service) Reply(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, userID uuid.UUID, body string) error {
	msg, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	a, ok := s.adapters.Get(msg.Channel)
	if !ok {
		return ErrAdapterNotFound
	}

	return a.HandleReply(ctx, msg.ID, userID, body)
}

// AssignMessage assigns an inbox message to a user.
// Returns ErrAlreadyAssigned if the message is already assigned.
func (s *Service) AssignMessage(ctx context.Context, id uuid.UUID, assigneeID uuid.UUID) error {
	assigned, err := s.repo.AssignMessage(ctx, id, assigneeID)
	if err != nil {
		return err
	}
	if !assigned {
		return ErrAlreadyAssigned
	}
	return nil
}

// GetUnreadCounts returns unread counts grouped by channel for a user.
func (s *Service) GetUnreadCounts(ctx context.Context, userID uuid.UUID) ([]models.UnreadCount, error) {
	return s.repo.GetUnreadCounts(ctx, userID)
}

// BulkMarkRead marks multiple inbox messages as read.
func (s *Service) BulkMarkRead(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	return s.repo.BulkMarkRead(ctx, ids)
}

// BulkArchive archives multiple inbox messages.
func (s *Service) BulkArchive(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	return s.repo.BulkArchive(ctx, ids)
}

// Update updates an inbox message.
func (s *Service) Update(ctx context.Context, msg *models.InboxMessage) error {
	return s.repo.Update(ctx, msg)
}

// StartSnoozeWorker runs a background goroutine that polls for expired snoozed items
// and un-snoozes them. On unsnooze, items reappear as unread in the inbox.
// The worker emits a pg_notify event so the gateway can push WebSocket updates.
func (s *Service) StartSnoozeWorker(ctx context.Context, interval time.Duration) {
	ctx = database.WithSystemContext(ctx)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		slog.Info("inbox snooze worker started", "interval", interval)

		for {
			select {
			case <-ctx.Done():
				slog.Info("inbox snooze worker stopped")
				return
			case <-ticker.C:
				count, err := s.repo.UnsnoozeExpired(ctx)
				if err != nil {
					slog.Error("snooze worker: failed to unsnooze expired items",
						"error", err,
					)
					continue
				}
				if count > 0 {
					slog.Info("snooze worker: unsnoozed expired items",
						"count", count,
					)

					// Emit notification so gateway can push WebSocket updates
					payload := struct {
						Type  string `json:"type"`
						Count int    `json:"count"`
					}{
						Type:  "inbox.snooze.expired",
						Count: count,
					}
					data, err := json.Marshal(payload)
					if err == nil {
						if err := s.repo.NotifyDelivery(ctx, string(data)); err != nil {
							slog.Warn("snooze worker: failed to send delivery notification",
								"error", err,
							)
						}
					}
				}
			}
		}
	}()
}
