package task

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MaxNestingDepth is the maximum allowed subtask nesting level
const MaxNestingDepth = 5

// Repository defines the interface for task persistence
type Repository interface {
	// Core CRUD
	Create(ctx context.Context, task *models.Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.TaskWithRelations, error)
	List(ctx context.Context, filters TaskFilters) ([]models.TaskWithRelations, int, error)
	Update(ctx context.Context, task *models.Task) error
	Delete(ctx context.Context, id uuid.UUID) error
	MoveTask(ctx context.Context, taskID uuid.UUID, newStatusID uuid.UUID, newSortOrder float64) error

	// Task number
	GetNextTaskNumber(ctx context.Context, projectID uuid.UUID) (int, error)

	// Nesting
	GetSubtasks(ctx context.Context, parentID uuid.UUID, maxDepth int) ([]models.TaskWithRelations, error)
	GetParentChain(ctx context.Context, taskID uuid.UUID) ([]models.Task, error)
	GetDepth(ctx context.Context, taskID uuid.UUID) (int, error)

	// Dependencies
	CreateDependency(ctx context.Context, dep *models.TaskDependency) error
	DeleteDependency(ctx context.Context, id uuid.UUID) error
	ListDependencies(ctx context.Context, taskID uuid.UUID) ([]models.TaskDependency, error)
	HasCycle(ctx context.Context, sourceID, targetID uuid.UUID) (bool, error)

	// Activity logging
	CreateActivity(ctx context.Context, activity *models.TaskActivity) error
	ListActivities(ctx context.Context, taskID uuid.UUID, page, pageSize int) ([]models.TaskActivity, int, error)

	// Entity links
	LinkEntity(ctx context.Context, link *models.TaskEntityLink) error
	UnlinkEntity(ctx context.Context, id uuid.UUID) error
	ListEntityLinks(ctx context.Context, taskID uuid.UUID) ([]models.TaskEntityLink, error)
	ListTasksForEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]models.TaskWithRelations, error)

	// Files
	AttachFile(ctx context.Context, file *models.TaskFile) error
	RemoveFile(ctx context.Context, id uuid.UUID) error
	ListFiles(ctx context.Context, taskID uuid.UUID) ([]models.TaskFile, error)

	// Custom fields
	SetCustomFieldValues(ctx context.Context, taskID uuid.UUID, values map[uuid.UUID]any) error
	GetCustomFieldValues(ctx context.Context, taskID uuid.UUID) (map[string]any, error)

	// Search
	Search(ctx context.Context, query string, filters TaskSearchFilters) ([]models.TaskWithRelations, int, error)

	// Template support
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.Task, error)
	ListDependenciesByProject(ctx context.Context, projectID uuid.UUID) ([]models.TaskDependency, error)

	// Status check
	GetStatusByID(ctx context.Context, statusID uuid.UUID) (statusName string, isClosed bool, err error)
}

// TaskFilters contains filtering and pagination parameters for task listing
type TaskFilters struct {
	ProjectID    *uuid.UUID
	AssigneeID   *uuid.UUID
	StatusID     *uuid.UUID
	Priority     *string
	ParentTaskID *uuid.UUID
	CreatedBy    *uuid.UUID
	DueDateFrom  *time.Time
	DueDateTo    *time.Time
	Search       string
	IsStandalone *bool // true = project_id IS NULL
	Page         int
	PageSize     int
	SortBy       string
	SortDesc     bool
}

// TaskSearchFilters contains additional filters for full-text search
type TaskSearchFilters struct {
	ProjectIDs  []uuid.UUID
	AssigneeIDs []uuid.UUID
	Priority    *string
	Page        int
	PageSize    int
}
