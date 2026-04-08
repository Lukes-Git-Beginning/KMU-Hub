package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/cache"
)

const ttlDashboardMetrics = 5 * time.Minute

// CachedRepository wraps a Repository with Redis cache-aside for dashboard metrics.
// Uses TTL-only expiration (no explicit invalidation) since dashboard data
// tolerates a few minutes of staleness.
type CachedRepository struct {
	inner Repository
	cache *cache.Client
}

// NewCachedRepository creates a cached wrapper around an existing Repository.
func NewCachedRepository(inner Repository, c *cache.Client) *CachedRepository {
	return &CachedRepository{inner: inner, cache: c}
}

func (r *CachedRepository) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time) (*Metrics, error) {
	key := fmt.Sprintf("cache:biz:dashboard:%s:%s:%s",
		tenantID.String(),
		dateFrom.Format("2006-01-02"),
		dateTo.Format("2006-01-02"),
	)
	return cache.Get(ctx, r.cache, key, ttlDashboardMetrics, func(ctx context.Context) (*Metrics, error) {
		return r.inner.GetDashboardMetrics(ctx, tenantID, dateFrom, dateTo)
	})
}

// Compile-time interface check.
var _ Repository = (*CachedRepository)(nil)
