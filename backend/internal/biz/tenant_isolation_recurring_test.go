package biz

// Cross-tenant isolation for the recurring invoice tables (Migration 000246).
// finance_recurring_invoices carries the customer snapshot and the billing
// template; finance_recurring_runs is the generation ledger. A leak here would
// expose another tenant's customers and their billed amounts, so both tables get
// the same read-side check as the rest of the finance schema.
//
// Skips cleanly without DATABASE_URL (CI runners without Postgres).

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_RecurringInvoices(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	today := time.Now().UTC().Format("2006-01-02")

	recurringID := testutil.SeedRow(t, pool, "finance_recurring_invoices", map[string]any{
		"tenant_id":           testutil.TenantA,
		"title":               "CRM-Lizenz",
		"recurrence_interval": "monthly",
		"start_date":          today,
		"next_run":            today,
	})
	defer testutil.CleanupRow(t, pool, "finance_recurring_invoices", recurringID)

	runID := testutil.SeedRow(t, pool, "finance_recurring_runs", map[string]any{
		"tenant_id":    testutil.TenantA,
		"recurring_id": recurringID,
		"period_date":  today,
	})
	defer testutil.CleanupRow(t, pool, "finance_recurring_runs", runID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"finance_recurring_invoices", "finance_recurring_invoices", recurringID},
		{"finance_recurring_runs", "finance_recurring_runs", runID},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
