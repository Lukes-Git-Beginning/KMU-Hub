package contact

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for contact persistence
type Repository interface {
	Create(ctx context.Context, contact *models.Contact) error
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Contact, error)
	GetByEmail(ctx context.Context, email string, tenantID uuid.UUID) (*models.Contact, error)
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Contact, int, error)
	ListWithVisibility(ctx context.Context, userID uuid.UUID, isAdmin bool, filter ListFilter, offset, limit int) ([]*models.Contact, int, error)
	ListByIDs(ctx context.Context, ids []uuid.UUID, tenantID uuid.UUID) ([]*models.Contact, error)
	ListAll(ctx context.Context, userID uuid.UUID, isAdmin bool, tenantID uuid.UUID) ([]*models.Contact, error)
	Update(ctx context.Context, contact *models.Contact, tenantID uuid.UUID) error
	UpdateVisibility(ctx context.Context, contactID uuid.UUID, visibility string, ownerID *uuid.UUID, tenantID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error

	// Relations
	GetCompanyName(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID) (string, error)
	GetTags(ctx context.Context, contactID uuid.UUID) ([]*models.Tag, error)
	AddTags(ctx context.Context, contactID uuid.UUID, tagIDs []uuid.UUID) error
	RemoveTags(ctx context.Context, contactID uuid.UUID, tagIDs []uuid.UUID) error

	// Batch relations (for list endpoints)
	GetCompanyNames(ctx context.Context, companyIDs []uuid.UUID, tenantID uuid.UUID) (map[uuid.UUID]string, error)
	GetTagsBatch(ctx context.Context, contactIDs []uuid.UUID) (map[uuid.UUID][]*models.Tag, error)
	GetCustomFieldValuesBatch(ctx context.Context, contactIDs []uuid.UUID) (map[uuid.UUID][]*models.CustomFieldValueRow, error)

	// Custom fields
	GetCustomFieldValues(ctx context.Context, contactID uuid.UUID) ([]*models.CustomFieldValueRow, error)
	SetCustomFieldValues(ctx context.Context, contactID uuid.UUID, values map[uuid.UUID]any) error

	// Checks
	// IsInUse reports whether the contact is referenced by records that block a
	// hard delete (dialer campaign history, advisory protocols — see
	// postgres_repository.go). The returned reason names what is in the way,
	// for use in the returned error; it is empty when inUse is false.
	IsInUse(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (inUse bool, reason string, err error)
	CompanyExists(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID) (bool, error)
	TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error)

	// Duplicate detection
	FindDuplicateCandidates(ctx context.Context, contactID uuid.UUID, tenantID uuid.UUID) ([]*DuplicateCandidate, error)
	MergeInto(ctx context.Context, primaryID, duplicateID uuid.UUID, tenantID uuid.UUID) error

	// Lead lifecycle (same contacts rows, filtered by lifecycle_stage)
	ListLeads(ctx context.Context, filter LeadFilter, offset, limit int) ([]*models.ContactWithRelations, int, error)
	UpdateLead(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, patch LeadPatch) (*models.ContactWithRelations, error)
}

// LeadFilter narrows the lead inbox. TenantID is always applied.
type LeadFilter struct {
	TenantID uuid.UUID
	Stage    string // "" = every non-customer stage
	Status   string // "" = every status
	Search   string // first_name, last_name, email, lead_company
}

// LeadPatch is a partial update of the lead columns. Nil fields stay as they
// are; ClearTemperature wins over Temperature and resets the manual override.
type LeadPatch struct {
	Stage            *string
	Status           *string
	Source           *string
	Score            *int16
	Temperature      *string
	ClearTemperature bool
}

// ListFilter contains filtering options for listing contacts
type ListFilter struct {
	TenantID         uuid.UUID
	CompanyID        *uuid.UUID
	TagIDs           []uuid.UUID
	Search           string // Searches first_name, last_name, email
	SortBy           string // created_at, first_name, last_name, email
	SortDesc         bool
	VisibilityFilter string // "", "shared", "personal"
}
