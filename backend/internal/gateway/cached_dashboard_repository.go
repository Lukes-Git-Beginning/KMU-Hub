package gateway

import (
	"context"
	"time"

	"github.com/google/uuid"
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

// GetDefaultLayout caches per tenant and role. The tenant belongs in the key
// since 000274: role defaults used to be installation-wide, so a role-only key
// would now serve the first tenant to warm it to every other tenant for the
// next 30 minutes — the same leak in Redis that the migration closed in the table.
func (r *CachedDashboardRepository) GetDefaultLayout(ctx context.Context, tenantID uuid.UUID, role string) (*models.DashboardDefault, error) {
	key := keyDashboardDefaults + tenantID.String() + ":" + role
	return cache.Get(ctx, r.cache, key, ttlDashboardLayout, func(ctx context.Context) (*models.DashboardDefault, error) {
		return r.inner.GetDefaultLayout(ctx, tenantID, role)
	})
}

func (r *CachedDashboardRepository) UpsertDefaultLayout(ctx context.Context, tenantID uuid.UUID, def *models.DashboardDefault) (*models.DashboardDefault, error) {
	result, err := r.inner.UpsertDefaultLayout(ctx, tenantID, def)
	if err != nil {
		return nil, err
	}
	cache.Delete(ctx, r.cache, keyDashboardDefaults+tenantID.String()+":"+def.Role)
	return result, nil
}

func (r *CachedDashboardRepository) GetUserLayout(ctx context.Context, tenantID uuid.UUID, userID string) (*models.UserDashboardLayout, error) {
	key := keyDashboardUser + tenantID.String() + ":" + userID
	return cache.Get(ctx, r.cache, key, ttlDashboardLayout, func(ctx context.Context) (*models.UserDashboardLayout, error) {
		return r.inner.GetUserLayout(ctx, tenantID, userID)
	})
}

func (r *CachedDashboardRepository) UpsertUserLayout(ctx context.Context, layout *models.UserDashboardLayout) (*models.UserDashboardLayout, error) {
	result, err := r.inner.UpsertUserLayout(ctx, layout)
	if err != nil {
		return nil, err
	}
	cache.Delete(ctx, r.cache, keyDashboardUser+layout.TenantID.String()+":"+layout.UserID)
	return result, nil
}

func (r *CachedDashboardRepository) DeleteUserLayout(ctx context.Context, tenantID uuid.UUID, userID string) error {
	if err := r.inner.DeleteUserLayout(ctx, tenantID, userID); err != nil {
		return err
	}
	cache.Delete(ctx, r.cache, keyDashboardUser+tenantID.String()+":"+userID)
	return nil
}

// Compile-time interface check.
var _ DashboardRepository = (*CachedDashboardRepository)(nil)
