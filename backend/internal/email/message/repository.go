package message

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ListOpts contains pagination and sorting options.
type ListOpts struct {
	Page    int
	PerPage int
	SortBy  string
	SortDir string
}

// Repository defines the interface for email message persistence.
type Repository interface {
	Create(ctx context.Context, msg *models.EmailMessage) error
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailMessage, error)
	GetByFolderUID(ctx context.Context, folderID uuid.UUID, uid uint32) (*models.EmailMessage, error)
	ListByFolder(ctx context.Context, folderID uuid.UUID, opts ListOpts) ([]*models.EmailMessage, int, error)
	ListByThread(ctx context.Context, threadID uuid.UUID) ([]*models.EmailMessage, error)
	Search(ctx context.Context, accountID uuid.UUID, query string, opts ListOpts) ([]*models.EmailMessage, int, error)
	UpdateFlags(ctx context.Context, id uuid.UUID, isRead, isStarred *bool) error
	UpdateThreadID(ctx context.Context, id uuid.UUID, threadID uuid.UUID) error
	MoveToFolder(ctx context.Context, id uuid.UUID, folderID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByFolder(ctx context.Context, folderID uuid.UUID) error
	GetHighestUID(ctx context.Context, folderID uuid.UUID) (uint32, error)
	GetByMessageIDHeader(ctx context.Context, messageIDHeader string) (*models.EmailMessage, error)
	FindThreadByReferences(ctx context.Context, accountID uuid.UUID, references []string, inReplyTo string) (uuid.UUID, error)
	CountUnreadByFolder(ctx context.Context, folderID uuid.UUID) (int, error)
	FindBySubjectAndParticipants(ctx context.Context, accountID uuid.UUID, normalizedSubject string, fromEmail string, since, until interface{}) (*models.EmailMessage, error)
}

// FolderRepository defines the interface for email folder persistence.
type FolderRepository interface {
	Create(ctx context.Context, folder *models.EmailFolder) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.EmailFolder, error)
	GetByIMAPName(ctx context.Context, accountID uuid.UUID, imapName string) (*models.EmailFolder, error)
	ListByAccount(ctx context.Context, accountID uuid.UUID) ([]*models.EmailFolder, error)
	// GetByAccountAndType returns the first folder of the given type (e.g.
	// models.FolderTypeArchive) for an account, or ErrFolderNotFound if the
	// account has none — not every synced account has an Archive/Spam folder.
	GetByAccountAndType(ctx context.Context, accountID uuid.UUID, folderType string) (*models.EmailFolder, error)
	UpdateCounts(ctx context.Context, folderID uuid.UUID, messageCount, unreadCount int) error
	UpdateUIDValidity(ctx context.Context, folderID uuid.UUID, uidValidity int64) error
	DeleteMessagesByFolder(ctx context.Context, folderID uuid.UUID) error
}
