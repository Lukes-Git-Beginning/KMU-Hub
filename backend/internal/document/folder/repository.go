package folder

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for folder persistence.
type Repository interface {
	Create(ctx context.Context, folder *models.DocumentFolder) error
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*models.DocumentFolder, error)
	List(ctx context.Context, filter ListFilter) ([]*models.DocumentFolder, int, error)
	Update(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, input UpdateInput) error
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
	GetPath(ctx context.Context, id uuid.UUID) ([]models.FolderPathSegment, error)
	GetChildren(ctx context.Context, tenantID uuid.UUID, parentID uuid.UUID) ([]*models.DocumentFolder, error)
	CountFiles(ctx context.Context, folderID uuid.UUID) (int, error)
	IsDescendant(ctx context.Context, folderID, potentialAncestorID uuid.UUID) (bool, error)
}

// ListFilter contains filtering options for listing folders.
type ListFilter struct {
	TenantID  uuid.UUID
	ParentID  *uuid.UUID
	SpaceType *string
	SpaceID   *uuid.UUID
}

// UpdateInput contains the fields that can be updated on a folder.
type UpdateInput struct {
	Name     *string
	ParentID *uuid.UUID
	Icon     *string
}
