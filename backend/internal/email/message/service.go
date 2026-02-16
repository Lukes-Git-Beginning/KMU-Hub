package message

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles email message business logic.
type Service struct {
	repo       Repository
	folderRepo FolderRepository
}

// NewService creates a new message service.
func NewService(repo Repository, folderRepo FolderRepository) *Service {
	return &Service{
		repo:       repo,
		folderRepo: folderRepo,
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

// GetByID returns a full message by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.EmailMessage, error) {
	return s.repo.GetByID(ctx, id)
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
func (s *Service) ToggleStar(ctx context.Context, id uuid.UUID) error {
	msg, err := s.repo.GetByID(ctx, id)
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
	return s.repo.Delete(ctx, id)
}

// GetFolderByID returns a folder by ID.
func (s *Service) GetFolderByID(ctx context.Context, id uuid.UUID) (*models.EmailFolder, error) {
	return s.folderRepo.GetByID(ctx, id)
}

// ListFolders returns all folders for an account.
func (s *Service) ListFolders(ctx context.Context, accountID uuid.UUID) ([]*models.EmailFolder, error) {
	return s.folderRepo.ListByAccount(ctx, accountID)
}
