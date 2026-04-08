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
	ListWithVisibility(ctx context.Context, userID uuid.UUID, isAdmin bool, filter ListFilter, offset, limit int) ([]*models.Contact, int, error)
	ListByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.Contact, error)
	ListAll(ctx context.Context, userID uuid.UUID, isAdmin bool) ([]*models.Contact, error)
	Update(ctx context.Context, contact *models.Contact) error
	UpdateVisibility(ctx context.Context, contactID uuid.UUID, visibility string, ownerID *uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Relations
	GetCompanyName(ctx context.Context, companyID uuid.UUID) (string, error)
	GetTags(ctx context.Context, contactID uuid.UUID) ([]*models.Tag, error)
	AddTags(ctx context.Context, contactID uuid.UUID, tagIDs []uuid.UUID) error
	RemoveTags(ctx context.Context, contactID uuid.UUID, tagIDs []uuid.UUID) error

	// Batch relations (for list endpoints)
	GetCompanyNames(ctx context.Context, companyIDs []uuid.UUID) (map[uuid.UUID]string, error)
	GetTagsBatch(ctx context.Context, contactIDs []uuid.UUID) (map[uuid.UUID][]*models.Tag, error)
	GetCustomFieldValuesBatch(ctx context.Context, contactIDs []uuid.UUID) (map[uuid.UUID][]*models.CustomFieldValueRow, error)

	// Custom fields
	GetCustomFieldValues(ctx context.Context, contactID uuid.UUID) ([]*models.CustomFieldValueRow, error)
	SetCustomFieldValues(ctx context.Context, contactID uuid.UUID, values map[uuid.UUID]any) error

	// Checks
	IsInUse(ctx context.Context, id uuid.UUID) (bool, error)
	CompanyExists(ctx context.Context, companyID uuid.UUID) (bool, error)
	TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error)

	// Duplicate detection
	FindDuplicateCandidates(ctx context.Context, contactID uuid.UUID) ([]*DuplicateCandidate, error)
	MergeInto(ctx context.Context, primaryID, duplicateID uuid.UUID) error
}

// ListFilter contains filtering options for listing contacts
type ListFilter struct {
	CompanyID        *uuid.UUID
	TagIDs           []uuid.UUID
	Search           string // Searches first_name, last_name, email
	SortBy           string // created_at, first_name, last_name, email
	SortDesc         bool
	VisibilityFilter string // "", "shared", "personal"
}
