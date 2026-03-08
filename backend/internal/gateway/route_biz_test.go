package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBizRoutes_ServiceName(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	if routes.ServiceName() != "biz" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "biz")
	}
}

// --- Invoices ---

func TestHandleCreateInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateInvoice)
}

func TestHandleCreateInvoice_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices", invalidJSON())
	routes.HandleCreateInvoice(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleListInvoices_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/invoices", nil)
	routes.HandleListInvoices(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/invoices/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Quotes ---

func TestHandleCreateQuote_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateQuote)
}

func TestHandleCreateQuote_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/quotes", invalidJSON())
	routes.HandleCreateQuote(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListQuotes_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/quotes", nil)
	routes.HandleListQuotes(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetQuote_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/quotes/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetQuote(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Credit Notes ---

func TestHandleCreateCreditNote_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateCreditNote)
}

func TestHandleCreateCreditNote_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/credit-notes", invalidJSON())
	routes.HandleCreateCreditNote(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListCreditNotes_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/credit-notes", nil)
	routes.HandleListCreditNotes(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Company Settings ---

func TestHandleGetCompanySettings_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/settings", nil)
	routes.HandleGetCompanySettings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateCompanySettings_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/settings", jsonBody(t, map[string]interface{}{}))
	routes.HandleUpdateCompanySettings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateCompanySettings_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/settings", invalidJSON())
	routes.HandleUpdateCompanySettings(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// --- Dunning ---

func TestHandleListDunnings_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/dunning", nil)
	routes.HandleListDunnings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
