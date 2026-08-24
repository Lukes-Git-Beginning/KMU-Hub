package dashboard

// GetDashboardMetrics runs six aggregation queries by hand, each with its own
// tenant_id = $1 filter (not RLS alone — the repository still scopes the read
// query itself). A mock repository test proves the service layer forwards
// whatever the repo returns; it cannot prove any single WHERE clause here is
// still doing its job. Every aggregation therefore gets a two-tenant case
// that would leak the other tenant's numbers if its filter were ever dropped.
//
// Skips cleanly without DATABASE_URL (CI runners without Postgres).

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func newDashboardFixture(t *testing.T) (*pgxpool.Pool, uuid.UUID, context.Context) {
	t.Helper()
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dashboard Tenant")

	return pool, tenantID, testutil.WithTenantCtx(context.Background(), tenantID)
}

func seedInvoice(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, cols map[string]any) uuid.UUID {
	t.Helper()
	base := map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"invoice_number": "RE-DASH-" + uuid.NewString()[:8],
		"status":         models.InvoiceStatusSent,
		"customer_name":  "Dashboard Test GmbH",
		"gross_total":    "0",
		"invoice_date":   time.Now().UTC().Format("2006-01-02"),
		"due_date":       time.Now().UTC().AddDate(0, 0, 30).Format("2006-01-02"),
		"created_by":     tenantID,
	}
	maps.Copy(base, cols)
	id := testutil.SeedRow(t, pool, "finance_invoices", base)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", id) })
	return id
}

func seedQuote(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, cols map[string]any) uuid.UUID {
	t.Helper()
	base := map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"quote_number":  "AN-DASH-" + uuid.NewString()[:8],
		"status":        models.QuoteStatusSent,
		"customer_name": "Dashboard Test GmbH",
		"gross_total":   "0",
		"created_by":    tenantID,
	}
	maps.Copy(base, cols)
	id := testutil.SeedRow(t, pool, "finance_quotes", base)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_quotes", id) })
	return id
}

func seedDunning(t *testing.T, pool *pgxpool.Pool, tenantID, invoiceID uuid.UUID, cols map[string]any) uuid.UUID {
	t.Helper()
	base := map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "invoice_id": invoiceID,
		"level": 1, "status": models.DunningStatusDraft, "created_by": tenantID,
	}
	maps.Copy(base, cols)
	id := testutil.SeedRow(t, pool, "finance_dunning_records", base)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_dunning_records", id) })
	return id
}

func mustDecimal(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("decimal %q: %v", s, err)
	}
	return d
}

// TestGetDashboardMetrics_RevenueScopedByTenantAndDateRange is the case that
// would catch a dropped "tenant_id = $1" or a dropped date bound on Query 1:
// a second tenant's invoices sit in the same table and the same date window.
func TestGetDashboardMetrics_RevenueScopedByTenantAndDateRange(t *testing.T) {
	pool, tenantA, ctxA := newDashboardFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Dashboard Tenant B")
	repo := NewPostgresRepository(pool)

	dateFrom := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	inRange := "2026-03-15"
	beforeRange := "2026-02-15"

	seedInvoice(t, pool, tenantA, map[string]any{"status": models.InvoiceStatusSent, "gross_total": "1000.00", "invoice_date": inRange})
	seedInvoice(t, pool, tenantA, map[string]any{"status": models.InvoiceStatusPaid, "gross_total": "500.00", "invoice_date": inRange})
	seedInvoice(t, pool, tenantA, map[string]any{"status": models.InvoiceStatusOverdue, "gross_total": "200.00", "invoice_date": inRange})
	seedInvoice(t, pool, tenantA, map[string]any{"status": models.InvoiceStatusDraft, "gross_total": "999.00", "invoice_date": inRange})
	seedInvoice(t, pool, tenantA, map[string]any{"status": models.InvoiceStatusSent, "gross_total": "5000.00", "invoice_date": beforeRange})
	// Same date window, same statuses, a different tenant — must not leak in.
	seedInvoice(t, pool, tenantB, map[string]any{"status": models.InvoiceStatusSent, "gross_total": "99999.00", "invoice_date": inRange})

	metrics, err := repo.GetDashboardMetrics(ctxA, tenantA, dateFrom, dateTo)
	if err != nil {
		t.Fatalf("GetDashboardMetrics: %v", err)
	}

	if want := mustDecimal(t, "1700.00"); !metrics.Revenue.TotalInvoiced.Equal(want) {
		t.Errorf("TotalInvoiced = %s, want %s", metrics.Revenue.TotalInvoiced, want)
	}
	if want := mustDecimal(t, "500.00"); !metrics.Revenue.TotalPaid.Equal(want) {
		t.Errorf("TotalPaid = %s, want %s", metrics.Revenue.TotalPaid, want)
	}
	if want := mustDecimal(t, "1200.00"); !metrics.Revenue.TotalOutstanding.Equal(want) {
		t.Errorf("TotalOutstanding = %s, want %s", metrics.Revenue.TotalOutstanding, want)
	}
	if want := mustDecimal(t, "200.00"); !metrics.Revenue.OverdueAmount.Equal(want) {
		t.Errorf("OverdueAmount = %s, want %s", metrics.Revenue.OverdueAmount, want)
	}
}

