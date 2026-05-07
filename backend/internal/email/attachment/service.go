package attachment

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service manages attachment DB records and MinIO storage.
type Service struct {
	repo  Repository
	store ObjectStore
}

// NewService creates a new attachment service.
func NewService(repo Repository, store ObjectStore) *Service {
	return &Service{
		repo:  repo,
		store: store,
	}
}

// CreateFromStream uploads an attachment to MinIO and creates a DB record.
func (s *Service) CreateFromStream(
	ctx context.Context,
	messageID uuid.UUID,
	accountID uuid.UUID,
	messageUID uint32,
	filename, contentType string,
	size int64,
	reader io.Reader,
	contentID string,
	isInline bool,
	tenantID uuid.UUID,
) (*models.EmailAttachment, error) {
	// Upload to MinIO (streaming, no full memory buffering)
	minioKey, err := s.store.Upload(ctx, accountID, messageUID, filename, reader, size, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload attachment: %w", err)
	}

	// Create DB record
	att := &models.EmailAttachment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		MessageID:   messageID,
		Filename:    filename,
		ContentType: contentType,
		SizeBytes:   size,
		MinIOKey:    minioKey,
		ContentID:   contentID,
		IsInline:    isInline,
		CreatedAt:   time.Now().UTC(),
	}

	if createErr := s.repo.Create(ctx, att); createErr != nil {
		// Cleanup MinIO on DB failure
		_ = s.store.Delete(ctx, minioKey)
		return nil, fmt.Errorf("failed to create attachment record: %w", createErr)
	}

	slog.Info("attachment created",
		"attachment_id", att.ID,
		"message_id", messageID,
		"tenant_id", tenantID,
		"filename", filename,
		"size", size,
	)

	return att, nil
}

// GetByMessage returns all attachments for a message scoped to a tenant.
func (s *Service) GetByMessage(ctx context.Context, messageID uuid.UUID, tenantID uuid.UUID) ([]*models.EmailAttachment, error) {
	return s.repo.GetByMessage(ctx, messageID, tenantID)
}

// GetDownloadURL returns a presigned URL for downloading an attachment.
func (s *Service) GetDownloadURL(ctx context.Context, attachmentID uuid.UUID, tenantID uuid.UUID) (string, error) {
	minioKey, err := s.repo.GetMinIOKeyByID(ctx, attachmentID, tenantID)
	if err != nil {
		return "", err
	}

	return s.store.GetPresignedURL(ctx, minioKey, 0)
}
