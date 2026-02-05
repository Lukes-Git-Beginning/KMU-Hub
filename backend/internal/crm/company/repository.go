package company

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for company persistence
type Repository interface {
	Create(ctx context.Context, company *models.Company) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Company, error)
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Company, int, error)
	Update(ctx context.Context, company *models.Company) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Relations
	GetContactCount(ctx context.Context, companyID uuid.UUID) (int, error)
	GetTags(ctx context.Context, companyID uuid.UUID) ([]*models.Tag, error)
	AddTags(ctx context.Context, companyID uuid.UUID, tagIDs []uuid.UUID) error
	RemoveTags(ctx context.Context, companyID uuid.UUID, tagIDs []uuid.UUID) error

	// Custom fields
	GetCustomFieldValues(ctx context.Context, companyID uuid.UUID) ([]*models.CustomFieldValueRow, error)
	SetCustomFieldValues(ctx context.Context, companyID uuid.UUID, values map[uuid.UUID]any) error

	// Checks
	HasContacts(ctx context.Context, companyID uuid.UUID) (bool, error)
	TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error)
}

// ListFilter contains filtering options for listing companies
type ListFilter struct {
	Search   string // Searches name, domain
	Industry *string
	TagIDs   []uuid.UUID
	SortBy   string // created_at, name
	SortDesc bool
}
