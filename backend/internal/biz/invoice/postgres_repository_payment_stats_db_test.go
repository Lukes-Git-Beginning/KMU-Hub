package invoice

// postgres_repository_payment_stats_db_test.go exercises AggregatePaymentStats
// (postgres_repository.go:530-570) against real Postgres. This aggregate feeds
// GetPaymentStats -> the dashboard KPI a business owner reads to decide whether
// to chase receivables. Aggregates without a test are the quietest bug class:
// they never error, they just return a wrong number. Untagged
// (testutil.SkipIfNoDB) so it runs in the local gate and counts toward this
// package's coverage number, unlike integration_test.go.
//
// FIXED (see BACKLOG.yml fix-payment-stats-outstanding-ignores-recorded-payments):
// AggregatePaymentStats used to sum the invoice's full gross_total for any
// non-paid, non-cancelled status, the same bug postgres_open_items.go's
// openItemsBase already fixed once for the Open-Items view by netting against
// finance_payments. It now joins finance_payments the same way: a partial
// payment lowers total_outstanding_amount, and total_paid_amount reflects
// cash actually collected (an overpayment surfaces) rather than the invoice's
// gross_total — except where a 'paid' invoice has no payment row at all, in
// which case gross_total is the only figure available and is used as-is.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/payment"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// newPaymentStatsTestInvoice creates and persists an invoice with the given
// status and gross total via the real Create path. Subtotal/tax are set
// consistently with gross so the row satisfies the finance_invoices check
// constraints; the exact split does not matter to AggregatePaymentStats,
// which only reads gross_total and status.
func newPaymentStatsTestInvoice(t *testing.T, repo *PostgresRepository, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID, status string, gross string, invoiceDate time.Time) *models.Invoice {
	t.Helper()
	grossDec := decimal.RequireFromString(gross)
	subtotal := grossDec.Div(decimal.NewFromInt(119)).Mul(decimal.NewFromInt(100)).Round(2)
	tax := grossDec.Sub(subtotal)

	inv := newListTestInvoice(tenantID, invoiceDate, invoiceDate.AddDate(0, 0, 30))
	inv.Status = status
	inv.Subtotal = subtotal
	inv.TotalTax = tax
	inv.GrossTotal = grossDec
	createListTestInvoice(t, repo, pool, ctx, inv)
	return inv
}

// setInvoiceUpdatedAt overwrites updated_at directly so AggregatePaymentStats'
// avg_days_to_pay (updated_at - invoice_date) is deterministic instead of
// depending on UpdateStatus's NOW().
func setInvoiceUpdatedAt(t *testing.T, pool *pgxpool.Pool, ctx context.Context, id uuid.UUID, updatedAt time.Time) {
	t.Helper()
	_, err := pool.Exec(ctx, `UPDATE finance_invoices SET updated_at = $1 WHERE id = $2`, updatedAt, id)
	require.NoError(t, err)
}

