package gateway

// route_biz_recurring_test.go covers the nine route_biz_recurring.go handlers
// (List, Create, Get, Update, Delete, Pause, Resume, Generate, and the shared
// setRecurringStatus helper reached through Pause/Resume). None of the nine
// had a test before this unit.
//
// This package has no bufconn stub for FinanceServiceClient (same boundary as
// every other gateway coverage unit this run: quotes, e-invoice, GoBD
// archive), so the *_ReachesRPC tests below only prove a handler with a valid,
// tenant-scoped request reaches the RPC layer (503 against the dummy
// localhost:0 address) — they cannot observe the exact request the handler
// built. Two things that matter are proven anyway, without needing a fake
// client:
//
//  1. Whether double-triggering HandleGenerateRecurringInvoice can emit two
//     invoices for the same period. It cannot: internal/biz/recurring
//     Service.Generate claims the period via a DB constraint before creating
//     the invoice and replays the existing one on a second call
//     (service.go:282, proven by TestGenerate_IsIdempotentPerPeriod). The
//     route is a thin pass-through with no additional idempotency logic of
//     its own to test.
//  2. Whether month-end intervals (31st in a 30-day month, Feb 29) land on
//     the right next run date. They already do:
//     TestNextRunFor_AnchorsAtStartAndClampsMonthEnd in
//     internal/biz/recurring/service_test.go. That is schedule-advancement
//     logic entirely inside the service, unreachable from the route layer
//     (the route only forwards start_date/interval strings), so it is not
//     duplicated here.
//
// What IS testable at the route layer without a fake client: the validation
// tags on createRecurringRequest/updateRecurringRequest. That includes an
// unexpected finding: the EndDate/ClearEndDate branch in
// HandleUpdateRecurringInvoice (an empty string is meant to clear the end
// date) turns out to be unreachable over HTTP — see the test below and
// fix-recurring-invoice-clear-end-date in BACKLOG.yml.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const testRecurringID = "660e8400-e29b-41d4-a716-446655440000"

func validCreateRecurringBody() map[string]interface{} {
	return map[string]interface{}{
		"customer":   map[string]interface{}{"name": "ACME"},
		"line_items": []interface{}{map[string]interface{}{"description": "x", "quantity": "1", "unit_price": "10.00", "tax_rate": "19.00"}},
		"tax_mode":   "standard",
		"interval":   "monthly",
		"start_date": "2026-01-01",
	}
}

// --- HandleListRecurringInvoices ---

func TestHandleListRecurringInvoices_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/recurring", nil)
	routes.HandleListRecurringInvoices(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListRecurringInvoices_ReachesRPCWithFilters(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/recurring?status=active&page=2&per_page=25", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListRecurringInvoices(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleCreateRecurringInvoice ---

func TestHandleCreateRecurringInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateRecurringInvoice)
}

