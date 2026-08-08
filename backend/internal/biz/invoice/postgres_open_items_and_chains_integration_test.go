//go:build integration

package invoice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testsupport/pgtc"
)

// ============================================================================
// Seed helpers (raw SQL — these tables belong to sibling packages, seeding
// directly avoids importing them and keeps this test focused on what the
// invoice repository itself reads).
// ============================================================================

func seedQuote(t *testing.T, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID, status string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO finance_quotes (
			id, tenant_id, quote_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, tax_breakdown,
			subtotal, total_tax, gross_total,
			created_by, created_at, updated_at, currency
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
		id, tenantID, "AN-TEST-"+id.String()[:8], status,
		"Test Kunde", "Musterstraße 1", "kunde@example.com", "",
		models.TaxModeStandard, nil,
		decimal.NewFromFloat(100), decimal.NewFromFloat(19), decimal.NewFromFloat(119),
		uuid.New(), time.Now().UTC(), time.Now().UTC(), "EUR",
	)
	if err != nil {
		t.Fatalf("seed quote: %v", err)
	}
	return id
}

func seedPayment(t *testing.T, pool *pgxpool.Pool, ctx context.Context, tenantID, invoiceID uuid.UUID, amount decimal.Decimal, paymentDate time.Time, reference string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO finance_payments (
			id, tenant_id, invoice_id, amount, payment_date,
			method, reference, notes, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id, tenantID, invoiceID, amount, paymentDate,
		"bank_transfer", reference, "", uuid.New(), time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("seed payment: %v", err)
	}
	return id
}

func seedDunning(t *testing.T, pool *pgxpool.Pool, ctx context.Context, tenantID, invoiceID uuid.UUID, level int, status string, sentAt *time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO finance_dunning_records (
			id, tenant_id, invoice_id, level, status,
			fee, interest, sent_at, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id, tenantID, invoiceID, level, status,
		decimal.NewFromFloat(5), decimal.Zero, sentAt, uuid.New(), time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("seed dunning: %v", err)
	}
	return id
}

