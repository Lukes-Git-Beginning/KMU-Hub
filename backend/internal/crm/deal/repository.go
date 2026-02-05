package deal

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for deal persistence
type Repository interface {
	Create(ctx context.Context, deal *models.Deal) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Deal, error)
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Deal, int, error)
	Update(ctx context.Context, deal *models.Deal) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Relations
	GetStageName(ctx context.Context, stageID uuid.UUID) (string, error)
	GetContactName(ctx context.Context, contactID uuid.UUID) (string, error)
	GetCompanyName(ctx context.Context, companyID uuid.UUID) (string, error)
	GetOwnerName(ctx context.Context, ownerID uuid.UUID) (string, error)
	GetTags(ctx context.Context, dealID uuid.UUID) ([]*models.Tag, error)
	AddTags(ctx context.Context, dealID uuid.UUID, tagIDs []uuid.UUID) error
	RemoveTags(ctx context.Context, dealID uuid.UUID, tagIDs []uuid.UUID) error

	// Custom fields
	GetCustomFieldValues(ctx context.Context, dealID uuid.UUID) ([]*models.CustomFieldValueRow, error)
	SetCustomFieldValues(ctx context.Context, dealID uuid.UUID, values map[uuid.UUID]any) error

	// Checks
	StageExists(ctx context.Context, stageID uuid.UUID) (bool, error)
	GetStage(ctx context.Context, stageID uuid.UUID) (*models.PipelineStage, error)
	ContactExists(ctx context.Context, contactID uuid.UUID) (bool, error)
	CompanyExists(ctx context.Context, companyID uuid.UUID) (bool, error)
	OwnerExists(ctx context.Context, ownerID uuid.UUID) (bool, error)
	TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error)

	// Closed at updates
	SetClosedAt(ctx context.Context, dealID uuid.UUID, closedAt *time.Time) error
}

// ListFilter contains filtering options for listing deals
type ListFilter struct {
	StageID   *uuid.UUID
	ContactID *uuid.UUID
	CompanyID *uuid.UUID
	OwnerID   *uuid.UUID
	TagIDs    []uuid.UUID
	Search    string // Searches deal name
	SortBy    string // created_at, name, value, expected_close_date
	SortDesc  bool
}
