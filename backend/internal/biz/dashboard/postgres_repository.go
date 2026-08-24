package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL dashboard repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// GetDashboardMetrics aggregates all dashboard data in a few efficient queries.
func (r *PostgresRepository) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time) (*Metrics, error) {
	metrics := &Metrics{}

	// Query 1: Revenue metrics from invoices within date range.
	// Monetary sums are scanned as strings + decimal to avoid float64 precision loss (ADR-0007).
	// lean: sums only the tenant's default currency and silently drops foreign-currency
	// invoices (currently only reachable via recurring invoices, see biz/recurring/service.go)
	// instead of building a per-currency dashboard; upgrade when a tenant actually books
	// outside their default currency and the single-figure metric stops being honest.
	var totalInvoiced, totalPaid, totalOutstanding, overdueAmount string
	err := r.pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(CASE WHEN status IN ('sent','paid','overdue') THEN gross_total ELSE 0 END), 0)::text as total_invoiced,
			COALESCE(SUM(CASE WHEN status = 'paid' THEN gross_total ELSE 0 END), 0)::text as total_paid,
			COALESCE(SUM(CASE WHEN status IN ('sent','overdue') THEN gross_total ELSE 0 END), 0)::text as total_outstanding,
			COALESCE(SUM(CASE WHEN status = 'overdue' THEN gross_total ELSE 0 END), 0)::text as overdue_amount
		FROM finance_invoices
		WHERE tenant_id = $1 AND invoice_date >= $2 AND invoice_date <= $3
			AND currency = COALESCE((SELECT default_currency FROM company_settings WHERE tenant_id = $1), 'EUR')`,
		tenantID, dateFrom, dateTo,
	).Scan(&totalInvoiced, &totalPaid, &totalOutstanding, &overdueAmount)
	if err != nil {
		return nil, err
	}
	invoicedDec, err := parseDecimal(totalInvoiced)
	if err != nil {
		return nil, err
	}
	paidDec, err := parseDecimal(totalPaid)
	if err != nil {
		return nil, err
	}
	outstandingDec, err := parseDecimal(totalOutstanding)
	if err != nil {
		return nil, err
	}
	overdueDec, err := parseDecimal(overdueAmount)
	if err != nil {
		return nil, err
	}
	metrics.Revenue = models.RevenueMetrics{
		TotalInvoiced:    invoicedDec,
		TotalPaid:        paidDec,
		TotalOutstanding: outstandingDec,
		OverdueAmount:    overdueDec,
	}
	metrics.MonthlyRevenue = paidDec

	// Calculate months in range for forecast
	months := monthsBetween(dateFrom, dateTo)
	if months < 1 {
		months = 1
	}
	metrics.MonthsInRange = months

	// Query 2: Pipeline metrics from quotes within date range.
	// lean: same tenant-default-currency scoping as Query 1 above — quotesPending/
	// quotesTotal/quotesAccepted stay currency-agnostic (they're counts, not sums),
	// only avg_deal_size needs the filter.
	var quotesPending int
	var quotesTotal, quotesAccepted int
	var avgDealSize string
	err = r.pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END), 0) as quotes_pending,
			COUNT(*) as quotes_total,
			COALESCE(SUM(CASE WHEN status = 'accepted' THEN 1 ELSE 0 END), 0) as quotes_accepted,
			COALESCE(AVG(CASE WHEN status IN ('sent','accepted') AND currency = COALESCE((SELECT default_currency FROM company_settings WHERE tenant_id = $1), 'EUR') THEN gross_total END), 0)::text as avg_deal_size
		FROM finance_quotes
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at <= $3`,
		tenantID, dateFrom, dateTo,
	).Scan(&quotesPending, &quotesTotal, &quotesAccepted, &avgDealSize)
	if err != nil {
		return nil, err
	}
	avgDealSizeDec, err := parseDecimal(avgDealSize)
	if err != nil {
		return nil, err
	}

	var conversionRate decimal.Decimal
	if quotesTotal > 0 {
		conversionRate = decimal.NewFromInt(int64(quotesAccepted)).
			Div(decimal.NewFromInt(int64(quotesTotal))).
			Mul(decimal.NewFromInt(100)).
			Round(1)
	}
	metrics.Pipeline = models.PipelineMetrics{
		QuotesPending:   quotesPending,
		ConversionRate:  conversionRate,
		AverageDealSize: avgDealSizeDec.Round(2),
	}

	// Query 3: Invoice status breakdown (all time for tenant, not date filtered)
	var draft, sent, overdue, paid, cancelled int
	err = r.pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(CASE WHEN status = 'draft' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'overdue' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END), 0)
		FROM finance_invoices
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(&draft, &sent, &overdue, &paid, &cancelled)
	if err != nil {
		return nil, err
	}
	metrics.StatusBreakdown = models.InvoiceStatusBreakdown{
		Draft:     draft,
		Sent:      sent,
		Overdue:   overdue,
		Paid:      paid,
		Cancelled: cancelled,
	}

	// Query 4: Recent invoices (last 10)
	recentRows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, invoice_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			company_snapshot, tax_mode, tax_breakdown,
			subtotal, total_tax, gross_total,
			invoice_date, delivery_date, due_date, payment_terms,
			snapshot_data, source_quote_id, notes,
			zugferd_profile, time_tracking_source, locked_at, locked_by,
			contact_id,
			created_by, created_at, updated_at
		FROM finance_invoices
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 10`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer recentRows.Close()

	for recentRows.Next() {
		inv, scanErr := scanInvoice(recentRows)
		if scanErr != nil {
			return nil, scanErr
		}
		metrics.RecentInvoices = append(metrics.RecentInvoices, inv)
	}
	if err = recentRows.Err(); err != nil {
		return nil, err
	}

	// Query 5: Expiring quotes (valid_until within 7 days)
	sevenDaysFromNow := time.Now().AddDate(0, 0, 7)
	quoteRows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, quote_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, tax_breakdown,
			subtotal, total_tax, gross_total,
			valid_until, notes, deal_id, source_quote_id,
			created_by, created_at, updated_at
		FROM finance_quotes
		WHERE tenant_id = $1 AND status = 'sent' AND valid_until IS NOT NULL AND valid_until <= $2
		ORDER BY valid_until ASC`,
		tenantID, sevenDaysFromNow,
	)
	if err != nil {
		return nil, err
	}
	defer quoteRows.Close()

	for quoteRows.Next() {
		q, scanErr := scanQuote(quoteRows)
		if scanErr != nil {
			return nil, scanErr
		}
		metrics.ExpiringQuotes = append(metrics.ExpiringQuotes, q)
	}
	if err = quoteRows.Err(); err != nil {
		return nil, err
	}

	// Query 6: Pending dunning records (status=draft)
	dunningRows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, invoice_id, level, status,
			fee, interest, sent_at, created_by, created_at
		FROM finance_dunning_records
		WHERE tenant_id = $1 AND status = 'draft'
		ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer dunningRows.Close()

	for dunningRows.Next() {
		d, scanErr := scanDunning(dunningRows)
		if scanErr != nil {
			return nil, scanErr
		}
		metrics.PendingDunnings = append(metrics.PendingDunnings, d)
	}
	if err = dunningRows.Err(); err != nil {
		return nil, err
	}

	return metrics, nil
}

