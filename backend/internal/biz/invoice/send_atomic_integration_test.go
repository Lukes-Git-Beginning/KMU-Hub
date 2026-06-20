//go:build integration

// External test package (invoice_test) so it can import the quote package for the
// real number-sequence repo without the import cycle that quote→invoice would
// create inside package invoice.
package invoice_test

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

func draftInvoice(tenantID uuid.UUID, lines []models.LineItem) *models.Invoice {
	raw, _ := json.Marshal(lines)
	now := time.Now().UTC().Truncate(time.Second)
	return &models.Invoice{
		ID:              uuid.New(),
		TenantID:        tenantID,
		InvoiceNumber:   "", // draft: number assigned on Send
		Status:          models.InvoiceStatusDraft,
		CustomerName:    "Atomic GmbH",
		CustomerAddress: "Teststr. 1, 10115 Berlin",
		CustomerEmail:   "atomic@example.com",
		TaxMode:         models.TaxModeStandard,
		LineItems:       raw,
		Subtotal:        decimal.NewFromInt(100),
		TotalTax:        decimal.NewFromInt(19),
		GrossTotal:      decimal.NewFromInt(119),
		InvoiceDate:     now,
		DueDate:         now.Add(14 * 24 * time.Hour),
		PaymentTerms:    "14 days",
		CreatedBy:       uuid.New(),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func oneValidLine() []models.LineItem {
	return []models.LineItem{{
		ID: "l1", Position: 1, Description: "Leistung",
		Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100),
		TaxRate: decimal.NewFromInt(19), LineTotal: decimal.NewFromInt(100),
	}}
}

// invoiceSeq reads the current_number for the invoice sequence (superuser, RLS-bypass).
// Returns (0, false) when no sequence row exists yet.
func invoiceSeq(t *testing.T, superPool *pgxpool.Pool, tenantID uuid.UUID, year int) (int, bool) {
	t.Helper()
	var n int
	err := superPool.QueryRow(context.Background(),
		`SELECT current_number FROM finance_number_sequences
		 WHERE tenant_id = $1 AND document_type = $2 AND fiscal_year = $3`,
		tenantID, models.DocumentTypeInvoice, year,
	).Scan(&n)
	if err != nil {
		return 0, false
	}
	return n, true
}

// TestSend_AtomicRollback_NumberNotConsumed proves F6: when the document update
// fails after the number is assigned, the sequence increment is rolled back in the
// same transaction, so a failed send does not burn a gap-free number.
func TestSend_AtomicRollback_NumberNotConsumed(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantExt(t, appPool, ctx, tenantID)
	year := time.Now().Year()

	repo := invoice.NewPostgresRepository(appPool)
	seqRepo := quote.NewPostgresNumberSequenceRepo(appPool)
	csRepo := quote.NewPostgresCompanySettingsRepo(appPool)
	svc := invoice.NewService(repo, seqRepo, csRepo, nil, appPool)

	// 1) Send a valid invoice A → the sequence advances to 1.
	invA := draftInvoice(tenantID, oneValidLine())
	require.NoError(t, repo.Create(ctx, invA))
	require.NoError(t, svc.Send(ctx, tenantID, invA.ID, uuid.New()))

	seq, ok := invoiceSeq(t, superPool, tenantID, year)
	require.True(t, ok, "sequence row must exist after the first send")
	require.Equal(t, 1, seq)

	// 2) Create invoice B, then corrupt its relational line to quantity 0 so the
	//    line re-insert inside UpdateInTx violates chk_invoice_lines_quantity > 0.
	//    GetByID loads lines from finance_invoice_lines, so corrupting the relational
	//    row means Send will attempt to re-insert a quantity=0 line and hit the constraint.
	invB := draftInvoice(tenantID, oneValidLine())
	require.NoError(t, repo.Create(ctx, invB))
	_, err := superPool.Exec(context.Background(),
		`UPDATE finance_invoice_lines SET quantity = '0' WHERE invoice_id = $1`, invB.ID)
	require.NoError(t, err)

	// 3) Send B → must fail (constraint violation during the coupled transaction).
	sendErr := svc.Send(ctx, tenantID, invB.ID, uuid.New())
	require.Error(t, sendErr, "send must fail when the line re-insert violates the quantity constraint")

	// 4) The sequence must be unchanged — number 2 was rolled back, not consumed.
	seqAfter, ok := invoiceSeq(t, superPool, tenantID, year)
	require.True(t, ok)
	assert.Equal(t, 1, seqAfter, "a failed send must not consume a sequence number (GoBD gap-free)")

	// 5) Invoice B must remain a draft without a number.
	gotB, err := repo.GetByID(ctx, tenantID, invB.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusDraft, gotB.Status)
	assert.Empty(t, gotB.InvoiceNumber)
}
