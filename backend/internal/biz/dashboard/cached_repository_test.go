package dashboard

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/cache"
	"github.com/kmuhub/kmuhub/internal/models"
)

type trackingDashboardRepo struct {
	*MockRepository
	getCalls int
}

func newTrackingDashboardRepo() *trackingDashboardRepo {
	return &trackingDashboardRepo{MockRepository: NewMockRepository()}
}

func (t *trackingDashboardRepo) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time) (*Metrics, error) {
	t.getCalls++
	return t.MockRepository.GetDashboardMetrics(ctx, tenantID, dateFrom, dateTo)
}

func setupCachedDashboardRepo(t *testing.T) (*CachedRepository, *trackingDashboardRepo, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rc.Close() })

	inner := newTrackingDashboardRepo()
	cached := NewCachedRepository(inner, cache.NewClient(rc))
	return cached, inner, mr
}

func sampleMetrics() *Metrics {
	return &Metrics{
		Revenue: models.RevenueMetrics{
			TotalInvoiced:    decimal.NewFromInt(50000),
			TotalPaid:        decimal.NewFromInt(30000),
			TotalOutstanding: decimal.NewFromInt(15000),
		},
		Pipeline: models.PipelineMetrics{
			QuotesPending:  10,
			ConversionRate: decimal.NewFromFloat(0.35),
		},
		MonthlyRevenue: decimal.NewFromInt(10000),
		MonthsInRange:  3,
	}
}

func TestCachedDashboardRepo_CacheMiss(t *testing.T) {
	repo, inner, _ := setupCachedDashboardRepo(t)
	inner.metrics = sampleMetrics()

	tenantID := uuid.New()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	m, err := repo.GetDashboardMetrics(context.Background(), tenantID, from, to)
	require.NoError(t, err)
	assert.Equal(t, 1, inner.getCalls)
	assert.True(t, m.Revenue.TotalInvoiced.Equal(decimal.NewFromInt(50000)))
}

func TestCachedDashboardRepo_CacheHit(t *testing.T) {
	repo, inner, _ := setupCachedDashboardRepo(t)
	inner.metrics = sampleMetrics()

	tenantID := uuid.New()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	// First call → miss
	_, _ = repo.GetDashboardMetrics(context.Background(), tenantID, from, to)
	assert.Equal(t, 1, inner.getCalls)

	// Second call → hit
	m, err := repo.GetDashboardMetrics(context.Background(), tenantID, from, to)
	require.NoError(t, err)
	assert.Equal(t, 1, inner.getCalls)
	assert.True(t, m.Revenue.TotalInvoiced.Equal(decimal.NewFromInt(50000)))
}

func TestCachedDashboardRepo_DifferentTenants_SeparateKeys(t *testing.T) {
	repo, inner, _ := setupCachedDashboardRepo(t)
	inner.metrics = sampleMetrics()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	_, _ = repo.GetDashboardMetrics(context.Background(), uuid.New(), from, to)
	_, _ = repo.GetDashboardMetrics(context.Background(), uuid.New(), from, to)
	assert.Equal(t, 2, inner.getCalls) // different tenants → separate cache entries
}

func TestCachedDashboardRepo_DifferentDateRanges_SeparateKeys(t *testing.T) {
	repo, inner, _ := setupCachedDashboardRepo(t)
	inner.metrics = sampleMetrics()

	tenantID := uuid.New()
	from1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	from2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	to2 := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)

	_, _ = repo.GetDashboardMetrics(context.Background(), tenantID, from1, to1)
	_, _ = repo.GetDashboardMetrics(context.Background(), tenantID, from2, to2)
	assert.Equal(t, 2, inner.getCalls)
}

func TestCachedDashboardRepo_RedisDown_FallsThrough(t *testing.T) {
	repo, inner, mr := setupCachedDashboardRepo(t)
	inner.metrics = sampleMetrics()
	mr.Close()

	m, err := repo.GetDashboardMetrics(context.Background(), uuid.New(),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	assert.Equal(t, 1, inner.getCalls)
	assert.True(t, m.Revenue.TotalInvoiced.Equal(decimal.NewFromInt(50000)))
}

func TestCachedDashboardRepo_DecimalRoundtrip(t *testing.T) {
	repo, inner, _ := setupCachedDashboardRepo(t)
	inner.metrics = &Metrics{
		Revenue: models.RevenueMetrics{
			TotalInvoiced: decimal.NewFromFloat(12345.67),
		},
		MonthlyRevenue: decimal.NewFromFloat(9876.54),
		MonthsInRange:  1,
	}

	tenantID := uuid.New()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	// Cache miss → populate
	_, _ = repo.GetDashboardMetrics(context.Background(), tenantID, from, to)

	// Cache hit → verify decimal precision survives JSON roundtrip
	m, err := repo.GetDashboardMetrics(context.Background(), tenantID, from, to)
	require.NoError(t, err)
	assert.True(t, m.Revenue.TotalInvoiced.Equal(decimal.NewFromFloat(12345.67)))
	assert.True(t, m.MonthlyRevenue.Equal(decimal.NewFromFloat(9876.54)))
}
