package company

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for company persistence
type Repository interface {
	Create(ctx context.Context, company *models.Company) error
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Company, error)
	GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*models.Company, error)
	GetNamesByIDs(ctx context.Context, ids []uuid.UUID, tenantID uuid.UUID) (map[uuid.UUID]string, error)
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Company, int, error)
	Update(ctx context.Context, company *models.Company, tenantID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error

	// Relations
	GetContactCount(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID) (int, error)
	GetTags(ctx context.Context, companyID uuid.UUID) ([]*models.Tag, error)
	AddTags(ctx context.Context, companyID uuid.UUID, tagIDs []uuid.UUID) error
	RemoveTags(ctx context.Context, companyID uuid.UUID, tagIDs []uuid.UUID) error

	// Custom fields
	GetCustomFieldValues(ctx context.Context, companyID uuid.UUID) ([]*models.CustomFieldValueRow, error)
	SetCustomFieldValues(ctx context.Context, companyID uuid.UUID, values map[uuid.UUID]any) error

	// Checks
	HasContacts(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID) (bool, error)
	TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error)

	// Duplicate detection
	FindDuplicateCandidates(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID) ([]*DuplicateCandidate, error)
	MergeInto(ctx context.Context, primaryID, duplicateID uuid.UUID, tenantID uuid.UUID) error
}

// ListFilter contains filtering options for listing companies
type ListFilter struct {
	TenantID uuid.UUID
	Search   string // Searches name, domain
	Industry *string
	TagIDs   []uuid.UUID
	SortBy   string // created_at, name
	SortDesc bool
}
