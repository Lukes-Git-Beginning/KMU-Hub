package share

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for share persistence.
type Repository interface {
	Create(ctx context.Context, share *models.DocumentShare) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.DocumentShare, error)
	ListSharedWithUser(ctx context.Context, userID uuid.UUID, entityType string) ([]*models.DocumentShare, error)
	GetUserPermission(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID) (*models.DocumentShare, error)
	HasAccess(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID) (bool, string, error)
}
