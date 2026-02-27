package activity

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for activity persistence
type Repository interface {
	Create(ctx context.Context, activity *models.Activity) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Activity, error)
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Activity, int, error)
	Update(ctx context.Context, activity *models.Activity) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Relations
	GetContactName(ctx context.Context, contactID uuid.UUID) (string, error)
	GetCompanyName(ctx context.Context, companyID uuid.UUID) (string, error)
	GetDealName(ctx context.Context, dealID uuid.UUID) (string, error)
	GetUserName(ctx context.Context, userID uuid.UUID) (string, error)
	GetTags(ctx context.Context, activityID uuid.UUID) ([]*models.Tag, error)
	AddTags(ctx context.Context, activityID uuid.UUID, tagIDs []uuid.UUID) error
	RemoveTags(ctx context.Context, activityID uuid.UUID, tagIDs []uuid.UUID) error

	// Custom fields
	GetCustomFieldValues(ctx context.Context, activityID uuid.UUID) ([]*models.CustomFieldValueRow, error)
	SetCustomFieldValues(ctx context.Context, activityID uuid.UUID, values map[uuid.UUID]any) error

	// Existence checks
	ContactExists(ctx context.Context, contactID uuid.UUID) (bool, error)
	CompanyExists(ctx context.Context, companyID uuid.UUID) (bool, error)
	DealExists(ctx context.Context, dealID uuid.UUID) (bool, error)
	UserExists(ctx context.Context, userID uuid.UUID) (bool, error)
	TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error)

	// Timeline
	GetContactTimeline(ctx context.Context, contactID uuid.UUID, offset, limit int) ([]*TimelineEvent, int, error)
}

// ListFilter contains filtering options for listing activities
type ListFilter struct {
	ActivityType *models.ActivityType
	ContactID    *uuid.UUID
	CompanyID    *uuid.UUID
	DealID       *uuid.UUID
	AssignedTo   *uuid.UUID
	CreatedBy    *uuid.UUID
	IsCompleted  *bool
	TagIDs       []uuid.UUID
	Search       string // Searches subject
	SortBy       string // created_at, due_date, subject
	SortDesc     bool
}
