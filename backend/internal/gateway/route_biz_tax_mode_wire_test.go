package gateway

// tax_mode round trip for invoices, quotes, recurring templates and credit notes. The desktop
// edit dialogs copy tax_mode from the entity they loaded straight into the PUT body, so what a
// GET answers must be accepted by the matching update. A real gRPC server on loopback plays the
// biz service and records the tax mode that arrives.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

type taxModeStub struct {
	bizv1.UnimplementedFinanceServiceServer
	mode bizv1.TaxMode // returned by every read, recorded from every write
}

func (s *taxModeStub) GetInvoice(context.Context, *bizv1.GetInvoiceRequest) (*bizv1.GetInvoiceResponse, error) {
	return &bizv1.GetInvoiceResponse{Invoice: &bizv1.Invoice{Id: "inv", TaxMode: s.mode}}, nil
}

func (s *taxModeStub) ListInvoices(context.Context, *bizv1.ListInvoicesRequest) (*bizv1.ListInvoicesResponse, error) {
	return &bizv1.ListInvoicesResponse{Invoices: []*bizv1.Invoice{{Id: "inv", TaxMode: s.mode}}, Total: 1}, nil
}

func (s *taxModeStub) UpdateInvoice(_ context.Context, r *bizv1.UpdateInvoiceRequest) (*bizv1.UpdateInvoiceResponse, error) {
	s.mode = r.GetTaxMode()
	return &bizv1.UpdateInvoiceResponse{Invoice: &bizv1.Invoice{Id: r.GetId(), TaxMode: s.mode}}, nil
}

func (s *taxModeStub) GetQuote(context.Context, *bizv1.GetQuoteRequest) (*bizv1.GetQuoteResponse, error) {
	return &bizv1.GetQuoteResponse{Quote: &bizv1.Quote{Id: "q", TaxMode: s.mode}}, nil
}

func (s *taxModeStub) UpdateQuote(_ context.Context, r *bizv1.UpdateQuoteRequest) (*bizv1.UpdateQuoteResponse, error) {
	s.mode = r.GetTaxMode()
	return &bizv1.UpdateQuoteResponse{Quote: &bizv1.Quote{Id: r.GetId(), TaxMode: s.mode}}, nil
}

func (s *taxModeStub) ListRecurringInvoices(context.Context, *bizv1.ListRecurringInvoicesRequest) (*bizv1.ListRecurringInvoicesResponse, error) {
	return &bizv1.ListRecurringInvoicesResponse{Recurring: []*bizv1.RecurringInvoice{{Id: "r", TaxMode: s.mode}}, Total: 1}, nil
}

func (s *taxModeStub) UpdateRecurringInvoice(_ context.Context, r *bizv1.UpdateRecurringInvoiceRequest) (*bizv1.UpdateRecurringInvoiceResponse, error) {
	if r.TaxMode != nil {
		s.mode = r.GetTaxMode()
	}
	return &bizv1.UpdateRecurringInvoiceResponse{Recurring: &bizv1.RecurringInvoice{Id: r.GetId(), TaxMode: s.mode}}, nil
}

func (s *taxModeStub) CreateCreditNote(_ context.Context, r *bizv1.CreateCreditNoteRequest) (*bizv1.CreateCreditNoteResponse, error) {
	s.mode = r.GetTaxMode()
	return &bizv1.CreateCreditNoteResponse{CreditNote: &bizv1.CreditNote{Id: "cn", TaxMode: s.mode}}, nil
}

func (s *taxModeStub) ListCreditNotes(context.Context, *bizv1.ListCreditNotesRequest) (*bizv1.ListCreditNotesResponse, error) {
	return &bizv1.ListCreditNotesResponse{CreditNotes: []*bizv1.CreditNote{{Id: "cn", TaxMode: s.mode}}, Total: 1}, nil
}

