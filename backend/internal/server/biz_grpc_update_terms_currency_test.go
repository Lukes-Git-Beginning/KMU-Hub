package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

func TestUpdateInvoice_CarriesDatesTermsAndCurrency(t *testing.T) {
	tenantID := uuid.New()
	repo := newStubInvoiceRepo()
	inv := draftInvoice(tenantID)
	quoteID := uuid.New()
	inv.SourceQuoteID = &quoteID // an ad-hoc draft is pinned to the tenant currency
	repo.invoices[inv.ID] = inv
	srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

	resp, err := srv.UpdateInvoice(context.Background(), &bizv1.UpdateInvoiceRequest{
		TenantId: tenantID.String(), Id: inv.ID.String(),
		InvoiceDate: "2026-05-01", DeliveryDate: "2026-04-28", PaymentTerms: "14 Tage netto", Currency: "CHF",
	})
	require.NoError(t, err)
	got := resp.GetInvoice()
	assert.Equal(t, "2026-05-01", got.GetInvoiceDate())
	assert.Equal(t, "2026-04-28", got.GetDeliveryDate())
	assert.Equal(t, "2026-05-15", got.GetDueDate())
	assert.Equal(t, "14 Tage netto", got.GetPaymentTerms())
	assert.Equal(t, "CHF", got.GetCurrency())
}

func TestUpdateInvoice_AdHocForeignCurrencyIs400(t *testing.T) {
	tenantID := uuid.New()
	repo := newStubInvoiceRepo()
	inv := draftInvoice(tenantID)
	repo.invoices[inv.ID] = inv
	srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

	_, err := srv.UpdateInvoice(context.Background(), &bizv1.UpdateInvoiceRequest{
		TenantId: tenantID.String(), Id: inv.ID.String(), Currency: "CHF",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
	assert.NotEqual(t, "CHF", repo.invoices[inv.ID].Currency)
}

func TestUpdateInvoice_ExplicitDueDateAndBadDates(t *testing.T) {
	tenantID := uuid.New()
	repo := newStubInvoiceRepo()
	inv := draftInvoice(tenantID)
	repo.invoices[inv.ID] = inv
	srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

	resp, err := srv.UpdateInvoice(context.Background(), &bizv1.UpdateInvoiceRequest{
		TenantId: tenantID.String(), Id: inv.ID.String(), InvoiceDate: "2026-05-01", DueDate: "2026-05-09",
	})
	require.NoError(t, err)
	assert.Equal(t, "2026-05-09", resp.GetInvoice().GetDueDate())

	for _, req := range []*bizv1.UpdateInvoiceRequest{
		{InvoiceDate: "nope"}, {DeliveryDate: "nope"}, {DueDate: "nope"},
	} {
		req.TenantId, req.Id = tenantID.String(), inv.ID.String()
		_, err := srv.UpdateInvoice(context.Background(), req)
		requireGRPCCode(t, err, codes.InvalidArgument)
	}
}

func TestUpdateQuote_CarriesCurrency(t *testing.T) {
	tenantID := uuid.New()
	repo := newStubQuoteRepo()
	q := draftQuote(tenantID)
	repo.quotes[q.ID] = q
	srv := newQuoteTestServer(repo, &stubCompanySettingsRepo{}, nil)

	resp, err := srv.UpdateQuote(context.Background(), &bizv1.UpdateQuoteRequest{
		TenantId: tenantID.String(), Id: q.ID.String(), Currency: "USD",
	})
	require.NoError(t, err)
	assert.Equal(t, "USD", resp.GetQuote().GetCurrency())
}

func TestUpdateInvoice_DueDateBeatsNumericTermsAndFreeTextIsKept(t *testing.T) {
	tenantID := uuid.New()
	repo := newStubInvoiceRepo()
	inv := draftInvoice(tenantID)
	repo.invoices[inv.ID] = inv
	srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

	resp, err := srv.UpdateInvoice(context.Background(), &bizv1.UpdateInvoiceRequest{
		TenantId: tenantID.String(), Id: inv.ID.String(),
		InvoiceDate: "2026-05-01", PaymentTerms: "14 Tage netto", DueDate: "2026-05-09",
	})
	require.NoError(t, err)
	assert.Equal(t, "2026-05-09", resp.GetInvoice().GetDueDate(), "due_date wins over the days in payment_terms")
	assert.Equal(t, "14 Tage netto", resp.GetInvoice().GetPaymentTerms())

	resp, err = srv.UpdateInvoice(context.Background(), &bizv1.UpdateInvoiceRequest{
		TenantId: tenantID.String(), Id: inv.ID.String(), PaymentTerms: "Sofort fällig",
	})
	require.NoError(t, err)
	assert.Equal(t, "Sofort fällig", resp.GetInvoice().GetPaymentTerms(), "terms without a number are kept as text")
	assert.Equal(t, "2026-05-09", resp.GetInvoice().GetDueDate())
}
