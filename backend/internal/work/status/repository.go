package status

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for project status persistence
type Repository interface {
	Create(ctx context.Context, status *models.ProjectStatus) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.ProjectStatus, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.ProjectStatus, error)
	Update(ctx context.Context, status *models.ProjectStatus) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Reorder updates sort_order for all provided status IDs within a project
	Reorder(ctx context.Context, projectID uuid.UUID, statusIDs []uuid.UUID) error

	// CountTasksWithStatus returns the number of tasks assigned to a given status
	CountTasksWithStatus(ctx context.Context, statusID uuid.UUID) (int, error)

	// HasDefault checks if a project already has a default status
	HasDefault(ctx context.Context, projectID uuid.UUID) (bool, error)

	// GetNextSortOrder returns the next available sort order for a project
	GetNextSortOrder(ctx context.Context, projectID uuid.UUID) (int, error)

	// NameExistsInProject checks if a status name already exists in a project (case-insensitive)
	NameExistsInProject(ctx context.Context, projectID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error)

	// CopyStatusesForProject copies all statuses from source project to target project
	CopyStatusesForProject(ctx context.Context, sourceProjectID, targetProjectID uuid.UUID) error
}
