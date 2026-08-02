package biz

// Document chains (Belegkette) join five tables — finance_quotes,
// finance_invoices, finance_payments, finance_dunning_records,
// finance_credit_notes — and every read is scoped by an explicit
// tenant_id = $1, not RLS alone. Two things only a real database proves:
//
//  1. the assembly is right: a quote that converted into an invoice becomes
//     its prefix node, payments/dunning/credit notes attach to the right
//     invoice, and an invoice not yet fully covered gets a trailing pending
//     placeholder for the remainder.
//  2. the query is tenant-scoped both ways: a leak here would expose another
//     tenant's customers and invoice amounts.
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

func TestTenantIsolation_DocumentChains(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Chains Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Chains Tenant B")

	now := time.Now().UTC().Truncate(24 * time.Hour)
	day := func(offset int) time.Time { return now.AddDate(0, 0, offset) }

	quoteID := testutil.SeedRow(t, pool, "finance_quotes", map[string]any{
		"tenant_id":     tenantA,
		"quote_number":  "AN-CHAINTEST-9001",
		"status":        models.QuoteStatusAccepted,
		"customer_name": "Kette GmbH",
		"gross_total":   "1000.00",
		"currency":      "EUR",
		"created_at":    day(-10),
		"created_by":    tenantA,
	})
	defer testutil.CleanupRow(t, pool, "finance_quotes", quoteID)

	invoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"tenant_id":       tenantA,
		"invoice_number":  "RE-CHAINTEST-9001",
		"status":          models.InvoiceStatusSent,
		"customer_name":   "Kette GmbH",
		"gross_total":     "1000.00",
		"currency":        "EUR",
		"invoice_date":    day(-7).Format("2006-01-02"),
		"due_date":        day(23).Format("2006-01-02"),
		"source_quote_id": quoteID,
		"created_by":      tenantA,
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

	dunningID := testutil.SeedRow(t, pool, "finance_dunning_records", map[string]any{
		"tenant_id":  tenantA,
		"invoice_id": invoiceID,
		"level":      1,
		"status":     models.DunningStatusSent,
		"sent_at":    day(-3),
		"created_by": tenantA,
	})
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", dunningID)

	creditNoteID := testutil.SeedRow(t, pool, "finance_credit_notes", map[string]any{
		"tenant_id":           tenantA,
		"credit_note_number":  "GS-CHAINTEST-9001",
		"status":              models.CreditNoteStatusSent,
		"original_invoice_id": invoiceID,
		"customer_name":       "Kette GmbH",
		"gross_total":         "100.00",
		"currency":            "EUR",
		"created_at":          day(-1),
		"created_by":          tenantA,
	})
	defer testutil.CleanupRow(t, pool, "finance_credit_notes", creditNoteID)

	// A quote that never became an invoice — its own one-node chain.
	standaloneQuoteID := testutil.SeedRow(t, pool, "finance_quotes", map[string]any{
		"tenant_id":     tenantA,
		"quote_number":  "AN-CHAINTEST-9002",
		"status":        models.QuoteStatusRejected,
		"customer_name": "Solo AG",
		"gross_total":   "50.00",
		"currency":      "CHF",
		"created_at":    day(-2),
		"created_by":    tenantA,
	})
	defer testutil.CleanupRow(t, pool, "finance_quotes", standaloneQuoteID)

	// Tenant B's own invoice — the isolation control.
	otherInvoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"tenant_id":      tenantB,
		"invoice_number": "RE-CHAINTEST-8001",
		"status":         models.InvoiceStatusSent,
		"customer_name":  "Andere AG",
		"gross_total":    "200.00",
		"currency":       "EUR",
		"invoice_date":   day(-1).Format("2006-01-02"),
		"due_date":       day(29).Format("2006-01-02"),
		"created_by":     tenantB,
	})
	defer testutil.CleanupRow(t, pool, "finance_invoices", otherInvoiceID)

	repo := invoice.NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	t.Run("owner sees the assembled chain", func(t *testing.T) {
		chains, err := repo.ListDocumentChains(ctxA, tenantA)
		if err != nil {
			t.Fatalf("ListDocumentChains: %v", err)
		}
		if len(chains) != 2 {
			t.Fatalf("expected 2 chains (invoice + standalone quote), got %d", len(chains))
		}

		var invoiceChain, quoteChain *models.DocumentChain
		for _, c := range chains {
			switch c.ID {
			case invoiceID:
				invoiceChain = c
			case standaloneQuoteID:
				quoteChain = c
			}
		}
		if invoiceChain == nil {
			t.Fatal("invoice-anchored chain missing")
			return
		}
		if quoteChain == nil {
			t.Fatal("standalone quote chain missing")
			return
		}

		if invoiceChain.Customer != "Kette GmbH" || invoiceChain.Currency != "EUR" {
			t.Errorf("invoice chain header = %+v", invoiceChain)
		}
		if invoiceChain.IsComplete {
			t.Error("invoice chain should not be complete: 500 of 1000 still open")
		}
		if !invoiceChain.TotalValue.Equal(decimal.RequireFromString("1000")) {
			t.Errorf("invoice chain total = %s, want 1000", invoiceChain.TotalValue)
		}

		wantTypes := []string{
			models.ChainNodeQuote, models.ChainNodeInvoice, models.ChainNodePayment,
			models.ChainNodeDunning, models.ChainNodeCreditNote, models.ChainNodePayment,
		}
		if len(invoiceChain.Nodes) != len(wantTypes) {
			t.Fatalf("node count = %d, want %d (%+v)", len(invoiceChain.Nodes), len(wantTypes), invoiceChain.Nodes)
		}
		for i, wantType := range wantTypes {
			if got := invoiceChain.Nodes[i].Type; got != wantType {
				t.Errorf("node[%d].Type = %s, want %s", i, got, wantType)
			}
		}

		quoteNode, invNode, payNode, dunNode, creditNode, placeholder :=
			invoiceChain.Nodes[0], invoiceChain.Nodes[1], invoiceChain.Nodes[2],
			invoiceChain.Nodes[3], invoiceChain.Nodes[4], invoiceChain.Nodes[5]

		if quoteNode.Number != "AN-CHAINTEST-9001" || quoteNode.Status != models.ChainNodeCompleted {
			t.Errorf("quote node = %+v", quoteNode)
		}
		if invNode.Number != "RE-CHAINTEST-9001" || invNode.Status != models.ChainNodeActive {
			t.Errorf("invoice node = %+v", invNode)
		}
		// Reference was left empty, so the payment node falls back to the
		// invoice number.
		if payNode.Number != "RE-CHAINTEST-9001" || payNode.Status != models.ChainNodeCompleted ||
			!payNode.Amount.Equal(decimal.RequireFromString("400")) {
			t.Errorf("payment node = %+v", payNode)
		}
		wantDunningNumber := "RE-CHAINTEST-9001/M1"
		if dunNode.Number != wantDunningNumber || dunNode.Status != models.ChainNodeActive ||
			!dunNode.Amount.Equal(decimal.RequireFromString("1000")) {
			t.Errorf("dunning node = %+v, want number %s", dunNode, wantDunningNumber)
		}
		if creditNode.Number != "GS-CHAINTEST-9001" || creditNode.Status != models.ChainNodeCompleted ||
			!creditNode.Amount.Equal(decimal.RequireFromString("100")) {
			t.Errorf("credit note node = %+v", creditNode)
		}
		if placeholder.Number != "" || placeholder.Date != nil || placeholder.Status != models.ChainNodePending ||
			!placeholder.Amount.Equal(decimal.RequireFromString("500")) {
			t.Errorf("pending placeholder = %+v, want amount 500 with no number/date", placeholder)
		}

		if quoteChain.IsComplete != true {
			t.Error("rejected standalone quote chain should be complete")
		}
		if len(quoteChain.Nodes) != 1 || quoteChain.Nodes[0].Number != "AN-CHAINTEST-9002" ||
			quoteChain.Nodes[0].Status != models.ChainNodeCancelled {
			t.Errorf("standalone quote chain nodes = %+v", quoteChain.Nodes)
		}
	})

	t.Run("other tenant sees none of it", func(t *testing.T) {
		chains, err := repo.ListDocumentChains(ctxB, tenantB)
		if err != nil {
			t.Fatalf("ListDocumentChains: %v", err)
		}
		if len(chains) != 1 {
			t.Fatalf("expected exactly tenant B's own chain, got %d", len(chains))
		}
		if chains[0].ID != otherInvoiceID {
			t.Fatalf("tenant B's chain id = %s, want %s", chains[0].ID, otherInvoiceID)
		}
		if chains[0].Customer != "Andere AG" {
			t.Errorf("tenant B chain leaked tenant A data: customer = %s", chains[0].Customer)
		}
	})
}