// TestGetDashboardMetrics_ForeignCurrencyInvoiceExcludedFromRevenueSum covers
// the fix for the blind currency sum: a tenant with no company_settings row
// (default currency implicitly EUR) has one EUR and one CHF invoice in range.
// The CHF invoice must not be blindly added into the EUR-denominated revenue
// figures — only the EUR invoice may count.
func TestGetDashboardMetrics_ForeignCurrencyInvoiceExcludedFromRevenueSum(t *testing.T) {
	pool, tenantID, ctx := newDashboardFixture(t)
	repo := NewPostgresRepository(pool)

	dateFrom := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	inRange := "2026-03-15"

	seedInvoice(t, pool, tenantID, map[string]any{"status": models.InvoiceStatusSent, "gross_total": "1000.00", "invoice_date": inRange, "currency": "EUR"})
	seedInvoice(t, pool, tenantID, map[string]any{"status": models.InvoiceStatusSent, "gross_total": "5000.00", "invoice_date": inRange, "currency": "CHF"})

	metrics, err := repo.GetDashboardMetrics(ctx, tenantID, dateFrom, dateTo)
	if err != nil {
		t.Fatalf("GetDashboardMetrics: %v", err)
	}

	if want := mustDecimal(t, "1000.00"); !metrics.Revenue.TotalInvoiced.Equal(want) {
		t.Errorf("TotalInvoiced = %s, want %s (CHF invoice must not be blindly summed into EUR)", metrics.Revenue.TotalInvoiced, want)
	}
	if want := mustDecimal(t, "1000.00"); !metrics.Revenue.TotalOutstanding.Equal(want) {
		t.Errorf("TotalOutstanding = %s, want %s", metrics.Revenue.TotalOutstanding, want)
	}
}

// TestGetDashboardMetrics_UsesTenantDefaultCurrencyNotHardcodedEUR covers a
// tenant whose company_settings.default_currency is CHF (set via
// biz/invoice/service.go at document creation time): their own CHF invoices
// must count, and a stray EUR invoice must be excluded — proving the filter
// resolves the tenant's actual default rather than hardcoding "EUR".
func TestGetDashboardMetrics_UsesTenantDefaultCurrencyNotHardcodedEUR(t *testing.T) {
	pool, tenantID, ctx := newDashboardFixture(t)
	repo := NewPostgresRepository(pool)
	settingsID := testutil.SeedRow(t, pool, "company_settings", map[string]any{"tenant_id": tenantID, "default_currency": "CHF"})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "company_settings", settingsID) })

	dateFrom := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	inRange := "2026-03-15"

	seedInvoice(t, pool, tenantID, map[string]any{"status": models.InvoiceStatusSent, "gross_total": "2000.00", "invoice_date": inRange, "currency": "CHF"})
	seedInvoice(t, pool, tenantID, map[string]any{"status": models.InvoiceStatusSent, "gross_total": "9999.00", "invoice_date": inRange, "currency": "EUR"})

	metrics, err := repo.GetDashboardMetrics(ctx, tenantID, dateFrom, dateTo)
	if err != nil {
		t.Fatalf("GetDashboardMetrics: %v", err)
	}

	if want := mustDecimal(t, "2000.00"); !metrics.Revenue.TotalInvoiced.Equal(want) {
		t.Errorf("TotalInvoiced = %s, want %s (tenant's default currency is CHF, the stray EUR invoice must be excluded)", metrics.Revenue.TotalInvoiced, want)
	}
}

