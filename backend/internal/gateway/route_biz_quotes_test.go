package gateway

// route_biz_quotes_test.go covers the eight route_biz_quotes.go handlers that
// route_biz_test.go does not touch (Update, Delete, Send, Accept, Reject,
// Convert-to-invoice, PDF generation, deal-to-quote) plus the remaining gaps
// on Create/List/Get (missing tenant, empty line items).
//
// Permission-guard wiring for the whole quotes group — including the additive
// legacy-vs-catalogue-key rollout (finance:write vs finance:quote:create,
// finance:quote:send, finance:invoice:create for convert, the delete-only
// finance:delete key) — is already exhaustively tested in
// route_capability_guard_test.go ("--- finance: quotes ---"); it is not
// duplicated here.
//
// Send/Accept/Reject/Convert forward straight to the RPC and let the quote
// service enforce the draft->sent->accepted/rejected transition, returning
// FailedPrecondition (-> 409 via grpcStatusToHTTP) for an invalid one. There
// is no bufconn stub for FinanceServiceClient in this package (same boundary
// as every other gateway coverage unit this run), so the *_ReachesRPC tests
// below only prove the handler reaches the RPC layer with a valid quote ID.
// The transition rejection itself is proven at two lower layers instead:
// internal/biz/quote/service_test.go (TestService_Accept_RejectsNonSent,
// TestService_Reject_RejectsNonSent, TestService_Send_RejectsNonDraft, ...)
// and internal/server/biz_grpc_errormap_settings_quotes_test.go, which pins
// quote.ErrQuoteNotDraft/ErrQuoteNotSent to codes.FailedPrecondition. The
// FailedPrecondition->409 HTTP mapping itself is proven generically in
// helpers_test.go (TestGrpcStatusToHTTP).

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const testQuoteID = "550e8400-e29b-41d4-a716-446655440000"

// --- HandleCreateQuote (gaps not covered by route_biz_test.go) ---

func TestHandleCreateQuote_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes", jsonBody(t, map[string]interface{}{
		"customer":   map[string]interface{}{"name": "ACME"},
		"line_items": []interface{}{map[string]interface{}{"description": "x"}},
		"tax_mode":   "standard",
	}))
	routes.HandleCreateQuote(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateQuote_MissingLineItems(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes", jsonBody(t, map[string]interface{}{
		"customer":   map[string]interface{}{"name": "ACME"},
		"line_items": []interface{}{},
		"tax_mode":   "standard",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateQuote(rec, req)
	assertValidationError(t, rec, "line_items")
}

func TestHandleCreateQuote_InvalidCustomerVAT(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	// DE123456789 is structurally valid (DE + 9 digits) but has a wrong check
	// digit, so it must be rejected before the gRPC call — same fixture as
	// TestHandleCreateInvoice_InvalidCustomerVAT.
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes", jsonBody(t, map[string]interface{}{
		"customer":   map[string]interface{}{"name": "ACME", "ust_id_nr": "DE123456789"},
		"line_items": []interface{}{map[string]interface{}{"description": "x"}},
		"tax_mode":   "standard",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateQuote(rec, req)
	assertValidationError(t, rec, "customer.ust_id_nr")
}

func TestHandleCreateQuote_ValidRequestReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes", jsonBody(t, map[string]interface{}{
		"customer":   map[string]interface{}{"name": "ACME"},
		"line_items": []interface{}{map[string]interface{}{"description": "x", "quantity": "1", "unit_price": "10.00", "tax_rate": "19.00"}},
		"tax_mode":   "standard",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateQuote(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListQuotes (gap: no tenant) ---

func TestHandleListQuotes_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/quotes", nil)
	routes.HandleListQuotes(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListQuotes_ReachesRPCWithFilters(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/quotes?status=sent&deal_id=550e8400-e29b-41d4-a716-446655440000&page=2&per_page=25", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListQuotes(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetQuote (gap: no tenant, RPC reach) ---

func TestHandleGetQuote_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/quotes/"+testQuoteID, nil)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleGetQuote(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetQuote_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/quotes/"+testQuoteID, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleGetQuote(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateQuote ---

func TestHandleUpdateQuote_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdateQuote)
}

func TestHandleUpdateQuote_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/quotes/"+testQuoteID, jsonBody(t, map[string]interface{}{"notes": "x"}))
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleUpdateQuote(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateQuote_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/quotes/"+testQuoteID, invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleUpdateQuote(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateQuote_InvalidTaxMode(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/quotes/"+testQuoteID, jsonBody(t, map[string]interface{}{
		"tax_mode": "bogus",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleUpdateQuote(rec, req)
	assertValidationError(t, rec, "tax_mode")
}

func TestHandleUpdateQuote_InvalidCustomerVAT(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/quotes/"+testQuoteID, jsonBody(t, map[string]interface{}{
		"customer": map[string]interface{}{"name": "ACME", "ust_id_nr": "DE123456789"},
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleUpdateQuote(rec, req)
	assertValidationError(t, rec, "customer.ust_id_nr")
}

func TestHandleUpdateQuote_ValidRequestReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/quotes/"+testQuoteID, jsonBody(t, map[string]interface{}{
		"notes": "updated",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleUpdateQuote(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteQuote ---

func TestHandleDeleteQuote_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleDeleteQuote)
}

func TestHandleDeleteQuote_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/finance/quotes/"+testQuoteID, nil)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleDeleteQuote(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteQuote_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/finance/quotes/"+testQuoteID, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleDeleteQuote(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleSendQuote ---
//
// draft->sent transition; see file header for why the 409 rejection itself
// is proven at the service/gRPC layers, not here.

func TestHandleSendQuote_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleSendQuote)
}

func TestHandleSendQuote_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes/"+testQuoteID+"/send", nil)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleSendQuote(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleSendQuote_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes/"+testQuoteID+"/send", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleSendQuote(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleAcceptQuote ---
//
// sent->accepted transition; see file header.

func TestHandleAcceptQuote_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleAcceptQuote)
}

func TestHandleAcceptQuote_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes/"+testQuoteID+"/accept", nil)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleAcceptQuote(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleAcceptQuote_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes/"+testQuoteID+"/accept", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleAcceptQuote(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleRejectQuote ---
//
// sent->rejected transition; see file header.

func TestHandleRejectQuote_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleRejectQuote)
}

func TestHandleRejectQuote_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes/"+testQuoteID+"/reject", nil)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleRejectQuote(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleRejectQuote_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes/"+testQuoteID+"/reject", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleRejectQuote(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleConvertQuoteToInvoice ---
//
// accepted->invoice transition, enforced server-side the same way; see file
// header for why the 409 rejection path is proven at the lower layers.

func TestHandleConvertQuoteToInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleConvertQuoteToInvoice)
}

