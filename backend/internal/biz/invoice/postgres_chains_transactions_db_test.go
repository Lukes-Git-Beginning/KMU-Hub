package invoice

// postgres_chains_transactions_db_test.go exercises the read-time assemblies
// (ListDocumentChains, ListTransactions, SummarizeOpenItems) and the write
// paths Update, UpsertImported and LockForCreditInTx against real Postgres as
// the app role. The tagged `integration` suite covers some of this through a
// Testcontainers database, but it is neither a PR gate nor part of the package
// coverage number; these tests are untagged and run wherever DATABASE_URL is set.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// chainTestDay is a UTC midnight n days before today, so date columns round-trip.
func chainTestDay(daysAgo int) time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -daysAgo)
}

// seedChainTenant creates a tenant and returns its id, repo and a tenant context.
func seedChainTenant(t *testing.T, pool *pgxpool.Pool, name string) (uuid.UUID, context.Context) {
	t.Helper()
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, name)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	return tenantID, testutil.WithTenantCtx(context.Background(), tenantID)
}

func seedChainQuote(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, status string, gross int64, createdAt time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	testutil.SeedRow(t, pool, "finance_quotes", map[string]any{
		"id": id, "tenant_id": tenantID,
		"quote_number": "AN-CH-" + id.String()[:8], "status": status,
		"customer_name": "Kette Kunde", "customer_address": "Musterstraße 1",
		"customer_email": "kette@example.com", "customer_ust_id_nr": "",
		"tax_mode": models.TaxModeStandard,
		"subtotal": decimal.NewFromInt(gross), "total_tax": decimal.Zero, "gross_total": decimal.NewFromInt(gross),
		"created_by": uuid.New(), "created_at": createdAt, "updated_at": createdAt, "currency": "EUR",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_quotes", id) })
	return id
}

func seedChainPayment(t *testing.T, pool *pgxpool.Pool, tenantID, invoiceID uuid.UUID, amount int64, date time.Time, reference string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "finance_payments", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "invoice_id": invoiceID,
		"amount": decimal.NewFromInt(amount), "payment_date": date,
		"method": "bank_transfer", "reference": reference, "notes": "",
		"created_by": uuid.New(), "created_at": date, "currency": "EUR",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_payments", id) })
	return id
}

func seedChainDunning(t *testing.T, pool *pgxpool.Pool, tenantID, invoiceID uuid.UUID, level int, status string, sentAt *time.Time, createdAt time.Time) {
	t.Helper()
	id := testutil.SeedRow(t, pool, "finance_dunning_records", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "invoice_id": invoiceID,
		"level": level, "status": status,
		"fee": decimal.NewFromInt(5), "interest": decimal.Zero,
		"sent_at": sentAt, "created_by": uuid.New(), "created_at": createdAt,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_dunning_records", id) })
}

func seedChainCreditNote(t *testing.T, pool *pgxpool.Pool, tenantID, invoiceID uuid.UUID, number, status string, amount int64, createdAt time.Time) {
	t.Helper()
	id := testutil.SeedRow(t, pool, "finance_credit_notes", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "credit_note_number": number, "status": status,
		"original_invoice_id": invoiceID,
		"customer_name":       "Kette Kunde", "customer_address": "Musterstraße 1",
		"customer_email": "kette@example.com", "customer_ust_id_nr": "",
		"tax_mode": models.TaxModeStandard,
		"subtotal": decimal.NewFromInt(amount), "total_tax": decimal.Zero, "gross_total": decimal.NewFromInt(amount),
		"reason": "Testgutschrift", "created_by": uuid.New(),
		"created_at": createdAt, "updated_at": createdAt, "currency": "EUR",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_credit_notes", id) })
}

func seedChainExpense(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, description, status, category string, amount int64, date time.Time) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "finance_expenses", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "description": description,
		"amount": decimal.NewFromInt(amount), "expense_date": date,
		"category": category, "status": status,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_expenses", id) })
	return id
}

// chainTestInvoice creates an invoice with the given status, gross and invoice date.
func chainTestInvoice(t *testing.T, repo *PostgresRepository, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID, status string, gross int64, invoiceDate time.Time) *models.Invoice {
	t.Helper()
	inv := newListTestInvoice(tenantID, invoiceDate, invoiceDate.AddDate(0, 0, 14))
	inv.Status = status
	inv.CustomerName = "Kette Kunde"
	inv.GrossTotal = decimal.NewFromInt(gross)
	createListTestInvoice(t, repo, pool, ctx, inv)
	return inv
}

func chainByID(chains []*models.DocumentChain, id uuid.UUID) *models.DocumentChain {
	for _, c := range chains {
		if c.ID == id {
			return c
		}
	}
	return nil
}

func nodeTypes(c *models.DocumentChain) []string {
	out := make([]string, 0, len(c.Nodes))
	for _, n := range c.Nodes {
		out = append(out, n.Type)
	}
	return out
}

func TestPostgresRepository_ListDocumentChains_AssemblesTheLifecycle(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, ctx := seedChainTenant(t, pool, "Chains Tenant")
	otherTenant, otherCtx := seedChainTenant(t, pool, "Chains Foreign Tenant")
	repo := NewPostgresRepository(pool)

	// Chain 1: quote -> invoice with a part payment, a dunning, a sent and a draft credit note.
	quote := seedChainQuote(t, pool, tenantID, models.QuoteStatusAccepted, 119, chainTestDay(30))
	open := newListTestInvoice(tenantID, chainTestDay(20), chainTestDay(6))
	open.Status = models.InvoiceStatusSent
	open.CustomerName = "Kette Kunde"
	open.GrossTotal = decimal.NewFromInt(119)
	open.SourceQuoteID = &quote
	createListTestInvoice(t, repo, pool, ctx, open)
	seedChainPayment(t, pool, tenantID, open.ID, 50, chainTestDay(10), "")
	sentAt := chainTestDay(8)
	seedChainDunning(t, pool, tenantID, open.ID, 1, models.DunningStatusSent, &sentAt, chainTestDay(9))
	seedChainCreditNote(t, pool, tenantID, open.ID, "GS-CH-SENT-"+uuid.NewString()[:6], models.CreditNoteStatusSent, 19, chainTestDay(5))
	seedChainCreditNote(t, pool, tenantID, open.ID, "GS-CH-DRAFT-"+uuid.NewString()[:6], models.CreditNoteStatusDraft, 10, chainTestDay(4))

	// Chain 2: cancelled invoice, nothing collected.
	cancelled := chainTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusCancelled, 200, chainTestDay(19))

	// Chain 3: fully paid, the payment names its own reference.
	paid := chainTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusPaid, 80, chainTestDay(18))
	seedChainPayment(t, pool, tenantID, paid.ID, 80, chainTestDay(17), "BANK-REF-1")

	// Chain 4+5: quotes that never became an invoice.
	rejected := seedChainQuote(t, pool, tenantID, models.QuoteStatusRejected, 500, chainTestDay(25))
	draftQuote := seedChainQuote(t, pool, tenantID, models.QuoteStatusDraft, 600, chainTestDay(24))

	// Foreign tenant noise: an invoice and a quote that must never surface.
	foreign := chainTestInvoice(t, repo, pool, otherCtx, otherTenant, models.InvoiceStatusSent, 999, chainTestDay(15))
	foreignQuote := seedChainQuote(t, pool, otherTenant, models.QuoteStatusDraft, 999, chainTestDay(14))

	chains, err := repo.ListDocumentChains(ctx, tenantID)
	require.NoError(t, err)
	require.Len(t, chains, 5, "three invoice chains plus two standalone quotes, nothing from the foreign tenant")
	assert.Nil(t, chainByID(chains, foreign.ID))
	assert.Nil(t, chainByID(chains, foreignQuote))

	// Invoices first by invoice date, then standalone quotes by creation date.
	assert.Equal(t, []uuid.UUID{open.ID, cancelled.ID, paid.ID, rejected, draftQuote},
		[]uuid.UUID{chains[0].ID, chains[1].ID, chains[2].ID, chains[3].ID, chains[4].ID})

	t.Run("open_chain_in_date_order_with_pending_remainder", func(t *testing.T) {
		c := chainByID(chains, open.ID)
		require.NotNil(t, c)
		assert.Equal(t, "Kette Kunde", c.Customer)
		assert.Equal(t, "EUR", c.Currency)
		assert.True(t, c.TotalValue.Equal(decimal.NewFromInt(119)))
		assert.False(t, c.IsComplete)
		assert.Equal(t, []string{
			models.ChainNodeQuote, models.ChainNodeInvoice, models.ChainNodePayment,
			models.ChainNodeDunning, models.ChainNodeCreditNote, models.ChainNodeCreditNote,
			models.ChainNodePayment,
		}, nodeTypes(c))

		assert.Equal(t, models.ChainNodeCompleted, c.Nodes[0].Status, "a converted quote is done whatever its own status says")
		assert.Equal(t, models.ChainNodeActive, c.Nodes[1].Status, "sent invoice")
		assert.Equal(t, open.InvoiceNumber, c.Nodes[2].Number, "payment without reference falls back to the invoice number")
		assert.True(t, c.Nodes[2].Amount.Equal(decimal.NewFromInt(50)))
		assert.Equal(t, open.InvoiceNumber+"/M1", c.Nodes[3].Number)
		assert.Equal(t, models.ChainNodeActive, c.Nodes[3].Status, "sent dunning")
		assert.True(t, c.Nodes[3].Amount.Equal(decimal.NewFromInt(119)), "dunning shows the invoice amount, not the fee")
		assert.Equal(t, models.ChainNodeCompleted, c.Nodes[4].Status, "sent credit note")
		assert.Equal(t, models.ChainNodePending, c.Nodes[5].Status, "draft credit note")

		rest := c.Nodes[6]
		assert.Equal(t, models.ChainNodePending, rest.Status)
		assert.Nil(t, rest.Date)
		assert.True(t, rest.Amount.Equal(decimal.NewFromInt(50)), "119 - 50 paid - 19 sent credit; the draft credit does not count, got %s", rest.Amount)
	})

	t.Run("dunning_dated_by_sent_at_else_created_at", func(t *testing.T) {
		c := chainByID(chains, open.ID)
		require.NotNil(t, c.Nodes[3].Date)
		assert.True(t, c.Nodes[3].Date.Equal(sentAt), "sent_at wins over created_at: got %s want %s", c.Nodes[3].Date, sentAt)
	})

	t.Run("cancelled_invoice_is_complete_without_remainder", func(t *testing.T) {
		c := chainByID(chains, cancelled.ID)
		require.NotNil(t, c)
		assert.True(t, c.IsComplete)
		assert.Equal(t, []string{models.ChainNodeInvoice}, nodeTypes(c))
		assert.Equal(t, models.ChainNodeCancelled, c.Nodes[0].Status)
	})

	t.Run("paid_invoice_is_complete_and_keeps_payment_reference", func(t *testing.T) {
		c := chainByID(chains, paid.ID)
		require.NotNil(t, c)
		assert.True(t, c.IsComplete)
		assert.Equal(t, []string{models.ChainNodeInvoice, models.ChainNodePayment}, nodeTypes(c))
		assert.Equal(t, models.ChainNodeCompleted, c.Nodes[0].Status)
		assert.Equal(t, "BANK-REF-1", c.Nodes[1].Number)
	})

	t.Run("standalone_quotes_are_one_node_chains", func(t *testing.T) {
		rej := chainByID(chains, rejected)
		require.NotNil(t, rej)
		assert.True(t, rej.IsComplete, "a rejected quote will never be followed by anything")
		require.Len(t, rej.Nodes, 1)
		assert.Equal(t, models.ChainNodeCancelled, rej.Nodes[0].Status)
		assert.True(t, rej.TotalValue.Equal(decimal.NewFromInt(500)))

		dq := chainByID(chains, draftQuote)
		require.NotNil(t, dq)
		assert.False(t, dq.IsComplete)
		assert.Equal(t, models.ChainNodeActive, dq.Nodes[0].Status)
	})

	t.Run("tenant_without_documents_gets_an_empty_slice", func(t *testing.T) {
		emptyTenant, emptyCtx := seedChainTenant(t, pool, "Chains Empty Tenant")
		got, err := repo.ListDocumentChains(emptyCtx, emptyTenant)
		require.NoError(t, err)
		assert.NotNil(t, got, "the wire shape needs [] not null")
		assert.Empty(t, got)
	})
}

func TestPostgresRepository_ListDocumentChains_OverpaymentLeavesNoRemainder(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, ctx := seedChainTenant(t, pool, "Chains Overpaid Tenant")
	repo := NewPostgresRepository(pool)

	inv := chainTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, 100, chainTestDay(10))
	seedChainPayment(t, pool, tenantID, inv.ID, 60, chainTestDay(5), "")
	seedChainPayment(t, pool, tenantID, inv.ID, 60, chainTestDay(3), "")
	seedChainDunning(t, pool, tenantID, inv.ID, 2, models.DunningStatusDraft, nil, chainTestDay(4))

	chains, err := repo.ListDocumentChains(ctx, tenantID)
	require.NoError(t, err)
	c := chainByID(chains, inv.ID)
	require.NotNil(t, c)
	assert.True(t, c.IsComplete, "paid beyond the gross total is complete")
	assert.Equal(t, []string{
		models.ChainNodeInvoice, models.ChainNodePayment, models.ChainNodeDunning, models.ChainNodePayment,
	}, nodeTypes(c), "no pending remainder node once the invoice is covered")
	assert.Equal(t, models.ChainNodePending, c.Nodes[2].Status, "draft dunning")
	assert.Equal(t, inv.InvoiceNumber+"/M2", c.Nodes[2].Number)
	require.NotNil(t, c.Nodes[2].Date)
	assert.True(t, c.Nodes[2].Date.Equal(chainTestDay(4)), "an unsent dunning is dated by created_at")
}