// TestGetDashboardMetrics_ForeignCurrencyQuoteExcludedFromAverageDealSize
// covers Query 2's avg_deal_size: a CHF quote must not drag the EUR average
// off, while quotesPending/quotesTotal/quotesAccepted stay currency-agnostic
// counts (unaffected by the fix, asserted here to pin that decision).
func TestGetDashboardMetrics_ForeignCurrencyQuoteExcludedFromAverageDealSize(t *testing.T) {
	pool, tenantID, ctx := newDashboardFixture(t)
	repo := NewPostgresRepository(pool)

	dateFrom := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	inRange := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)

	seedQuote(t, pool, tenantID, map[string]any{"status": models.QuoteStatusSent, "gross_total": "1000.00", "created_at": inRange, "currency": "EUR"})
	seedQuote(t, pool, tenantID, map[string]any{"status": models.QuoteStatusAccepted, "gross_total": "50000.00", "created_at": inRange, "currency": "CHF"})

	metrics, err := repo.GetDashboardMetrics(ctx, tenantID, dateFrom, dateTo)
	if err != nil {
		t.Fatalf("GetDashboardMetrics: %v", err)
	}

	if want := mustDecimal(t, "1000.00"); !metrics.Pipeline.AverageDealSize.Equal(want) {
		t.Errorf("AverageDealSize = %s, want %s (CHF quote must not enter the EUR average)", metrics.Pipeline.AverageDealSize, want)
	}
	if metrics.Pipeline.QuotesPending != 1 {
		t.Errorf("QuotesPending = %d, want 1 (count stays currency-agnostic)", metrics.Pipeline.QuotesPending)
	}
}

// TestGetDashboardMetrics_EmptyDateRangeReturnsZeroNotError covers a tenant
// with zero invoices in range: the SUM(...) aggregates must COALESCE to 0,
// not surface a scan error from a NULL, and MonthsInRange must still be >= 1
// so a caller never divides by zero building a forecast.
func TestGetDashboardMetrics_EmptyDateRangeReturnsZeroNotError(t *testing.T) {
	pool, tenantID, ctx := newDashboardFixture(t)
	repo := NewPostgresRepository(pool)

	dateFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	metrics, err := repo.GetDashboardMetrics(ctx, tenantID, dateFrom, dateTo)
	if err != nil {
		t.Fatalf("GetDashboardMetrics on an empty tenant: %v", err)
	}
	if !metrics.Revenue.TotalInvoiced.IsZero() || !metrics.Revenue.TotalPaid.IsZero() ||
		!metrics.Revenue.TotalOutstanding.IsZero() || !metrics.Revenue.OverdueAmount.IsZero() {
		t.Errorf("revenue on an empty tenant = %+v, want all zero", metrics.Revenue)
	}
	if metrics.Pipeline.QuotesPending != 0 || !metrics.Pipeline.AverageDealSize.IsZero() {
		t.Errorf("pipeline on an empty tenant = %+v, want all zero", metrics.Pipeline)
	}
	if metrics.MonthsInRange < 1 {
		t.Errorf("MonthsInRange = %d, want >= 1 (a forecast divides by this)", metrics.MonthsInRange)
	}
}

