package downstream

// Integration test for the cross-module KPI aggregation (Migration-independent —
// reads finance_invoices, deals, tickets, stock_warnings).
//
// The assertions are deltas around a seed, not absolute values: other packages'
// RLS tests run in parallel against the same database, and a leftover row from
// an aborted run must not turn this test red. A delta also proves the point the
// test exists for — the foreign tenant's rows must move nothing.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// Dedicated tenants so parallel package tests seeding TenantA/TenantB cannot
// perturb the deltas measured here.
var (
	kpiTenant      = uuid.MustParse("cccc0000-0000-0000-0000-000000000001")
	kpiOtherTenant = uuid.MustParse("cccc0000-0000-0000-0000-000000000002")
)

func TestKPISnapshot_AggregatesOnlyOwnTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	testutil.EnsureTenant(t, pool, kpiTenant, "KPI Tenant")
	testutil.EnsureTenant(t, pool, kpiOtherTenant, "KPI Other Tenant")

	repo := NewPostgresKPIRepo(pool)
	ctx := testutil.WithTenantCtx(context.Background(), kpiTenant)

	// Period bounds come from the database clock: the point-in-time values
	// compare against created_at, which Postgres stamps. Taking `to` from the Go
	// clock makes the assertions hostage to host-vs-container clock skew.
	now := dbNow(t, pool)
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	before, err := repo.KPISnapshot(ctx, kpiTenant, from, now)
	if err != nil {
		t.Fatalf("baseline snapshot: %v", err)
	}

	// Seed the same four rows for both tenants, with different amounts.
	seedKPIFixtures(t, pool, kpiTenant, "1000.00", "500.00")
	seedKPIFixtures(t, pool, kpiOtherTenant, "777.00", "999.00")

	after, err := repo.KPISnapshot(ctx, kpiTenant, from, dbNow(t, pool))
	if err != nil {
		t.Fatalf("snapshot after seed: %v", err)
	}

	if got, want := after.Revenue.Sub(before.Revenue), decimal.RequireFromString("1000.00"); !got.Equal(want) {
		t.Errorf("revenue delta = %s, want %s (foreign tenant must not count)", got, want)
	}
	if got, want := after.PipelineVolume.Sub(before.PipelineVolume), decimal.RequireFromString("500.00"); !got.Equal(want) {
		t.Errorf("pipeline delta = %s, want %s (foreign tenant must not count)", got, want)
	}
	if got := after.OpenTickets - before.OpenTickets; got != 1 {
		t.Errorf("open ticket delta = %d, want 1", got)
	}
	if got := after.StockWarnings - before.StockWarnings; got != 1 {
		t.Errorf("stock warning delta = %d, want 1", got)
	}
}

