package gateway

// route_biz_ext_test.go covers route_biz_ext.go, the sole remaining untested
// route_biz_*.go file in this neighbourhood: HandleCreateInvoiceFromTime,
// the thin gRPC proxy that turns aggregated HR time entries into a draft
// invoice. NewBizExtRoutes and getBizClient are exercised implicitly by
// every test below; registerTimeExtRoutes (route wiring into the /hr/time
// subrouter) is already exercised by openapi_drift_test.go, which builds
// the full router including BizExtRoutes.
//
// Same infrastructure boundary as every other gateway coverage unit this
// run (recurring invoices, quotes, e-invoice, GoBD archive, banking):
// there is no bufconn stub for FinanceServiceClient in this package, so
// *_ReachesRPC below only proves a valid request reaches the RPC layer
// (503 against the dummy localhost:0 address) — it cannot observe what a
// successful RPC would return.
//
// DOUBLE-BILLING FINDING (root-caused, not fixed here — this is a
// coverage-only unit and the fix sits one layer down, in
// internal/server/biz_grpc.go and internal/biz/hr/timetracking, not in
// this thin gateway proxy): HandleCreateInvoiceFromTime forwards
// employee_id/date_from/date_to verbatim to
// BizGRPCServer.CreateInvoiceFromTimeEntries (biz_grpc.go:1981), which
// aggregates time via
// PostgresWorkTimeRepo.AggregateWorkTimeForInvoice
// (postgres_repository.go:532). That query filters only on tenant,
// employee, status and the clock_in window — nothing marks an
// hr_work_time_entries row as billed, and nothing excludes rows already
// referenced by a prior invoice's time_tracking_source (biz_grpc.go:2101,
// LinkTimeTracking, is write-only best-effort audit trail, never read
// back). Calling this endpoint twice with the same employee/date range
// therefore creates two separate draft invoices billing the identical
// hours. Confirmed at the schema level too: hr_work_time_entries
// (migrations/000046_create_hr_tables.up.sql:120) has no billed/invoice_id
// column at all — unlike internal/work/timeentry, which does track
// `billed` for a different table entirely (route_biz_time_entries.go).
// Not reproducible as a gateway-level test: the gateway has no
// bufconn-backed FinanceServiceClient to drive two real RPCs against, and
// the handler itself has no idempotency logic to assert on (it is a pure
// passthrough). Filed as fix-biz-time-entry-invoice-double-billing at the
// end of BACKLOG.yml per the "found a gap one layer down -> new unit, not
// a test wrapped around it" rule.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func validCreateInvoiceFromTimeBody() map[string]interface{} {
	return map[string]interface{}{
		"employee_id":   "550e8400-e29b-41d4-a716-446655440000",
		"customer_name": "ACME GmbH",
		"date_from":     "2026-08-01",
		"date_to":       "2026-08-31",
		"hourly_rate":   "85.00",
		"tax_mode":      "standard",
	}
}

