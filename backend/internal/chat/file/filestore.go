package file

import (
	"context"
	"io"
	"time"
)

// FileStore abstracts file storage operations (MinIO, S3, etc.)
type FileStore interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}