func TestPostgresRepository_ListTransactions_MergesPaymentsAndApprovedExpenses(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, ctx := seedChainTenant(t, pool, "Transactions Tenant")
	otherTenant, otherCtx := seedChainTenant(t, pool, "Transactions Foreign Tenant")
	repo := NewPostgresRepository(pool)

	inv := chainTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, 500, chainTestDay(30))
	payOld := seedChainPayment(t, pool, tenantID, inv.ID, 100, chainTestDay(10), "")
	payNew := seedChainPayment(t, pool, tenantID, inv.ID, 20, chainTestDay(2), "")
	expApproved := seedChainExpense(t, pool, tenantID, "Büromaterial", "approved", "Bürobedarf", 30, chainTestDay(6))
	seedChainExpense(t, pool, tenantID, "Noch offen", "pending", "Reisekosten", 70, chainTestDay(5))
	seedChainExpense(t, pool, tenantID, "Abgelehnt", "rejected", "Reisekosten", 80, chainTestDay(4))

	foreignInv := chainTestInvoice(t, repo, pool, otherCtx, otherTenant, models.InvoiceStatusSent, 900, chainTestDay(20))
	seedChainPayment(t, pool, otherTenant, foreignInv.ID, 900, chainTestDay(1), "")
	seedChainExpense(t, pool, otherTenant, "Fremd", "approved", "Sonstiges", 5, chainTestDay(1))

	got, err := repo.ListTransactions(ctx, tenantID)
	require.NoError(t, err)
	require.Len(t, got, 3, "two payments and the approved expense; pending, rejected and foreign rows stay out")

	assert.Equal(t, "pay-"+payNew.String(), got[0].ID, "newest first")
	assert.Equal(t, "exp-"+expApproved.String(), got[1].ID)
	assert.Equal(t, "pay-"+payOld.String(), got[2].ID)

	income := got[0]
	assert.Equal(t, "income", income.Type)
	assert.Equal(t, "Rechnung "+inv.InvoiceNumber+" Kette Kunde", income.Description)
	assert.True(t, income.Amount.Equal(decimal.NewFromInt(20)))
	assert.Equal(t, financeTransactionIncomeCategory, income.Category)
	assert.Equal(t, "completed", income.Status)
	assert.Equal(t, inv.InvoiceNumber, income.Reference)
	require.NotNil(t, income.InvoiceID)
	assert.Equal(t, inv.ID, *income.InvoiceID)

	expense := got[1]
	assert.Equal(t, "expense", expense.Type)
	assert.Equal(t, "Büromaterial", expense.Description)
	assert.Equal(t, "Bürobedarf", expense.Category)
	assert.True(t, expense.Amount.Equal(decimal.NewFromInt(30)))
	assert.Nil(t, expense.InvoiceID)
	assert.True(t, expense.Date.Equal(chainTestDay(6)))
}

