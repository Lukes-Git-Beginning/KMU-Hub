package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MockRepository implements Repository for testing.
type MockRepository struct {
	metrics    *Metrics
	metricsErr error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{}
}

func (m *MockRepository) GetDashboardMetrics(_ context.Context, _ uuid.UUID, _, _ time.Time) (*Metrics, error) {
	if m.metricsErr != nil {
		return nil, m.metricsErr
	}
	return m.metrics, nil
}

// ============================================================================
// GetDashboard Tests
// ============================================================================

func TestService_GetDashboard_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	invoice := &models.Invoice{ID: uuid.New()}
	quote := &models.Quote{ID: uuid.New()}
	dunning := &models.DunningRecord{ID: uuid.New()}

	repo.metrics = &Metrics{
		Revenue: models.RevenueMetrics{
			TotalInvoiced:    decimal.NewFromInt(50000),
			TotalPaid:        decimal.NewFromInt(30000),
			TotalOutstanding: decimal.NewFromInt(15000),
			OverdueAmount:    decimal.NewFromInt(5000),
		},
		Pipeline: models.PipelineMetrics{
			QuotesPending:  5,
			ConversionRate: decimal.NewFromFloat(0.65),
			AverageDealSize: decimal.NewFromInt(10000),
		},
		StatusBreakdown: models.InvoiceStatusBreakdown{
			Draft: 2, Sent: 5, Overdue: 3, Paid: 10, Cancelled: 1,
		},
		RecentInvoices:  []*models.Invoice{invoice},
		ExpiringQuotes:  []*models.Quote{quote},
		PendingDunnings: []*models.DunningRecord{dunning},
		MonthlyRevenue:  decimal.NewFromInt(30000),
		MonthsInRange:   3,
	}

	tenantID := uuid.New()
	dateFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	dashboard, err := svc.GetDashboard(context.Background(), tenantID, dateFrom, dateTo)

	require.NoError(t, err)
	assert.Equal(t, decimal.NewFromInt(50000), dashboard.Revenue.TotalInvoiced)
	assert.Equal(t, decimal.NewFromInt(30000), dashboard.Revenue.TotalPaid)
	assert.Equal(t, decimal.NewFromInt(15000), dashboard.Revenue.TotalOutstanding)
	assert.Equal(t, decimal.NewFromInt(5000), dashboard.Revenue.OverdueAmount)
	assert.Equal(t, 5, dashboard.Pipeline.QuotesPending)
	assert.True(t, decimal.NewFromFloat(0.65).Equal(dashboard.Pipeline.ConversionRate))
	assert.Equal(t, decimal.NewFromInt(10000), dashboard.Pipeline.AverageDealSize)
	assert.Len(t, dashboard.RecentInvoices, 1)
	assert.Len(t, dashboard.ExpiringQuotes, 1)
	assert.Len(t, dashboard.PendingDunnings, 1)
}

func TestService_GetDashboard_NilSlices(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	repo.metrics = &Metrics{
		Revenue:         models.RevenueMetrics{},
		Pipeline:        models.PipelineMetrics{},
		StatusBreakdown: models.InvoiceStatusBreakdown{},
		RecentInvoices:  nil,
		ExpiringQuotes:  nil,
		PendingDunnings: nil,
		MonthlyRevenue:  decimal.Zero,
		MonthsInRange:   0,
	}

	dashboard, err := svc.GetDashboard(context.Background(), uuid.New(),
		time.Now().AddDate(0, -1, 0), time.Now())

	require.NoError(t, err)
	assert.NotNil(t, dashboard.RecentInvoices, "nil RecentInvoices should become empty slice")
	assert.NotNil(t, dashboard.ExpiringQuotes, "nil ExpiringQuotes should become empty slice")
	assert.NotNil(t, dashboard.PendingDunnings, "nil PendingDunnings should become empty slice")
	assert.Empty(t, dashboard.RecentInvoices)
	assert.Empty(t, dashboard.ExpiringQuotes)
	assert.Empty(t, dashboard.PendingDunnings)
}

func TestService_GetDashboard_RepoError(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	repoErr := errors.New("database connection failed")
	repo.metricsErr = repoErr

	dashboard, err := svc.GetDashboard(context.Background(), uuid.New(),
		time.Now().AddDate(0, -1, 0), time.Now())

	assert.Nil(t, dashboard)
	assert.ErrorIs(t, err, repoErr)
}

