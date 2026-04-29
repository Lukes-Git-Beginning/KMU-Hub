package report

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for report data queries
type Repository interface {
	// GetPipelineReport returns deal aggregations grouped by stage
	GetPipelineReport(ctx context.Context, filter PipelineFilter) (*models.PipelineReport, error)

	// GetConversionReport returns stage transition metrics
	GetConversionReport(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*models.ConversionReport, error)

	// GetActivityReport returns activity metrics grouped by type
	GetActivityReport(ctx context.Context, filter ActivityFilter) (*models.ActivityReport, error)
}

// PipelineFilter contains filtering options for pipeline report
type PipelineFilter struct {
	TenantID  uuid.UUID
	OwnerID   *uuid.UUID
	StartDate time.Time
	EndDate   time.Time
}

// ActivityFilter contains filtering options for activity report
type ActivityFilter struct {
	TenantID  uuid.UUID
	UserID    *uuid.UUID
	StartDate time.Time
	EndDate   time.Time
}
