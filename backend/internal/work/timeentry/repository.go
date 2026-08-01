package timeentry

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for time entry persistence
type Repository interface {
	// Core CRUD
	Create(ctx context.Context, entry *models.TimeEntry) error
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.TimeEntryWithUser, error)
	Update(ctx context.Context, entry *models.TimeEntry) error
	Delete(ctx context.Context, id, tenantID uuid.UUID) error

	// Listing
	ListByTask(ctx context.Context, taskID, tenantID uuid.UUID, page, pageSize int) ([]models.TimeEntryWithUser, int, error)
	ListByUser(ctx context.Context, userID, tenantID uuid.UUID, page, pageSize int) ([]models.TimeEntryWithUser, int, error)

	// ListBillable returns completed entries across all tasks/projects of the
	// tenant, for invoicing (Stunden -> Rechnung).
	ListBillable(ctx context.Context, tenantID uuid.UUID) ([]models.BillableTimeEntry, error)

	// ListByProject returns completed entries for a single project's tasks,
	// for the project-level "Stunden abrechnen" roll-up.
	ListByProject(ctx context.Context, projectID, tenantID uuid.UUID) ([]models.ProjectTimeEntry, error)

	// AggregateProjectHours sums completed entries per member per period
	// bucket ("week" or "month"), since the given time, for the project
	// team-utilization roll-up. Bucketing happens in SQL via date_trunc, not
	// by pulling every row into Go.
	AggregateProjectHours(ctx context.Context, projectID, tenantID uuid.UUID, trunc string, since time.Time) ([]models.UtilizationBucket, error)

	// Timer
	GetActiveTimer(ctx context.Context, userID, tenantID uuid.UUID) (*models.ActiveTimer, error)
	StopActiveTimer(ctx context.Context, userID, tenantID uuid.UUID) (*models.TimeEntry, error)

	// Summary
	GetTaskTimeSummary(ctx context.Context, taskID, tenantID uuid.UUID) (*models.TimeEntrySummary, error)
}
