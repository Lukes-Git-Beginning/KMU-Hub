package dashboard

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Metrics holds the raw aggregated data from the database.
type Metrics struct {
	Revenue         models.RevenueMetrics
	Pipeline        models.PipelineMetrics
	StatusBreakdown models.InvoiceStatusBreakdown
	RecentInvoices  []*models.Invoice
	ExpiringQuotes  []*models.Quote
	PendingDunnings []*models.DunningRecord
	// MonthlyRevenue is used for forecast calculation (sum of paid invoices in date range).
	MonthlyRevenue decimal.Decimal
	// MonthsInRange is the number of months covered by the date range.
	MonthsInRange int
}

// Repository defines the interface for dashboard data aggregation.
type Repository interface {
	GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time) (*Metrics, error)
}
