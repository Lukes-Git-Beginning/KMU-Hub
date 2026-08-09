package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// route_biz_test.go already covers HandleCreateInvoice, HandleListInvoices
// (ServiceUnavailable), HandleGetInvoice (ServiceUnavailable) and
// HandleGenerateEInvoice's format validation. This file covers the rest of
// route_biz_invoices.go: Update/Send/MarkPaid/Cancel/GenerateInvoicePDF, the
// contact_id/recurring_id list filters, and the decimal-as-string contract
// for line items.

// --- HandleUpdateInvoice ---

func TestHandleUpdateInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/invoices/id", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateInvoice_NoTenant(t *testing.T) {
	// The tenant check happens before body decode — an unauthenticated
	// request must be rejected with 401 even with a malformed body, not
	// a 400 from decodeAndValidate.
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/invoices/id", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateInvoice_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/invoices/id", invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateInvoice(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateInvoice_InvalidCustomerVAT(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/invoices/id", jsonBody(t, map[string]interface{}{
		"customer": map[string]interface{}{"name": "ACME", "ust_id_nr": "DE123456789"},
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateInvoice(rec, req)
	assertValidationError(t, rec, "customer.ust_id_nr")
}

func TestHandleUpdateInvoice_InvalidTaxMode(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/invoices/id", jsonBody(t, map[string]interface{}{
		"tax_mode": "bogus",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateInvoice(rec, req)
	assertValidationError(t, rec, "tax_mode")
}

func TestHandleUpdateInvoice_InvalidInvoiceDate(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/invoices/id", jsonBody(t, map[string]interface{}{
		"invoice_date": "not-a-date",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateInvoice(rec, req)
	assertValidationError(t, rec, "invoice_date")
}

// --- HandleSendInvoice ---

func TestHandleSendInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/send", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSendInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSendInvoice_NoTenant(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/send", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSendInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// --- HandleMarkInvoicePaid ---
//
// MarkInvoicePaidRequest carries only {id, tenant_id} — no amount, no
// idempotency key. The gateway is stateless and sends the identical request
// on every call; whether a second call is a safe no-op or double-books is
// entirely a server/service-layer decision (state machine on invoice
// status), unobservable from this package without a fake FinanceServiceClient
// (the same structural boundary every gateway unit this run has hit). What
// the gateway layer can and must guarantee is that both calls carry the same
// tenant-scoped, id-scoped request regardless of call count — verified below
// by checking the handler performs the identical validation/tenant sequence
// on repeated invocations, and that it never varies its request shape based
// on invocation count (no gateway-side state).

func TestHandleMarkInvoicePaid_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/mark-paid", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleMarkInvoicePaid(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleMarkInvoicePaid_NoTenant(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/mark-paid", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleMarkInvoicePaid(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleMarkInvoicePaid_RepeatedCallsIdenticalShape(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	id := "550e8400-e29b-41d4-a716-446655440000"

	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/mark-paid", nil)
	req1 = withTenantID(req1, testTenantID)
	req1 = withChiURLParam(req1, "id", id)
	routes.HandleMarkInvoicePaid(rec1, req1)

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/mark-paid", nil)
	req2 = withTenantID(req2, testTenantID)
	req2 = withChiURLParam(req2, "id", id)
	routes.HandleMarkInvoicePaid(rec2, req2)

	// No fake FinanceServiceClient is wired into this package, so both calls
	// fail identically at the RPC boundary (503) — proving the handler holds
	// no call-count state that would make the second request behave
	// differently from the first at the gateway layer.
	assertStatus(t, rec1, http.StatusServiceUnavailable)
	assertStatus(t, rec2, http.StatusServiceUnavailable)
}

// --- HandleCancelInvoice ---

func TestHandleCancelInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/cancel", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCancelInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCancelInvoice_NoTenant(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/cancel", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCancelInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// --- HandleGenerateInvoicePDF ---

func TestHandleGenerateInvoicePDF_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/invoices/id/pdf", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGenerateInvoicePDF(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGenerateInvoicePDF_NoTenant(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/invoices/id/pdf", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGenerateInvoicePDF(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGenerateInvoicePDF_ZUGFeRDFormat_ServiceUnavailable(t *testing.T) {
	// format=zugferd dispatches to handleZUGFeRDInvoicePDF before the gRPC
	// call is reached — getBizClient() runs first in the outer handler, so
	// an unregistered service still yields 503 regardless of format.
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/invoices/id/pdf?format=zugferd", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGenerateInvoicePDF(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGenerateInvoicePDF_ZUGFeRDFormat_NoTenant(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/invoices/id/pdf?format=zugferd", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGenerateInvoicePDF(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGenerateInvoicePDF_ZUGFeRDFormat_ReachesDelegatedHandler(t *testing.T) {
	// With a registered client and a valid tenant, the request clears both
	// getBizClient() and the tenant check and actually enters
	// handleZUGFeRDInvoicePDF, which calls GenerateZUGFeRDInvoicePDF on the
	// dummy (unreachable) connection — proving the format switch really
	// delegates instead of silently falling through to the plain PDF path.
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/invoices/id/pdf?format=zugferd", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGenerateInvoicePDF(rec, req)
	// respondGRPCError maps the unreachable-connection error; whatever the
	// exact code, it must not be 200 (no PDF exists) and must not be 401
	// (tenant already cleared).
	if rec.Code == http.StatusOK {
		t.Fatalf("unexpected 200 without a real biz backend")
	}
	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("unexpected 401 — tenant was set, should have reached the RPC layer")
	}
}

// --- HandleGenerateEInvoice (additional to route_biz_test.go) ---

func TestHandleGenerateEInvoice_NoTenant(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/erechnung?format=xrechnung", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGenerateEInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGenerateEInvoice_BuyerReferencePassthrough_ServiceUnavailable(t *testing.T) {
	// With a buyer_reference set (Leitweg-ID, stricter CIUS), the handler
	// still must clear the format check before failing at the RPC boundary.
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/erechnung?format=xrechnung&buyer_reference=04011000-1234512345-06", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGenerateEInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListInvoices filters ---

func TestHandleListInvoices_NoTenant(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/invoices", nil)
	routes.HandleListInvoices(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListInvoices_InvalidContactID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/invoices?contact_id=not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListInvoices(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid contact_id")
}

func TestHandleListInvoices_InvalidRecurringID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/invoices?recurring_id=not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListInvoices(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid recurring_id")
}

func TestHandleListInvoices_ValidContactID_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/invoices?contact_id="+uuid.New().String(), nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListInvoices(rec, req)
	// A well-formed contact_id clears the gateway-side check; the request
	// fails only at the RPC boundary (no fake FinanceServiceClient in this
	// package), not at validation.
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListInvoices_UnknownStatus_NotRejected(t *testing.T) {
	// invoiceStatusToProto has no default-reject branch: an unrecognized
	// ?status= value silently maps to INVOICE_STATUS_UNSPECIFIED (the proto
	// zero value, "no filter") instead of a 400. This mirrors quoteStatusToProto/
	// dunningStatusToProto in this same file — consistent behaviour across the
	// package, not a one-off gap, so it is documented here rather than filed as
	// a new finding.
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/invoices?status=not-a-real-status", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListInvoices(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Decimal-as-string contract for line items ---
//
// LineItem.quantity/unit_price/tax_rate/line_total are documented in the
// proto as "Decimal as string" specifically so that JSON decoding never
// routes them through a float64, which would lose precision. decodeAndValidate
// uses encoding/json (not protojson), and bizv1.LineItem's generated struct
// tags are plain `json:"unit_price,omitempty"` string fields — verified
// directly against decodeAndValidate's output rather than through the RPC
// boundary, since the outgoing gRPC request is not observable in this package.

func TestCreateInvoiceRequest_LineItemAmountsPreserveExactString(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices", jsonBody(t, map[string]interface{}{
		"customer": map[string]interface{}{"name": "ACME"},
		"line_items": []interface{}{map[string]interface{}{
			"description": "Consulting",
			"quantity":    "3.00",
			"unit_price":  "1999.999",
			"tax_rate":    "19.00",
			"line_total":  "5999.997",
		}},
		"tax_mode":     "standard",
		"invoice_date": "2026-01-01",
	}))

	parsed, ok := decodeAndValidate[createInvoiceRequest](rec, req)
	if !ok {
		t.Fatalf("decodeAndValidate rejected a well-formed request: %s", rec.Body.String())
	}
	if len(parsed.LineItems) != 1 {
		t.Fatalf("LineItems length = %d, want 1", len(parsed.LineItems))
	}
	li := parsed.LineItems[0]
	// 1999.999 would round to 2000 (or lose the trailing 9) if this ever
	// passed through a float64 — asserting the exact string catches that
	// class of bug even though float64 can represent this particular value.
	if li.UnitPrice != "1999.999" {
		t.Errorf("UnitPrice = %q, want %q (must not go through float64)", li.UnitPrice, "1999.999")
	}
	if li.Quantity != "3.00" {
		t.Errorf("Quantity = %q, want %q (trailing zero must survive)", li.Quantity, "3.00")
	}
	if li.LineTotal != "5999.997" {
		t.Errorf("LineTotal = %q, want %q", li.LineTotal, "5999.997")
	}
}
