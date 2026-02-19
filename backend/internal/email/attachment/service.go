package attachment

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service manages attachment DB records and MinIO storage.
type Service struct {
	pool  *pgxpool.Pool
	store *Store
}

// NewService creates a new attachment service.
func NewService(pool *pgxpool.Pool, store *Store) *Service {
	return &Service{
		pool:  pool,
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
) (*models.EmailAttachment, error) {
	// Upload to MinIO (streaming, no full memory buffering)
	minioKey, err := s.store.Upload(ctx, accountID, messageUID, filename, reader, size, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload attachment: %w", err)
	}

	// Create DB record
	att := &models.EmailAttachment{
		ID:          uuid.New(),
		MessageID:   messageID,
		Filename:    filename,
		ContentType: contentType,
		SizeBytes:   size,
		MinIOKey:    minioKey,
		ContentID:   contentID,
		IsInline:    isInline,
		CreatedAt:   time.Now().UTC(),
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO email_attachments (id, message_id, filename, content_type,
			size_bytes, minio_key, content_id, is_inline, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		att.ID, att.MessageID, att.Filename, att.ContentType,
		att.SizeBytes, att.MinIOKey, att.ContentID, att.IsInline, att.CreatedAt,
	)
	if err != nil {
		// Cleanup MinIO on DB failure
		_ = s.store.Delete(ctx, minioKey)
		return nil, fmt.Errorf("failed to create attachment record: %w", err)
	}

	slog.Info("attachment created",
		"attachment_id", att.ID,
		"message_id", messageID,
		"filename", filename,
		"size", size,
	)

	return att, nil
}

// GetByMessage returns all attachments for a message.
func (s *Service) GetByMessage(ctx context.Context, messageID uuid.UUID) ([]*models.EmailAttachment, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, message_id, filename, content_type, size_bytes, minio_key,
			content_id, is_inline, created_at
		 FROM email_attachments WHERE message_id = $1
		 ORDER BY created_at ASC`, messageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []*models.EmailAttachment
	for rows.Next() {
		var att models.EmailAttachment
		if err := rows.Scan(&att.ID, &att.MessageID, &att.Filename, &att.ContentType,
			&att.SizeBytes, &att.MinIOKey, &att.ContentID, &att.IsInline,
			&att.CreatedAt); err != nil {
			return nil, err
		}
		attachments = append(attachments, &att)
	}
	return attachments, rows.Err()
}

// GetDownloadURL returns a presigned URL for downloading an attachment.
func (s *Service) GetDownloadURL(ctx context.Context, attachmentID uuid.UUID) (string, error) {
	var minioKey string
	err := s.pool.QueryRow(ctx,
		`SELECT minio_key FROM email_attachments WHERE id = $1`, attachmentID,
	).Scan(&minioKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrAttachmentNotFound
		}
		return "", err
	}

	return s.store.GetPresignedURL(ctx, minioKey, 0)
}
