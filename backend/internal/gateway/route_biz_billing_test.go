package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ============================================================================
// ServiceUnavailable — every handler in route_biz_billing.go calls
// getBizClient() before doing anything else, so a single generic request
// proves the 503 path for all 24 of them at once (route_biz_test.go already
// covers this individually for the handful defined before this unit).
// ============================================================================

func TestBizBillingRoutes_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())

	handlers := map[string]http.HandlerFunc{
		"HandleGetCreditNote":         routes.HandleGetCreditNote,
		"HandleSendCreditNote":        routes.HandleSendCreditNote,
		"HandleGenerateCreditNotePDF": routes.HandleGenerateCreditNotePDF,
		"HandleRecordPayment":         routes.HandleRecordPayment,
		"HandleListPayments":          routes.HandleListPayments,
		"HandleDeletePayment":         routes.HandleDeletePayment,
		"HandleCreateDunning":         routes.HandleCreateDunning,
		"HandleSendDunning":           routes.HandleSendDunning,
		"HandleEscalateDunning":       routes.HandleEscalateDunning,
		"HandleGenerateDunningPDF":    routes.HandleGenerateDunningPDF,
		"HandleGetDunningConfig":      routes.HandleGetDunningConfig,
		"HandleUpdateDunningConfig":   routes.HandleUpdateDunningConfig,
		"HandleGetFinanceDashboard":   routes.HandleGetFinanceDashboard,
		"HandleExportDATEV":           routes.HandleExportDATEV,
		"HandleGetJournalSummary":     routes.HandleGetJournalSummary,
		"HandleValidateInvoiceNumber": routes.HandleValidateInvoiceNumber,
		"HandleLockInvoice":           routes.HandleLockInvoice,
		"HandleGetPaymentStats":       routes.HandleGetPaymentStats,
		"HandleUpdateDunningStatus":   routes.HandleUpdateDunningStatus,
		"HandleSendDunningNotice":     routes.HandleSendDunningNotice,
		"HandleGenerateGoBDExport":    routes.HandleGenerateGoBDExport,
	}

	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			testServiceUnavailable(t, h)
		})
	}
}

// ============================================================================
// NoTenant — the client check runs before the tenant check in every handler
// here (unlike route_crm_activities.go, where the order flips per handler),
// so registering the service and omitting the tenant is enough to prove the
// 401 path without needing a valid body or URL param for any of them.
// ============================================================================

func TestBizBillingRoutes_NoTenant(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))

	handlers := map[string]http.HandlerFunc{
		"HandleGetCreditNote":         routes.HandleGetCreditNote,
		"HandleSendCreditNote":        routes.HandleSendCreditNote,
		"HandleGenerateCreditNotePDF": routes.HandleGenerateCreditNotePDF,
		"HandleListCreditNotes":       routes.HandleListCreditNotes,
		"HandleCreateCreditNote":      routes.HandleCreateCreditNote,
		"HandleRecordPayment":         routes.HandleRecordPayment,
		"HandleListPayments":          routes.HandleListPayments,
		"HandleDeletePayment":         routes.HandleDeletePayment,
		"HandleListDunnings":          routes.HandleListDunnings,
		"HandleCreateDunning":         routes.HandleCreateDunning,
		"HandleSendDunning":           routes.HandleSendDunning,
		"HandleEscalateDunning":       routes.HandleEscalateDunning,
		"HandleGenerateDunningPDF":    routes.HandleGenerateDunningPDF,
		"HandleGetDunningConfig":      routes.HandleGetDunningConfig,
		"HandleUpdateDunningConfig":   routes.HandleUpdateDunningConfig,
		"HandleGetFinanceDashboard":   routes.HandleGetFinanceDashboard,
		"HandleExportDATEV":           routes.HandleExportDATEV,
		"HandleGetJournalSummary":     routes.HandleGetJournalSummary,
		"HandleValidateInvoiceNumber": routes.HandleValidateInvoiceNumber,
		"HandleLockInvoice":           routes.HandleLockInvoice,
		"HandleGetPaymentStats":       routes.HandleGetPaymentStats,
		"HandleUpdateDunningStatus":   routes.HandleUpdateDunningStatus,
		"HandleSendDunningNotice":     routes.HandleSendDunningNotice,
		"HandleGenerateGoBDExport":    routes.HandleGenerateGoBDExport,
	}

	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			// Deliberately no withTenantID/withAuth — simulates a token
			// without a tid claim reaching the handler.
			h(rec, req)
			assertStatus(t, rec, http.StatusUnauthorized)
			assertErrorContains(t, rec, "missing or invalid tenant")
		})
	}
}