// A closed deal and a resolved ticket are gone from the point-in-time values.
func TestKPISnapshot_ExcludesClosedDealsAndResolvedTickets(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	testutil.EnsureTenant(t, pool, kpiTenant, "KPI Tenant")

	repo := NewPostgresKPIRepo(pool)
	ctx := testutil.WithTenantCtx(context.Background(), kpiTenant)

	now := dbNow(t, pool)
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	before, err := repo.KPISnapshot(ctx, kpiTenant, from, now)
	if err != nil {
		t.Fatalf("baseline snapshot: %v", err)
	}

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("kpi-closed-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     kpiTenant,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": kpiTenant,
		"name":      fmt.Sprintf("KPI-Stage-%s", uuid.New()),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)

	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  kpiTenant,
		"name":       "Closed Deal",
		"value":      "4200.00",
		"stage_id":   stageID,
		"created_by": userID,
		"closed_at":  now.Add(-time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "deals", dealID)

	ticketID := testutil.SeedRow(t, pool, "tickets", map[string]any{
		"id":           uuid.New(),
		"tenant_id":    kpiTenant,
		"subject":      "Resolved ticket",
		"status":       "solved",
		"requester_id": uuid.New(),
		"resolved_at":  now.Add(-time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "tickets", ticketID)

	after, err := repo.KPISnapshot(ctx, kpiTenant, from, dbNow(t, pool))
	if err != nil {
		t.Fatalf("snapshot after seed: %v", err)
	}

	if got := after.PipelineVolume.Sub(before.PipelineVolume); !got.IsZero() {
		t.Errorf("pipeline delta = %s, want 0 — a closed deal is not open pipeline", got)
	}
	if got := after.OpenTickets - before.OpenTickets; got != 0 {
		t.Errorf("open ticket delta = %d, want 0 — a resolved ticket is not open", got)
	}
}

// TestKPISeries_BucketsByCalendarMonthAndExcludesForeignTenant proves the
// series query's period bucketing, not just its aggregates: a fixture dated
// into last month, closed/resolved exactly at this month's boundary, must
// land in the second-to-last bucket and move nothing in the last (current)
// one — the boundary itself is the part a Go-side loop over KPISnapshot
// calls would get subtly wrong (off-by-one on which side of midnight a
// closed_at exactly on the boundary falls).
func TestKPISeries_BucketsByCalendarMonthAndExcludesForeignTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	testutil.EnsureTenant(t, pool, kpiTenant, "KPI Tenant")
	testutil.EnsureTenant(t, pool, kpiOtherTenant, "KPI Other Tenant")

	repo := NewPostgresKPIRepo(pool)
	ctx := testutil.WithTenantCtx(context.Background(), kpiTenant)

	// .UTC() matters: dbNow's pgx scan can carry a non-UTC Location (the test
	// host's local zone) even though the underlying instant is correct. The
	// query buckets in the database's session time zone (UTC, verified by
	// TestKPISnapshot's own tenant-scoped assertions running clean); building
	// currentMonthStart from a non-UTC Location would shift these fixtures'
	// timestamps by the zone offset and land them on the wrong side of a
	// boundary that is supposed to be exact.
	now := dbNow(t, pool).UTC()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastMonthStart := currentMonthStart.AddDate(0, -1, 0)

	before, err := repo.KPISeries(ctx, kpiTenant, now, 8)
	if err != nil {
		t.Fatalf("baseline series: %v", err)
	}
	if len(before) != 8 {
		t.Fatalf("expected 8 buckets, got %d", len(before))
	}
	if !before[7].PeriodStart.Equal(currentMonthStart) {
		t.Errorf("last bucket period_start = %s, want %s (current month)", before[7].PeriodStart, currentMonthStart)
	}
	for i := 1; i < len(before); i++ {
		want := before[i-1].PeriodStart.AddDate(0, 1, 0)
		if !before[i].PeriodStart.Equal(want) {
			t.Errorf("bucket %d period_start = %s, want %s (one calendar month after bucket %d)", i, before[i].PeriodStart, want, i-1)
		}
	}

	seedKPISeriesFixture(t, pool, kpiTenant, lastMonthStart, currentMonthStart)
	seedKPISeriesFixture(t, pool, kpiOtherTenant, lastMonthStart, currentMonthStart)

	after, err := repo.KPISeries(ctx, kpiTenant, now, 8)
	if err != nil {
		t.Fatalf("series after seed: %v", err)
	}

	if got := after[7].Revenue.Sub(before[7].Revenue); !got.IsZero() {
		t.Errorf("current-month revenue delta = %s, want 0 — the invoice is dated last month", got)
	}
	if got := after[7].PipelineVolume.Sub(before[7].PipelineVolume); !got.IsZero() {
		t.Errorf("current-month pipeline delta = %s, want 0 — the deal closed exactly at this month's start", got)
	}
	if got := after[7].OpenTickets - before[7].OpenTickets; got != 0 {
		t.Errorf("current-month open ticket delta = %d, want 0 — the ticket resolved exactly at this month's start", got)
	}

	if got, want := after[6].Revenue.Sub(before[6].Revenue), decimal.RequireFromString("321.00"); !got.Equal(want) {
		t.Errorf("last-month revenue delta = %s, want %s", got, want)
	}
	if got, want := after[6].PipelineVolume.Sub(before[6].PipelineVolume), decimal.RequireFromString("654.00"); !got.Equal(want) {
		t.Errorf("last-month pipeline delta = %s, want %s — the deal was still open through last month's end", got, want)
	}
	if got := after[6].OpenTickets - before[6].OpenTickets; got != 1 {
		t.Errorf("last-month open ticket delta = %d, want 1 — the ticket was still open through last month's end", got)
	}
}

// TestKPISnapshot_ForeignCurrencyInvoiceAndDealExcluded covers
// fix-berichte-kpi-blind-currency-sum: a tenant with no company_settings row
// (default currency implicitly EUR) has one EUR and one CHF invoice, and one
// EUR and one CHF deal, all in range. The CHF rows must not be blindly summed
// into the EUR-labeled revenue and pipeline_volume KPI tiles (executor.go
// hardcodes Unit: "EUR" on both).
func TestKPISnapshot_ForeignCurrencyInvoiceAndDealExcluded(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	testutil.EnsureTenant(t, pool, kpiTenant, "KPI Tenant")

	repo := NewPostgresKPIRepo(pool)
	ctx := testutil.WithTenantCtx(context.Background(), kpiTenant)

	now := dbNow(t, pool)
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	before, err := repo.KPISnapshot(ctx, kpiTenant, from, now)
	if err != nil {
		t.Fatalf("baseline snapshot: %v", err)
	}

	seedKPICurrencyFixtures(t, pool, kpiTenant, "EUR", "111.00", "222.00")
	seedKPICurrencyFixtures(t, pool, kpiTenant, "CHF", "9999.00", "8888.00")

	after, err := repo.KPISnapshot(ctx, kpiTenant, from, dbNow(t, pool))
	if err != nil {
		t.Fatalf("snapshot after seed: %v", err)
	}

	if got, want := after.Revenue.Sub(before.Revenue), decimal.RequireFromString("111.00"); !got.Equal(want) {
		t.Errorf("revenue delta = %s, want %s (CHF invoice must not be blindly summed into EUR)", got, want)
	}
	if got, want := after.PipelineVolume.Sub(before.PipelineVolume), decimal.RequireFromString("222.00"); !got.Equal(want) {
		t.Errorf("pipeline delta = %s, want %s (CHF deal must not be blindly summed into EUR)", got, want)
	}
}

// TestKPISnapshot_UsesTenantDefaultCurrencyNotHardcodedEUR covers a tenant
// whose company_settings.default_currency is CHF: their own CHF invoice and
// deal must count, and a stray EUR invoice and deal must be excluded —
// proving the filter resolves the tenant's actual default rather than
// hardcoding "EUR".
func TestKPISnapshot_UsesTenantDefaultCurrencyNotHardcodedEUR(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	testutil.EnsureTenant(t, pool, kpiTenant, "KPI Tenant")

	settingsID := testutil.SeedRow(t, pool, "company_settings", map[string]any{"tenant_id": kpiTenant, "default_currency": "CHF"})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "company_settings", settingsID) })

	repo := NewPostgresKPIRepo(pool)
	ctx := testutil.WithTenantCtx(context.Background(), kpiTenant)

	now := dbNow(t, pool)
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	before, err := repo.KPISnapshot(ctx, kpiTenant, from, now)
	if err != nil {
		t.Fatalf("baseline snapshot: %v", err)
	}

	seedKPICurrencyFixtures(t, pool, kpiTenant, "CHF", "333.00", "444.00")
	seedKPICurrencyFixtures(t, pool, kpiTenant, "EUR", "7777.00", "6666.00")

	after, err := repo.KPISnapshot(ctx, kpiTenant, from, dbNow(t, pool))
	if err != nil {
		t.Fatalf("snapshot after seed: %v", err)
	}

	if got, want := after.Revenue.Sub(before.Revenue), decimal.RequireFromString("333.00"); !got.Equal(want) {
		t.Errorf("revenue delta = %s, want %s (tenant's default currency is CHF, the stray EUR invoice must be excluded)", got, want)
	}
	if got, want := after.PipelineVolume.Sub(before.PipelineVolume), decimal.RequireFromString("444.00"); !got.Equal(want) {
		t.Errorf("pipeline delta = %s, want %s (tenant's default currency is CHF, the stray EUR deal must be excluded)", got, want)
	}
}

