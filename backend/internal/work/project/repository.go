package project

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for project persistence
type Repository interface {
	// Project CRUD
	Create(ctx context.Context, project *models.Project) error
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Project, error)
	List(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, isAdmin bool, includeArchived bool) ([]models.ProjectWithDetails, error)
	Update(ctx context.Context, project *models.Project) error
	Archive(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID) error
	GetProjectKey(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID) (string, error)
	KeyExists(ctx context.Context, tenantID uuid.UUID, key string) (bool, error)

	// Member management
	AddMember(ctx context.Context, projectID, userID uuid.UUID, role string) error
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
	UpdateMemberRole(ctx context.Context, projectID, userID uuid.UUID, role string) error
	GetMember(ctx context.Context, projectID, userID uuid.UUID) (*models.ProjectMember, error)
	ListMembers(ctx context.Context, projectID uuid.UUID) ([]models.ProjectMember, error)
	IsMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
	CountOwners(ctx context.Context, projectID uuid.UUID) (int, error)

	// Template operations
	SaveAsTemplate(ctx context.Context, tenantID uuid.UUID, sourceID uuid.UUID, newName string, newKey string, createdBy uuid.UUID) (*models.Project, error)
	GetForTemplate(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID) (*models.Project, []models.ProjectStatus, error)

	// User preferences
	GetUserPreference(ctx context.Context, userID, projectID uuid.UUID) (*models.UserProjectPreference, error)
	SetUserPreference(ctx context.Context, pref *models.UserProjectPreference) error
}
