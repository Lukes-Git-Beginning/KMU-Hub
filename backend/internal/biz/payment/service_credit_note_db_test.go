package payment

// service_credit_note_db_test.go proves the paid transition counts sent credit
// notes like payments: invoice 1000, sent credit note 400, payment 600 is settled.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedCreditNoteOn(t *testing.T, pool *pgxpool.Pool, tenantID, invoiceID uuid.UUID, status, gross string) {
	t.Helper()
	now := time.Now()
	id := testutil.SeedRow(t, pool, "finance_credit_notes", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"credit_note_number": "GS-" + uuid.NewString()[:8], "status": status,
		"original_invoice_id": invoiceID,
		"customer_name":       "Gutschrift Kunde", "customer_address": "Musterstraße 1",
		"customer_email": "gs@example.com", "customer_ust_id_nr": "",
		"tax_mode": models.TaxModeStandard,
		"subtotal": gross, "total_tax": "0", "gross_total": gross,
		"reason": "Testgutschrift", "created_by": uuid.New(),
		"created_at": now, "updated_at": now, "currency": "EUR",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_credit_notes", id) })
}

func TestService_Record_SentCreditNoteCountsTowardPaid(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Payment Credit Note")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	repo := NewPostgresRepository(pool)
	invRepo := invoice.NewPostgresRepository(pool)
	svc := NewService(repo, invRepo, invRepo, pool)

	record := func(t *testing.T, invoiceID uuid.UUID, amount string) *models.Payment {
		t.Helper()
		p, err := svc.Record(ctx, RecordInput{
			TenantID: tenantID, InvoiceID: invoiceID, UserID: uuid.New(),
			Amount: decimal.RequireFromString(amount), Method: "bank_transfer",
		})
		require.NoError(t, err)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_payments", p.ID) })
		return p
	}

	t.Run("sent credit note plus payment settles the invoice, delete reverts it", func(t *testing.T) {
		invoiceID := seedLockInvoice(t, pool, tenantID, models.InvoiceStatusSent, "1000.00")
		seedCreditNoteOn(t, pool, tenantID, invoiceID, models.CreditNoteStatusSent, "400.00")

		p := record(t, invoiceID, "600.00")
		assert.Equal(t, models.InvoiceStatusPaid, invoiceStatus(t, pool, ctx, invoiceID))

		require.NoError(t, svc.Delete(ctx, tenantID, p.ID))
		assert.NotEqual(t, models.InvoiceStatusPaid, invoiceStatus(t, pool, ctx, invoiceID),
			"without the payment the credit note alone does not cover the invoice")
	})

	t.Run("deleting a payment keeps the invoice paid while credit note and rest still cover it", func(t *testing.T) {
		invoiceID := seedLockInvoice(t, pool, tenantID, models.InvoiceStatusSent, "1000.00")
		seedCreditNoteOn(t, pool, tenantID, invoiceID, models.CreditNoteStatusSent, "400.00")

		record(t, invoiceID, "600.00")
		extra := record(t, invoiceID, "200.00")
		require.Equal(t, models.InvoiceStatusPaid, invoiceStatus(t, pool, ctx, invoiceID))

		require.NoError(t, svc.Delete(ctx, tenantID, extra.ID))
		assert.Equal(t, models.InvoiceStatusPaid, invoiceStatus(t, pool, ctx, invoiceID),
			"600 paid + 400 credited still cover 1000")
	})

	t.Run("draft credit note does not reduce the amount due", func(t *testing.T) {
		invoiceID := seedLockInvoice(t, pool, tenantID, models.InvoiceStatusSent, "1000.00")
		seedCreditNoteOn(t, pool, tenantID, invoiceID, models.CreditNoteStatusDraft, "400.00")

		record(t, invoiceID, "600.00")
		assert.Equal(t, models.InvoiceStatusSent, invoiceStatus(t, pool, ctx, invoiceID))
	})
}
