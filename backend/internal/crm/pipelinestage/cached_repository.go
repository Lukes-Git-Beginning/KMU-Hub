package pipelinestage

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/cache"
	"github.com/kmuhub/kmuhub/internal/models"
)

const (
	keyStageList = "cache:pipelinestages:list"
	keyStageByID = "cache:pipelinestage:"
	ttlStageList = 10 * time.Minute
	ttlStageByID = 10 * time.Minute
)

// CachedRepository wraps a Repository with Redis cache-aside for read-heavy operations.
type CachedRepository struct {
	inner Repository
	cache *cache.Client
}

// NewCachedRepository creates a cached wrapper around an existing Repository.
func NewCachedRepository(inner Repository, c *cache.Client) *CachedRepository {
	return &CachedRepository{inner: inner, cache: c}
}

func (r *CachedRepository) List(ctx context.Context) ([]*models.PipelineStage, error) {
	return cache.Get(ctx, r.cache, keyStageList, ttlStageList, func(ctx context.Context) ([]*models.PipelineStage, error) {
		return r.inner.List(ctx)
	})
}

func (r *CachedRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.PipelineStage, error) {
	key := keyStageByID + id.String()
	return cache.Get(ctx, r.cache, key, ttlStageByID, func(ctx context.Context) (*models.PipelineStage, error) {
		return r.inner.GetByID(ctx, id)
	})
}

// ListWithStats is not cached — contains live deal counts that change frequently.
func (r *CachedRepository) ListWithStats(ctx context.Context) ([]*models.PipelineStageWithStats, error) {
	return r.inner.ListWithStats(ctx)
}

func (r *CachedRepository) Create(ctx context.Context, stage *models.PipelineStage) error {
	if err := r.inner.Create(ctx, stage); err != nil {
		return err
	}
	cache.Delete(ctx, r.cache, keyStageList)
	return nil
}

func (r *CachedRepository) Update(ctx context.Context, stage *models.PipelineStage) error {
	if err := r.inner.Update(ctx, stage); err != nil {
		return err
	}
	cache.Delete(ctx, r.cache, keyStageList, keyStageByID+stage.ID.String())
	return nil
}

func (r *CachedRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.inner.Delete(ctx, id); err != nil {
		return err
	}
	cache.Delete(ctx, r.cache, keyStageList, keyStageByID+id.String())
	return nil
}

func (r *CachedRepository) Reorder(ctx context.Context, stageIDs []uuid.UUID) error {
	if err := r.inner.Reorder(ctx, stageIDs); err != nil {
		return err
	}
	keys := make([]string, 0, len(stageIDs)+1)
	keys = append(keys, keyStageList)
	for _, id := range stageIDs {
		keys = append(keys, keyStageByID+id.String())
	}
	cache.Delete(ctx, r.cache, keys...)
	return nil
}

// Pass-through methods — not cached, used only during mutations.

func (r *CachedRepository) HasDeals(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.inner.HasDeals(ctx, id)
}

func (r *CachedRepository) HasWonStage(ctx context.Context, excludeID *uuid.UUID) (bool, error) {
	return r.inner.HasWonStage(ctx, excludeID)
}

func (r *CachedRepository) HasLostStage(ctx context.Context, excludeID *uuid.UUID) (bool, error) {
	return r.inner.HasLostStage(ctx, excludeID)
}

func (r *CachedRepository) GetNextSortOrder(ctx context.Context) (int, error) {
	return r.inner.GetNextSortOrder(ctx)
}

func (r *CachedRepository) CountStages(ctx context.Context) (int, error) {
	return r.inner.CountStages(ctx)
}

// Compile-time interface check.
var _ Repository = (*CachedRepository)(nil)