func TestPostgresRepository_ListTransactions_EmptyLedgerIsNotNil(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, ctx := seedChainTenant(t, pool, "Transactions Empty Tenant")
	got, err := NewPostgresRepository(pool).ListTransactions(ctx, tenantID)
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestPostgresRepository_SummarizeOpenItems_BucketsByCurrencyAndAge(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, ctx := seedChainTenant(t, pool, "Summary Tenant")
	otherTenant, otherCtx := seedChainTenant(t, pool, "Summary Foreign Tenant")
	repo := NewPostgresRepository(pool)
	today := chainTestDay(0)

	// due date = today - daysOverdue; gross 119 each.
	mk := func(c context.Context, tenant uuid.UUID, status string, daysOverdue int, currency string) *models.Invoice {
		inv := newListTestInvoice(tenant, today.AddDate(0, 0, -daysOverdue-14), today.AddDate(0, 0, -daysOverdue))
		inv.Status = status
		inv.Currency = currency
		createListTestInvoice(t, repo, pool, c, inv)
		return inv
	}
	mk(ctx, tenantID, models.InvoiceStatusSent, -5, "EUR")      // current, 0 days counted
	mk(ctx, tenantID, models.InvoiceStatusSent, 10, "EUR")      // d30
	mk(ctx, tenantID, models.InvoiceStatusOverdue, 25, "EUR")   // d30
	mk(ctx, tenantID, models.InvoiceStatusSent, 45, "EUR")      // d60
	mk(ctx, tenantID, models.InvoiceStatusOverdue, 90, "EUR")   // d60plus
	mk(ctx, tenantID, models.InvoiceStatusSent, 10, "CHF")      // d30, other currency
	mk(ctx, tenantID, models.InvoiceStatusDraft, 50, "EUR")     // drafts are no receivable
	mk(ctx, tenantID, models.InvoiceStatusPaid, 50, "EUR")      // neither are paid ones
	mk(ctx, tenantID, models.InvoiceStatusCancelled, 50, "EUR") // nor cancelled ones
	mk(otherCtx, otherTenant, models.InvoiceStatusSent, 10, "EUR")

	// A part payment lowers the open amount of the 45-day invoice by 19.
	var d60 *models.Invoice
	{
		invs, _, err := repo.List(ctx, tenantID, ListFilter{Limit: 50})
		require.NoError(t, err)
		for _, inv := range invs {
			if inv.Status == models.InvoiceStatusSent && inv.Currency == "EUR" && inv.DueDate.Equal(today.AddDate(0, 0, -45)) {
				d60 = inv
			}
		}
	}
	require.NotNil(t, d60)
	seedChainPayment(t, pool, tenantID, d60.ID, 19, today, "")

	totals, err := repo.SummarizeOpenItems(ctx, tenantID, today)
	require.NoError(t, err)

	type key struct {
		currency string
		bucket   int
	}
	got := map[key]*models.OpenItemBucketTotal{}
	for _, b := range totals {
		got[key{b.Currency, b.BucketIndex}] = b
	}
	idx := models.AgingBucketIndexOf
	require.Len(t, got, 5, "EUR current/d30/d60/d60plus plus CHF d30; got %d buckets", len(got))

	check := func(currency, bucket string, count int, amount int64, days int) {
		t.Helper()
		b := got[key{currency, idx(bucket)}]
		require.NotNil(t, b, "%s %s missing", currency, bucket)
		assert.Equal(t, count, b.Count, "%s %s count", currency, bucket)
		assert.True(t, b.Amount.Equal(decimal.NewFromInt(amount)), "%s %s amount: got %s want %d", currency, bucket, b.Amount, amount)
		assert.Equal(t, days, b.DaysOverdueSum, "%s %s days", currency, bucket)
	}
	check("EUR", models.AgingBucketCurrent, 1, 119, 0)
	check("EUR", models.AgingBucketD30, 2, 238, 35)
	check("EUR", models.AgingBucketD60, 1, 100, 45)
	check("EUR", models.AgingBucketD60Plus, 1, 119, 90)
	check("CHF", models.AgingBucketD30, 1, 119, 10)

	// Ordered by currency, then bucket.
	for i := 1; i < len(totals); i++ {
		prev, cur := totals[i-1], totals[i]
		assert.True(t, prev.Currency < cur.Currency || (prev.Currency == cur.Currency && prev.BucketIndex < cur.BucketIndex),
			"summary must be sorted by currency then bucket")
	}

	t.Run("zero_as_of_defaults_to_now", func(t *testing.T) {
		now, err := repo.SummarizeOpenItems(ctx, tenantID, time.Time{})
		require.NoError(t, err)
		assert.Len(t, now, 5)
	})

	t.Run("tenant_without_receivables_gets_an_empty_slice", func(t *testing.T) {
		emptyTenant, emptyCtx := seedChainTenant(t, pool, "Summary Empty Tenant")
		empty, err := repo.SummarizeOpenItems(emptyCtx, emptyTenant, today)
		require.NoError(t, err)
		assert.NotNil(t, empty)
		assert.Empty(t, empty)
	})
}

func TestPostgresRepository_Update_ReplacesHeaderAndLines(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, ctx := seedChainTenant(t, pool, "Update Tenant")
	repo := NewPostgresRepository(pool)
	now := time.Now().UTC()

	inv := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	createListTestInvoice(t, repo, pool, ctx, inv)
	neighbour := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	createListTestInvoice(t, repo, pool, ctx, neighbour)

	lines := []models.LineItem{
		{ID: "a", Position: 1, Description: "Erste Position", Quantity: decimal.NewFromInt(2), UnitPrice: decimal.NewFromInt(50), TaxRate: decimal.NewFromInt(19), LineTotal: decimal.NewFromInt(100)},
		{ID: "b", Position: 2, Description: "Zweite Position", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(30), TaxRate: decimal.NewFromInt(7), LineTotal: decimal.NewFromInt(30)},
	}
	raw, err := json.Marshal(lines)
	require.NoError(t, err)

	updatedAt := now.Add(time.Hour).Truncate(time.Second)
	inv.CustomerName = "Neuer Kunde AG"
	inv.CustomerEmail = "neu@example.com"
	inv.Notes = "geändert"
	inv.Subtotal = decimal.NewFromInt(130)
	inv.TotalTax = decimal.NewFromFloat(21.1)
	inv.GrossTotal = decimal.NewFromFloat(151.1)
	inv.LineItems = raw
	inv.UpdatedAt = updatedAt
	require.NoError(t, repo.Update(ctx, inv))

	got, err := repo.GetByID(ctx, tenantID, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, "Neuer Kunde AG", got.CustomerName)
	assert.Equal(t, "neu@example.com", got.CustomerEmail)
	assert.Equal(t, "geändert", got.Notes)
	assert.True(t, got.GrossTotal.Equal(decimal.NewFromFloat(151.1)))
	assert.True(t, got.UpdatedAt.Equal(updatedAt))

	gotLines, err := ParseLineItems(got.LineItems)
	require.NoError(t, err)
	require.Len(t, gotLines, 2, "the single original line is replaced, not appended to")
	assert.Equal(t, "Erste Position", gotLines[0].Description)
	assert.Equal(t, "Zweite Position", gotLines[1].Description)
	assert.True(t, gotLines[1].TaxRate.Equal(decimal.NewFromInt(7)))

	other, err := repo.GetByID(ctx, tenantID, neighbour.ID)
	require.NoError(t, err)
	assert.Equal(t, "List Filter Test GmbH", other.CustomerName, "a neighbouring invoice is untouched")
	otherLines, err := ParseLineItems(other.LineItems)
	require.NoError(t, err)
	assert.Len(t, otherLines, 1)

	t.Run("unparsable_line_items_roll_the_whole_update_back", func(t *testing.T) {
		broken := *got
		broken.CustomerName = "Darf nicht ankommen"
		broken.LineItems = json.RawMessage(`{"not":"an array"}`)
		err := repo.Update(ctx, &broken)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshal line items")

		after, err := repo.GetByID(ctx, tenantID, inv.ID)
		require.NoError(t, err)
		assert.Equal(t, "Neuer Kunde AG", after.CustomerName, "header write rolled back with the failed line insert")
		afterLines, err := ParseLineItems(after.LineItems)
		require.NoError(t, err)
		assert.Len(t, afterLines, 2, "the delete of the old lines rolled back too")
	})
}

func TestPostgresRepository_UpsertImported_KeepsTheCanonicalRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, ctx := seedChainTenant(t, pool, "Upsert Tenant")
	repo := NewPostgresRepository(pool)
	now := time.Now().UTC().Truncate(time.Second)

	extID := "bexio-" + uuid.NewString()[:8]
	extNumber := "RE-EXT-1"
	build := func(number string, gross int64, lineText string) *models.Invoice {
		lines := []models.LineItem{{
			ID: "l1", Position: 1, Description: lineText,
			Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(gross), TaxRate: decimal.Zero, LineTotal: decimal.NewFromInt(gross),
		}}
		raw, _ := json.Marshal(lines)
		return &models.Invoice{
			ID: uuid.New(), TenantID: tenantID, InvoiceNumber: number, Status: models.InvoiceStatusSent,
			CustomerName: "Import Kunde", TaxMode: models.TaxModeStandard, LineItems: raw,
			Subtotal: decimal.NewFromInt(gross), TotalTax: decimal.Zero, GrossTotal: decimal.NewFromInt(gross),
			Currency: "EUR", InvoiceDate: now, DueDate: now.AddDate(0, 0, 14),
			CreatedBy: uuid.New(), CreatedAt: now, UpdatedAt: now,
			Source: models.InvoiceSourceBexio, ExternalID: &extID, ExternalNumber: &extNumber,
		}
	}

	first := build("EXT-0001", 100, "Import Zeile alt")
	firstID := first.ID
	require.NoError(t, repo.UpsertImported(ctx, first))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", firstID) })
	assert.Equal(t, firstID, first.ID)

	second := build("EXT-0002", 250, "Import Zeile neu")
	secondGeneratedID := second.ID
	extNumber = "RE-EXT-2"
	require.NoError(t, repo.UpsertImported(ctx, second))
	assert.Equal(t, firstID, second.ID, "the canonical row id replaces the caller's fresh id")
	assert.NotEqual(t, secondGeneratedID, second.ID)

	got, err := repo.GetByID(ctx, tenantID, firstID)
	require.NoError(t, err)
	assert.Equal(t, "EXT-0002", got.InvoiceNumber)
	assert.True(t, got.GrossTotal.Equal(decimal.NewFromInt(250)))
	assert.Equal(t, models.InvoiceSourceBexio, got.Source)
	require.NotNil(t, got.ExternalNumber)
	assert.Equal(t, "RE-EXT-2", *got.ExternalNumber)

	lines, err := ParseLineItems(got.LineItems)
	require.NoError(t, err)
	require.Len(t, lines, 1, "re-import replaces the lines instead of stacking them")
	assert.Equal(t, "Import Zeile neu", lines[0].Description)

	var rows int
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT count(*) FROM finance_invoices WHERE tenant_id = $1 AND source = $2 AND external_id = $3`,
		tenantID, models.InvoiceSourceBexio, extID).Scan(&rows))
	assert.Equal(t, 1, rows, "one external id is one invoice")

	t.Run("the_same_external_id_in_another_tenant_is_a_separate_invoice", func(t *testing.T) {
		otherTenant, otherCtx := seedChainTenant(t, pool, "Upsert Other Tenant")
		foreign := build("EXT-F", 10, "Fremd")
		foreign.TenantID = otherTenant
		require.NoError(t, repo.UpsertImported(otherCtx, foreign))
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", foreign.ID) })
		assert.NotEqual(t, firstID, foreign.ID)

		again, err := repo.GetByID(ctx, tenantID, firstID)
		require.NoError(t, err)
		assert.Equal(t, "EXT-0002", again.InvoiceNumber)
	})

	t.Run("a_row_the_database_rejects_surfaces_as_an_error", func(t *testing.T) {
		bad := build("EXT-BAD", 1, "x")
		otherExt := "bexio-" + uuid.NewString()[:8]
		bad.ExternalID = &otherExt
		bad.Currency = "TOOLONG"
		err := repo.UpsertImported(ctx, bad)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upsert imported invoice")
	})
}

func TestPostgresRepository_LockForCreditInTx_LocksTheRowAndReturnsStatusAndGross(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, ctx := seedChainTenant(t, pool, "Credit Lock Tenant")
	otherTenant, otherCtx := seedChainTenant(t, pool, "Credit Lock Foreign Tenant")
	repo := NewPostgresRepository(pool)

	inv := chainTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusPaid, 119, chainTestDay(3))
	foreign := chainTestInvoice(t, repo, pool, otherCtx, otherTenant, models.InvoiceStatusSent, 50, chainTestDay(3))

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx) //nolint:errcheck

	status, gross, err := repo.LockForCreditInTx(ctx, tx, tenantID, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusPaid, status)
	assert.True(t, gross.Equal(decimal.NewFromInt(119)))

	// A second transaction cannot take the same row lock while the first is open.
	rival, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer rival.Rollback(ctx) //nolint:errcheck
	_, err = rival.Exec(ctx, `SELECT 1 FROM finance_invoices WHERE tenant_id = $1 AND id = $2 FOR UPDATE NOWAIT`, tenantID, inv.ID)
	require.Error(t, err)
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "55P03", pgErr.Code, "lock_not_available: the credit-note race is serialised on this row")

	t.Run("unknown_invoice", func(t *testing.T) {
		tx2, err := pool.Begin(ctx)
		require.NoError(t, err)
		defer tx2.Rollback(ctx) //nolint:errcheck
		_, _, err = repo.LockForCreditInTx(ctx, tx2, tenantID, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lock invoice for credit note")
	})

	t.Run("foreign_tenant_invoice_cannot_be_locked", func(t *testing.T) {
		tx3, err := pool.Begin(ctx)
		require.NoError(t, err)
		defer tx3.Rollback(ctx) //nolint:errcheck
		_, _, err = repo.LockForCreditInTx(ctx, tx3, tenantID, foreign.ID)
		require.Error(t, err, "RLS and the tenant filter both hide the row")
	})
}

func TestParseTaxBreakdown(t *testing.T) {
	got, err := ParseTaxBreakdown(nil)
	require.NoError(t, err)
	assert.Nil(t, got, "no stored breakdown is nil, not an empty struct")

	tb := &models.TaxBreakdown{}
	raw, err := json.Marshal(tb)
	require.NoError(t, err)
	got, err = ParseTaxBreakdown(raw)
	require.NoError(t, err)
	require.NotNil(t, got)

	_, err = ParseTaxBreakdown(json.RawMessage(`[1,2`))
	require.Error(t, err)
}
