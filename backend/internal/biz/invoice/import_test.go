package invoice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

func TestUpsertImported_CreatesReadOnlyMirror(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	due := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)

	in := models.ImportedInvoiceInput{
		ExternalID:     "4711",
		ExternalNumber: "2026-042",
		Status:         models.InvoiceStatusPaid,
		CustomerName:   "Muster AG",
		InvoiceDate:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		DueDate:        &due,
		Currency:       "CHF",
		Subtotal:       decimal.RequireFromString("100.00"),
		TotalTax:       decimal.RequireFromString("8.10"),
		GrossTotal:     decimal.RequireFromString("108.10"),
		Lines: []models.ImportedInvoiceLine{{
			Position: 1, Description: "Beratung",
			Quantity: decimal.RequireFromString("1"), UnitPrice: decimal.RequireFromString("100.00"),
			LineTotal: decimal.RequireFromString("100.00"),
		}},
	}

	inv, err := svc.UpsertImported(context.Background(), tenantID, in)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceSourceBexio, inv.Source)
	assert.Equal(t, "2026-042", inv.InvoiceNumber)
	require.NotNil(t, inv.ExternalID)
	assert.Equal(t, "4711", *inv.ExternalID)
	assert.Equal(t, models.InvoiceStatusPaid, inv.Status)
	assert.Equal(t, "CHF", inv.Currency)

	stored, ok := repo.invoices[inv.ID]
	require.True(t, ok)
	assert.Equal(t, models.InvoiceSourceBexio, stored.Source)
}

func TestUpsertImported_RequiresExternalIDAndNumber(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	tenantID := uuid.New()

	_, err := svc.UpsertImported(context.Background(), tenantID, models.ImportedInvoiceInput{ExternalNumber: "X"})
	require.Error(t, err) // missing external id

	_, err = svc.UpsertImported(context.Background(), tenantID, models.ImportedInvoiceInput{ExternalID: "1"})
	require.Error(t, err) // missing external number
}

func TestUpsertImported_Idempotent(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	in := models.ImportedInvoiceInput{
		ExternalID: "4711", ExternalNumber: "2026-042", Status: models.InvoiceStatusSent,
		GrossTotal: decimal.RequireFromString("108.10"),
	}

	first, err := svc.UpsertImported(context.Background(), tenantID, in)
	require.NoError(t, err)
	second, err := svc.UpsertImported(context.Background(), tenantID, in)
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID)
	assert.Len(t, repo.invoices, 1)
}

// TestImportedInvoiceRejectsMutations verifies every Cosmi mutation path refuses a
// source='bexio' mirror with ErrExternalReadOnly (the external system stays the book
// of record).
func TestImportedInvoiceRejectsMutations(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	extID, extNum := "4711", "2026-042"
	id := uuid.New()
	repo.invoices[id] = &models.Invoice{
		ID: id, TenantID: tenantID, InvoiceNumber: extNum, Status: models.InvoiceStatusSent,
		Source: models.InvoiceSourceBexio, ExternalID: &extID, ExternalNumber: &extNum,
		InvoiceDate: time.Now(), DueDate: time.Now(),
	}
	ctx := context.Background()

	_, updErr := svc.Update(ctx, tenantID, id, UpdateInput{})
	assert.ErrorIs(t, updErr, ErrExternalReadOnly)

	assert.ErrorIs(t, svc.Send(ctx, tenantID, id, uuid.New()), ErrExternalReadOnly)
	assert.ErrorIs(t, svc.MarkPaid(ctx, tenantID, id), ErrExternalReadOnly)
	assert.ErrorIs(t, svc.Cancel(ctx, tenantID, id, uuid.New()), ErrExternalReadOnly)

	_, lockErr := svc.LockInvoice(ctx, tenantID, id, uuid.New())
	assert.ErrorIs(t, lockErr, ErrExternalReadOnly)
}

// TestJournalIgnoresImportedInvoices verifies the GoBD journal counts only Cosmi-issued
// invoices, so an imported Bexio mirror never distorts the gap-free sequence check.
func TestJournalIgnoresImportedInvoices(t *testing.T) {
	svc, repo, numSeq, _, _ := newTestService()
	tenantID := uuid.New()
	year := 2026
	numSeq.seqInfo = &SequenceInfo{CurrentNumber: 1, FiscalYear: year}

	cosmiID := uuid.New()
	repo.invoices[cosmiID] = &models.Invoice{
		ID: cosmiID, TenantID: tenantID, InvoiceNumber: "RE-2026-0001",
		Status: models.InvoiceStatusSent, Source: models.InvoiceSourceCosmi,
		InvoiceDate: time.Date(year, 3, 1, 0, 0, 0, 0, time.UTC),
	}

	extID, bexioNum := "9000", "B-2026-77"
	bexioID := uuid.New()
	repo.invoices[bexioID] = &models.Invoice{
		ID: bexioID, TenantID: tenantID, InvoiceNumber: bexioNum,
		Status: models.InvoiceStatusPaid, Source: models.InvoiceSourceBexio,
		ExternalID: &extID, ExternalNumber: &bexioNum,
		InvoiceDate: time.Date(year, 4, 1, 0, 0, 0, 0, time.UTC),
	}

	summary, err := svc.GetJournalSummary(context.Background(), tenantID, year)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.TotalInvoicesIssued) // only the Cosmi invoice
	assert.Equal(t, 0, summary.GapsDetected)        // seq=1, count=1 → no gap
}
