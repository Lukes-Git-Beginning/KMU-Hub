package savedfilter

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for saved filter persistence
type Repository interface {
	Create(ctx context.Context, filter *models.SavedFilter) error
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.SavedFilter, error)
	List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.SavedFilter, error)
	Update(ctx context.Context, filter *models.SavedFilter) error
	Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error

	// ClearDefault clears the is_default flag for a user's filters of a specific entity type
	ClearDefault(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, entityType models.EntityType) error
}

// ListFilter contains filtering options for listing saved filters
type ListFilter struct {
	EntityType *models.EntityType
	UserID     *uuid.UUID
}