// TestPostgresRepository_AggregatePaymentStats_EmptyPeriodReturnsZeroes proves
// the COALESCE guards actually fire: SUM/AVG over zero matching rows must
// return 0, not NULL scanned into an error or a zero-valued-by-accident struct.
func TestPostgresRepository_AggregatePaymentStats_EmptyPeriodReturnsZeroes(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Payment Stats Empty Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	stats, err := repo.AggregatePaymentStats(ctx, tenantID, time.Date(2040, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2040, 1, 31, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 0, stats.TotalInvoices)
	assert.Equal(t, 0, stats.TotalPaid)
	assert.Equal(t, 0, stats.TotalOutstanding)
	assert.True(t, decimal.Zero.Equal(stats.TotalPaidAmount), "expected 0, got %s", stats.TotalPaidAmount)
	assert.True(t, decimal.Zero.Equal(stats.TotalOutstandingAmount), "expected 0, got %s", stats.TotalOutstandingAmount)
	assert.True(t, decimal.Zero.Equal(stats.AverageDaysToPay), "expected 0, got %s", stats.AverageDaysToPay)
}

// TestPostgresRepository_AggregatePaymentStats_ClassifiesByStatusAndSumsGross
// proves the status buckets and the amount sums independently: a paid, a
// sent, an overdue and a cancelled invoice all in range. Cancelled must count
// toward total_invoices (it happened, it consumed a number) but toward
// neither total_paid/total_outstanding nor either amount sum.
func TestPostgresRepository_AggregatePaymentStats_ClassifiesByStatusAndSumsGross(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Payment Stats Classify Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	when := time.Date(2041, 4, 10, 0, 0, 0, 0, time.UTC)

	newPaymentStatsTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusPaid, "500.00", when)
	newPaymentStatsTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "300.00", when)
	newPaymentStatsTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusOverdue, "200.00", when)
	newPaymentStatsTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusCancelled, "9000.00", when)

	stats, err := repo.AggregatePaymentStats(ctx, tenantID, when.AddDate(0, 0, -1), when.AddDate(0, 0, 1))
	require.NoError(t, err)

	// Independently derived expectations, not a second copy of the aggregate SQL.
	assert.Equal(t, 4, stats.TotalInvoices, "all four invoices fall in range regardless of status")
	assert.Equal(t, 1, stats.TotalPaid)
	assert.Equal(t, 2, stats.TotalOutstanding, "sent + overdue, cancelled excluded")
	assert.True(t, decimal.RequireFromString("500.00").Equal(stats.TotalPaidAmount), "got %s", stats.TotalPaidAmount)
	assert.True(t, decimal.RequireFromString("500.00").Equal(stats.TotalOutstandingAmount), "300 (sent) + 200 (overdue), the 9000 cancelled invoice must not appear; got %s", stats.TotalOutstandingAmount)
}

// TestPostgresRepository_AggregatePaymentStats_PartialPaymentIsNettedFromOutstanding
// proves a sent invoice's recorded partial payment is netted off
// total_outstanding_amount, the same way postgres_open_items.go's
// openItemsBase already nets partial payments for the Open-Items view.
func TestPostgresRepository_AggregatePaymentStats_PartialPaymentIsNettedFromOutstanding(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Payment Stats Partial Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	payRepo := payment.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	when := time.Date(2042, 6, 1, 0, 0, 0, 0, time.UTC)

	inv := newPaymentStatsTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "1000.00", when)

	p := &models.Payment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		InvoiceID:   inv.ID,
		Amount:      decimal.RequireFromString("400.00"),
		PaymentDate: when,
		Method:      "bank_transfer",
		Reference:   "PARTIAL-1",
		CreatedBy:   uuid.New(),
		CreatedAt:   time.Now(),
	}
	require.NoError(t, payRepo.Create(ctx, p))
	defer testutil.CleanupRow(t, pool, "finance_payments", p.ID)

	stats, err := repo.AggregatePaymentStats(ctx, tenantID, when.AddDate(0, 0, -1), when.AddDate(0, 0, 1))
	require.NoError(t, err)

	assert.Equal(t, 1, stats.TotalOutstanding)
	assert.True(t, decimal.RequireFromString("600.00").Equal(stats.TotalOutstandingAmount),
		"the recorded 400.00 partial payment must be netted off the 1000.00 gross_total; got %s", stats.TotalOutstandingAmount)
}

