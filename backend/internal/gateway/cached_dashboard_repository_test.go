package gateway

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/cache"
	"github.com/kmuhub/kmuhub/internal/models"
)

// testDashTenant is the tenant the dashboard cache tests operate on.
var testDashTenant = uuid.New()

// mockDashboardRepo is a hand-written mock for DashboardRepository.
type mockDashboardRepo struct {
	defaults map[string]*models.DashboardDefault
	users    map[string]*models.UserDashboardLayout

	getDefaultCalls    int
	getUserCalls       int
	upsertDefaultCalls int
	upsertUserCalls    int
	deleteUserCalls    int

	// Error injection for route_dashboard_test.go's repository-failure cases.
	// Left nil by every cache test below, so existing behaviour is unchanged.
	getDefaultErr    error
	getUserErr       error
	upsertDefaultErr error
	upsertUserErr    error
	deleteUserErr    error
}

func newMockDashboardRepo() *mockDashboardRepo {
	return &mockDashboardRepo{
		defaults: make(map[string]*models.DashboardDefault),
		users:    make(map[string]*models.UserDashboardLayout),
	}
}

// defaultKey mirrors the (tenant, role) uniqueness the table has had since
// 000274. Keying the mock on role alone would let a test pass that reads
// another tenant's preset.
func defaultKey(tenantID uuid.UUID, role string) string {
	return tenantID.String() + ":" + role
}

func (m *mockDashboardRepo) setDefault(tenantID uuid.UUID, def *models.DashboardDefault) {
	m.defaults[defaultKey(tenantID, def.Role)] = def
}

func (m *mockDashboardRepo) GetDefaultLayout(_ context.Context, tenantID uuid.UUID, role string) (*models.DashboardDefault, error) {
	m.getDefaultCalls++
	if m.getDefaultErr != nil {
		return nil, m.getDefaultErr
	}
	if d, ok := m.defaults[defaultKey(tenantID, role)]; ok {
		return d, nil
	}
	return nil, ErrDashboardNotFound
}

func (m *mockDashboardRepo) UpsertDefaultLayout(_ context.Context, tenantID uuid.UUID, def *models.DashboardDefault) (*models.DashboardDefault, error) {
	m.upsertDefaultCalls++
	if m.upsertDefaultErr != nil {
		return nil, m.upsertDefaultErr
	}
	m.setDefault(tenantID, def)
	return def, nil
}

func (m *mockDashboardRepo) GetUserLayout(_ context.Context, _ uuid.UUID, userID string) (*models.UserDashboardLayout, error) {
	m.getUserCalls++
	if m.getUserErr != nil {
		return nil, m.getUserErr
	}
	if l, ok := m.users[userID]; ok {
		return l, nil
	}
	return nil, ErrDashboardNotFound
}

func (m *mockDashboardRepo) UpsertUserLayout(_ context.Context, layout *models.UserDashboardLayout) (*models.UserDashboardLayout, error) {
	m.upsertUserCalls++
	if m.upsertUserErr != nil {
		return nil, m.upsertUserErr
	}
	m.users[layout.UserID] = layout
	return layout, nil
}

func (m *mockDashboardRepo) DeleteUserLayout(_ context.Context, _ uuid.UUID, userID string) error {
	m.deleteUserCalls++
	if m.deleteUserErr != nil {
		return m.deleteUserErr
	}
	if _, ok := m.users[userID]; !ok {
		return ErrDashboardNotFound
	}
	delete(m.users, userID)
	return nil
}

func setupCachedDashboard(t *testing.T) (*CachedDashboardRepository, *mockDashboardRepo) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rc.Close() })

	inner := newMockDashboardRepo()
	cached := NewCachedDashboardRepository(inner, cache.NewClient(rc))
	return cached, inner
}

func TestCachedDashboard_GetDefaultLayout_CacheMiss(t *testing.T) {
	repo, inner := setupCachedDashboard(t)
	inner.setDefault(testDashTenant, &models.DashboardDefault{
		ID:     "1",
		Role:   "admin",
		Layout: json.RawMessage(`{"cols":3}`),
	})

	d, err := repo.GetDefaultLayout(context.Background(), testDashTenant, "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", d.Role)
	assert.Equal(t, 1, inner.getDefaultCalls)
}

func TestCachedDashboard_GetDefaultLayout_CacheHit(t *testing.T) {
	repo, inner := setupCachedDashboard(t)
	inner.setDefault(testDashTenant, &models.DashboardDefault{
		ID:     "1",
		Role:   "admin",
		Layout: json.RawMessage(`{"cols":3}`),
	})

	_, _ = repo.GetDefaultLayout(context.Background(), testDashTenant, "admin")
	d, err := repo.GetDefaultLayout(context.Background(), testDashTenant, "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", d.Role)
	assert.Equal(t, 1, inner.getDefaultCalls) // not incremented
}

