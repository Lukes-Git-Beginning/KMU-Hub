package invoice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

func TestService_Create_ExplicitPaymentTermsBeatTenantDefault(t *testing.T) {
	svc, _, _, cs, _ := newTestService()
	cs.settings = &models.CompanySettings{DefaultPaymentTermsDays: 45}
	input := testCreateInput(uuid.New(), uuid.New())
	input.PaymentTermsDays = 14

	inv, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "14 Tage netto", inv.PaymentTerms)
	assert.Equal(t, input.InvoiceDate.AddDate(0, 0, 14), inv.DueDate)
}

func TestService_Create_CurrencyExplicitBeatsTenantDefault(t *testing.T) {
	svc, _, _, cs, _ := newTestService()
	cs.settings = &models.CompanySettings{DefaultCurrency: "EUR"}
	input := testCreateInput(uuid.New(), uuid.New())

	inv, err := svc.Create(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "EUR", inv.Currency, "no explicit currency takes the tenant default")

	// A quote-sourced invoice keeps the requested currency (ad-hoc ones are pinned,
	// see service_currency_test.go).
	quoteID := uuid.New()
	input.SourceQuoteID = &quoteID
	input.Currency = "CHF"
	inv, err = svc.Create(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "CHF", inv.Currency)
}

func TestService_Create_FreeTextPaymentTermsReplaceTheGeneratedLabel(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	input := testCreateInput(uuid.New(), uuid.New())
	input.PaymentTerms = "Sofort fällig"

	inv, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "Sofort fällig", inv.PaymentTerms)
	assert.Equal(t, input.InvoiceDate.AddDate(0, 0, input.PaymentTermsDays), inv.DueDate, "the due date still follows the days")
}
