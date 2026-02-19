package attachment

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

const (
	// defaultPresignExpiry is the default expiry for presigned download URLs.
	defaultPresignExpiry = 1 * time.Hour

	// keyPrefix is the MinIO key prefix for email attachments.
	keyPrefix = "email-attachments"
)

// Store handles streaming email attachments to/from MinIO.
type Store struct {
	client *minio.Client
	bucket string
}

// NewStore creates a new attachment store.
func NewStore(client *minio.Client, bucket string) *Store {
	return &Store{
		client: client,
		bucket: bucket,
	}
}

// Upload streams an attachment directly to MinIO (no full memory buffering).
// Returns the MinIO object key.
func (s *Store) Upload(ctx context.Context, accountID uuid.UUID, messageUID uint32, filename string, reader io.Reader, size int64, contentType string) (string, error) {
	key := fmt.Sprintf("%s/%s/%d/%s", keyPrefix, accountID.String(), messageUID, filename)

	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}

	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, opts)
	if err != nil {
		return "", fmt.Errorf("failed to upload attachment: %w", err)
	}

	slog.Info("attachment uploaded",
		"key", key,
		"size", size,
		"content_type", contentType,
	)

	return key, nil
}

// Download returns a reader for streaming download of an attachment.
func (s *Store) Download(ctx context.Context, minioKey string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, minioKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download attachment: %w", err)
	}

	return obj, nil
}

// GetPresignedURL generates a presigned GET URL for an attachment.
func (s *Store) GetPresignedURL(ctx context.Context, minioKey string, expiry time.Duration) (string, error) {
	if expiry == 0 {
		expiry = defaultPresignExpiry
	}

	url, err := s.client.PresignedGetObject(ctx, s.bucket, minioKey, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url.String(), nil
}

// Delete removes an attachment from MinIO.
func (s *Store) Delete(ctx context.Context, minioKey string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, minioKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	slog.Info("attachment deleted", "key", minioKey)
	return nil
}
