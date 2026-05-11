package biz

// Cross-tenant isolation tests for finance tables (Migration 000122).
// company_settings, finance_number_sequences, finance_quotes, finance_invoices,
// finance_credit_notes, finance_payments, finance_dunning_records,
// finance_dunning_config — all have tenant_id NOT NULL and RLS enabled in mig 000122.
//
// Note: finance_credit_notes has a NOT NULL FK to finance_invoices (original_invoice_id),
// finance_payments and finance_dunning_records also FK to finance_invoices.
// We seed the parent first and cascade cleanup.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Finance(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	today := time.Now().UTC().Format("2006-01-02")
	due := time.Now().UTC().Add(30 * 24 * time.Hour).Format("2006-01-02")

	// Seed company_settings (UNIQUE on tenant_id — use conflict-safe insert via SeedRow).
	settingsID := testutil.SeedRow(t, pool, "company_settings", map[string]any{
		"tenant_id": testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "company_settings", settingsID)

	// Seed finance_number_sequences (UNIQUE on tenant_id, document_type, fiscal_year).
	seqID := testutil.SeedRow(t, pool, "finance_number_sequences", map[string]any{
		"tenant_id":     testutil.TenantA,
		"document_type": "invoice",
		"fiscal_year":   2026,
	})
	defer testutil.CleanupRow(t, pool, "finance_number_sequences", seqID)

	// Seed finance_quotes.
	quoteID := testutil.SeedRow(t, pool, "finance_quotes", map[string]any{
		"tenant_id":  testutil.TenantA,
		"created_by": uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "finance_quotes", quoteID)

	// Seed finance_invoices (basis for credit notes, payments, dunning).
	invoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"tenant_id":    testutil.TenantA,
		"invoice_date": today,
		"due_date":     due,
		"created_by":   uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "finance_invoices", invoiceID)

	// Seed finance_credit_notes (references invoiceID).
	creditID := testutil.SeedRow(t, pool, "finance_credit_notes", map[string]any{
		"tenant_id":           testutil.TenantA,
		"original_invoice_id": invoiceID,
		"created_by":          uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "finance_credit_notes", creditID)

	// Seed finance_payments.
	paymentID := testutil.SeedRow(t, pool, "finance_payments", map[string]any{
		"tenant_id":    testutil.TenantA,
		"invoice_id":   invoiceID,
		"amount":       "100.00",
		"payment_date": today,
		"created_by":   uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "finance_payments", paymentID)

	// Seed finance_dunning_records.
	dunningID := testutil.SeedRow(t, pool, "finance_dunning_records", map[string]any{
		"tenant_id":  testutil.TenantA,
		"invoice_id": invoiceID,
		"level":      1,
		"created_by": uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", dunningID)

	// Seed finance_dunning_config (UNIQUE on tenant_id — one per tenant).
	dunningCfgID := testutil.SeedRow(t, pool, "finance_dunning_config", map[string]any{
		"tenant_id":               testutil.TenantA,
		"level1_days_after_due":   7,
		"level2_days_after_level1": 14,
		"level3_days_after_level2": 21,
	})
	defer testutil.CleanupRow(t, pool, "finance_dunning_config", dunningCfgID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"company_settings", "company_settings", settingsID},
		{"finance_number_sequences", "finance_number_sequences", seqID},
		{"finance_quotes", "finance_quotes", quoteID},
		{"finance_invoices", "finance_invoices", invoiceID},
		{"finance_credit_notes", "finance_credit_notes", creditID},
		{"finance_payments", "finance_payments", paymentID},
		{"finance_dunning_records", "finance_dunning_records", dunningID},
		{"finance_dunning_config", "finance_dunning_config", dunningCfgID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