// seedKPICurrencyFixtures seeds one open invoice and one open deal in the
// given currency, registering cleanup for each and their FK dependencies.
func seedKPICurrencyFixtures(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, currency, grossTotal, dealValue string) {
	t.Helper()

	today := time.Now().UTC().Format("2006-01-02")
	due := time.Now().UTC().Add(30 * 24 * time.Hour).Format("2006-01-02")

	invoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"tenant_id":      tenantID,
		"invoice_number": fmt.Sprintf("KPICUR-%s", uuid.New().String()[:16]), // column is varchar(30)
		"invoice_date":   today,
		"due_date":       due,
		"status":         "sent",
		"gross_total":    grossTotal,
		"currency":       currency,
		"created_by":     uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", invoiceID) })

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("kpi-cur-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     tenantID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": tenantID,
		"name":      fmt.Sprintf("KPI-Cur-Stage-%s", uuid.New()),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "pipeline_stages", stageID) })

	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  tenantID,
		"name":       "KPI Currency Deal",
		"value":      dealValue,
		"currency":   currency,
		"stage_id":   stageID,
		"created_by": userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "deals", dealID) })
}

// dbNow reads the database clock, the same one that stamps created_at.
func dbNow(t *testing.T, pool *pgxpool.Pool) time.Time {
	t.Helper()
	var now time.Time
	if err := pool.QueryRow(testutil.WithSystemCtx(context.Background()), "SELECT NOW()").Scan(&now); err != nil {
		t.Fatalf("read database clock: %v", err)
	}
	return now
}

