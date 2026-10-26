package file

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"
)

// FileStore abstracts file storage operations (MinIO, S3, etc.)
type FileStore interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	GetPresignedUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// UnavailableStore is a FileStore that always returns an error.
// Used when MinIO is not available at startup so the gateway can start
// and return proper errors instead of panicking.
type UnavailableStore struct{}

func NewUnavailableStore() FileStore { return &UnavailableStore{} }

func (s *UnavailableStore) Upload(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return errors.New("file storage unavailable")
}
func (s *UnavailableStore) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, errors.New("file storage unavailable")
}
func (s *UnavailableStore) Delete(_ context.Context, _ string) error {
	return errors.New("file storage unavailable")
}
func (s *UnavailableStore) GetPresignedURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", errors.New("file storage unavailable")
}
func (s *UnavailableStore) GetPresignedUploadURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", errors.New("file storage unavailable")
}

// NewMinIOStoreOrUnavailable is for services that only delete objects as a side
// effect of another operation (erasure in crm/auth): a MinIO that is down at
// startup must not keep the whole service from starting, so it degrades to an
// UnavailableStore whose Delete returns an error -- the erasure that needs it
// fails and is retried, every other request is unaffected.
func NewMinIOStoreOrUnavailable(endpoint, accessKey, secretKey, bucket string, useSSL bool) FileStore {
	store, err := NewMinIOStore(endpoint, accessKey, secretKey, bucket, useSSL)
	if err != nil {
		slog.Warn("minio unavailable, object deletion will fail until restart", "error", err)
		return NewUnavailableStore()
	}
	return store
}
