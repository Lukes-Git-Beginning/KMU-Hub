package pipelinestage

// fix-crm-pipeline-totals-blind-currency-sum: deals.currency is free per deal, so the
// Kanban header total must only add deals in the tenant default currency, while
// deal_count keeps counting every deal of the stage.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestListWithStats_SumsOnlyTenantDefaultCurrencyButCountsAllDeals(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Stage Currency Tenant")
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "email": fmt.Sprintf("stage-cur-%s@test.invalid", uuid.New()),
		"password_hash": "x", "tenant_id": tenantID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "name": "Stage-" + uuid.NewString()[:8],
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "pipeline_stages", stageID) })

	seedDeal := func(value, currency string) {
		id := testutil.SeedRow(t, pool, "deals", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "name": "Deal-" + uuid.NewString()[:8],
			"value": value, "currency": currency, "stage_id": stageID, "created_by": userID,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "deals", id) })
	}
	seedDeal("10000.00", "EUR")
	seedDeal("10000.00", "USD")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	find := func() (count int, total decimal.Decimal) {
		t.Helper()
		stages, err := repo.ListWithStats(ctx, tenantID)
		if err != nil {
			t.Fatalf("ListWithStats: %v", err)
		}
		for _, s := range stages {
			if s.ID == stageID {
				return s.DealCount, s.TotalValue
			}
		}
		t.Fatalf("stage %s missing from ListWithStats", stageID)
		return 0, decimal.Zero
	}

	// No company_settings row: default is EUR.
	count, total := find()
	if count != 2 {
		t.Errorf("deal_count = %d, want 2 (the column shows every deal of the stage)", count)
	}
	if !total.Equal(decimal.RequireFromString("10000")) {
		t.Errorf("total_value = %s, want 10000 (EUR deal only, the USD deal must not be added)", total)
	}

	// default_currency USD flips which deal counts, proving the default is resolved, not hardcoded.
	settingsID := testutil.SeedRow(t, pool, "company_settings", map[string]any{"tenant_id": tenantID, "default_currency": "USD"})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "company_settings", settingsID) })
	seedDeal("250.00", "USD")

	count, total = find()
	if count != 3 {
		t.Errorf("deal_count = %d, want 3", count)
	}
	if !total.Equal(decimal.RequireFromString("10250")) {
		t.Errorf("total_value = %s, want 10250 (both USD deals, EUR deal excluded)", total)
	}
}
