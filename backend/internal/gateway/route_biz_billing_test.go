package gateway

import (
	"net/http"
	"net/http/httptest"
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
// as a decimal string. A value with more precision than float64 can hold
// exactly (20 digits) reaching the gRPC layer (503, not 400) proves no float
// conversion happens anywhere before the RPC call.
// ============================================================================

func TestHandleRecordPayment_AmountStaysStringHighPrecision(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices/id/payments", jsonBody(t, map[string]interface{}{
		"amount":       "12345678901234567890.123456789",
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
