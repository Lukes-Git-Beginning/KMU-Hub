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
	defer pool.Close()

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
	defer pool.Close()

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
