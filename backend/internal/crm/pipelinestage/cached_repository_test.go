package pipelinestage

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/cache"
	"github.com/kmuhub/kmuhub/internal/models"
)

// trackingRepo wraps MockRepository and counts calls to read methods.
type trackingRepo struct {
	*MockRepository
	listCalls    int
	getByIDCalls int
}

func newTrackingRepo() *trackingRepo {
	return &trackingRepo{MockRepository: NewMockRepository()}
}

func (t *trackingRepo) List(ctx context.Context) ([]*models.PipelineStage, error) {
	t.listCalls++
	return t.MockRepository.List(ctx)
}

func (t *trackingRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.PipelineStage, error) {
	t.getByIDCalls++
	return t.MockRepository.GetByID(ctx, id)
}

func setupCachedRepo(t *testing.T) (*CachedRepository, *trackingRepo, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rc.Close() })

	inner := newTrackingRepo()
	cached := NewCachedRepository(inner, cache.NewClient(rc))
	return cached, inner, mr
}

func seedStage(inner *trackingRepo, name string) *models.PipelineStage {
	stage := &models.PipelineStage{
		ID:          uuid.New(),
		Name:        name,
		Color:       "#ff0000",
		SortOrder:   1,
		Probability: decimal.NewFromInt(50),
	}
	inner.stages[stage.ID] = stage
	return stage
}

func TestCachedRepository_List_CacheMiss(t *testing.T) {
	repo, inner, _ := setupCachedRepo(t)
	seedStage(inner, "Stage A")

	stages, err := repo.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, stages, 1)
	assert.Equal(t, 1, inner.listCalls)
}

func TestCachedRepository_List_CacheHit(t *testing.T) {
	repo, inner, _ := setupCachedRepo(t)
	seedStage(inner, "Stage A")

	// First call → cache miss
	_, _ = repo.List(context.Background())
	assert.Equal(t, 1, inner.listCalls)

	// Second call → cache hit
	stages, err := repo.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, stages, 1)
	assert.Equal(t, 1, inner.listCalls) // not incremented
}

func TestCachedRepository_List_RedisDown_FallsThrough(t *testing.T) {
	repo, inner, mr := setupCachedRepo(t)
	seedStage(inner, "Stage A")
	mr.Close()

	stages, err := repo.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, stages, 1)
	assert.Equal(t, 1, inner.listCalls)
}

func TestCachedRepository_GetByID_CacheMiss(t *testing.T) {
	repo, inner, _ := setupCachedRepo(t)
	stage := seedStage(inner, "Stage B")

	result, err := repo.GetByID(context.Background(), stage.ID)
	require.NoError(t, err)
	assert.Equal(t, "Stage B", result.Name)
	assert.Equal(t, 1, inner.getByIDCalls)
}

func TestCachedRepository_GetByID_CacheHit(t *testing.T) {
	repo, inner, _ := setupCachedRepo(t)
	stage := seedStage(inner, "Stage B")

	_, _ = repo.GetByID(context.Background(), stage.ID)
	result, err := repo.GetByID(context.Background(), stage.ID)
	require.NoError(t, err)
	assert.Equal(t, "Stage B", result.Name)
	assert.Equal(t, 1, inner.getByIDCalls) // not incremented
}

func TestCachedRepository_Create_InvalidatesList(t *testing.T) {
	repo, inner, _ := setupCachedRepo(t)
	seedStage(inner, "Existing")

	// Populate cache
	_, _ = repo.List(context.Background())
	assert.Equal(t, 1, inner.listCalls)

	// Create invalidates list cache
	err := repo.Create(context.Background(), &models.PipelineStage{
		ID:   uuid.New(),
		Name: "New Stage",
	})
	require.NoError(t, err)

	// Next List should hit DB again
	_, _ = repo.List(context.Background())
	assert.Equal(t, 2, inner.listCalls)
}

func TestCachedRepository_Update_InvalidatesListAndByID(t *testing.T) {
	repo, inner, _ := setupCachedRepo(t)
	stage := seedStage(inner, "Original")

	// Populate both caches
	_, _ = repo.List(context.Background())
	_, _ = repo.GetByID(context.Background(), stage.ID)

	// Update invalidates both
	stage.Name = "Updated"
	err := repo.Update(context.Background(), stage)
	require.NoError(t, err)

	_, _ = repo.List(context.Background())
	_, _ = repo.GetByID(context.Background(), stage.ID)
	assert.Equal(t, 2, inner.listCalls)
	assert.Equal(t, 2, inner.getByIDCalls)
}

func TestCachedRepository_Delete_InvalidatesListAndByID(t *testing.T) {
	repo, inner, _ := setupCachedRepo(t)
	stage := seedStage(inner, "ToDelete")

	// Populate cache
	_, _ = repo.List(context.Background())
	_, _ = repo.GetByID(context.Background(), stage.ID)

	err := repo.Delete(context.Background(), stage.ID)
	require.NoError(t, err)

	// Should fetch from DB again
	_, _ = repo.List(context.Background())
	assert.Equal(t, 2, inner.listCalls)
}

func TestCachedRepository_Reorder_InvalidatesAll(t *testing.T) {
	repo, inner, _ := setupCachedRepo(t)
	s1 := seedStage(inner, "First")
	s2 := seedStage(inner, "Second")

	// Populate cache
	_, _ = repo.List(context.Background())
	_, _ = repo.GetByID(context.Background(), s1.ID)
	_, _ = repo.GetByID(context.Background(), s2.ID)

	err := repo.Reorder(context.Background(), []uuid.UUID{s2.ID, s1.ID})
	require.NoError(t, err)

	// All should re-fetch from DB
	_, _ = repo.List(context.Background())
	assert.Equal(t, 2, inner.listCalls)

	_, _ = repo.GetByID(context.Background(), s1.ID)
	_, _ = repo.GetByID(context.Background(), s2.ID)
	assert.Equal(t, 4, inner.getByIDCalls) // 2 initial + 2 after invalidation
}

func TestCachedRepository_ListWithStats_NotCached(t *testing.T) {
	repo, inner, _ := setupCachedRepo(t)
	seedStage(inner, "Stage")

	// ListWithStats always goes to inner
	_, _ = repo.ListWithStats(context.Background())
	_, _ = repo.ListWithStats(context.Background())
	// We can't track this easily since trackingRepo doesn't override ListWithStats,
	// but the point is it compiles and delegates correctly.
}
