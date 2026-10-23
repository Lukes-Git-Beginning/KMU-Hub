package gateway

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// bizCreateStub records what the handlers forward to the biz service, so the
// assertions run against the real gRPC request rather than a decoded struct.
type bizCreateStub struct {
	bizv1.UnimplementedFinanceServiceServer
	gotInvoice *bizv1.CreateInvoiceRequest
	gotQuote   *bizv1.CreateQuoteRequest
}

func (s *bizCreateStub) CreateInvoice(_ context.Context, r *bizv1.CreateInvoiceRequest) (*bizv1.CreateInvoiceResponse, error) {
	s.gotInvoice = r
	return &bizv1.CreateInvoiceResponse{Invoice: &bizv1.Invoice{Id: uuid.NewString()}}, nil
}

func (s *bizCreateStub) CreateQuote(_ context.Context, r *bizv1.CreateQuoteRequest) (*bizv1.CreateQuoteResponse, error) {
	s.gotQuote = r
	return &bizv1.CreateQuoteResponse{Quote: &bizv1.Quote{Id: uuid.NewString()}}, nil
}

func newBizCreateStubRoutes(t *testing.T) (*BizRoutes, *bizCreateStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &bizCreateStub{}
	srv := grpc.NewServer()
	bizv1.RegisterFinanceServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("biz", lis.Addr().String())
	return NewBizRoutes(reg), stub
}

func createDocBody(extra map[string]any) map[string]any {
	body := map[string]any{
		"customer": map[string]any{"name": "ACME"},
		"line_items": []any{map[string]any{
			"description": "Consulting", "quantity": "1", "unit_price": "100.00", "tax_rate": "19.00",
		}},
		"tax_mode":     "standard",
		"invoice_date": "2026-01-01",
	}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func postCreateDoc(t *testing.T, h http.HandlerFunc, path string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := withAuth(httptest.NewRequest(http.MethodPost, path, jsonBody(t, body)), uuid.NewString(), testTenantID)
	h(rec, req)
	return rec
}

func TestHandleCreateInvoice_ForwardsPaymentTermsDaysAndCurrency(t *testing.T) {
	routes, stub := newBizCreateStubRoutes(t)

	rec := postCreateDoc(t, routes.HandleCreateInvoice, "/api/v1/finance/invoices",
		createDocBody(map[string]any{"payment_terms_days": 14, "currency": "CHF", "exchange_rate": "0.95"}))

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.NotNil(t, stub.gotInvoice)
	assert.Equal(t, "14 Tage netto", stub.gotInvoice.GetPaymentTerms())
	assert.Equal(t, "CHF", stub.gotInvoice.GetCurrency())
}

func TestHandleCreateInvoice_WithoutTermsOrCurrencyKeepsTenantDefaults(t *testing.T) {
	routes, stub := newBizCreateStubRoutes(t)

	rec := postCreateDoc(t, routes.HandleCreateInvoice, "/api/v1/finance/invoices", createDocBody(nil))

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Empty(t, stub.gotInvoice.GetPaymentTerms())
	assert.Empty(t, stub.gotInvoice.GetCurrency())
}

func TestHandleCreateInvoice_ExplicitPaymentTermsStringWinsOverDays(t *testing.T) {
	routes, stub := newBizCreateStubRoutes(t)

	rec := postCreateDoc(t, routes.HandleCreateInvoice, "/api/v1/finance/invoices",
		createDocBody(map[string]any{"payment_terms": "7 Tage netto", "payment_terms_days": 14}))

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(t, "7 Tage netto", stub.gotInvoice.GetPaymentTerms())
}

func TestHandleCreateInvoice_RejectsBadTermsAndCurrency(t *testing.T) {
	routes, stub := newBizCreateStubRoutes(t)

	for name, extra := range map[string]map[string]any{
		"negative days":     {"payment_terms_days": -3},
		"absurd days":       {"payment_terms_days": 100000},
		"unknown currency":  {"currency": "XXXX"},
		"lowercase isocode": {"currency": "chf"},
	} {
		rec := postCreateDoc(t, routes.HandleCreateInvoice, "/api/v1/finance/invoices", createDocBody(extra))
		assert.Equal(t, http.StatusBadRequest, rec.Code, name)
	}
	assert.Nil(t, stub.gotInvoice, "invalid requests must not reach the biz service")
}

func TestHandleCreateQuote_ForwardsCurrency(t *testing.T) {
	routes, stub := newBizCreateStubRoutes(t)

	rec := postCreateDoc(t, routes.HandleCreateQuote, "/api/v1/finance/quotes",
		createDocBody(map[string]any{"currency": "USD", "exchange_rate": "1.08"}))

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.NotNil(t, stub.gotQuote)
	assert.Equal(t, "USD", stub.gotQuote.GetCurrency())

	rec = postCreateDoc(t, routes.HandleCreateQuote, "/api/v1/finance/quotes", createDocBody(map[string]any{"currency": "dollars"}))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
