package tag

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for tag persistence
type Repository interface {
	Create(ctx context.Context, tag *models.Tag) error
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Tag, error)
	GetByEntityAndName(ctx context.Context, tenantID uuid.UUID, entityType models.EntityType, name string) (*models.Tag, error)
	List(ctx context.Context, tenantID uuid.UUID, entityType *models.EntityType, offset, limit int) ([]*models.Tag, int, error)
	Update(ctx context.Context, tag *models.Tag) error
	Delete(ctx context.Context, id, tenantID uuid.UUID) error
	IsInUse(ctx context.Context, id, tenantID uuid.UUID) (bool, error)
}
