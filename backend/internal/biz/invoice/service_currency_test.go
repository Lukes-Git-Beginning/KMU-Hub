package invoice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

func TestService_Create_AdHocCurrencyIsPinnedToTenantCurrency(t *testing.T) {
	quoteID := uuid.New()
	recurringID := uuid.New()

	tests := []struct {
		name           string
		tenantCurrency string // "" = no company settings
		requested      string
		sourceQuoteID  *uuid.UUID
		recurringID    *uuid.UUID
		wantErr        bool
		wantCurrency   string
	}{
		{name: "no currency takes the default", requested: "", wantCurrency: "EUR"},
		{name: "same currency as the default", requested: "EUR", wantCurrency: "EUR"},
		{name: "same currency in lower case", requested: "eur", wantCurrency: "eur"},
		{name: "foreign currency is rejected", requested: "CHF", wantErr: true},
		{name: "tenant currency from settings is accepted", tenantCurrency: "CHF", requested: "CHF", wantCurrency: "CHF"},
		{name: "default currency is rejected for a CHF tenant", tenantCurrency: "CHF", requested: "EUR", wantErr: true},
		{name: "invoice from a quote keeps the requested currency", requested: "CHF", sourceQuoteID: &quoteID, wantCurrency: "CHF"},
		{name: "recurring invoice keeps its own currency", requested: "USD", recurringID: &recurringID, wantCurrency: "USD"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo, _, cs, _ := newTestService()
			if tc.tenantCurrency != "" {
				cs.settings = &models.CompanySettings{DefaultCurrency: tc.tenantCurrency}
			}
			input := testCreateInput(uuid.New(), uuid.New())
			input.Currency = tc.requested
			input.SourceQuoteID = tc.sourceQuoteID
			input.RecurringID = tc.recurringID

			inv, err := svc.Create(context.Background(), input)

			if tc.wantErr {
				require.ErrorIs(t, err, ErrCurrencyNotTenantCurrency)
				assert.Empty(t, repo.invoices, "a rejected invoice must not be stored")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantCurrency, inv.Currency)
		})
	}
}

func TestService_Update_AdHocCurrencyChangeIsPinnedToTenantCurrency(t *testing.T) {
	quoteID := uuid.New()
	ptr := func(s string) *string { return &s }

	tests := []struct {
		name          string
		stored        string
		sourceQuoteID *uuid.UUID
		requested     string
		wantErr       bool
		wantCurrency  string
	}{
		{name: "echoing the stored currency is a no-op", stored: "EUR", requested: "EUR", wantCurrency: "EUR"},
		{name: "changing to a foreign currency is rejected", stored: "EUR", requested: "CHF", wantErr: true},
		{name: "a legacy CHF draft stays editable when the form echoes CHF", stored: "CHF", requested: "CHF", wantCurrency: "CHF"},
		{name: "a legacy CHF draft can move back to the tenant currency", stored: "CHF", requested: "EUR", wantCurrency: "EUR"},
		{name: "a draft converted from a quote may change currency", stored: "EUR", sourceQuoteID: &quoteID, requested: "CHF", wantCurrency: "CHF"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo, _, _, _ := newTestService()
			tenantID := uuid.New()
			inv := createDraftInvoice(t, repo, tenantID)
			inv.Currency = tc.stored
			inv.SourceQuoteID = tc.sourceQuoteID

			got, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{Currency: ptr(tc.requested)})

			if tc.wantErr {
				require.ErrorIs(t, err, ErrCurrencyNotTenantCurrency)
				assert.Equal(t, tc.stored, repo.invoices[inv.ID].Currency, "a rejected update must not change the stored currency")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantCurrency, got.Currency)
		})
	}
}