func TestCachedDashboard_GetDefaultLayout_NotFound_NotCached(t *testing.T) {
	repo, inner := setupCachedDashboard(t)

	// First call → ErrDashboardNotFound (not cached)
	_, err := repo.GetDefaultLayout(context.Background(), testDashTenant, "viewer")
	assert.ErrorIs(t, err, ErrDashboardNotFound)
	assert.Equal(t, 1, inner.getDefaultCalls)

	// Second call → should still hit DB (not cached)
	_, err = repo.GetDefaultLayout(context.Background(), testDashTenant, "viewer")
	assert.ErrorIs(t, err, ErrDashboardNotFound)
	assert.Equal(t, 2, inner.getDefaultCalls)
}

func TestCachedDashboard_UpsertDefaultLayout_InvalidatesCache(t *testing.T) {
	repo, inner := setupCachedDashboard(t)
	inner.setDefault(testDashTenant, &models.DashboardDefault{
		ID:     "1",
		Role:   "admin",
		Layout: json.RawMessage(`{"cols":3}`),
	})

	// Populate cache
	_, _ = repo.GetDefaultLayout(context.Background(), testDashTenant, "admin")
	assert.Equal(t, 1, inner.getDefaultCalls)

	// Upsert invalidates
	_, err := repo.UpsertDefaultLayout(context.Background(), testDashTenant, &models.DashboardDefault{
		Role:   "admin",
		Layout: json.RawMessage(`{"cols":4}`),
	})
	require.NoError(t, err)

	// Next get should hit DB
	_, _ = repo.GetDefaultLayout(context.Background(), testDashTenant, "admin")
	assert.Equal(t, 2, inner.getDefaultCalls)
}

// The Redis key used to be role-only, so the first tenant to warm the cache
// served its role preset to every other tenant for the next 30 minutes. This
// guards the tenant-scoped key.
func TestCachedDashboard_GetDefaultLayout_NotSharedAcrossTenants(t *testing.T) {
	repo, inner := setupCachedDashboard(t)
	other := uuid.New()

	inner.setDefault(testDashTenant, &models.DashboardDefault{
		ID:     "own",
		Role:   "admin",
		Layout: json.RawMessage(`{"cols":3}`),
	})
	inner.setDefault(other, &models.DashboardDefault{
		ID:     "foreign",
		Role:   "admin",
		Layout: json.RawMessage(`{"cols":9}`),
	})

	// Warm the cache from the other tenant first.
	foreign, err := repo.GetDefaultLayout(context.Background(), other, "admin")
	require.NoError(t, err)
	assert.Equal(t, "foreign", foreign.ID)

	own, err := repo.GetDefaultLayout(context.Background(), testDashTenant, "admin")
	require.NoError(t, err)
	assert.Equal(t, "own", own.ID, "must not be served the other tenant's cached preset")
	assert.Equal(t, 2, inner.getDefaultCalls, "second tenant must miss the cache")
}

func TestCachedDashboard_GetUserLayout_CacheHit(t *testing.T) {
	repo, inner := setupCachedDashboard(t)
	inner.users["user-1"] = &models.UserDashboardLayout{
		ID:     "1",
		UserID: "user-1",
		Layout: json.RawMessage(`{"custom":true}`),
	}

	_, _ = repo.GetUserLayout(context.Background(), uuid.Nil, "user-1")
	l, err := repo.GetUserLayout(context.Background(), uuid.Nil, "user-1")
	require.NoError(t, err)
	assert.Equal(t, "user-1", l.UserID)
	assert.Equal(t, 1, inner.getUserCalls)
}

func TestCachedDashboard_UpsertUserLayout_InvalidatesCache(t *testing.T) {
	repo, inner := setupCachedDashboard(t)
	inner.users["user-1"] = &models.UserDashboardLayout{
		ID:     "1",
		UserID: "user-1",
		Layout: json.RawMessage(`{"old":true}`),
	}

	// Populate cache
	_, _ = repo.GetUserLayout(context.Background(), uuid.Nil, "user-1")

	// Upsert invalidates
	_, _ = repo.UpsertUserLayout(context.Background(), &models.UserDashboardLayout{
		UserID: "user-1",
		Layout: json.RawMessage(`{"new":true}`),
	})

	_, _ = repo.GetUserLayout(context.Background(), uuid.Nil, "user-1")
	assert.Equal(t, 2, inner.getUserCalls)
}

func TestCachedDashboard_DeleteUserLayout_InvalidatesCache(t *testing.T) {
	repo, inner := setupCachedDashboard(t)
	inner.users["user-1"] = &models.UserDashboardLayout{
		ID:     "1",
		UserID: "user-1",
		Layout: json.RawMessage(`{}`),
	}

	// Populate cache
	_, _ = repo.GetUserLayout(context.Background(), uuid.Nil, "user-1")

	// Delete invalidates
	err := repo.DeleteUserLayout(context.Background(), uuid.Nil, "user-1")
	require.NoError(t, err)

	// Should hit DB again (and get NotFound since deleted)
	_, err = repo.GetUserLayout(context.Background(), uuid.Nil, "user-1")
	assert.ErrorIs(t, err, ErrDashboardNotFound)
	assert.Equal(t, 2, inner.getUserCalls)
}