// ============================================================================
// InvalidUUID — the eleven handlers below used to read chi.URLParam(r, "id")
// straight into the gRPC request, so a malformed id (typo, truncated copy-
// paste) reached the RPC layer as a downstream error instead of a local 400.
// They now go through validateUUIDParam like every other id-bearing handler
// in this package (route_auth.go, route_work_test.go's HandleGetTask, ...).
// ============================================================================

func TestBizBillingRoutes_InvalidUUID(t *testing.T) {
	handlers := map[string]http.HandlerFunc{}
	routes := NewBizRoutes(registryWithService("biz"))
	handlers["HandleGetCreditNote"] = routes.HandleGetCreditNote
	handlers["HandleSendCreditNote"] = routes.HandleSendCreditNote
	handlers["HandleGenerateCreditNotePDF"] = routes.HandleGenerateCreditNotePDF
	handlers["HandleRecordPayment"] = routes.HandleRecordPayment
	handlers["HandleListPayments"] = routes.HandleListPayments
	handlers["HandleDeletePayment"] = routes.HandleDeletePayment
	handlers["HandleSendDunning"] = routes.HandleSendDunning
	handlers["HandleGenerateDunningPDF"] = routes.HandleGenerateDunningPDF
	handlers["HandleLockInvoice"] = routes.HandleLockInvoice
	handlers["HandleUpdateDunningStatus"] = routes.HandleUpdateDunningStatus
	handlers["HandleSendDunningNotice"] = routes.HandleSendDunningNotice

	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = withTenantID(req, testTenantID)
			req = withChiURLParam(req, "id", "not-a-uuid")
			h(rec, req)
			assertStatus(t, rec, http.StatusBadRequest)
			assertErrorContains(t, rec, "invalid id")
		})
	}
}

// ============================================================================
// Credit Notes — validation branches not already exercised by route_biz_test.go
// (which only covers InvalidJSON/MissingCustomer/MissingReason).
// ============================================================================

func TestHandleCreateCreditNote_MissingLineItems(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/credit-notes", jsonBody(t, map[string]interface{}{
		"customer": map[string]interface{}{"name": "ACME"},
		"tax_mode": "standard",
		"reason":   "duplicate",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateCreditNote(rec, req)
	assertValidationError(t, rec, "line_items")
}

func TestHandleCreateCreditNote_InvalidTaxMode(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/credit-notes", jsonBody(t, map[string]interface{}{
		"customer":   map[string]interface{}{"name": "ACME"},
		"line_items": []interface{}{map[string]interface{}{"description": "x"}},
		"tax_mode":   "bogus",
		"reason":     "duplicate",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateCreditNote(rec, req)
	assertValidationError(t, rec, "tax_mode")
}

func TestHandleCreateCreditNote_InvalidOriginalInvoiceID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/credit-notes", jsonBody(t, map[string]interface{}{
		"original_invoice_id": "not-a-uuid",
		"customer":            map[string]interface{}{"name": "ACME"},
		"line_items":          []interface{}{map[string]interface{}{"description": "x"}},
		"tax_mode":            "standard",
		"reason":              "duplicate",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateCreditNote(rec, req)
	assertValidationError(t, rec, "original_invoice_id")
}

