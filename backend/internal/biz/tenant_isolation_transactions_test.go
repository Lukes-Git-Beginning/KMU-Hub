package biz

// Finance transactions (Transaktionen) merge two tables read by explicit
// tenant_id = $1, not RLS alone -- finance_payments (joined to
// finance_invoices for the description/reference) and finance_expenses
// (approved only). Two things only a real database proves:
//
//  1. the merge is right: a payment becomes an income entry described by its
//     invoice, an approved expense becomes an expense entry, and a
//     pending/rejected expense is excluded entirely.
//  2. the query is tenant-scoped both ways: a leak here would expose another
//     tenant's customers, payments and expenses.
//
// Skips cleanly without DATABASE_URL (CI runners without Postgres).

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Transactions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Transactions Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Transactions Tenant B")

	now := time.Now().UTC().Truncate(24 * time.Hour)
	day := func(offset int) time.Time { return now.AddDate(0, 0, offset) }

	invoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"tenant_id":      tenantA,
		"invoice_number": "RE-TXTEST-9001",
		"status":         models.InvoiceStatusSent,
		"customer_name":  "Ledger GmbH",
		"gross_total":    "1000.00",
		"currency":       "EUR",
		"invoice_date":   day(-7).Format("2006-01-02"),
		"due_date":       day(23).Format("2006-01-02"),
		"created_by":     tenantA,
	})
	defer testutil.CleanupRow(t, pool, "finance_invoices", invoiceID)

	paymentID := testutil.SeedRow(t, pool, "finance_payments", map[string]any{
		"tenant_id":    tenantA,
		"invoice_id":   invoiceID,
		"amount":       "400.00",
		"payment_date": day(-5).Format("2006-01-02"),
		"created_by":   tenantA,
	})
	defer testutil.CleanupRow(t, pool, "finance_payments", paymentID)

	approvedExpenseID := testutil.SeedRow(t, pool, "finance_expenses", map[string]any{
		"tenant_id":    tenantA,
		"description":  "Bürobedarf Staples",
		"amount":       "234.50",
		"expense_date": day(-3).Format("2006-01-02"),
		"category":     "Büromaterial",
		"status":       models.ExpenseStatusApproved,
	})
	defer testutil.CleanupRow(t, pool, "finance_expenses", approvedExpenseID)

	// Neither ever moved money and must not appear in the ledger.
	pendingExpenseID := testutil.SeedRow(t, pool, "finance_expenses", map[string]any{
		"tenant_id":    tenantA,
		"description":  "Noch nicht entschieden",
		"amount":       "50.00",
		"expense_date": day(-2).Format("2006-01-02"),
		"category":     "Sonstiges",
		"status":       models.ExpenseStatusPending,
	})
	defer testutil.CleanupRow(t, pool, "finance_expenses", pendingExpenseID)

	rejectedExpenseID := testutil.SeedRow(t, pool, "finance_expenses", map[string]any{
		"tenant_id":    tenantA,
		"description":  "Abgelehnt",
		"amount":       "75.00",
		"expense_date": day(-1).Format("2006-01-02"),
		"category":     "Sonstiges",
		"status":       models.ExpenseStatusRejected,
	})
	defer testutil.CleanupRow(t, pool, "finance_expenses", rejectedExpenseID)

	// Tenant B's own payment and approved expense — the isolation control.
	otherInvoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"tenant_id":      tenantB,
		"invoice_number": "RE-TXTEST-8001",
		"status":         models.InvoiceStatusSent,
		"customer_name":  "Andere AG",
		"gross_total":    "200.00",
		"currency":       "EUR",
		"invoice_date":   day(-1).Format("2006-01-02"),
		"due_date":       day(29).Format("2006-01-02"),
		"created_by":     tenantB,
	})
	defer testutil.CleanupRow(t, pool, "finance_invoices", otherInvoiceID)

	otherPaymentID := testutil.SeedRow(t, pool, "finance_payments", map[string]any{
		"tenant_id":    tenantB,
		"invoice_id":   otherInvoiceID,
		"amount":       "50.00",
		"payment_date": day(-1).Format("2006-01-02"),
		"created_by":   tenantB,
	})
	defer testutil.CleanupRow(t, pool, "finance_payments", otherPaymentID)

	otherExpenseID := testutil.SeedRow(t, pool, "finance_expenses", map[string]any{
		"tenant_id":    tenantB,
		"description":  "Andere Ausgabe",
		"amount":       "10.00",
		"expense_date": day(-1).Format("2006-01-02"),
		"category":     "Sonstiges",
		"status":       models.ExpenseStatusApproved,
	})
	defer testutil.CleanupRow(t, pool, "finance_expenses", otherExpenseID)

	repo := invoice.NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	t.Run("owner sees payment and approved expense only, newest first", func(t *testing.T) {
		transactions, err := repo.ListTransactions(ctxA, tenantA)
		if err != nil {
			t.Fatalf("ListTransactions: %v", err)
		}
		if len(transactions) != 2 {
			t.Fatalf("expected 2 transactions (payment + approved expense), got %d: %+v", len(transactions), transactions)
		}

		// expense_date (-3) is more recent than payment_date (-5): expense first.
		expenseTx, incomeTx := transactions[0], transactions[1]

		if expenseTx.ID != "exp-"+approvedExpenseID.String() {
			t.Errorf("expenseTx.ID = %s, want exp- prefix on %s", expenseTx.ID, approvedExpenseID)
		}
		if expenseTx.Type != "expense" || expenseTx.Description != "Bürobedarf Staples" ||
			expenseTx.Category != "Büromaterial" || expenseTx.Status != "completed" {
			t.Errorf("expense transaction = %+v", expenseTx)
		}
		if !expenseTx.Amount.Equal(decimal.RequireFromString("234.50")) {
			t.Errorf("expenseTx.Amount = %s, want 234.50", expenseTx.Amount)
		}
		if expenseTx.Reference != "" || expenseTx.InvoiceID != nil {
			t.Errorf("expense transaction should carry no invoice reference, got %+v", expenseTx)
		}

		if incomeTx.ID != "pay-"+paymentID.String() {
			t.Errorf("incomeTx.ID = %s, want pay- prefix on %s", incomeTx.ID, paymentID)
		}
		if incomeTx.Type != "income" || incomeTx.Category != "Umsatzerlöse" || incomeTx.Status != "completed" {
			t.Errorf("income transaction = %+v", incomeTx)
		}
		wantDescription := "Rechnung RE-TXTEST-9001 Ledger GmbH"
		if incomeTx.Description != wantDescription {
			t.Errorf("incomeTx.Description = %q, want %q", incomeTx.Description, wantDescription)
		}
		if incomeTx.Reference != "RE-TXTEST-9001" {
			t.Errorf("incomeTx.Reference = %q, want the invoice number", incomeTx.Reference)
		}
		if incomeTx.InvoiceID == nil || *incomeTx.InvoiceID != invoiceID {
			t.Errorf("incomeTx.InvoiceID = %v, want %s", incomeTx.InvoiceID, invoiceID)
		}
		if !incomeTx.Amount.Equal(decimal.RequireFromString("400.00")) {
			t.Errorf("incomeTx.Amount = %s, want 400.00", incomeTx.Amount)
		}
	})

	t.Run("other tenant sees none of it", func(t *testing.T) {
		transactions, err := repo.ListTransactions(ctxB, tenantB)
		if err != nil {
			t.Fatalf("ListTransactions: %v", err)
		}
		if len(transactions) != 2 {
			t.Fatalf("expected exactly tenant B's own 2 transactions, got %d: %+v", len(transactions), transactions)
		}
		for _, tx := range transactions {
			if tx.ID == "pay-"+paymentID.String() || tx.ID == "exp-"+approvedExpenseID.String() {
				t.Fatalf("tenant B saw tenant A's transaction: %+v", tx)
			}
		}
	})
}