func newTaxModeRoutes(t *testing.T) (*BizRoutes, *taxModeStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &taxModeStub{mode: bizv1.TaxMode_TAX_MODE_REVERSE_CHARGE}
	srv := grpc.NewServer()
	bizv1.RegisterFinanceServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("biz", lis.Addr().String())
	t.Cleanup(reg.Close)
	return NewBizRoutes(reg), stub
}

func taxModeRequest(method, body string) *http.Request {
	req := httptest.NewRequest(method, "/x/11111111-1111-1111-1111-111111111111", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenantID(req, testTenantID)
	return withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), rec.Body.String())
	return out
}

func TestTaxModeWire_InvoiceRoundTrip(t *testing.T) {
	routes, stub := newTaxModeRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleGetInvoice(rec, taxModeRequest(http.MethodGet, ""))
	assertStatus(t, rec, http.StatusOK)
	got := decodeBody(t, rec)
	require.Equal(t, "reverse_charge", got["tax_mode"])

	// What the edit dialog does: send the loaded value back unchanged.
	stub.mode = bizv1.TaxMode_TAX_MODE_UNSPECIFIED
	rec = httptest.NewRecorder()
	routes.HandleUpdateInvoice(rec, taxModeRequest(http.MethodPut, `{"tax_mode":"`+got["tax_mode"].(string)+`"}`))
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, bizv1.TaxMode_TAX_MODE_REVERSE_CHARGE, stub.mode)
	assert.Equal(t, "reverse_charge", decodeBody(t, rec)["tax_mode"])
}

func TestTaxModeWire_InvoiceUpdateStillAcceptsNumber(t *testing.T) {
	routes, stub := newTaxModeRoutes(t)
	for body, want := range map[string]bizv1.TaxMode{
		`{"tax_mode":1}`:          bizv1.TaxMode_TAX_MODE_STANDARD,
		`{"tax_mode":2}`:          bizv1.TaxMode_TAX_MODE_REVERSE_CHARGE,
		`{"tax_mode":3}`:          bizv1.TaxMode_TAX_MODE_KLEINUNTERNEHMER,
		`{"tax_mode":"standard"}`: bizv1.TaxMode_TAX_MODE_STANDARD,
		`{"tax_mode":null}`:       bizv1.TaxMode_TAX_MODE_UNSPECIFIED,
		`{"notes":"only notes"}`:  bizv1.TaxMode_TAX_MODE_UNSPECIFIED,
	} {
		// Start from a mode that differs from the expected one so a no-op cannot pass.
		stub.mode = bizv1.TaxMode_TAX_MODE_KLEINUNTERNEHMER
		if want == bizv1.TaxMode_TAX_MODE_KLEINUNTERNEHMER {
			stub.mode = bizv1.TaxMode_TAX_MODE_STANDARD
		}
		rec := httptest.NewRecorder()
		routes.HandleUpdateInvoice(rec, taxModeRequest(http.MethodPut, body))
		assertStatus(t, rec, http.StatusOK)
		assert.Equal(t, want, stub.mode, body)
	}
}

func TestTaxModeWire_InvalidValuesStay400(t *testing.T) {
	routes, _ := newTaxModeRoutes(t)
	for _, body := range []string{`{"tax_mode":"bogus"}`, `{"tax_mode":9}`, `{"tax_mode":-1}`, `{"tax_mode":true}`} {
		rec := httptest.NewRecorder()
		routes.HandleUpdateInvoice(rec, taxModeRequest(http.MethodPut, body))
		assertStatus(t, rec, http.StatusBadRequest)
	}
}

func TestTaxModeWire_InvoiceListUsesString(t *testing.T) {
	routes, _ := newTaxModeRoutes(t)
	rec := httptest.NewRecorder()
	routes.HandleListInvoices(rec, taxModeRequest(http.MethodGet, ""))
	assertStatus(t, rec, http.StatusOK)
	items := decodeBody(t, rec)["invoices"].([]any)
	require.Len(t, items, 1)
	assert.Equal(t, "reverse_charge", items[0].(map[string]any)["tax_mode"])
}

