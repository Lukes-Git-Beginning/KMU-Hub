package timeentry

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for time entry persistence
type Repository interface {
	// Core CRUD
	Create(ctx context.Context, entry *models.TimeEntry) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.TimeEntryWithUser, error)
	Update(ctx context.Context, entry *models.TimeEntry) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Listing
	ListByTask(ctx context.Context, taskID uuid.UUID, page, pageSize int) ([]models.TimeEntryWithUser, int, error)
	ListByUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]models.TimeEntryWithUser, int, error)

	// Timer
	GetActiveTimer(ctx context.Context, userID uuid.UUID) (*models.ActiveTimer, error)
	StopActiveTimer(ctx context.Context, userID uuid.UUID) (*models.TimeEntry, error)

	// Summary
	GetTaskTimeSummary(ctx context.Context, taskID uuid.UUID) (*models.TimeEntrySummary, error)
}
