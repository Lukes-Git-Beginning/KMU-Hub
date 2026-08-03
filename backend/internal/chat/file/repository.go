package file

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for file persistence
type Repository interface {
	// File CRUD
	CreateFile(ctx context.Context, file *models.ChatFile) error
	GetFileByID(ctx context.Context, id uuid.UUID) (*models.ChatFile, error)
	ListChannelFiles(ctx context.Context, channelID uuid.UUID, limit, offset int) ([]*models.ChatFileWithUploader, int, error)
	GetFilesByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]models.ChatFileWithUploader, error)
	DeleteFile(ctx context.Context, id uuid.UUID) error // soft delete + set deleted_at

	// Channel checks
	IsChannelMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error)
	IsChannelArchived(ctx context.Context, channelID uuid.UUID) (bool, error)
	GetChannelRole(ctx context.Context, channelID, userID uuid.UUID) (models.ChannelRole, error)

	// Quota (tenant-scoped since 000275)
	GetStorageQuota(ctx context.Context, tenantID uuid.UUID) (*models.StorageQuota, error)
	IncrementUsedBytes(ctx context.Context, tenantID uuid.UUID, bytes int64) error
	DecrementUsedBytes(ctx context.Context, tenantID uuid.UUID, bytes int64) error

	// User info
	GetUserInfo(ctx context.Context, userID uuid.UUID) (firstName, lastName string, err error)
}
