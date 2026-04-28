package pipelinestage

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/cache"
	"github.com/kmuhub/kmuhub/internal/models"
)

const (
	keyStageList = "cache:pipelinestages:list:"  // suffix: tenantID
	keyStageByID = "cache:pipelinestage:"        // suffix: tenantID:id
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

func (r *CachedRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*models.PipelineStage, error) {
	key := keyStageList + tenantID.String()
	return cache.Get(ctx, r.cache, key, ttlStageList, func(ctx context.Context) ([]*models.PipelineStage, error) {
		return r.inner.List(ctx, tenantID)
	})
}

func (r *CachedRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.PipelineStage, error) {
	key := keyStageByID + tenantID.String() + ":" + id.String()
	return cache.Get(ctx, r.cache, key, ttlStageByID, func(ctx context.Context) (*models.PipelineStage, error) {
		return r.inner.GetByID(ctx, id, tenantID)
	})
}

// ListWithStats is not cached — contains live deal counts that change frequently.
func (r *CachedRepository) ListWithStats(ctx context.Context, tenantID uuid.UUID) ([]*models.PipelineStageWithStats, error) {
	return r.inner.ListWithStats(ctx, tenantID)
}

func (r *CachedRepository) Create(ctx context.Context, stage *models.PipelineStage) error {
	if err := r.inner.Create(ctx, stage); err != nil {
		return err
	}
	cache.Delete(ctx, r.cache, keyStageList+stage.TenantID.String())
	return nil
}

func (r *CachedRepository) Update(ctx context.Context, stage *models.PipelineStage) error {
	if err := r.inner.Update(ctx, stage); err != nil {
		return err
	}
	tenantPrefix := stage.TenantID.String()
	cache.Delete(ctx, r.cache,
		keyStageList+tenantPrefix,
		keyStageByID+tenantPrefix+":"+stage.ID.String(),
	)
	return nil
}

func (r *CachedRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if err := r.inner.Delete(ctx, id, tenantID); err != nil {
		return err
	}
	tenantPrefix := tenantID.String()
	cache.Delete(ctx, r.cache,
		keyStageList+tenantPrefix,
		keyStageByID+tenantPrefix+":"+id.String(),
	)
	return nil
}

func (r *CachedRepository) Reorder(ctx context.Context, tenantID uuid.UUID, stageIDs []uuid.UUID) error {
	if err := r.inner.Reorder(ctx, tenantID, stageIDs); err != nil {
		return err
	}
	tenantPrefix := tenantID.String()
	keys := make([]string, 0, len(stageIDs)+1)
	keys = append(keys, keyStageList+tenantPrefix)
	for _, id := range stageIDs {
		keys = append(keys, keyStageByID+tenantPrefix+":"+id.String())
	}
	cache.Delete(ctx, r.cache, keys...)
	return nil
}

// Pass-through methods — not cached, used only during mutations.

func (r *CachedRepository) HasDeals(ctx context.Context, id, tenantID uuid.UUID) (bool, error) {
	return r.inner.HasDeals(ctx, id, tenantID)
}

func (r *CachedRepository) HasWonStage(ctx context.Context, tenantID uuid.UUID, excludeID *uuid.UUID) (bool, error) {
	return r.inner.HasWonStage(ctx, tenantID, excludeID)
}

func (r *CachedRepository) HasLostStage(ctx context.Context, tenantID uuid.UUID, excludeID *uuid.UUID) (bool, error) {
	return r.inner.HasLostStage(ctx, tenantID, excludeID)
}

func (r *CachedRepository) GetNextSortOrder(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return r.inner.GetNextSortOrder(ctx, tenantID)
}

func (r *CachedRepository) CountStages(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return r.inner.CountStages(ctx, tenantID)
}

// Compile-time interface check.
var _ Repository = (*CachedRepository)(nil)
