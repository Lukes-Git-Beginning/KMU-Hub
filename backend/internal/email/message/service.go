package message

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
)

// Service handles email message business logic.
type Service struct {
	repo       Repository
	folderRepo FolderRepository
	emitter    EventEmitter
}

// NewService creates a new message service.
func NewService(repo Repository, folderRepo FolderRepository) *Service {
	return &Service{
		repo:       repo,
		folderRepo: folderRepo,
	}
}

// SetEventEmitter sets the optional event emitter for notification events.
func (s *Service) SetEventEmitter(emitter EventEmitter) {
	s.emitter = emitter
}

// emitEvent emits a notification event if the emitter is configured.
// Nil-safe: does nothing if no emitter is set.
func (s *Service) emitEvent(ctx context.Context, eventType, resourceID string, targetUserIDs []string, title, body, deepLink string) {
	if s.emitter == nil {
		return
	}
	payload := models.EventPayload{
		Type:          eventType,
		Priority:      "normal",
		ModuleID:      event.ModuleEmail,
		ResourceID:    resourceID,
		TargetUserIDs: targetUserIDs,
		Title:         title,
		Body:          body,
		DeepLink:      deepLink,
		Timestamp:     time.Now(),
	}
	if err := s.emitter.EmitEmailEvent(ctx, payload); err != nil {
		slog.Error("failed to emit email event",
			"type", eventType,
			"resource_id", resourceID,
			"error", err,
		)
	}
}

// Create stores a message and assigns a thread ID.
func (s *Service) Create(ctx context.Context, msg *models.EmailMessage) error {
	if err := s.repo.Create(ctx, msg); err != nil {
		return err
	}

	if err := s.AssignThread(ctx, msg); err != nil {
		slog.Warn("thread assignment failed, message stored without thread",
			"message_id", msg.ID,
			"error", err,
		)
	}

	s.emitEvent(ctx, event.EventEmailReceived, msg.ID.String(), nil,
		"E-Mail empfangen", msg.Subject, "/mail/"+msg.ID.String())

	return nil
}

// CreateSynced stores a synced message (used by sync engine).
// Same as Create but implements the sync.MessageSyncer interface.
func (s *Service) CreateSynced(ctx context.Context, msg *models.EmailMessage) error {
	return s.Create(ctx, msg)
}

// GetHighestUID returns the highest UID stored for a folder.
// Implements the sync.MessageSyncer interface.
func (s *Service) GetHighestUID(ctx context.Context, folderID uuid.UUID) (uint32, error) {
	return s.repo.GetHighestUID(ctx, folderID)
}

// GetByID returns a full message by ID, scoped to the tenant.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailMessage, error) {
	return s.repo.GetByID(ctx, id, tenantID)
}

// ListByFolder returns paginated messages for a folder.
func (s *Service) ListByFolder(ctx context.Context, folderID uuid.UUID, opts ListOpts) ([]*models.EmailMessage, int, error) {
	return s.repo.ListByFolder(ctx, folderID, opts)
}

// ListByThread returns all messages in a conversation thread.
func (s *Service) ListByThread(ctx context.Context, threadID uuid.UUID) ([]*models.EmailMessage, error) {
	return s.repo.ListByThread(ctx, threadID)
}

// Search performs full-text search across messages.
func (s *Service) Search(ctx context.Context, accountID uuid.UUID, query string, opts ListOpts) ([]*models.EmailMessage, int, error) {
	return s.repo.Search(ctx, accountID, query, opts)
}

// MarkRead marks a message as read.
func (s *Service) MarkRead(ctx context.Context, id uuid.UUID) error {
	isRead := true
	return s.repo.UpdateFlags(ctx, id, &isRead, nil)
}

// MarkUnread marks a message as unread.
func (s *Service) MarkUnread(ctx context.Context, id uuid.UUID) error {
	isRead := false
	return s.repo.UpdateFlags(ctx, id, &isRead, nil)
}

// ToggleStar toggles the starred state of a message.
// tenantID is used for the initial GetByID isolation check.
func (s *Service) ToggleStar(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	msg, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}
	toggled := !msg.IsStarred
	return s.repo.UpdateFlags(ctx, id, nil, &toggled)
}

// MoveToFolder moves a message to a different folder.
func (s *Service) MoveToFolder(ctx context.Context, id uuid.UUID, folderID uuid.UUID) error {
	return s.repo.MoveToFolder(ctx, id, folderID)
}

// Delete deletes a message.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	s.emitEvent(ctx, event.EventEmailDeleted, id.String(), nil,
		"E-Mail geloescht", "", "")

	return nil
}

// GetFolderByID returns a folder by ID.
func (s *Service) GetFolderByID(ctx context.Context, id uuid.UUID) (*models.EmailFolder, error) {
	return s.folderRepo.GetByID(ctx, id)
}

// ListFolders returns all folders for an account.
func (s *Service) ListFolders(ctx context.Context, accountID uuid.UUID) ([]*models.EmailFolder, error) {
	return s.folderRepo.ListByAccount(ctx, accountID)
}
