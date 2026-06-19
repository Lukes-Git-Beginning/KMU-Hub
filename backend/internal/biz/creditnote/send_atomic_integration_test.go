//go:build integration

// External test package so it can import the quote package (real number-sequence
// repo) without the import cycle quote→invoice would create inside package creditnote.
package creditnote_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/creditnote"
	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/quote"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testsupport/pgtc"
)

func seedTenantExt(t *testing.T, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID) {
	t.Helper()
	_, err := pool.Exec(ctx,
		`INSERT INTO tenants (id, name, created_at, updated_at)
		 VALUES ($1, $2, NOW(), NOW()) ON CONFLICT (id) DO NOTHING`,
		tenantID, fmt.Sprintf("Test Tenant %s", tenantID.String()[:8]),
	)
	if err != nil {
		t.Logf("seedTenant (non-fatal): %v", err)
	}
}

func oneValidLine() []models.LineItem {
	return []models.LineItem{{
		ID: "l1", Position: 1, Description: "Gutschrift-Posten",
		Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100),
		TaxRate: decimal.NewFromInt(19), LineTotal: decimal.NewFromInt(100),
	}}
}

// seedInvoice creates a minimal invoice row to satisfy the credit note FK.
func seedInvoice(t *testing.T, ctx context.Context, repo *invoice.PostgresRepository, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	raw, _ := json.Marshal(oneValidLine())
	now := time.Now().UTC().Truncate(time.Second)
	inv := &models.Invoice{
		ID: uuid.New(), TenantID: tenantID, InvoiceNumber: "RE-FK-" + uuid.New().String()[:8],
		Status: models.InvoiceStatusSent, CustomerName: "FK GmbH",
		TaxMode: models.TaxModeStandard, LineItems: raw,
		Subtotal: decimal.NewFromInt(100), TotalTax: decimal.NewFromInt(19), GrossTotal: decimal.NewFromInt(119),
		InvoiceDate: now, DueDate: now.Add(14 * 24 * time.Hour), PaymentTerms: "14 days",
		CreatedBy: uuid.New(), CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, repo.Create(ctx, inv))
	return inv.ID
}

func draftCreditNote(tenantID, originalInvoiceID uuid.UUID, lines []models.LineItem) *models.CreditNote {
	raw, _ := json.Marshal(lines)
	now := time.Now().UTC().Truncate(time.Second)
	return &models.CreditNote{
		ID:                uuid.New(),
		TenantID:          tenantID,
		CreditNoteNumber:  "", // draft: number assigned on Send
		Status:            models.CreditNoteStatusDraft,
		OriginalInvoiceID: originalInvoiceID,
		CustomerName:      "Atomic GmbH",
		TaxMode:           models.TaxModeStandard,
		LineItems:         raw,
		Subtotal:          decimal.NewFromInt(100),
		TotalTax:          decimal.NewFromInt(19),
		GrossTotal:        decimal.NewFromInt(119),
		Reason:            "Teilstorno",
		CreatedBy:         uuid.New(),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func creditNoteSeq(t *testing.T, superPool *pgxpool.Pool, tenantID uuid.UUID, year int) (int, bool) {
	t.Helper()
	var n int
	err := superPool.QueryRow(context.Background(),
		`SELECT current_number FROM finance_number_sequences
		 WHERE tenant_id = $1 AND document_type = $2 AND fiscal_year = $3`,
		tenantID, models.DocumentTypeCreditNote, year,
	).Scan(&n)
	if err != nil {
		return 0, false
	}
	return n, true
}

// TestCreditNoteSend_AtomicRollback_NumberNotConsumed proves F6 for credit notes:
// a failed update after number assignment rolls the sequence increment back.
func TestCreditNoteSend_AtomicRollback_NumberNotConsumed(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantExt(t, appPool, ctx, tenantID)
	year := time.Now().Year()

	invRepo := invoice.NewPostgresRepository(appPool)
	originalInvoiceID := seedInvoice(t, ctx, invRepo, tenantID)

	repo := creditnote.NewPostgresRepository(appPool)
	seqRepo := quote.NewPostgresNumberSequenceRepo(appPool)
	svc := creditnote.NewService(repo, nil, seqRepo, appPool)

	// 1) Send a valid credit note A → the GS sequence advances to 1.
	cnA := draftCreditNote(tenantID, originalInvoiceID, oneValidLine())
	require.NoError(t, repo.Create(ctx, cnA))
	require.NoError(t, svc.Send(ctx, tenantID, cnA.ID, uuid.New()))

	seq, ok := creditNoteSeq(t, superPool, tenantID, year)
	require.True(t, ok, "sequence row must exist after the first send")
	require.Equal(t, 1, seq)

	// 2) Create credit note B, then corrupt its line_items to quantity 0 so the line
	//    re-insert inside UpdateInTx violates chk_credit_note_lines_quantity > 0.
	cnB := draftCreditNote(tenantID, originalInvoiceID, oneValidLine())
	require.NoError(t, repo.Create(ctx, cnB))
	_, err := superPool.Exec(context.Background(),
		`UPDATE finance_credit_notes SET line_items =
		   '[{"id":"l1","position":1,"description":"Bad","quantity":"0","unit_price":"100","tax_rate":"19","line_total":"0"}]'::jsonb
		 WHERE id = $1`, cnB.ID)
	require.NoError(t, err)

	// 3) Send B → must fail during the coupled transaction.
	require.Error(t, svc.Send(ctx, tenantID, cnB.ID, uuid.New()),
		"send must fail when the line re-insert violates the quantity constraint")

	// 4) Sequence unchanged — number 2 rolled back, not consumed.
	seqAfter, ok := creditNoteSeq(t, superPool, tenantID, year)
	require.True(t, ok)
	assert.Equal(t, 1, seqAfter, "a failed send must not consume a sequence number (GoBD gap-free)")

	// 5) Credit note B must remain a draft without a number.
	gotB, err := repo.GetByID(ctx, tenantID, cnB.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CreditNoteStatusDraft, gotB.Status)
	assert.Empty(t, gotB.CreditNoteNumber)
}