// seedKPIFixtures seeds one invoice, one open deal, one open ticket and one
// active stock warning for the given tenant, registering cleanup for each.
func seedKPIFixtures(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, grossTotal, dealValue string) {
	t.Helper()

	today := time.Now().UTC().Format("2006-01-02")
	due := time.Now().UTC().Add(30 * 24 * time.Hour).Format("2006-01-02")

	invoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"tenant_id":      tenantID,
		"invoice_number": fmt.Sprintf("KPI-%s", uuid.New().String()[:20]), // column is varchar(30)
		"invoice_date":   today,
		"due_date":       due,
		"status":         "sent",
		"gross_total":    grossTotal,
		"created_by":     uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", invoiceID) })

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("kpi-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     tenantID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": tenantID,
		"name":      fmt.Sprintf("KPI-Stage-%s", uuid.New()),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "pipeline_stages", stageID) })

	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  tenantID,
		"name":       "KPI Deal",
		"value":      dealValue,
		"stage_id":   stageID,
		"created_by": userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "deals", dealID) })

	ticketID := testutil.SeedRow(t, pool, "tickets", map[string]any{
		"id":           uuid.New(),
		"tenant_id":    tenantID,
		"subject":      "KPI ticket",
		"requester_id": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tickets", ticketID) })

	itemID := testutil.SeedRow(t, pool, "inventory_items", map[string]any{
		"id":        uuid.New(),
		"tenant_id": tenantID,
		"name":      "KPI Item",
		"sku":       fmt.Sprintf("KPI-%s", uuid.New()),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventory_items", itemID) })

	warningID := testutil.SeedRow(t, pool, "stock_warnings", map[string]any{
		"id":        uuid.New(),
		"tenant_id": tenantID,
		"item_id":   itemID,
		"status":    "active",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "stock_warnings", warningID) })
}

// TestKPISeries_ForeignCurrencyExcludedFromBucket covers
// fix-berichte-kpi-blind-currency-sum for kpiSeriesQuery: a CHF invoice and
// deal dated into last month's bucket must not be summed into that bucket's
// EUR-labeled revenue/pipeline_volume, mirroring
// TestKPISnapshot_ForeignCurrencyInvoiceAndDealExcluded for the snapshot
// query.
func TestKPISeries_ForeignCurrencyExcludedFromBucket(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	testutil.EnsureTenant(t, pool, kpiTenant, "KPI Tenant")

	repo := NewPostgresKPIRepo(pool)
	ctx := testutil.WithTenantCtx(context.Background(), kpiTenant)

	now := dbNow(t, pool).UTC()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastMonthStart := currentMonthStart.AddDate(0, -1, 0)

	before, err := repo.KPISeries(ctx, kpiTenant, now, 8)
	if err != nil {
		t.Fatalf("baseline series: %v", err)
	}

	seedKPISeriesCurrencyFixture(t, pool, kpiTenant, lastMonthStart, currentMonthStart, "EUR", "111.00", "222.00")
	seedKPISeriesCurrencyFixture(t, pool, kpiTenant, lastMonthStart, currentMonthStart, "CHF", "9999.00", "8888.00")

	after, err := repo.KPISeries(ctx, kpiTenant, now, 8)
	if err != nil {
		t.Fatalf("series after seed: %v", err)
	}

	if got, want := after[6].Revenue.Sub(before[6].Revenue), decimal.RequireFromString("111.00"); !got.Equal(want) {
		t.Errorf("last-month revenue delta = %s, want %s (CHF invoice must not be blindly summed into EUR bucket)", got, want)
	}
	if got, want := after[6].PipelineVolume.Sub(before[6].PipelineVolume), decimal.RequireFromString("222.00"); !got.Equal(want) {
		t.Errorf("last-month pipeline delta = %s, want %s (CHF deal must not be blindly summed into EUR bucket)", got, want)
	}
}

// seedKPISeriesCurrencyFixture is seedKPISeriesFixture with an explicit
// currency and amounts, for the foreign-currency exclusion test above.
func seedKPISeriesCurrencyFixture(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, lastMonthStart, currentMonthStart time.Time, currency, grossTotal, dealValue string) {
	t.Helper()

	invoiceDate := lastMonthStart.AddDate(0, 0, 5).Format("2006-01-02")
	dueDate := currentMonthStart.AddDate(0, 0, 30).Format("2006-01-02")

	invoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"tenant_id":      tenantID,
		"invoice_number": fmt.Sprintf("KPISERCUR-%s", uuid.New().String()[:12]), // column is varchar(30)
		"invoice_date":   invoiceDate,
		"due_date":       dueDate,
		"status":         "sent",
		"gross_total":    grossTotal,
		"currency":       currency,
		"created_by":     uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", invoiceID) })

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("kpi-series-cur-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     tenantID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": tenantID,
		"name":      fmt.Sprintf("KPI-Series-Cur-Stage-%s", uuid.New()),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "pipeline_stages", stageID) })

	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  tenantID,
		"name":       "KPI Series Currency Deal",
		"value":      dealValue,
		"currency":   currency,
		"stage_id":   stageID,
		"created_by": userID,
		"created_at": lastMonthStart,
		"closed_at":  currentMonthStart,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "deals", dealID) })
}

