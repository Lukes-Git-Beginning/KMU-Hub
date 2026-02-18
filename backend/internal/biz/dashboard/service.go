// Package dashboard provides the business logic for the finance dashboard.
// Aggregates revenue, pipeline, and status metrics with date range filtering.
// Includes a simple revenue forecast based on average monthly revenue.
package dashboard

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles finance dashboard business logic.
type Service struct {
	repo Repository
}

// NewService creates a new dashboard service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetDashboard retrieves the finance dashboard with aggregated metrics.
// Adds a simple revenue forecast: average monthly revenue * remaining months in year.
func (s *Service) GetDashboard(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time) (*models.FinanceDashboard, error) {
	metrics, err := s.repo.GetDashboardMetrics(ctx, tenantID, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	// Calculate revenue forecast: average monthly paid revenue * remaining months in year
	forecast := calculateForecast(metrics.MonthlyRevenue, metrics.MonthsInRange)

	dashboard := &models.FinanceDashboard{
		Revenue:         metrics.Revenue,
		Pipeline:        metrics.Pipeline,
		StatusBreakdown: metrics.StatusBreakdown,
		RecentInvoices:  metrics.RecentInvoices,
		ExpiringQuotes:  metrics.ExpiringQuotes,
		PendingDunnings: metrics.PendingDunnings,
		RevenueForecast: forecast,
	}

	// Ensure nil slices become empty arrays for clean JSON serialization
	if dashboard.RecentInvoices == nil {
		dashboard.RecentInvoices = []*models.Invoice{}
	}
	if dashboard.ExpiringQuotes == nil {
		dashboard.ExpiringQuotes = []*models.Quote{}
	}
	if dashboard.PendingDunnings == nil {
		dashboard.PendingDunnings = []*models.DunningRecord{}
	}

	slog.Info("dashboard loaded",
		"tenant_id", tenantID,
		"total_invoiced", metrics.Revenue.TotalInvoiced,
		"total_paid", metrics.Revenue.TotalPaid,
		"forecast", forecast,
	)

	return dashboard, nil
}

// calculateForecast computes a simple revenue forecast.
// Formula: (total_paid / months_in_range) * remaining_months_in_year
func calculateForecast(totalPaid decimal.Decimal, monthsInRange int) decimal.Decimal {
	if monthsInRange <= 0 || totalPaid.IsZero() {
		return decimal.Zero
	}

	avgMonthly := totalPaid.Div(decimal.NewFromInt(int64(monthsInRange)))
	now := time.Now()
	remainingMonths := 12 - int(now.Month())
	if remainingMonths <= 0 {
		return decimal.Zero
	}

	return avgMonthly.Mul(decimal.NewFromInt(int64(remainingMonths))).Round(2)
}