func TestService_GetDashboard_WithForecast(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	// totalPaid=12000, monthsInRange=3 -> avgMonthly=4000
	// forecast = 4000 * remainingMonthsInYear
	repo.metrics = &Metrics{
		Revenue: models.RevenueMetrics{
			TotalPaid: decimal.NewFromInt(12000),
		},
		MonthlyRevenue: decimal.NewFromInt(12000),
		MonthsInRange:  3,
	}

	dashboard, err := svc.GetDashboard(context.Background(), uuid.New(),
		time.Now().AddDate(0, -3, 0), time.Now())

	require.NoError(t, err)

	// Calculate expected forecast based on current month
	remainingMonths := 12 - int(time.Now().Month())
	if remainingMonths <= 0 {
		assert.True(t, dashboard.RevenueForecast.IsZero(),
			"forecast should be zero when no remaining months in year")
	} else {
		expected := decimal.NewFromInt(4000).Mul(decimal.NewFromInt(int64(remainingMonths))).Round(2)
		assert.True(t, expected.Equal(dashboard.RevenueForecast),
			"expected forecast %s but got %s", expected, dashboard.RevenueForecast)
	}
}

func TestService_GetDashboard_ZeroRevenue(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	repo.metrics = &Metrics{
		Revenue:        models.RevenueMetrics{},
		MonthlyRevenue: decimal.Zero,
		MonthsInRange:  6,
	}

	dashboard, err := svc.GetDashboard(context.Background(), uuid.New(),
		time.Now().AddDate(0, -6, 0), time.Now())

	require.NoError(t, err)
	assert.True(t, dashboard.RevenueForecast.IsZero(),
		"forecast should be zero when revenue is zero")
}

func TestService_GetDashboard_ZeroMonths(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	repo.metrics = &Metrics{
		Revenue: models.RevenueMetrics{
			TotalPaid: decimal.NewFromInt(50000),
		},
		MonthlyRevenue: decimal.NewFromInt(50000),
		MonthsInRange:  0,
	}

	dashboard, err := svc.GetDashboard(context.Background(), uuid.New(),
		time.Now().AddDate(0, -1, 0), time.Now())

	require.NoError(t, err)
	assert.True(t, dashboard.RevenueForecast.IsZero(),
		"forecast should be zero when months in range is zero")
}

func TestService_GetDashboard_WithData(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	invoices := []*models.Invoice{
		{ID: uuid.New()},
		{ID: uuid.New()},
		{ID: uuid.New()},
	}
	quotes := []*models.Quote{
		{ID: uuid.New()},
		{ID: uuid.New()},
	}
	dunnings := []*models.DunningRecord{
		{ID: uuid.New()},
	}

	repo.metrics = &Metrics{
		Revenue:         models.RevenueMetrics{},
		Pipeline:        models.PipelineMetrics{},
		StatusBreakdown: models.InvoiceStatusBreakdown{},
		RecentInvoices:  invoices,
		ExpiringQuotes:  quotes,
		PendingDunnings: dunnings,
		MonthlyRevenue:  decimal.Zero,
		MonthsInRange:   1,
	}

	dashboard, err := svc.GetDashboard(context.Background(), uuid.New(),
		time.Now().AddDate(0, -1, 0), time.Now())

	require.NoError(t, err)
	assert.Len(t, dashboard.RecentInvoices, 3, "should pass through all invoices")
	assert.Len(t, dashboard.ExpiringQuotes, 2, "should pass through all quotes")
	assert.Len(t, dashboard.PendingDunnings, 1, "should pass through all dunnings")

	// Verify the same slice contents are returned
	assert.Equal(t, invoices[0].ID, dashboard.RecentInvoices[0].ID)
	assert.Equal(t, quotes[0].ID, dashboard.ExpiringQuotes[0].ID)
	assert.Equal(t, dunnings[0].ID, dashboard.PendingDunnings[0].ID)
}

func TestService_GetDashboard_StatusBreakdown(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	repo.metrics = &Metrics{
		Revenue:  models.RevenueMetrics{},
		Pipeline: models.PipelineMetrics{},
		StatusBreakdown: models.InvoiceStatusBreakdown{
			Draft:     4,
			Sent:      8,
			Overdue:   2,
			Paid:      15,
			Cancelled: 3,
		},
		MonthlyRevenue: decimal.Zero,
		MonthsInRange:  1,
	}

	dashboard, err := svc.GetDashboard(context.Background(), uuid.New(),
		time.Now().AddDate(0, -1, 0), time.Now())

	require.NoError(t, err)
	assert.Equal(t, 4, dashboard.StatusBreakdown.Draft)
	assert.Equal(t, 8, dashboard.StatusBreakdown.Sent)
	assert.Equal(t, 2, dashboard.StatusBreakdown.Overdue)
	assert.Equal(t, 15, dashboard.StatusBreakdown.Paid)
	assert.Equal(t, 3, dashboard.StatusBreakdown.Cancelled)
}
