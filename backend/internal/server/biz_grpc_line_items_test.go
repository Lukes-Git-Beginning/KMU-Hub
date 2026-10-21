package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
	"github.com/kmuhub/kmuhub/internal/biz/recurring"
	"github.com/kmuhub/kmuhub/internal/models"
)

func TestProtoLineItemsToModel(t *testing.T) {
	item := func(qty, price, rate, total string) []*bizv1.LineItem {
		return []*bizv1.LineItem{lineItem(), {Description: "x", Quantity: qty, UnitPrice: price, TaxRate: rate, LineTotal: total}}
	}

	cases := []struct {
		name    string
		items   []*bizv1.LineItem
		mode    string
		wantMsg string // empty = success
	}{
		{"valid standard", item("2", "10.50", "19", ""), models.TaxModeStandard, ""},
		{"german comma quantity", item("1,5", "10", "19", ""), models.TaxModeStandard, "line_items[1].quantity"},
		{"german comma price", item("1", "9,99", "19", ""), models.TaxModeStandard, "line_items[1].unit_price"},
		{"garbage tax rate", item("1", "10", "19%", ""), models.TaxModeKleinunternehmer, "line_items[1].tax_rate"},
		{"garbage line total", item("1", "10", "19", "abc"), models.TaxModeStandard, "line_items[1].line_total"},
		{"empty quantity", item("", "10", "19", ""), models.TaxModeStandard, "line_items[1].quantity is required"},
		{"empty price", item("1", "", "19", ""), models.TaxModeStandard, "line_items[1].unit_price is required"},
		{"empty rate in standard", item("1", "10", "", ""), models.TaxModeStandard, "line_items[1].tax_rate is required"},
		{"empty rate in kleinunternehmer", item("1", "10", "", ""), models.TaxModeKleinunternehmer, ""},
		{"empty rate in reverse charge", item("1", "10", "", ""), models.TaxModeReverseCharge, ""},
		{"empty rate with unknown update mode", item("1", "10", "", ""), "", ""},
		{"negative amounts stay legal", item("-1", "-10", "19", ""), models.TaxModeStandard, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := protoLineItemsToModel(tc.items, tc.mode)
			if tc.wantMsg == "" {
				require.NoError(t, err)
				assert.Len(t, got, len(tc.items))
				return
			}
			requireGRPCCode(t, err, codes.InvalidArgument)
			assert.Contains(t, status.Convert(err).Message(), tc.wantMsg)
			assert.Nil(t, got)
		})
	}
}

// The seven RPCs that take positions must surface the parse error instead of
// persisting a 0 EUR line.
func TestLineItemParseErrorsReachTheClient(t *testing.T) {
	tenant := uuid.New().String()
	user := uuid.New().String()
	id := uuid.New().String()
	bad := []*bizv1.LineItem{{Description: "x", Quantity: "1,5", UnitPrice: "10.00", TaxRate: "19.00"}}

	quoteSrv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
	finSrv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
	recSrv := &BizGRPCServer{recurringSvc: recurring.NewService(nil, nil, nil)}

	calls := map[string]func() error{
		"CreateQuote": func() error {
			_, err := quoteSrv.CreateQuote(context.Background(), &bizv1.CreateQuoteRequest{TenantId: tenant, CreatedBy: user, LineItems: bad})
			return err
		},
		"UpdateQuote": func() error {
			_, err := quoteSrv.UpdateQuote(context.Background(), &bizv1.UpdateQuoteRequest{TenantId: tenant, Id: id, LineItems: bad})
			return err
		},
		"CreateInvoice": func() error {
			_, err := finSrv.CreateInvoice(context.Background(), &bizv1.CreateInvoiceRequest{TenantId: tenant, CreatedBy: user, LineItems: bad})
			return err
		},
		"UpdateInvoice": func() error {
			_, err := finSrv.UpdateInvoice(context.Background(), &bizv1.UpdateInvoiceRequest{TenantId: tenant, Id: id, LineItems: bad})
			return err
		},
		"CreateCreditNote": func() error {
			_, err := finSrv.CreateCreditNote(context.Background(), &bizv1.CreateCreditNoteRequest{
				TenantId: tenant, CreatedBy: user, OriginalInvoiceId: uuid.New().String(), LineItems: bad,
			})
			return err
		},
		"CreateRecurringInvoice": func() error {
			_, err := recSrv.CreateRecurringInvoice(context.Background(), &bizv1.CreateRecurringInvoiceRequest{
				TenantId: tenant, StartDate: "2026-01-01", LineItems: bad,
			})
			return err
		},
		"UpdateRecurringInvoice": func() error {
			_, err := recSrv.UpdateRecurringInvoice(context.Background(), &bizv1.UpdateRecurringInvoiceRequest{
				TenantId: tenant, Id: id, LineItems: bad,
			})
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			err := call()
			requireGRPCCode(t, err, codes.InvalidArgument)
			assert.Contains(t, status.Convert(err).Message(), "line_items[0].quantity")
		})
	}
}

func TestUpdateDunningConfigRejectsNegativeFees(t *testing.T) {
	tenantID := uuid.New()
	for _, cfg := range []*bizv1.DunningConfig{
		{Level1Fee: "-5.00"},
		{Level2Fee: "-0.01"},
		{Level3Fee: "-10"},
	} {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{config: defaultDunningConfig(tenantID)}, &stubDunningInvoiceReader{})
		_, err := srv.UpdateDunningConfig(context.Background(), &bizv1.UpdateDunningConfigRequest{TenantId: tenantID.String(), Config: cfg})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, status.Convert(err).Message(), "must not be negative")
	}

	srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{config: defaultDunningConfig(tenantID)}, &stubDunningInvoiceReader{})
	_, err := srv.UpdateDunningConfig(context.Background(), &bizv1.UpdateDunningConfigRequest{
		TenantId: tenantID.String(), Config: &bizv1.DunningConfig{Level1Fee: "0", Level2Fee: "5", Level3Fee: "10.50"},
	})
	require.NoError(t, err, "zero and positive fees stay valid")
}

func TestUpdateCompanySettingsBasiszinssatzBounds(t *testing.T) {
	tenantID := uuid.New()
	update := func(rate string) error {
		settings := &stubCompanySettingsRepo{settings: &models.CompanySettings{TenantID: tenantID}}
		srv := newQuoteTestServer(newStubQuoteRepo(), settings, nil)
		_, err := srv.UpdateCompanySettings(context.Background(), &bizv1.UpdateCompanySettingsRequest{
			TenantId: tenantID.String(), Settings: &bizv1.CompanySettings{Name: "Acme GmbH", Basiszinssatz: rate},
		})
		return err
	}

	for _, rate := range []string{"350", "-9", "25.01", "-5.01"} {
		requireGRPCCode(t, update(rate), codes.InvalidArgument)
	}
	// The Bundesbank rate was -0.88 % until mid-2022: negative is legal.
	for _, rate := range []string{"-0.88", "0", "3.62", "25", "-5"} {
		require.NoError(t, update(rate), rate)
	}
}