func TestHandleConvertQuoteToInvoice_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes/"+testQuoteID+"/convert", jsonBody(t, map[string]interface{}{
		"invoice_date": "2026-01-01",
	}))
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleConvertQuoteToInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleConvertQuoteToInvoice_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes/"+testQuoteID+"/convert", invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleConvertQuoteToInvoice(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleConvertQuoteToInvoice_MissingInvoiceDate(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes/"+testQuoteID+"/convert", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleConvertQuoteToInvoice(rec, req)
	assertValidationError(t, rec, "invoice_date")
}

func TestHandleConvertQuoteToInvoice_InvalidDueDate(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes/"+testQuoteID+"/convert", jsonBody(t, map[string]interface{}{
		"invoice_date": "2026-01-01",
		"due_date":     "not-a-date",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleConvertQuoteToInvoice(rec, req)
	assertValidationError(t, rec, "due_date")
}

func TestHandleConvertQuoteToInvoice_ValidRequestReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes/"+testQuoteID+"/convert", jsonBody(t, map[string]interface{}{
		"invoice_date":  "2026-01-01",
		"payment_terms": "30 Tage netto",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleConvertQuoteToInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGenerateQuotePDF ---

func TestHandleGenerateQuotePDF_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGenerateQuotePDF)
}

func TestHandleGenerateQuotePDF_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/quotes/"+testQuoteID+"/pdf", nil)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleGenerateQuotePDF(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGenerateQuotePDF_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/quotes/"+testQuoteID+"/pdf", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleGenerateQuotePDF(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleCreateQuoteFromDeal ---

func TestHandleCreateQuoteFromDeal_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/deals/"+testQuoteID+"/quote", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "dealId", testQuoteID)
	routes.HandleCreateQuoteFromDeal(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateQuoteFromDeal_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/deals/"+testQuoteID+"/quote", nil)
	req = withChiURLParam(req, "dealId", testQuoteID)
	routes.HandleCreateQuoteFromDeal(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateQuoteFromDeal_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/deals/"+testQuoteID+"/quote", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "dealId", testQuoteID)
	routes.HandleCreateQuoteFromDeal(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
