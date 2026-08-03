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

// SetStarred sets the starred state explicitly, unlike ToggleStar which flips
// whatever the current value is. Bulk star/unstar need every selected message
// to land in the same state regardless of where it started.
func (s *Service) SetStarred(ctx context.Context, id uuid.UUID, starred bool) error {
	return s.repo.UpdateFlags(ctx, id, nil, &starred)
}

// bulkActions are the actions BulkAction accepts. "move" additionally requires
// target; the rest ignore it (archive/spam resolve their own target below).
var bulkActions = map[string]bool{
	"read": true, "unread": true, "star": true, "unstar": true,
	"archive": true, "spam": true, "move": true, "delete": true,
}

// BulkAction applies action to every message in ids that belongs to tenantID.
// An id that does not exist or belongs to another tenant is silently skipped
// — not counted in the returned affected total, and not treated as a fatal
// error for the rest of the batch. archive/spam resolve their destination
// folder per message from that message's OWN account (a selection spanning
// the unified inbox can touch several accounts, each with its own Archive/
// Spam folder) and are skipped, not failed, if that account has none.
func (s *Service) BulkAction(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, target string) (int, error) {
	if !bulkActions[action] {
		return 0, ErrUnknownBulkAction
	}

	var moveFolderID uuid.UUID
	if action == "move" {
		if target == "" {
			return 0, ErrBulkTargetRequired
		}
		parsed, err := uuid.Parse(target)
		if err != nil {
			return 0, ErrBulkTargetRequired
		}
		if _, err := s.folderRepo.GetByID(ctx, parsed); err != nil {
			return 0, err
		}
		moveFolderID = parsed
	}

	folderType := ""
	switch action {
	case "archive":
		folderType = models.FolderTypeArchive
	case "spam":
		folderType = models.FolderTypeSpam
	}
	resolvedAccountFolders := make(map[uuid.UUID]uuid.UUID)

	affected := 0
	for _, id := range ids {
		msg, err := s.repo.GetByID(ctx, id, tenantID)
		if err != nil {
			continue
		}

		switch action {
		case "read":
			err = s.MarkRead(ctx, id)
		case "unread":
			err = s.MarkUnread(ctx, id)
		case "star":
			err = s.SetStarred(ctx, id, true)
		case "unstar":
			err = s.SetStarred(ctx, id, false)
		case "delete":
			err = s.Delete(ctx, id)
		case "move":
			err = s.MoveToFolder(ctx, id, moveFolderID)
		case "archive", "spam":
			folderID, ok := resolvedAccountFolders[msg.AccountID]
			if !ok {
				f, ferr := s.folderRepo.GetByAccountAndType(ctx, msg.AccountID, folderType)
				if ferr != nil {
					continue
				}
				folderID = f.ID
				resolvedAccountFolders[msg.AccountID] = folderID
			}
			err = s.MoveToFolder(ctx, id, folderID)
		}
		if err != nil {
			continue
		}
		affected++
	}

	return affected, nil
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
