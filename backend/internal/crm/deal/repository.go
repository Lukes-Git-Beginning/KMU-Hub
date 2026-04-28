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
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Deal, error)
	List(ctx context.Context, tenantID uuid.UUID, filter ListFilter, offset, limit int) ([]*models.Deal, int, error)
	Update(ctx context.Context, deal *models.Deal) error
	Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error

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

	// Batch relations (for list endpoints — eliminates N+1 queries)
	GetStageNames(ctx context.Context, stageIDs []uuid.UUID) (map[uuid.UUID]string, error)
	GetContactNames(ctx context.Context, contactIDs []uuid.UUID) (map[uuid.UUID]string, error)
	GetCompanyNames(ctx context.Context, companyIDs []uuid.UUID) (map[uuid.UUID]string, error)
	GetOwnerNames(ctx context.Context, ownerIDs []uuid.UUID) (map[uuid.UUID]string, error)
	GetTagsBatch(ctx context.Context, dealIDs []uuid.UUID) (map[uuid.UUID][]*models.Tag, error)
	GetCustomFieldValuesBatch(ctx context.Context, dealIDs []uuid.UUID) (map[uuid.UUID][]*models.CustomFieldValueRow, error)

	// Checks
	StageExists(ctx context.Context, stageID, tenantID uuid.UUID) (bool, error)
	GetStage(ctx context.Context, stageID uuid.UUID) (*models.PipelineStage, error)
	ContactExists(ctx context.Context, contactID uuid.UUID) (bool, error)
	CompanyExists(ctx context.Context, companyID uuid.UUID) (bool, error)
	OwnerExists(ctx context.Context, ownerID uuid.UUID) (bool, error)
	TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error)

	// Closed at updates
	SetClosedAt(ctx context.Context, dealID, tenantID uuid.UUID, closedAt *time.Time) error
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