// HandleCreateInvoiceFromTime validates the request body before reaching
// the gRPC client (tenant check, then decodeAndValidate, then
// getBizClient) — the reverse order of most other biz handlers, which
// check the client first. The generic testServiceUnavailable helper sends
// an empty "{}" body and therefore does not apply here: an empty body
// fails validation with 400 before the missing-service 503 is ever
// reached, so this test supplies a valid body explicitly.
func TestHandleCreateInvoiceFromTime_ServiceUnavailable(t *testing.T) {
	routes := NewBizExtRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", jsonBody(t, validCreateInvoiceFromTimeBody()))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateInvoiceFromTime_NoTenantID(t *testing.T) {
	routes := NewBizExtRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", jsonBody(t, validCreateInvoiceFromTimeBody()))
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateInvoiceFromTime_InvalidJSON(t *testing.T) {
	routes := NewBizExtRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateInvoiceFromTime_MissingEmployeeID(t *testing.T) {
	routes := NewBizExtRoutes(registryWithService("biz"))
	body := validCreateInvoiceFromTimeBody()
	delete(body, "employee_id")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertValidationError(t, rec, "employee_id")
}

func TestHandleCreateInvoiceFromTime_InvalidEmployeeIDNotUUID(t *testing.T) {
	routes := NewBizExtRoutes(registryWithService("biz"))
	body := validCreateInvoiceFromTimeBody()
	body["employee_id"] = "not-a-uuid"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertValidationError(t, rec, "employee_id")
}

func TestHandleCreateInvoiceFromTime_MissingCustomerName(t *testing.T) {
	routes := NewBizExtRoutes(registryWithService("biz"))
	body := validCreateInvoiceFromTimeBody()
	delete(body, "customer_name")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertValidationError(t, rec, "customer_name")
}

func TestHandleCreateInvoiceFromTime_InvalidCustomerEmail(t *testing.T) {
	routes := NewBizExtRoutes(registryWithService("biz"))
	body := validCreateInvoiceFromTimeBody()
	body["customer_email"] = "not-an-email"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertValidationError(t, rec, "customer_email")
}

func TestHandleCreateInvoiceFromTime_InvalidDateFromFormat(t *testing.T) {
	routes := NewBizExtRoutes(registryWithService("biz"))
	body := validCreateInvoiceFromTimeBody()
	body["date_from"] = "01.08.2026"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertValidationError(t, rec, "date_from")
}

func TestHandleCreateInvoiceFromTime_MissingDateTo(t *testing.T) {
	routes := NewBizExtRoutes(registryWithService("biz"))
	body := validCreateInvoiceFromTimeBody()
	delete(body, "date_to")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertValidationError(t, rec, "date_to")
}

func TestHandleCreateInvoiceFromTime_MissingHourlyRate(t *testing.T) {
	routes := NewBizExtRoutes(registryWithService("biz"))
	body := validCreateInvoiceFromTimeBody()
	delete(body, "hourly_rate")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertValidationError(t, rec, "hourly_rate")
}

func TestHandleCreateInvoiceFromTime_ZeroHourlyRateRejected(t *testing.T) {
	routes := NewBizExtRoutes(registryWithService("biz"))
	body := validCreateInvoiceFromTimeBody()
	body["hourly_rate"] = "0"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertValidationError(t, rec, "hourly_rate")
}

func TestHandleCreateInvoiceFromTime_NegativeHourlyRateRejected(t *testing.T) {
	routes := NewBizExtRoutes(registryWithService("biz"))
	body := validCreateInvoiceFromTimeBody()
	body["hourly_rate"] = "-10.00"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertValidationError(t, rec, "hourly_rate")
}

func TestHandleCreateInvoiceFromTime_NonNumericHourlyRateRejected(t *testing.T) {
	routes := NewBizExtRoutes(registryWithService("biz"))
	body := validCreateInvoiceFromTimeBody()
	body["hourly_rate"] = "eighty-five"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertValidationError(t, rec, "hourly_rate")
}

func TestHandleCreateInvoiceFromTime_InvalidTaxMode(t *testing.T) {
	routes := NewBizExtRoutes(registryWithService("biz"))
	body := validCreateInvoiceFromTimeBody()
	body["tax_mode"] = "invalid_mode"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertValidationError(t, rec, "tax_mode")
}

func TestHandleCreateInvoiceFromTime_EmptyTaxModeAllowed(t *testing.T) {
	// tax_mode is validate:"omitempty,oneof=..." — the service derives it
	// from company settings (Kleinunternehmer check) when absent, so an
	// empty tax_mode must pass gateway-side validation and reach the RPC.
	routes := NewBizExtRoutes(registryWithService("biz"))
	body := validCreateInvoiceFromTimeBody()
	delete(body, "tax_mode")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", jsonBody(t, body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateInvoiceFromTime_ValidRequestReachesRPC(t *testing.T) {
	routes := NewBizExtRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/hr/time/create-invoice", jsonBody(t, validCreateInvoiceFromTimeBody()))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInvoiceFromTime(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
