package invoice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func day(y int, m time.Month, d int) *time.Time {
	t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return &t
}

func TestService_Update_SetsCurrency(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	quoteID := uuid.New()
	inv.SourceQuoteID = &quoteID // ad-hoc drafts are pinned to the tenant currency

	chf, empty := "CHF", ""
	got, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{Currency: &chf})
	require.NoError(t, err)
	assert.Equal(t, "CHF", got.Currency)

	got, err = svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{Currency: &empty})
	require.NoError(t, err)
	assert.Equal(t, "CHF", got.Currency, "empty currency leaves the stored one")
}

func TestService_Update_PaymentTermsDaysRecomputesDueDate(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID) // invoice date 2026-03-01

	days := 14
	got, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{PaymentTermsDays: &days})
	require.NoError(t, err)
	assert.Equal(t, "2026-03-15", got.DueDate.Format("2006-01-02"))
	assert.Equal(t, "14 Tage netto", got.PaymentTerms)
}

func TestService_Update_InvoiceDateMovesDueDateAlong(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID) // 2026-03-01 -> due 2026-03-31

	got, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{InvoiceDate: day(2026, time.April, 1)})
	require.NoError(t, err)
	assert.Equal(t, "2026-04-01", got.InvoiceDate.Format("2006-01-02"))
	assert.Equal(t, "2026-05-01", got.DueDate.Format("2006-01-02"), "due date keeps its 30-day distance")
}

func TestService_Update_InvoiceDateWithTermsUsesTheTerms(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)

	days := 10
	got, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{
		InvoiceDate: day(2026, time.April, 1), PaymentTermsDays: &days,
	})
	require.NoError(t, err)
	assert.Equal(t, "2026-04-11", got.DueDate.Format("2006-01-02"))
}

func TestService_Update_ExplicitDueDateAndDeliveryDate(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)

	got, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{
		InvoiceDate: day(2026, time.April, 1), DueDate: day(2026, time.April, 20), DeliveryDate: day(2026, time.March, 28),
	})
	require.NoError(t, err)
	assert.Equal(t, "2026-04-20", got.DueDate.Format("2006-01-02"))
	require.NotNil(t, got.DeliveryDate)
	assert.Equal(t, "2026-03-28", got.DeliveryDate.Format("2006-01-02"))
}

func TestService_Update_ExplicitDueDateBeatsPaymentTermsDays(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID) // invoice date 2026-03-01

	days := 14
	got, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{
		PaymentTermsDays: &days, DueDate: day(2026, time.March, 9),
	})
	require.NoError(t, err)
	assert.Equal(t, "2026-03-09", got.DueDate.Format("2006-01-02"), "the named due date is not overwritten by the terms")
	assert.Equal(t, "14 Tage netto", got.PaymentTerms)
}

func TestService_Update_FreeTextTermsKeepTheDueDate(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID) // due 2026-03-31

	terms := "Sofort fällig"
	got, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{PaymentTerms: &terms})
	require.NoError(t, err)
	assert.Equal(t, "Sofort fällig", got.PaymentTerms)
	assert.Equal(t, "2026-03-31", got.DueDate.Format("2006-01-02"))

	empty := ""
	got, err = svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{PaymentTerms: &empty})
	require.NoError(t, err)
	assert.Equal(t, "Sofort fällig", got.PaymentTerms, "an empty label leaves the stored one")
}
