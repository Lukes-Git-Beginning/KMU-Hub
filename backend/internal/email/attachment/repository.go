package attachment

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for email attachment persistence.
type Repository interface {
	Create(ctx context.Context, att *models.EmailAttachment) error
	GetByMessage(ctx context.Context, messageID uuid.UUID) ([]*models.EmailAttachment, error)
	GetMinIOKeyByID(ctx context.Context, id uuid.UUID) (string, error)
}

// ObjectStore defines the interface for attachment file storage (MinIO).
type ObjectStore interface {
	Upload(ctx context.Context, accountID uuid.UUID, messageUID uint32, filename string, reader io.Reader, size int64, contentType string) (string, error)
	GetPresignedURL(ctx context.Context, minioKey string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, minioKey string) error
}