// parseDecimal parses a NUMERIC column (scanned as a string) into a decimal,
// avoiding the float64 precision loss that decimal.NewFromFloat introduces on
// monetary values (ADR-0007).
func parseDecimal(s string) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse decimal %q: %w", s, err)
	}
	return d, nil
}

// monthsBetween calculates the approximate number of months between two dates.
func monthsBetween(from, to time.Time) int {
	months := (to.Year()-from.Year())*12 + int(to.Month()) - int(from.Month())
	if months < 1 {
		months = 1
	}
	return months
}

// scanInvoice scans an invoice row from a query result.
// Does NOT scan line_items (dropped in Migration 000217).
func scanInvoice(rows interface{ Scan(dest ...any) error }) (*models.Invoice, error) {
	var inv models.Invoice
	var subtotalStr, totalTaxStr, grossTotalStr string
	err := rows.Scan(
		&inv.ID, &inv.TenantID, &inv.InvoiceNumber, &inv.Status,
		&inv.CustomerName, &inv.CustomerAddress, &inv.CustomerEmail, &inv.CustomerUStIDNr,
		&inv.CompanySnapshotRaw, &inv.TaxMode, &inv.TaxBreakdownRaw,
		&subtotalStr, &totalTaxStr, &grossTotalStr,
		&inv.InvoiceDate, &inv.DeliveryDate, &inv.DueDate, &inv.PaymentTerms,
		&inv.SnapshotData, &inv.SourceQuoteID, &inv.Notes,
		&inv.ZUGFeRDProfile, &inv.TimeTrackingSource, &inv.LockedAt, &inv.LockedBy,
		&inv.ContactID,
		&inv.CreatedBy, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	// Scan monetary values as string + decimal to avoid float64 precision loss (ADR-0007).
	if inv.Subtotal, err = parseDecimal(subtotalStr); err != nil {
		return nil, err
	}
	if inv.TotalTax, err = parseDecimal(totalTaxStr); err != nil {
		return nil, err
	}
	if inv.GrossTotal, err = parseDecimal(grossTotalStr); err != nil {
		return nil, err
	}
	return &inv, nil
}

// scanQuote scans a quote row from a query result.
// Does NOT scan line_items (dropped in Migration 000217).
func scanQuote(rows interface{ Scan(dest ...any) error }) (*models.Quote, error) {
	var q models.Quote
	var subtotalStr, totalTaxStr, grossTotalStr string
	err := rows.Scan(
		&q.ID, &q.TenantID, &q.QuoteNumber, &q.Status,
		&q.CustomerName, &q.CustomerAddress, &q.CustomerEmail, &q.CustomerUStIDNr,
		&q.TaxMode, &q.TaxBreakdownRaw,
		&subtotalStr, &totalTaxStr, &grossTotalStr,
		&q.ValidUntil, &q.Notes, &q.DealID, &q.SourceQuoteID,
		&q.CreatedBy, &q.CreatedAt, &q.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	// Scan monetary values as string + decimal to avoid float64 precision loss (ADR-0007).
	if q.Subtotal, err = parseDecimal(subtotalStr); err != nil {
		return nil, err
	}
	if q.TotalTax, err = parseDecimal(totalTaxStr); err != nil {
		return nil, err
	}
	if q.GrossTotal, err = parseDecimal(grossTotalStr); err != nil {
		return nil, err
	}
	return &q, nil
}

// scanDunning scans a dunning record row from a query result.
func scanDunning(rows interface{ Scan(dest ...any) error }) (*models.DunningRecord, error) {
	var d models.DunningRecord
	var feeStr, interestStr string
	err := rows.Scan(
		&d.ID, &d.TenantID, &d.InvoiceID, &d.Level, &d.Status,
		&feeStr, &interestStr, &d.SentAt, &d.CreatedBy, &d.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	// Scan monetary values as string + decimal to avoid float64 precision loss (ADR-0007).
	if d.Fee, err = parseDecimal(feeStr); err != nil {
		return nil, err
	}
	if d.Interest, err = parseDecimal(interestStr); err != nil {
		return nil, err
	}
	return &d, nil
}

// Ensure json import is used (for JSONB types in scan results).
var _ = json.RawMessage{}