// TestGetDashboardMetrics_PipelineScopedByTenant covers Query 2: pending
// count, accepted count and average deal size must all come from tenant A's
// quotes only, in-range, with the draft quote excluded from the average.
func TestGetDashboardMetrics_PipelineScopedByTenant(t *testing.T) {
	pool, tenantA, ctxA := newDashboardFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Dashboard Tenant B")
	repo := NewPostgresRepository(pool)

	dateFrom := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	inRange := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)

	seedQuote(t, pool, tenantA, map[string]any{"status": models.QuoteStatusSent, "gross_total": "1000.00", "created_at": inRange})
	seedQuote(t, pool, tenantA, map[string]any{"status": models.QuoteStatusAccepted, "gross_total": "2000.00", "created_at": inRange})
	seedQuote(t, pool, tenantA, map[string]any{"status": models.QuoteStatusDraft, "gross_total": "999999.00", "created_at": inRange})
	seedQuote(t, pool, tenantB, map[string]any{"status": models.QuoteStatusSent, "gross_total": "99999.00", "created_at": inRange})

	metrics, err := repo.GetDashboardMetrics(ctxA, tenantA, dateFrom, dateTo)
	if err != nil {
		t.Fatalf("GetDashboardMetrics: %v", err)
	}

	if metrics.Pipeline.QuotesPending != 1 {
		t.Errorf("QuotesPending = %d, want 1", metrics.Pipeline.QuotesPending)
	}
	if want := mustDecimal(t, "1500.00"); !metrics.Pipeline.AverageDealSize.Equal(want) {
		t.Errorf("AverageDealSize = %s, want %s (draft quote must not enter the average)", metrics.Pipeline.AverageDealSize, want)
	}
	// 1 accepted out of 3 total quotes = 33.3%.
	if want := mustDecimal(t, "33.3"); !metrics.Pipeline.ConversionRate.Equal(want) {
		t.Errorf("ConversionRate = %s, want %s", metrics.Pipeline.ConversionRate, want)
	}
}

// TestGetDashboardMetrics_StatusBreakdownIsAllTimeButTenantScoped covers
// Query 3, the one aggregation that deliberately ignores the date range
// (comment in the source: "all time for tenant, not date filtered"). The
// date window below excludes every invoice on purpose to prove the filter
// really is absent; a second tenant's invoice proves the tenant filter is
// still present.
func TestGetDashboardMetrics_StatusBreakdownIsAllTimeButTenantScoped(t *testing.T) {
	pool, tenantA, ctxA := newDashboardFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Dashboard Tenant B")
	repo := NewPostgresRepository(pool)

	oldDate := "2020-01-01"
	seedInvoice(t, pool, tenantA, map[string]any{"status": models.InvoiceStatusDraft, "invoice_date": oldDate})
	seedInvoice(t, pool, tenantA, map[string]any{"status": models.InvoiceStatusSent, "invoice_date": oldDate})
	seedInvoice(t, pool, tenantA, map[string]any{"status": models.InvoiceStatusSent, "invoice_date": oldDate})
	seedInvoice(t, pool, tenantA, map[string]any{"status": models.InvoiceStatusOverdue, "invoice_date": oldDate})
	seedInvoice(t, pool, tenantA, map[string]any{"status": models.InvoiceStatusCancelled, "invoice_date": oldDate})
	seedInvoice(t, pool, tenantB, map[string]any{"status": models.InvoiceStatusSent, "invoice_date": oldDate})

	// A date range far in the future — the breakdown must ignore it entirely.
	dateFrom := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)

	metrics, err := repo.GetDashboardMetrics(ctxA, tenantA, dateFrom, dateTo)
	if err != nil {
		t.Fatalf("GetDashboardMetrics: %v", err)
	}

	want := models.InvoiceStatusBreakdown{Draft: 1, Sent: 2, Overdue: 1, Paid: 0, Cancelled: 1}
	if metrics.StatusBreakdown != want {
		t.Errorf("StatusBreakdown = %+v, want %+v", metrics.StatusBreakdown, want)
	}
}

// TestGetDashboardMetrics_RecentInvoicesScopedByTenantAndLimitedToTen covers
// Query 4: newest-first, capped at 10, and never another tenant's invoice.
func TestGetDashboardMetrics_RecentInvoicesScopedByTenantAndLimitedToTen(t *testing.T) {
	pool, tenantA, ctxA := newDashboardFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Dashboard Tenant B")
	repo := NewPostgresRepository(pool)

	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	var newestID uuid.UUID
	for i := range 11 {
		id := seedInvoice(t, pool, tenantA, map[string]any{"created_at": base.Add(time.Duration(i) * time.Hour)})
		newestID = id // last iteration has the latest created_at
	}
	otherID := seedInvoice(t, pool, tenantB, map[string]any{"created_at": base.Add(100 * time.Hour)})

	dateFrom := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	metrics, err := repo.GetDashboardMetrics(ctxA, tenantA, dateFrom, dateTo)
	if err != nil {
		t.Fatalf("GetDashboardMetrics: %v", err)
	}

	if len(metrics.RecentInvoices) != 10 {
		t.Fatalf("RecentInvoices count = %d, want 10 (capped, 11 seeded)", len(metrics.RecentInvoices))
	}
	if metrics.RecentInvoices[0].ID != newestID {
		t.Errorf("RecentInvoices[0].ID = %s, want the most recently created invoice %s", metrics.RecentInvoices[0].ID, newestID)
	}
	for _, inv := range metrics.RecentInvoices {
		if inv.ID == otherID {
			t.Fatalf("RecentInvoices leaked another tenant's invoice %s", otherID)
		}
	}
}

