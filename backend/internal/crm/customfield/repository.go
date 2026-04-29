package customfield

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for custom field persistence
type Repository interface {
	Create(ctx context.Context, field *models.CustomFieldDefinition) error
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.CustomFieldDefinition, error)
	GetByEntityAndName(ctx context.Context, tenantID uuid.UUID, entityType models.EntityType, fieldName string) (*models.CustomFieldDefinition, error)
	List(ctx context.Context, tenantID uuid.UUID, entityType *models.EntityType, offset, limit int) ([]*models.CustomFieldDefinition, int, error)
	Update(ctx context.Context, field *models.CustomFieldDefinition) error
	Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error
}
