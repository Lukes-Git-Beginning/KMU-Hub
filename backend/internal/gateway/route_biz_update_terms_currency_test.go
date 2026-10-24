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

// bizUpdateStub records the update requests the handlers forward to biz.
type bizUpdateStub struct {
	bizv1.UnimplementedFinanceServiceServer
	gotInvoice *bizv1.UpdateInvoiceRequest
	gotQuote   *bizv1.UpdateQuoteRequest
}

func (s *bizUpdateStub) UpdateInvoice(_ context.Context, r *bizv1.UpdateInvoiceRequest) (*bizv1.UpdateInvoiceResponse, error) {
	s.gotInvoice = r
	return &bizv1.UpdateInvoiceResponse{Invoice: &bizv1.Invoice{Id: r.GetId()}}, nil
}

func (s *bizUpdateStub) UpdateQuote(_ context.Context, r *bizv1.UpdateQuoteRequest) (*bizv1.UpdateQuoteResponse, error) {
	s.gotQuote = r
	return &bizv1.UpdateQuoteResponse{Quote: &bizv1.Quote{Id: r.GetId()}}, nil
}

func newBizUpdateStubRoutes(t *testing.T) (*BizRoutes, *bizUpdateStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &bizUpdateStub{}
	srv := grpc.NewServer()
	bizv1.RegisterFinanceServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("biz", lis.Addr().String())
	return NewBizRoutes(reg), stub
}

func putDoc(t *testing.T, h http.HandlerFunc, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := withAuth(httptest.NewRequest(http.MethodPut, "/api/v1/finance/doc/x", jsonBody(t, body)), uuid.NewString(), testTenantID)
	req = withChiURLParam(req, "id", uuid.NewString())
	h(rec, req)
	return rec
}

func TestHandleUpdateInvoice_ForwardsPaymentTermsDaysAndCurrency(t *testing.T) {
	routes, stub := newBizUpdateStubRoutes(t)

	rec := putDoc(t, routes.HandleUpdateInvoice, map[string]any{
		"invoice_date": "2026-05-01", "payment_terms_days": 14, "currency": "CHF", "exchange_rate": "0.95",
	})

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, stub.gotInvoice)
	assert.Equal(t, "14 Tage netto", stub.gotInvoice.GetPaymentTerms())
	assert.Equal(t, "CHF", stub.gotInvoice.GetCurrency())
	assert.Equal(t, "2026-05-01", stub.gotInvoice.GetInvoiceDate())
}

func TestHandleUpdateInvoice_ExplicitTermsStringWinsAndAbsentStaysEmpty(t *testing.T) {
	routes, stub := newBizUpdateStubRoutes(t)

	rec := putDoc(t, routes.HandleUpdateInvoice, map[string]any{"payment_terms": "7 Tage netto", "payment_terms_days": 14})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "7 Tage netto", stub.gotInvoice.GetPaymentTerms())

	rec = putDoc(t, routes.HandleUpdateInvoice, map[string]any{"notes": "x"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Empty(t, stub.gotInvoice.GetPaymentTerms())
	assert.Empty(t, stub.gotInvoice.GetCurrency())
}

func TestHandleUpdateInvoice_RejectsBadTermsAndCurrency(t *testing.T) {
	routes, stub := newBizUpdateStubRoutes(t)

	for name, body := range map[string]map[string]any{
		"negative days":    {"payment_terms_days": -3},
		"absurd days":      {"payment_terms_days": 100000},
		"unknown currency": {"currency": "XXXX"},
	} {
		rec := putDoc(t, routes.HandleUpdateInvoice, body)
		assert.Equal(t, http.StatusBadRequest, rec.Code, name)
	}
	assert.Nil(t, stub.gotInvoice, "invalid requests must not reach the biz service")
}

func TestHandleUpdateQuote_ForwardsCurrency(t *testing.T) {
	routes, stub := newBizUpdateStubRoutes(t)

	rec := putDoc(t, routes.HandleUpdateQuote, map[string]any{"currency": "USD", "exchange_rate": "1.08"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, stub.gotQuote)
	assert.Equal(t, "USD", stub.gotQuote.GetCurrency())

	rec = putDoc(t, routes.HandleUpdateQuote, map[string]any{"currency": "dollars"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