// TestGetDashboardMetrics_ExpiringQuotesWithinSevenDays covers Query 5's
// three-way filter: status=sent, valid_until set, and within 7 days from now.
func TestGetDashboardMetrics_ExpiringQuotesWithinSevenDays(t *testing.T) {
	pool, tenantA, ctxA := newDashboardFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Dashboard Tenant B")
	repo := NewPostgresRepository(pool)

	soon := time.Now().Add(3 * 24 * time.Hour).Format("2006-01-02")
	tooFar := time.Now().Add(10 * 24 * time.Hour).Format("2006-01-02")

	expiring := seedQuote(t, pool, tenantA, map[string]any{"status": models.QuoteStatusSent, "valid_until": soon})
	seedQuote(t, pool, tenantA, map[string]any{"status": models.QuoteStatusSent, "valid_until": tooFar})
	seedQuote(t, pool, tenantA, map[string]any{"status": models.QuoteStatusDraft, "valid_until": soon})
	seedQuote(t, pool, tenantB, map[string]any{"status": models.QuoteStatusSent, "valid_until": soon})

	dateFrom := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	metrics, err := repo.GetDashboardMetrics(ctxA, tenantA, dateFrom, dateTo)
	if err != nil {
		t.Fatalf("GetDashboardMetrics: %v", err)
	}

	if len(metrics.ExpiringQuotes) != 1 {
		t.Fatalf("ExpiringQuotes count = %d, want 1", len(metrics.ExpiringQuotes))
	}
	if metrics.ExpiringQuotes[0].ID != expiring {
		t.Errorf("ExpiringQuotes[0].ID = %s, want %s", metrics.ExpiringQuotes[0].ID, expiring)
	}
}

// TestGetDashboardMetrics_PendingDunningRecordsScopedByTenant covers Query 6:
// only status='draft', only this tenant.
func TestGetDashboardMetrics_PendingDunningRecordsScopedByTenant(t *testing.T) {
	pool, tenantA, ctxA := newDashboardFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Dashboard Tenant B")
	repo := NewPostgresRepository(pool)

	invA1 := seedInvoice(t, pool, tenantA, nil)
	invA2 := seedInvoice(t, pool, tenantA, nil)
	invB := seedInvoice(t, pool, tenantB, nil)

	pending := seedDunning(t, pool, tenantA, invA1, map[string]any{"status": models.DunningStatusDraft})
	seedDunning(t, pool, tenantA, invA2, map[string]any{"status": models.DunningStatusSent})
	seedDunning(t, pool, tenantB, invB, map[string]any{"status": models.DunningStatusDraft})

	dateFrom := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	metrics, err := repo.GetDashboardMetrics(ctxA, tenantA, dateFrom, dateTo)
	if err != nil {
		t.Fatalf("GetDashboardMetrics: %v", err)
	}

	if len(metrics.PendingDunnings) != 1 {
		t.Fatalf("PendingDunnings count = %d, want 1", len(metrics.PendingDunnings))
	}
	if metrics.PendingDunnings[0].ID != pending {
		t.Errorf("PendingDunnings[0].ID = %s, want %s", metrics.PendingDunnings[0].ID, pending)
	}
}

// TestGetDashboardMetrics_CanceledContextReturnsError is the error path: a
// query against an already-canceled context must surface as an error, not a
// panic and not a silently empty result.
func TestGetDashboardMetrics_CanceledContextReturnsError(t *testing.T) {
	pool, tenantID, ctx := newDashboardFixture(t)
	repo := NewPostgresRepository(pool)

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	_, err := repo.GetDashboardMetrics(canceledCtx, tenantID,
		time.Now().AddDate(0, -1, 0), time.Now())
	if err == nil {
		t.Fatal("GetDashboardMetrics with a canceled context: got nil error, want one")
	}
}
