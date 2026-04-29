package file

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for file persistence.
type Repository interface {
	Create(ctx context.Context, file *models.DocumentFile) error
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.DocumentFile, error)
	List(ctx context.Context, filter ListFilter) ([]*models.DocumentFile, int, error)
	Update(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, input UpdateInput) error
	SoftDelete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error

	// Versioning
	CreateVersion(ctx context.Context, version *models.DocumentFileVersion) error
	ListVersions(ctx context.Context, fileID uuid.UUID) ([]*models.DocumentFileVersion, error)
	GetVersion(ctx context.Context, fileID uuid.UUID, versionNumber int) (*models.DocumentFileVersion, error)
	UpdateCurrentVersion(ctx context.Context, fileID uuid.UUID, versionNumber int) error

	// Search content
	UpdateSearchContent(ctx context.Context, id uuid.UUID, contentText string) error

	// Entity links
	CreateEntityLink(ctx context.Context, link *models.DocumentEntityLink) error
	DeleteEntityLink(ctx context.Context, id uuid.UUID) error
	ListEntityLinks(ctx context.Context, fileID uuid.UUID) ([]*models.DocumentEntityLink, error)
	ListFilesByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.DocumentFile, error)
}

// ListFilter contains filtering options for listing files.
type ListFilter struct {
	TenantID   uuid.UUID // Required: always filters by tenant
	FolderID   *uuid.UUID
	OwnerID    *uuid.UUID
	IsFavorite *bool
	IsDeleted  *bool
	TagIDs     []uuid.UUID
	SortField  string // "name", "size", "date", "type"
	SortDir    string // "asc", "desc"
	Limit      int
	Offset     int
}

// UpdateInput contains the fields that can be updated on a file.
type UpdateInput struct {
	Filename   *string
	FolderID   *uuid.UUID
	IsFavorite *bool
}