func seedCreditNote(t *testing.T, pool *pgxpool.Pool, ctx context.Context, tenantID, invoiceID uuid.UUID, amount decimal.Decimal, status string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO finance_credit_notes (
			id, tenant_id, credit_note_number, status,
			original_invoice_id,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, tax_breakdown,
			subtotal, total_tax, gross_total,
			reason, created_by, created_at, updated_at, currency
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`,
		id, tenantID, "GS-TEST-"+id.String()[:8], status,
		invoiceID,
		"Test Kunde", "Musterstraße 1", "kunde@example.com", "",
		models.TaxModeStandard, nil,
		amount, decimal.Zero, amount,
		"Testgutschrift", uuid.New(), time.Now().UTC(), time.Now().UTC(), "EUR",
	)
	if err != nil {
		t.Fatalf("seed credit note: %v", err)
	}
	return id
}

// overdueInvoice creates a sent invoice due daysOverdue days ago (or in the
// future for a negative value) and returns it after Create.
func overdueInvoice(t *testing.T, ctx context.Context, repo *PostgresRepository, tenantID uuid.UUID, daysOverdue int) *models.Invoice {
	t.Helper()
	inv := makeInvoice(tenantID, twoLines())
	inv.Status = models.InvoiceStatusSent
	inv.Currency = "EUR"
	inv.DueDate = time.Now().UTC().AddDate(0, 0, -daysOverdue)
	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("create overdue invoice: %v", err)
	}
	return inv
}

// ============================================================================
// ListOpenItems / SummarizeOpenItems
// ============================================================================

func TestListOpenItems_FiltersPaginationAndDunning(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	otherTenant := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	otherCtx := pgtc.TenantCtx(context.Background(), otherTenant)
	seedTenant(t, appPool, ctx, tenantID)
	seedTenant(t, appPool, otherCtx, otherTenant)

	repo := NewPostgresRepository(appPool)
	asOf := time.Now().UTC()

	current := overdueInvoice(t, ctx, repo, tenantID, -5) // not yet due
	d30 := overdueInvoice(t, ctx, repo, tenantID, 10)
	d60 := overdueInvoice(t, ctx, repo, tenantID, 45)
	d60plus := overdueInvoice(t, ctx, repo, tenantID, 90)

	// Draft invoices never appear in open items, however overdue their due date.
	draft := makeInvoice(tenantID, twoLines())
	draft.DueDate = time.Now().UTC().AddDate(0, 0, -30)
	if err := repo.Create(ctx, draft); err != nil {
		t.Fatalf("create draft: %v", err)
	}

	// Fully paid invoice: open_amount <= 0, must drop out of the list entirely.
	paidOff := overdueInvoice(t, ctx, repo, tenantID, 20)
	seedPayment(t, appPool, ctx, tenantID, paidOff.ID, paidOff.GrossTotal, asOf, "full-payoff")

	// Partially paid invoice: stays, but OpenAmount is the residual.
	partial := overdueInvoice(t, ctx, repo, tenantID, 15)
	partialPaid := decimal.NewFromFloat(100)
	seedPayment(t, appPool, ctx, tenantID, partial.ID, partialPaid, asOf, "partial")

	// Two dunning records on the same invoice — ORDER BY level DESC must surface
	// the higher one, not the most recently inserted.
	sentAt1 := asOf.Add(-72 * time.Hour)
	sentAt2 := asOf.Add(-24 * time.Hour)
	seedDunning(t, appPool, ctx, tenantID, d60.ID, 1, models.DunningStatusSent, &sentAt1)
	seedDunning(t, appPool, ctx, tenantID, d60.ID, 2, models.DunningStatusSent, &sentAt2)

	// Cross-tenant noise: another tenant's overdue invoice must never surface.
	overdueInvoice(t, otherCtx, repo, otherTenant, 50)

	t.Run("no_filter_excludes_draft_and_fully_paid", func(t *testing.T) {
		items, total, err := repo.ListOpenItems(ctx, tenantID, models.OpenItemFilter{AsOf: asOf, Limit: 50})
		if err != nil {
			t.Fatalf("ListOpenItems: %v", err)
		}
		if total != 5 {
			t.Fatalf("total: got %d, want 5 (current, d30, d60, d60plus, partial)", total)
		}
		ids := make(map[uuid.UUID]*models.OpenItem, len(items))
		for _, it := range items {
			ids[it.InvoiceID] = it
			if it.InvoiceID == draft.ID {
				t.Error("draft invoice leaked into open items")
			}
			if it.InvoiceID == paidOff.ID {
				t.Error("fully paid invoice leaked into open items")
			}
		}
		p, ok := ids[partial.ID]
		if !ok {
			t.Fatal("partially paid invoice missing from open items")
		}
		wantOpen := partial.GrossTotal.Sub(partialPaid)
		if !p.OpenAmount.Equal(wantOpen) {
			t.Errorf("partial OpenAmount: got %s, want %s", p.OpenAmount, wantOpen)
		}
		if !p.PaidAmount.Equal(partialPaid) {
			t.Errorf("partial PaidAmount: got %s, want %s", p.PaidAmount, partialPaid)
		}
	})

	t.Run("overdue_only_drops_not_yet_due", func(t *testing.T) {
		items, total, err := repo.ListOpenItems(ctx, tenantID, models.OpenItemFilter{AsOf: asOf, OverdueOnly: true, Limit: 50})
		if err != nil {
			t.Fatalf("ListOpenItems: %v", err)
		}
		if total != 4 {
			t.Fatalf("total: got %d, want 4", total)
		}
		for _, it := range items {
			if it.InvoiceID == current.ID {
				t.Error("not-yet-due invoice leaked into overdue-only filter")
			}
		}
	})

	t.Run("bucket_filter_isolates_d60", func(t *testing.T) {
		items, total, err := repo.ListOpenItems(ctx, tenantID, models.OpenItemFilter{
			AsOf: asOf, Bucket: models.AgingBucketD60, Limit: 50,
		})
		if err != nil {
			t.Fatalf("ListOpenItems: %v", err)
		}
		if total != 1 || len(items) != 1 {
			t.Fatalf("d60 bucket: got %d results, want exactly 1 (%s)", total, d60.ID)
		}
		if items[0].InvoiceID != d60.ID {
			t.Errorf("d60 bucket returned wrong invoice: got %s, want %s", items[0].InvoiceID, d60.ID)
		}
		if items[0].DunningLevel != 2 {
			t.Errorf("dunning level: got %d, want 2 (highest level, not most recent insert)", items[0].DunningLevel)
		}
	})

	t.Run("unknown_bucket_key_is_an_error", func(t *testing.T) {
		_, _, err := repo.ListOpenItems(ctx, tenantID, models.OpenItemFilter{AsOf: asOf, Bucket: "not-a-bucket"})
		if err == nil {
			t.Fatal("expected error for unknown bucket key, got nil")
		}
	})

	t.Run("pagination_limits_page_but_not_total", func(t *testing.T) {
		items, total, err := repo.ListOpenItems(ctx, tenantID, models.OpenItemFilter{AsOf: asOf, Limit: 2})
		if err != nil {
			t.Fatalf("ListOpenItems: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("page size: got %d, want 2", len(items))
		}
		if total != 5 {
			t.Fatalf("total must reflect the full filtered set, not the page: got %d, want 5", total)
		}
	})

	t.Run("cross_tenant_isolation", func(t *testing.T) {
		items, _, err := repo.ListOpenItems(ctx, tenantID, models.OpenItemFilter{AsOf: asOf, Limit: 50})
		if err != nil {
			t.Fatalf("ListOpenItems: %v", err)
		}
		for _, it := range items {
			// None of tenant A's items may belong to otherTenant's invoices.
			if it.InvoiceID == uuid.Nil {
				t.Error("unexpected nil invoice id")
			}
		}
		otherItems, otherTotal, err := repo.ListOpenItems(otherCtx, otherTenant, models.OpenItemFilter{AsOf: asOf, Limit: 50})
		if err != nil {
			t.Fatalf("ListOpenItems (other tenant): %v", err)
		}
		if otherTotal != 1 {
			t.Fatalf("other tenant total: got %d, want 1 (its own overdue invoice only)", otherTotal)
		}
		for _, it := range otherItems {
			if it.InvoiceID == d30.ID || it.InvoiceID == d60.ID || it.InvoiceID == d60plus.ID {
				t.Error("cross-tenant leak: tenant A's invoice visible to otherTenant")
			}
		}
	})

	t.Run("canceled_context_returns_error", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()
		_, _, err := repo.ListOpenItems(canceledCtx, tenantID, models.OpenItemFilter{AsOf: asOf, Limit: 50})
		if err == nil {
			t.Fatal("ListOpenItems with a canceled context: got nil error, want one")
		}
	})
}

func TestSummarizeOpenItems_AggregatesByCurrencyAndBucket(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	otherTenant := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	otherCtx := pgtc.TenantCtx(context.Background(), otherTenant)
	seedTenant(t, appPool, ctx, tenantID)
	seedTenant(t, appPool, otherCtx, otherTenant)

	repo := NewPostgresRepository(appPool)
	asOf := time.Now().UTC()

	invA := overdueInvoice(t, ctx, repo, tenantID, 10) // bucket d30
	invB := overdueInvoice(t, ctx, repo, tenantID, 10) // same bucket, doubles the count

	// Cross-tenant noise in the same bucket — must not bleed into tenant A's sum.
	overdueInvoice(t, otherCtx, repo, otherTenant, 10)

	totals, err := repo.SummarizeOpenItems(ctx, tenantID, asOf)
	if err != nil {
		t.Fatalf("SummarizeOpenItems: %v", err)
	}

	var bucket *models.OpenItemBucketTotal
	for _, b := range totals {
		if b.Currency == invA.Currency && b.BucketIndex == models.AgingBucketIndexOf(models.AgingBucketD30) {
			bucket = b
		}
	}
	if bucket == nil {
		t.Fatal("d30 bucket missing from summary")
	}
	if bucket.Count != 2 {
		t.Fatalf("bucket count: got %d, want 2 (tenant A only)", bucket.Count)
	}
	wantAmount := invA.GrossTotal.Add(invB.GrossTotal)
	if !bucket.Amount.Equal(wantAmount) {
		t.Errorf("bucket amount: got %s, want %s", bucket.Amount, wantAmount)
	}
	if bucket.DaysOverdueSum != 20 {
		t.Errorf("days overdue sum: got %d, want 20 (2 x 10 days)", bucket.DaysOverdueSum)
	}

	t.Run("empty_tenant_returns_empty_not_nil", func(t *testing.T) {
		emptyTenant := uuid.New()
		emptyCtx := pgtc.TenantCtx(context.Background(), emptyTenant)
		seedTenant(t, appPool, emptyCtx, emptyTenant)
		emptyTotals, err := repo.SummarizeOpenItems(emptyCtx, emptyTenant, asOf)
		if err != nil {
			t.Fatalf("SummarizeOpenItems: %v", err)
		}
		if emptyTotals == nil {
			t.Fatal("SummarizeOpenItems returned nil, want an empty slice")
		}
		if len(emptyTotals) != 0 {
			t.Fatalf("empty tenant totals: got %d entries, want 0", len(emptyTotals))
		}
	})

	t.Run("canceled_context_returns_error", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()
		_, err := repo.SummarizeOpenItems(canceledCtx, tenantID, asOf)
		if err == nil {
			t.Fatal("SummarizeOpenItems with a canceled context: got nil error, want one")
		}
	})
}

// ============================================================================
// ListDocumentChains
// ============================================================================

func TestListDocumentChains_FullLifecycleAndBranches(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	otherTenant := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	otherCtx := pgtc.TenantCtx(context.Background(), otherTenant)
	seedTenant(t, appPool, ctx, tenantID)
	seedTenant(t, appPool, otherCtx, otherTenant)

	repo := NewPostgresRepository(appPool)

	// Chain 1: quote -> invoice -> partial payment -> dunning -> still-pending remainder.
	quoteID := seedQuote(t, appPool, ctx, tenantID, models.QuoteStatusAccepted)
	invPartial := makeInvoice(tenantID, twoLines())
	invPartial.Status = models.InvoiceStatusSent
	invPartial.SourceQuoteID = &quoteID
	if err := repo.Create(ctx, invPartial); err != nil {
		t.Fatalf("create invPartial: %v", err)
	}
	partialAmount := invPartial.GrossTotal.Sub(decimal.NewFromFloat(50))
	seedPayment(t, appPool, ctx, tenantID, invPartial.ID, partialAmount, time.Now().UTC(), "part-1")
	seedDunning(t, appPool, ctx, tenantID, invPartial.ID, 1, models.DunningStatusSent, nil)

	// Chain 2: invoice fully paid -> complete, no pending node.
	invPaid := makeInvoice(tenantID, twoLines())
	invPaid.Status = models.InvoiceStatusPaid
	if err := repo.Create(ctx, invPaid); err != nil {
		t.Fatalf("create invPaid: %v", err)
	}
	seedPayment(t, appPool, ctx, tenantID, invPaid.ID, invPaid.GrossTotal, time.Now().UTC(), "full")

	// Chain 3: cancelled invoice, unpaid -> complete regardless of remaining amount.
	invCancelled := makeInvoice(tenantID, twoLines())
	invCancelled.Status = models.InvoiceStatusCancelled
	if err := repo.Create(ctx, invCancelled); err != nil {
		t.Fatalf("create invCancelled: %v", err)
	}

	// Chain 4: invoice with a pending (not-yet-sent) credit note — must NOT reduce
	// the remaining amount or the chain's completeness.
	invCreditPending := makeInvoice(tenantID, twoLines())
	invCreditPending.Status = models.InvoiceStatusSent
	if err := repo.Create(ctx, invCreditPending); err != nil {
		t.Fatalf("create invCreditPending: %v", err)
	}
	seedCreditNote(t, appPool, ctx, tenantID, invCreditPending.ID, invCreditPending.GrossTotal, models.CreditNoteStatusDraft)

	// Chain 5: invoice with a sent credit note covering the full amount -> complete.
	invCreditSent := makeInvoice(tenantID, twoLines())
	invCreditSent.Status = models.InvoiceStatusSent
	if err := repo.Create(ctx, invCreditSent); err != nil {
		t.Fatalf("create invCreditSent: %v", err)
	}
	seedCreditNote(t, appPool, ctx, tenantID, invCreditSent.ID, invCreditSent.GrossTotal, models.CreditNoteStatusSent)

	// Chain 6: standalone rejected quote, never converted to an invoice.
	standaloneQuoteID := seedQuote(t, appPool, ctx, tenantID, models.QuoteStatusRejected)

	// Cross-tenant noise.
	otherQuoteID := seedQuote(t, appPool, otherCtx, otherTenant, models.QuoteStatusAccepted)
	_ = otherQuoteID

	chains, err := repo.ListDocumentChains(ctx, tenantID)
	if err != nil {
		t.Fatalf("ListDocumentChains: %v", err)
	}

	byID := make(map[uuid.UUID]*models.DocumentChain, len(chains))
	for _, c := range chains {
		byID[c.ID] = c
	}

	t.Run("partial_chain_has_quote_invoice_payment_dunning_and_pending_node", func(t *testing.T) {
		c, ok := byID[invPartial.ID]
		if !ok {
			t.Fatal("partial chain missing")
		}
		if c.IsComplete {
			t.Error("partially paid, non-cancelled chain must not be complete")
		}
		var sawQuote, sawInvoice, sawPayment, sawDunning, sawPending bool
		for _, n := range c.Nodes {
			switch n.Type {
			case models.ChainNodeQuote:
				sawQuote = true
				if n.Status != models.ChainNodeCompleted {
					t.Errorf("quote node status: got %q, want completed (it produced an invoice)", n.Status)
				}
			case models.ChainNodeInvoice:
				sawInvoice = true
			case models.ChainNodePayment:
				if n.Status == models.ChainNodePending {
					sawPending = true
					wantRemaining := invPartial.GrossTotal.Sub(partialAmount)
					if !n.Amount.Equal(wantRemaining) {
						t.Errorf("pending remainder amount: got %s, want %s", n.Amount, wantRemaining)
					}
				} else {
					sawPayment = true
				}
			case models.ChainNodeDunning:
				sawDunning = true
			}
		}
		if !sawQuote || !sawInvoice || !sawPayment || !sawDunning || !sawPending {
			t.Errorf("chain missing expected node types: quote=%v invoice=%v payment=%v dunning=%v pending=%v",
				sawQuote, sawInvoice, sawPayment, sawDunning, sawPending)
		}
		// Nodes must be date-ordered.
		for i := 1; i < len(c.Nodes); i++ {
			if c.Nodes[i].Date == nil || c.Nodes[i-1].Date == nil {
				continue
			}
			if c.Nodes[i].Date.Before(*c.Nodes[i-1].Date) {
				t.Errorf("nodes not sorted by date at index %d", i)
			}
		}
	})

	t.Run("fully_paid_chain_is_complete_without_pending_node", func(t *testing.T) {
		c, ok := byID[invPaid.ID]
		if !ok {
			t.Fatal("paid chain missing")
		}
		if !c.IsComplete {
			t.Error("fully paid chain must be complete")
		}
		for _, n := range c.Nodes {
			if n.Type == models.ChainNodePayment && n.Status == models.ChainNodePending {
				t.Error("fully paid chain must not carry a pending remainder node")
			}
		}
	})

	t.Run("cancelled_chain_is_complete_regardless_of_remaining", func(t *testing.T) {
		c, ok := byID[invCancelled.ID]
		if !ok {
			t.Fatal("cancelled chain missing")
		}
		if !c.IsComplete {
			t.Error("cancelled invoice chain must be complete even though nothing was paid")
		}
	})

	t.Run("pending_credit_note_does_not_reduce_remaining", func(t *testing.T) {
		c, ok := byID[invCreditPending.ID]
		if !ok {
			t.Fatal("pending-credit chain missing")
		}
		if c.IsComplete {
			t.Error("a draft credit note must not count toward completeness")
		}
		for _, n := range c.Nodes {
			if n.Type == models.ChainNodeCreditNote && n.Status != models.ChainNodePending {
				t.Errorf("draft credit note node status: got %q, want pending", n.Status)
			}
		}
	})

	t.Run("sent_credit_note_covering_full_amount_completes_chain", func(t *testing.T) {
		c, ok := byID[invCreditSent.ID]
		if !ok {
			t.Fatal("sent-credit chain missing")
		}
		if !c.IsComplete {
			t.Error("a sent credit note covering the full invoice amount must complete the chain")
		}
		for _, n := range c.Nodes {
			if n.Type == models.ChainNodeCreditNote && n.Status != models.ChainNodeCompleted {
				t.Errorf("sent credit note node status: got %q, want completed", n.Status)
			}
		}
	})

	t.Run("standalone_quote_is_its_own_one_node_chain", func(t *testing.T) {
		c, ok := byID[standaloneQuoteID]
		if !ok {
			t.Fatal("standalone quote chain missing")
		}
		if len(c.Nodes) != 1 {
			t.Fatalf("standalone quote node count: got %d, want 1", len(c.Nodes))
		}
		if c.Nodes[0].Type != models.ChainNodeQuote {
			t.Errorf("standalone quote node type: got %q, want quote", c.Nodes[0].Type)
		}
		if !c.IsComplete {
			t.Error("a rejected standalone quote must be marked complete (nothing more will happen to it)")
		}
	})

	t.Run("cross_tenant_isolation", func(t *testing.T) {
		if _, leaked := byID[otherQuoteID]; leaked {
			t.Error("cross-tenant quote leaked into tenant A's document chains")
		}
		otherChains, err := repo.ListDocumentChains(otherCtx, otherTenant)
		if err != nil {
			t.Fatalf("ListDocumentChains (other tenant): %v", err)
		}
		for _, c := range otherChains {
			if c.ID == invPartial.ID || c.ID == invPaid.ID {
				t.Error("tenant A's document leaked into otherTenant's chains")
			}
		}
	})

	t.Run("canceled_context_returns_error", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()
		_, err := repo.ListDocumentChains(canceledCtx, tenantID)
		if err == nil {
			t.Fatal("ListDocumentChains with a canceled context: got nil error, want one")
		}
	})
}
