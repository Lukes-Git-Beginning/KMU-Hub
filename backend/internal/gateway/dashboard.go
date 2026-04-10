package gateway

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/cache"
)

// NewDashboardStack wires the dashboard repository → cached repository → service
// stack internally so callers only need a pool and cache client.
func NewDashboardStack(pool *pgxpool.Pool, cacheClient *cache.Client) *DashboardService {
	repo := NewCachedDashboardRepository(
		NewPostgresDashboardRepository(pool), cacheClient,
	)
	return NewDashboardService(repo)
}
