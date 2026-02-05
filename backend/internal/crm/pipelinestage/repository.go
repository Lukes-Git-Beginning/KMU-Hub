package pipelinestage

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for pipeline stage persistence
type Repository interface {
	Create(ctx context.Context, stage *models.PipelineStage) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PipelineStage, error)
	List(ctx context.Context) ([]*models.PipelineStage, error)
	ListWithStats(ctx context.Context) ([]*models.PipelineStageWithStats, error)
	Update(ctx context.Context, stage *models.PipelineStage) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Reorder updates the sort_order for multiple stages
	Reorder(ctx context.Context, stageIDs []uuid.UUID) error

	// Checks
	HasDeals(ctx context.Context, id uuid.UUID) (bool, error)
	HasWonStage(ctx context.Context, excludeID *uuid.UUID) (bool, error)
	HasLostStage(ctx context.Context, excludeID *uuid.UUID) (bool, error)
	GetNextSortOrder(ctx context.Context) (int, error)
	CountStages(ctx context.Context) (int, error)
}
