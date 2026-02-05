package contact

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for contact persistence
type Repository interface {
	Create(ctx context.Context, contact *models.Contact) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Contact, error)
	GetByEmail(ctx context.Context, email string) (*models.Contact, error)
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Contact, int, error)
	Update(ctx context.Context, contact *models.Contact) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Relations
	GetCompanyName(ctx context.Context, companyID uuid.UUID) (string, error)
	GetTags(ctx context.Context, contactID uuid.UUID) ([]*models.Tag, error)
	AddTags(ctx context.Context, contactID uuid.UUID, tagIDs []uuid.UUID) error
	RemoveTags(ctx context.Context, contactID uuid.UUID, tagIDs []uuid.UUID) error

	// Custom fields
	GetCustomFieldValues(ctx context.Context, contactID uuid.UUID) ([]*models.CustomFieldValueRow, error)
	SetCustomFieldValues(ctx context.Context, contactID uuid.UUID, values map[uuid.UUID]any) error

	// Checks
	IsInUse(ctx context.Context, id uuid.UUID) (bool, error)
	CompanyExists(ctx context.Context, companyID uuid.UUID) (bool, error)
	TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error)
}

// ListFilter contains filtering options for listing contacts
type ListFilter struct {
	CompanyID *uuid.UUID
	TagIDs    []uuid.UUID
	Search    string // Searches first_name, last_name, email
	SortBy    string // created_at, first_name, last_name, email
	SortDesc  bool
}
