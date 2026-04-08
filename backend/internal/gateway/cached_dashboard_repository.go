package gateway

import (
	"context"
	"time"

	"github.com/kmuhub/kmuhub/internal/cache"
	"github.com/kmuhub/kmuhub/internal/models"
)

const (
	keyDashboardDefaults = "cache:dashboard:defaults:"
	keyDashboardUser     = "cache:dashboard:user:"
	ttlDashboardLayout   = 30 * time.Minute
)

// CachedDashboardRepository wraps a DashboardRepository with Redis cache-aside.
// ErrDashboardNotFound is never cached (no negative caching).
type CachedDashboardRepository struct {
	inner DashboardRepository
	cache *cache.Client
}

// NewCachedDashboardRepository creates a cached wrapper around an existing DashboardRepository.
func NewCachedDashboardRepository(inner DashboardRepository, c *cache.Client) *CachedDashboardRepository {
	return &CachedDashboardRepository{inner: inner, cache: c}
}

func (r *CachedDashboardRepository) GetDefaultLayout(ctx context.Context, role string) (*models.DashboardDefault, error) {
	key := keyDashboardDefaults + role
	return cache.Get(ctx, r.cache, key, ttlDashboardLayout, func(ctx context.Context) (*models.DashboardDefault, error) {
		return r.inner.GetDefaultLayout(ctx, role)
	})
}

func (r *CachedDashboardRepository) UpsertDefaultLayout(ctx context.Context, def *models.DashboardDefault) (*models.DashboardDefault, error) {
	result, err := r.inner.UpsertDefaultLayout(ctx, def)
	if err != nil {
		return nil, err
	}
	cache.Delete(ctx, r.cache, keyDashboardDefaults+def.Role)
	return result, nil
}

func (r *CachedDashboardRepository) GetUserLayout(ctx context.Context, userID string) (*models.UserDashboardLayout, error) {
	key := keyDashboardUser + userID
	return cache.Get(ctx, r.cache, key, ttlDashboardLayout, func(ctx context.Context) (*models.UserDashboardLayout, error) {
		return r.inner.GetUserLayout(ctx, userID)
	})
}

func (r *CachedDashboardRepository) UpsertUserLayout(ctx context.Context, layout *models.UserDashboardLayout) (*models.UserDashboardLayout, error) {
	result, err := r.inner.UpsertUserLayout(ctx, layout)
	if err != nil {
		return nil, err
	}
	cache.Delete(ctx, r.cache, keyDashboardUser+layout.UserID)
	return result, nil
}

func (r *CachedDashboardRepository) DeleteUserLayout(ctx context.Context, userID string) error {
	if err := r.inner.DeleteUserLayout(ctx, userID); err != nil {
		return err
	}
	cache.Delete(ctx, r.cache, keyDashboardUser+userID)
	return nil
}

// Compile-time interface check.
var _ DashboardRepository = (*CachedDashboardRepository)(nil)