func TestHandleCreateRecurringInvoice_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/recurring", jsonBody(t, validCreateRecurringBody()))
	routes.HandleCreateRecurringInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateRecurringInvoice_MissingLineItems(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := validCreateRecurringBody()
	body["line_items"] = []interface{}{}
	req := httptest.NewRequest("POST", "/api/v1/finance/recurring", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateRecurringInvoice(rec, req)
	assertValidationError(t, rec, "line_items")
}

func TestHandleCreateRecurringInvoice_InvalidInterval(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := validCreateRecurringBody()
	body["interval"] = "daily"
	req := httptest.NewRequest("POST", "/api/v1/finance/recurring", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateRecurringInvoice(rec, req)
	assertValidationError(t, rec, "interval")
}

func TestHandleCreateRecurringInvoice_InvalidStartDate(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := validCreateRecurringBody()
	body["start_date"] = "01/01/2026"
	req := httptest.NewRequest("POST", "/api/v1/finance/recurring", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateRecurringInvoice(rec, req)
	assertValidationError(t, rec, "start_date")
}

func TestHandleCreateRecurringInvoice_InvalidCustomerVAT(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := validCreateRecurringBody()
	// DE123456789 is structurally valid (DE + 9 digits) but has a wrong check
	// digit, same fixture as TestHandleCreateInvoice_InvalidCustomerVAT.
	body["customer"] = map[string]interface{}{"name": "ACME", "ust_id_nr": "DE123456789"}
	req := httptest.NewRequest("POST", "/api/v1/finance/recurring", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateRecurringInvoice(rec, req)
	assertValidationError(t, rec, "customer.ust_id_nr")
}

func TestHandleCreateRecurringInvoice_ValidRequestReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/recurring", jsonBody(t, validCreateRecurringBody()))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateRecurringInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetRecurringInvoice ---

func TestHandleGetRecurringInvoice_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/recurring/"+testRecurringID, nil)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandleGetRecurringInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetRecurringInvoice_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/recurring/"+testRecurringID, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandleGetRecurringInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateRecurringInvoice ---

func TestHandleUpdateRecurringInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdateRecurringInvoice)
}

func TestHandleUpdateRecurringInvoice_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/recurring/"+testRecurringID, jsonBody(t, map[string]interface{}{"notes": "x"}))
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandleUpdateRecurringInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateRecurringInvoice_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/recurring/"+testRecurringID, invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandleUpdateRecurringInvoice(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateRecurringInvoice_InvalidTaxMode(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/recurring/"+testRecurringID, jsonBody(t, map[string]interface{}{
		"tax_mode": "bogus",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandleUpdateRecurringInvoice(rec, req)
	assertValidationError(t, rec, "tax_mode")
}

func TestHandleUpdateRecurringInvoice_InvalidCustomerVAT(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/recurring/"+testRecurringID, jsonBody(t, map[string]interface{}{
		"customer": map[string]interface{}{"name": "ACME", "ust_id_nr": "DE123456789"},
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandleUpdateRecurringInvoice(rec, req)
	assertValidationError(t, rec, "customer.ust_id_nr")
}

func TestHandleUpdateRecurringInvoice_InvalidEndDateFormat(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/recurring/"+testRecurringID, jsonBody(t, map[string]interface{}{
		"end_date": "not-a-date",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandleUpdateRecurringInvoice(rec, req)
	assertValidationError(t, rec, "end_date")
}

// The handler's own logic reads an empty end_date as the sentinel for
// "clear the end date" (grpcReq.ClearEndDate, route_biz_recurring.go:179),
// but that branch is unreachable over HTTP: decodeAndValidate runs the
// omitempty,datetime tag first, and go-playground/validator's omitempty only
// skips a *string field when the pointer itself is nil — not when it points
// at an empty string. A present-but-empty end_date is therefore rejected
// with 400 before the handler ever sees it, and there is no other way for a
// JSON client to send "the pointer is non-nil and empty". This is a real,
// verified bug (the clear-the-end-date feature is dead code), not something
// a coverage unit is allowed to fix; see fix-recurring-invoice-clear-end-date
// at the end of BACKLOG.yml.
func TestHandleUpdateRecurringInvoice_EmptyEndDateRejectedByValidationBeforeClearLogicRuns(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/recurring/"+testRecurringID, jsonBody(t, map[string]interface{}{
		"end_date": "",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandleUpdateRecurringInvoice(rec, req)
	assertValidationError(t, rec, "end_date")
}

func TestHandleUpdateRecurringInvoice_ValidRequestReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/recurring/"+testRecurringID, jsonBody(t, map[string]interface{}{
		"notes":    "updated",
		"end_date": "2026-06-01",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandleUpdateRecurringInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteRecurringInvoice ---

func TestHandleDeleteRecurringInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleDeleteRecurringInvoice)
}

func TestHandleDeleteRecurringInvoice_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/finance/recurring/"+testRecurringID, nil)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandleDeleteRecurringInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteRecurringInvoice_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/finance/recurring/"+testRecurringID, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandleDeleteRecurringInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandlePauseRecurringInvoice / HandleResumeRecurringInvoice ---
//
// Both go through the shared setRecurringStatus helper; testing them
// separately exercises the helper via each of its two call sites.

func TestHandlePauseRecurringInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandlePauseRecurringInvoice)
}

func TestHandlePauseRecurringInvoice_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/recurring/"+testRecurringID+"/pause", nil)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandlePauseRecurringInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandlePauseRecurringInvoice_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/recurring/"+testRecurringID+"/pause", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandlePauseRecurringInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleResumeRecurringInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleResumeRecurringInvoice)
}

func TestHandleResumeRecurringInvoice_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/recurring/"+testRecurringID+"/resume", nil)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandleResumeRecurringInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleResumeRecurringInvoice_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/recurring/"+testRecurringID+"/resume", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandleResumeRecurringInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGenerateRecurringInvoice ---
//
// See file header: double-triggering this route cannot emit two invoices for
// the same period — that guarantee lives in Service.Generate's ClaimPeriod
// call, already proven by TestGenerate_IsIdempotentPerPeriod.

func TestHandleGenerateRecurringInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGenerateRecurringInvoice)
}

func TestHandleGenerateRecurringInvoice_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/recurring/"+testRecurringID+"/generate", nil)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandleGenerateRecurringInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGenerateRecurringInvoice_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/recurring/"+testRecurringID+"/generate", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testRecurringID)
	routes.HandleGenerateRecurringInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
