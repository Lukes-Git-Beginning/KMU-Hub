package workflow

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the data access interface for automation workflows.
type Repository interface {
	Create(ctx context.Context, automation *models.Automation) error
	Update(ctx context.Context, automation *models.Automation) error
	Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Automation, error)
	List(ctx context.Context, filter ListFilter) ([]*models.Automation, int, error)
	ListActiveByTriggerType(ctx context.Context, triggerType string) ([]*models.Automation, error)
	ListActiveTimeBased(ctx context.Context) ([]*models.Automation, error)
	SetActive(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, active bool) error
	UpdateLastTriggered(ctx context.Context, id uuid.UUID, at time.Time) error
}

// ExecutionRepository defines the data access interface for automation execution logs.
type ExecutionRepository interface {
	CreateExecution(ctx context.Context, execution *models.AutomationExecution) error
	UpdateExecution(ctx context.Context, execution *models.AutomationExecution) error
	GetExecution(ctx context.Context, id uuid.UUID) (*models.AutomationExecution, error)
	ListExecutions(ctx context.Context, filter ExecutionFilter) ([]*models.AutomationExecution, int, error)
	CleanupOldExecutions(ctx context.Context, olderThan time.Duration) (int64, error)
}

// TemplateRepository defines the data access interface for automation templates.
type TemplateRepository interface {
	ListTemplates(ctx context.Context, category *string) ([]*models.AutomationTemplate, error)
	GetTemplate(ctx context.Context, id string) (*models.AutomationTemplate, error)
	UpsertTemplate(ctx context.Context, template *models.AutomationTemplate) error
}

// ListFilter defines filtering criteria for listing automations.
type ListFilter struct {
	TenantID    uuid.UUID  // Required: always filters by tenant
	OwnerID     *uuid.UUID
	Scope       *string
	TriggerType *string
	IsActive    *bool
	Limit       int
	Offset      int
}

// ExecutionFilter defines filtering criteria for listing executions.
type ExecutionFilter struct {
	AutomationID *uuid.UUID
	Status       *string
	StartedAfter *time.Time
	StartedBefore *time.Time
	Limit        int
	Offset       int
}
