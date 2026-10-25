package creditnote

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// newTestLineItems credits 100 net at 19 % = 119.00 gross.
var testCreditGross = decimal.NewFromInt(119)

func createOnInvoice(t *testing.T, svc *Service, tenantID, invoiceID uuid.UUID) (*models.CreditNote, error) {
	t.Helper()
	return svc.Create(context.Background(), CreateInput{
		TenantID:          tenantID,
		OriginalInvoiceID: invoiceID,
		LineItems:         newTestLineItems(),
		TaxMode:           models.TaxModeStandard,
		Reason:            "limit test",
		UserID:            uuid.New(),
	})
}

func TestService_Create_CreditExactlyAtInvoiceGrossIsAccepted(t *testing.T) {
	svc, repo, invReader, _ := newTestService()
	tenantID := uuid.New()
	inv := newTestInvoice(tenantID, models.InvoiceStatusSent)
	inv.GrossTotal = testCreditGross
	invReader.AddInvoice(inv)

	cn, err := createOnInvoice(t, svc, tenantID, inv.ID)

	require.NoError(t, err)
	assert.True(t, testCreditGross.Equal(cn.GrossTotal))
	assert.Len(t, repo.creditNotes, 1)
}

func TestService_Create_OneCentOverInvoiceGrossIsRefused(t *testing.T) {
	svc, repo, invReader, _ := newTestService()
	tenantID := uuid.New()
	inv := newTestInvoice(tenantID, models.InvoiceStatusSent)
	inv.GrossTotal = decimal.RequireFromString("118.99")
	invReader.AddInvoice(inv)

	_, err := createOnInvoice(t, svc, tenantID, inv.ID)

	require.ErrorIs(t, err, ErrCreditExceedsInvoice)
	assert.Contains(t, err.Error(), "118.99", "the message names what is still creditable")
	assert.Empty(t, repo.creditNotes, "a refused credit note must not be stored")
}

func TestService_Create_DraftsCountTowardsTheLimit(t *testing.T) {
	svc, repo, invReader, _ := newTestService()
	tenantID := uuid.New()
	inv := newTestInvoice(tenantID, models.InvoiceStatusSent)
	inv.GrossTotal = decimal.NewFromInt(200) // fits one 119, not two
	invReader.AddInvoice(inv)

	_, err := createOnInvoice(t, svc, tenantID, inv.ID)
	require.NoError(t, err)

	_, err = createOnInvoice(t, svc, tenantID, inv.ID)
	require.ErrorIs(t, err, ErrCreditExceedsInvoice)
	assert.Len(t, repo.creditNotes, 1, "the second draft is refused, only the first exists")
}

func TestService_Create_LimitIsPerInvoice(t *testing.T) {
	svc, _, invReader, _ := newTestService()
	tenantID := uuid.New()
	a := newTestInvoice(tenantID, models.InvoiceStatusSent)
	a.GrossTotal = testCreditGross
	b := newTestInvoice(tenantID, models.InvoiceStatusSent)
	b.GrossTotal = testCreditGross
	invReader.AddInvoice(a)
	invReader.AddInvoice(b)

	_, err := createOnInvoice(t, svc, tenantID, a.ID)
	require.NoError(t, err)

	_, err = createOnInvoice(t, svc, tenantID, b.ID)
	assert.NoError(t, err, "credit notes of invoice A must not eat into invoice B's limit")
}

// lockedStatusReader reports a different status under the row lock than the earlier
// plain read did — an invoice cancelled between the two.
type lockedStatusReader struct {
	*MockInvoiceReader
	status string
}

func (l lockedStatusReader) LockForCreditInTx(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID) (string, decimal.Decimal, error) {
	_, gross, err := l.MockInvoiceReader.LockForCreditInTx(ctx, tx, tenantID, id)
	return l.status, gross, err
}

func TestService_Create_StatusIsRecheckedUnderTheLock(t *testing.T) {
	repo := NewMockRepository()
	reader := NewMockInvoiceReader()
	svc := NewService(repo, lockedStatusReader{reader, models.InvoiceStatusCancelled}, NewMockNumberSequenceRepo(), noopTxBeginner{})
	tenantID := uuid.New()
	inv := newTestInvoice(tenantID, models.InvoiceStatusSent)
	reader.AddInvoice(inv)

	_, err := createOnInvoice(t, svc, tenantID, inv.ID)

	require.ErrorIs(t, err, ErrInvalidInvoiceForCredit)
	assert.Empty(t, repo.creditNotes)
}

func TestService_Create_RequiresPool(t *testing.T) {
	reader := NewMockInvoiceReader()
	svc := NewService(NewMockRepository(), reader, NewMockNumberSequenceRepo(), nil)
	tenantID := uuid.New()
	inv := newTestInvoice(tenantID, models.InvoiceStatusSent)
	reader.AddInvoice(inv)

	_, err := createOnInvoice(t, svc, tenantID, inv.ID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pool not configured")
}

func TestService_StornoInvoice_MirrorsAnInvoiceAtTheLimit(t *testing.T) {
	svc, repo, invReader, _ := newTestService()
	tenantID := uuid.New()
	inv := sentInvoiceWithLines(tenantID)
	inv.GrossTotal = testCreditGross // exactly what the mirrored credit note costs
	invReader.AddInvoice(inv)
	svc.SetInvoiceStatusUpdater(&fakeInvoiceCanceller{})

	cnID, _, err := svc.StornoInvoice(context.Background(), tenantID, inv.ID, uuid.New())

	require.NoError(t, err)
	assert.True(t, testCreditGross.Equal(repo.creditNotes[cnID].GrossTotal))
}

func TestService_StornoInvoice_RefusedWhenADraftAlreadyEatsTheLimit(t *testing.T) {
	svc, _, invReader, _ := newTestService()
	tenantID := uuid.New()
	inv := sentInvoiceWithLines(tenantID)
	inv.GrossTotal = testCreditGross
	invReader.AddInvoice(inv)
	svc.SetInvoiceStatusUpdater(&fakeInvoiceCanceller{})

	_, err := createOnInvoice(t, svc, tenantID, inv.ID) // a draft for the full amount
	require.NoError(t, err)

	_, _, err = svc.StornoInvoice(context.Background(), tenantID, inv.ID, uuid.New())

	assert.ErrorIs(t, err, ErrCreditExceedsInvoice, "a full storno on top of a full draft would credit twice")
}