func TestHandleCreateCreditNote_InvalidCustomerVAT(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	// DE123456789 is structurally valid (DE + 9 digits) but has a wrong check
	// digit — same fixture route_biz_test.go uses for HandleCreateInvoice.
	req := httptest.NewRequest("POST", "/api/v1/finance/credit-notes", jsonBody(t, map[string]interface{}{
		"customer":   map[string]interface{}{"name": "ACME", "ust_id_nr": "DE123456789"},
		"line_items": []interface{}{map[string]interface{}{"description": "x"}},
		"tax_mode":   "standard",
		"reason":     "duplicate",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateCreditNote(rec, req)
	assertValidationError(t, rec, "customer.ust_id_nr")
}

// ============================================================================
// Payments — amounts must survive the JSON->validate->proto path unchanged
// as a decimal string. A value whose integer part has more precision than
// float64 can hold exactly (20 digits) reaching the gRPC layer (503, not
// 400) proves no float conversion happens anywhere before the RPC call.
// The two decimal places are load-bearing here, not incidental — see
// max_2dp below; a float64 parse would also mangle a 20-digit integer part
// regardless of scale, which is what this test actually proves.
// ============================================================================

func TestHandleRecordPayment_AmountStaysStringHighPrecision(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/payments", jsonBody(t, map[string]interface{}{
		"amount":       "12345678901234567890.12",
		"payment_date": "2026-01-01",
		"method":       "bank_transfer",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRecordPayment(rec, req)
	// decimal_gt0 uses shopspring/decimal (arbitrary precision), not a float
	// parse — a value this large must pass validation and reach the
	// (unreachable, dummy-backend) RPC call, not get rejected as malformed.
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleRecordPayment_IdempotencyKeyForwarded documents that the
// Idempotency-Key header is a pure pass-through into RecordPaymentRequest —
// the gateway performs no local dedup, so a request carrying the header must
// reach the gRPC layer exactly like one without it. The actual dedup
// happens server-side (biz service, DB-level per the handler's own comment);
// asserting the wire value made it into the proto request would need a fake
// FinanceServiceClient, which does not exist for this package yet (same
// boundary noted in Lauf 7 Iteration 6/7 for InboxServiceClient/CRMServiceClient).
func TestHandleRecordPayment_IdempotencyKeyForwarded(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/payments", jsonBody(t, map[string]interface{}{
		"amount":       "100.00",
		"payment_date": "2026-01-01",
		"method":       "bank_transfer",
	}))
	req.Header.Set("Idempotency-Key", "a-client-generated-key")
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRecordPayment(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// A request with NO Idempotency-Key header is not this package's concern to
// prove: the header is enforced by middleware.Idempotency, which wraps the
// router in front of every handler here and is never in the call path when a
// test invokes routes.HandleRecordPayment directly (as every test in this
// file does). That behaviour is proven at its own layer:
// TestIdempotency_MissingKey_WarnMode_Passes and
// TestIdempotency_MissingKey_HardMode_Blocks / TestHardMode_MissingKey_Returns400
// (internal/middleware/idempotency_test.go). The mode actually running in
// production is IDEMPOTENCY_MODE=hard (deploy/docker/docker-compose.prod.yml)
// — a POST without the header gets 400 there, not a silent pass-through.

// TestHandleRecordPayment_InvalidAmountFormats proves every malformed amount
// at this trust boundary lands on 400 through the decimal_gt0/max_2dp
// validator tags (route_biz_billing.go:196), never reaches the RPC layer as
// a downstream 500. payments.amount is NUMERIC(15,2) (migrations/000045) —
// the "more than two decimal places" cases exist because that scale would
// otherwise silently round the value away instead of rejecting it here.
func TestHandleRecordPayment_InvalidAmountFormats(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))

	cases := map[string]string{
		"Negative":                     "-10.00",
		"Zero":                         "0",
		"NonNumeric":                   "not-a-number",
		"Empty":                        "",
		"MoreThanTwoDecimalPlaces":     "10.999",
		"ScientificNotationTooPrecise": "1.005e0", // = 1.005, three decimal places
	}
	for name, amount := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/payments", jsonBody(t, map[string]interface{}{
				"amount":       amount,
				"payment_date": "2026-01-01",
				"method":       "bank_transfer",
			}))
			req = withTenantID(req, testTenantID)
			req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
			routes.HandleRecordPayment(rec, req)
			assertValidationError(t, rec, "amount")
		})
	}
}