// seedKPISeriesFixture seeds one invoice dated 5 days into lastMonthStart's
// month, and one deal and one ticket created at lastMonthStart and
// closed/resolved exactly at currentMonthStart — so all three belong to last
// month's bucket and must not move the current one.
func seedKPISeriesFixture(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, lastMonthStart, currentMonthStart time.Time) {
	t.Helper()

	invoiceDate := lastMonthStart.AddDate(0, 0, 5).Format("2006-01-02")
	dueDate := currentMonthStart.AddDate(0, 0, 30).Format("2006-01-02")

	invoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"tenant_id":      tenantID,
		"invoice_number": fmt.Sprintf("KPISER-%s", uuid.New().String()[:18]), // column is varchar(30)
		"invoice_date":   invoiceDate,
		"due_date":       dueDate,
		"status":         "sent",
		"gross_total":    "321.00",
		"created_by":     uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", invoiceID) })

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("kpi-series-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     tenantID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id":        uuid.New(),
		"tenant_id": tenantID,
		"name":      fmt.Sprintf("KPI-Series-Stage-%s", uuid.New()),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "pipeline_stages", stageID) })

	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  tenantID,
		"name":       "KPI Series Deal",
		"value":      "654.00",
		"stage_id":   stageID,
		"created_by": userID,
		"created_at": lastMonthStart,
		"closed_at":  currentMonthStart,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "deals", dealID) })

	ticketID := testutil.SeedRow(t, pool, "tickets", map[string]any{
		"id":           uuid.New(),
		"tenant_id":    tenantID,
		"subject":      "KPI Series Ticket",
		"requester_id": uuid.New(),
		"created_at":   lastMonthStart,
		"resolved_at":  currentMonthStart,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tickets", ticketID) })
}