// TestPostgresRepository_AggregatePaymentStats_PaidAmountReflectsOverpayment
// is the paid-side counterpart: an invoice marked paid after an overpayment
// (450.00 collected against a 400.00 invoice) contributes the actually
// collected 450.00 to total_paid_amount, not its own gross_total.
func TestPostgresRepository_AggregatePaymentStats_PaidAmountReflectsOverpayment(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Payment Stats Overpaid Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	payRepo := payment.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	when := time.Date(2043, 9, 15, 0, 0, 0, 0, time.UTC)

	inv := newPaymentStatsTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusPaid, "400.00", when)

	p := &models.Payment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		InvoiceID:   inv.ID,
		Amount:      decimal.RequireFromString("450.00"),
		PaymentDate: when,
		Method:      "bank_transfer",
		Reference:   "OVERPAY-1",
		CreatedBy:   uuid.New(),
		CreatedAt:   time.Now(),
	}
	require.NoError(t, payRepo.Create(ctx, p))
	defer testutil.CleanupRow(t, pool, "finance_payments", p.ID)

	stats, err := repo.AggregatePaymentStats(ctx, tenantID, when.AddDate(0, 0, -1), when.AddDate(0, 0, 1))
	require.NoError(t, err)

	assert.Equal(t, 1, stats.TotalPaid)
	assert.True(t, decimal.RequireFromString("450.00").Equal(stats.TotalPaidAmount),
		"the 450.00 actually collected must be reported, not the invoice's 400.00 gross_total; got %s", stats.TotalPaidAmount)
}

// TestPostgresRepository_AggregatePaymentStats_AverageDaysToPayIsArithmeticMean
// proves the average is a real mean over the paid invoices, not e.g. the
// duration of one row or a value skewed by unpaid rows in the same period.
// invoice_date is a plain DATE column and the local Postgres session runs
// with TimeZone=UTC (confirmed), so a DATE and a UTC-midnight-based
// TIMESTAMPTZ subtract without a timezone-offset surprise.
func TestPostgresRepository_AggregatePaymentStats_AverageDaysToPayIsArithmeticMean(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Payment Stats AvgDays Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	when := time.Date(2044, 2, 1, 0, 0, 0, 0, time.UTC)

	fast := newPaymentStatsTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusPaid, "100.00", when)
	setInvoiceUpdatedAt(t, pool, ctx, fast.ID, when.AddDate(0, 0, 5))

	slow := newPaymentStatsTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusPaid, "100.00", when)
	setInvoiceUpdatedAt(t, pool, ctx, slow.ID, when.AddDate(0, 0, 15))

	// An unpaid invoice in the same period must not pull the average toward
	// its own (irrelevant, still-open) updated_at.
	newPaymentStatsTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "100.00", when)

	stats, err := repo.AggregatePaymentStats(ctx, tenantID, when.AddDate(0, 0, -1), when.AddDate(0, 0, 20))
	require.NoError(t, err)

	avg, _ := stats.AverageDaysToPay.Float64()
	assert.InDelta(t, 10.0, avg, 0.01, "arithmetic mean of 5 and 15 days must be 10, got %v", avg)
}

// TestPostgresRepository_AggregatePaymentStats_IsTenantScoped proves a second
// tenant's invoices, even with much larger amounts, never move the first
// tenant's KPI — the dashboard is per-tenant, and a forgotten tenant_id
// filter here would leak revenue figures across customers.
func TestPostgresRepository_AggregatePaymentStats_IsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Payment Stats CrossTenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Payment Stats CrossTenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	when := time.Date(2045, 7, 20, 0, 0, 0, 0, time.UTC)

	newPaymentStatsTestInvoice(t, repo, pool, ctxA, tenantA, models.InvoiceStatusPaid, "119.00", when)
	newPaymentStatsTestInvoice(t, repo, pool, ctxB, tenantB, models.InvoiceStatusPaid, "999999.00", when)
	newPaymentStatsTestInvoice(t, repo, pool, ctxB, tenantB, models.InvoiceStatusSent, "999999.00", when)

	statsA, err := repo.AggregatePaymentStats(ctxA, tenantA, when.AddDate(0, 0, -1), when.AddDate(0, 0, 1))
	require.NoError(t, err)

	assert.Equal(t, 1, statsA.TotalInvoices, "tenant A must not see tenant B's invoices")
	assert.True(t, decimal.RequireFromString("119.00").Equal(statsA.TotalPaidAmount), "got %s", statsA.TotalPaidAmount)
	assert.True(t, decimal.Zero.Equal(statsA.TotalOutstandingAmount), "got %s", statsA.TotalOutstandingAmount)
}