func TestTaxModeWire_QuoteRoundTrip(t *testing.T) {
	routes, stub := newTaxModeRoutes(t)
	rec := httptest.NewRecorder()
	routes.HandleGetQuote(rec, taxModeRequest(http.MethodGet, ""))
	assertStatus(t, rec, http.StatusOK)
	require.Equal(t, "reverse_charge", decodeBody(t, rec)["tax_mode"])

	stub.mode = bizv1.TaxMode_TAX_MODE_UNSPECIFIED
	rec = httptest.NewRecorder()
	routes.HandleUpdateQuote(rec, taxModeRequest(http.MethodPut, `{"tax_mode":"reverse_charge"}`))
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, bizv1.TaxMode_TAX_MODE_REVERSE_CHARGE, stub.mode)

	stub.mode = bizv1.TaxMode_TAX_MODE_UNSPECIFIED
	rec = httptest.NewRecorder()
	routes.HandleUpdateQuote(rec, taxModeRequest(http.MethodPut, `{"tax_mode":3}`))
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, bizv1.TaxMode_TAX_MODE_KLEINUNTERNEHMER, stub.mode)
}

func TestTaxModeWire_RecurringListAndUpdate(t *testing.T) {
	routes, stub := newTaxModeRoutes(t)
	rec := httptest.NewRecorder()
	routes.HandleListRecurringInvoices(rec, taxModeRequest(http.MethodGet, ""))
	assertStatus(t, rec, http.StatusOK)
	items := decodeBody(t, rec)["recurring"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, "reverse_charge", items[0].(map[string]any)["tax_mode"])

	stub.mode = bizv1.TaxMode_TAX_MODE_STANDARD
	rec = httptest.NewRecorder()
	routes.HandleUpdateRecurringInvoice(rec, taxModeRequest(http.MethodPut, `{"tax_mode":"reverse_charge"}`))
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, bizv1.TaxMode_TAX_MODE_REVERSE_CHARGE, stub.mode)

	stub.mode = bizv1.TaxMode_TAX_MODE_STANDARD
	rec = httptest.NewRecorder()
	routes.HandleUpdateRecurringInvoice(rec, taxModeRequest(http.MethodPut, `{"tax_mode":2}`))
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, bizv1.TaxMode_TAX_MODE_REVERSE_CHARGE, stub.mode)

	// An omitted tax_mode must not touch the stored mode.
	stub.mode = bizv1.TaxMode_TAX_MODE_KLEINUNTERNEHMER
	rec = httptest.NewRecorder()
	routes.HandleUpdateRecurringInvoice(rec, taxModeRequest(http.MethodPut, `{"title":"x"}`))
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, bizv1.TaxMode_TAX_MODE_KLEINUNTERNEHMER, stub.mode)
}

func TestTaxModeWire_CreditNoteListAndCreate(t *testing.T) {
	routes, stub := newTaxModeRoutes(t)
	rec := httptest.NewRecorder()
	routes.HandleListCreditNotes(rec, taxModeRequest(http.MethodGet, ""))
	assertStatus(t, rec, http.StatusOK)
	items := decodeBody(t, rec)["credit_notes"].([]any)
	require.Len(t, items, 1)
	assert.Equal(t, "reverse_charge", items[0].(map[string]any)["tax_mode"])

	stub.mode = bizv1.TaxMode_TAX_MODE_UNSPECIFIED
	rec = httptest.NewRecorder()
	routes.HandleCreateCreditNote(rec, taxModeRequest(http.MethodPost,
		`{"customer":{"name":"A"},"line_items":[{"description":"x"}],"tax_mode":2,"reason":"r"}`))
	assertStatus(t, rec, http.StatusCreated)
	assert.Equal(t, bizv1.TaxMode_TAX_MODE_REVERSE_CHARGE, stub.mode)
}