// TestHandleRecordPayment_ScientificNotationWithinPrecision documents the
// other side of the case above: "1e2" == 100, zero decimal places — a
// legitimate (if unusual) value that must reach the RPC layer like any other
// valid amount, not be rejected merely for its notation.
func TestHandleRecordPayment_ScientificNotationWithinPrecision(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/payments", jsonBody(t, map[string]interface{}{
		"amount":       "1e2",
		"payment_date": "2026-01-01",
		"method":       "bank_transfer",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRecordPayment(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleRecordPayment_AmountAsJSONNumber: the field is typed string, so a
// bare JSON number ("amount": 100.5 instead of "100.50") fails to unmarshal
// before validation ever runs — a decode error (400 "invalid request body"),
// not a validation error naming the field.
func TestHandleRecordPayment_AmountAsJSONNumber(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"amount": 100.5, "payment_date": "2026-01-01", "method": "bank_transfer"}`
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/payments", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRecordPayment(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// HandleDeletePayment on a payment belonging to a GoBD-locked (LockedAt !=
// nil) invoice: the row itself is always deleted (payment.Service.Delete
// only skips the coupled invoice-status revert when locked, per its own
// GoBD §146 comment — internal/biz/payment/service.go:270-278). This gateway
// package has no fake FinanceServiceClient to script that distinction
// end-to-end (same boundary noted for IdempotencyKeyForwarded above), so the
// behaviour is proven where the logic actually lives:
// TestDelete_NoRevertWhenInvoiceLocked and TestDelete_RevertsFromPaid
// (internal/biz/payment/service_test.go). Whether deleting the payment row
// itself should also be blocked on a locked invoice (not just the status
// revert) is a compliance question for Luke, not a gateway coverage fix —
// noted here so the next person touching this handler sees it.

// HandleGenerateCreditNotePDF without company settings: the server layer
// already turns this into a comprehensible error, not a 500 or an empty
// PDF — BizGRPCServer.requireCompanySettings (internal/server/biz_grpc.go:147)
// returns codes.FailedPrecondition("company settings not configured") when
// none exist, which internal/gateway/helpers.go maps to HTTP 409 (proven
// generically in helpers_test.go, same mapping used by
// TestHandleGenerateCreditNotePDF's sibling handlers). No local gateway logic
// to add or test here.

// ============================================================================
// Dunning
// ============================================================================

func TestHandleCreateDunning_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/dunning/detect", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleCreateDunning(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateDunning_LevelOutOfRange(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))

	cases := map[string]int{"TooLow": 0, "TooHigh": 4}
	for name, level := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/v1/finance/dunning/detect", jsonBody(t, map[string]interface{}{
				"invoice_id": "550e8400-e29b-41d4-a716-446655440000",
				"level":      level,
			}))
			req = withTenantID(req, testTenantID)
			routes.HandleCreateDunning(rec, req)
			assertValidationError(t, rec, "level")
		})
	}
}

func TestHandleCreateDunning_InvalidFee(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/dunning/detect", jsonBody(t, map[string]interface{}{
		"invoice_id": "550e8400-e29b-41d4-a716-446655440000",
		"level":      1,
		"fee":        "-5.00",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateDunning(rec, req)
	assertValidationError(t, rec, "fee")
}

func TestHandleEscalateDunning_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/dunning/id/escalate", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleEscalateDunning(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleEscalateDunning_MissingInvoiceID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/dunning/id/escalate", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	routes.HandleEscalateDunning(rec, req)
	assertValidationError(t, rec, "invoice_id")
}

func TestHandleEscalateDunning_InvalidInvoiceIDFormat(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/dunning/id/escalate", jsonBody(t, map[string]interface{}{
		"invoice_id": "not-a-uuid",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleEscalateDunning(rec, req)
	assertValidationError(t, rec, "invoice_id")
}

func TestHandleUpdateDunningConfig_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/dunning/config", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleUpdateDunningConfig(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateDunningConfig_NegativeDays(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/dunning/config", jsonBody(t, map[string]interface{}{
		"level1_days_after_due": -1,
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleUpdateDunningConfig(rec, req)
	assertValidationError(t, rec, "level1_days_after_due")
}

func TestHandleUpdateDunningStatus_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/dunning/id/status", invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateDunningStatus(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateDunningStatus_InvalidStatus(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/finance/dunning/id/status", jsonBody(t, map[string]interface{}{
		"status": "overdue",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateDunningStatus(rec, req)
	assertValidationError(t, rec, "status")
}

// ============================================================================
// GoBD Journal & Compliance — the year bound check (2000..2100) is local
// logic unique to this handler, not shared validation-tag machinery, so it
// gets its own direct coverage.
// ============================================================================

func TestHandleGetJournalSummary_MissingYear(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/journal/summary", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGetJournalSummary(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "year query parameter is required")
}

func TestHandleGetJournalSummary_InvalidYearFormat(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/journal/summary?year=abc", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGetJournalSummary(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "4-digit number")
}

func TestHandleGetJournalSummary_YearOutOfRange(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))

	cases := map[string]string{"TooLow": "1999", "TooHigh": "2101"}
	for name, year := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/v1/finance/journal/summary?year="+year, nil)
			req = withTenantID(req, testTenantID)
			routes.HandleGetJournalSummary(rec, req)
			assertStatus(t, rec, http.StatusBadRequest)
			assertErrorContains(t, rec, "4-digit number")
		})
	}
}

func TestHandleGetJournalSummary_ValidYearReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/journal/summary?year=2026", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGetJournalSummary(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleValidateInvoiceNumber_MissingNumber(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/finance/invoices/validate-number", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleValidateInvoiceNumber(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "number query parameter is required")
}

func TestHandleGetPaymentStats_MissingDates(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))

	cases := map[string]string{
		"MissingBoth": "",
		"MissingTo":   "?from=2026-01-01",
		"MissingFrom": "?to=2026-12-31",
	}
	for name, query := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/v1/finance/stats/payments"+query, nil)
			req = withTenantID(req, testTenantID)
			routes.HandleGetPaymentStats(rec, req)
			assertStatus(t, rec, http.StatusBadRequest)
			assertErrorContains(t, rec, "from and to")
		})
	}
}

func TestHandleGenerateGoBDExport_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/export/gobd", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleGenerateGoBDExport(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleGenerateGoBDExport_MissingDates(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/export/gobd", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	routes.HandleGenerateGoBDExport(rec, req)
	assertValidationError(t, rec, "from_date")
}
