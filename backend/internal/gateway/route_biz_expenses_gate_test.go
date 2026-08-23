package gateway

// Boundary and validation coverage for route_biz_expenses.go: before this unit
// all seven handlers had zero HTTP-level tests (only the wire-shape mapping in
// route_biz_expenses_test.go was covered). Follows the established pattern
// from route_biz_billing_test.go / route_datev_upload_test.go: ServiceUnavailable
// and NoTenant tables prove the boundary for every handler at once, then each
// handler's one interesting failure mode gets its own test.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func expenseTestHandlers(routes *BizRoutes) map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"HandleListExpenses":         routes.HandleListExpenses,
		"HandleCreateExpense":        routes.HandleCreateExpense,
		"HandleUpdateExpense":        routes.HandleUpdateExpense,
		"HandleApproveExpense":       routes.HandleApproveExpense,
		"HandleRejectExpense":        routes.HandleRejectExpense,
		"HandleAttachExpenseReceipt": routes.HandleAttachExpenseReceipt,
		"HandleDeleteExpense":        routes.HandleDeleteExpense,
	}
}

func TestBizExpensesRoutes_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	for name, h := range expenseTestHandlers(routes) {
		t.Run(name, func(t *testing.T) {
			testServiceUnavailable(t, h)
		})
	}
}

func TestBizExpensesRoutes_NoTenant(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	for name, h := range expenseTestHandlers(routes) {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
			// Deliberately no withTenantID — simulates a token without a tid claim.
			h(rec, req)
			assertStatus(t, rec, http.StatusUnauthorized)
			assertErrorContains(t, rec, "missing or invalid tenant")
		})
	}
}

// ============================================================================
// HandleCreateExpense — the interesting cases are the two rejection rules the
// backlog unit asked to verify: a non-positive amount (an "Erstattung"/refund
// as a negative expense is not supported today — validate:"required,gt=0"
// rejects it locally, same as a missing description) and a malformed date.
// ============================================================================

func TestHandleCreateExpense_MissingDescription(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"amount":10.5,"date":"2026-05-20"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/expenses", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateExpense(rec, req)
	assertValidationError(t, rec, "description")
}

// TestHandleCreateExpense_NegativeAmountRejected pins that a refund cannot be
// recorded as a negative expense: the handler's validate:"gt=0" tag rejects it
// before the request ever reaches the service, which enforces the same rule
// independently (parseAmount -> ErrInvalidAmount). If Erstattungen need their
// own negative-amount path, that is a data-model decision, not a bug fix.
func TestHandleCreateExpense_NegativeAmountRejected(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"description":"Erstattung Kunde","amount":-15.00,"date":"2026-05-20"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/expenses", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateExpense(rec, req)
	assertValidationError(t, rec, "amount")
}

func TestHandleCreateExpense_ZeroAmountRejected(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"description":"Nichts","amount":0,"date":"2026-05-20"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/expenses", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateExpense(rec, req)
	assertValidationError(t, rec, "amount")
}

func TestHandleCreateExpense_InvalidDateFormatRejected(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"description":"Bürobedarf","amount":10.5,"date":"20.05.2026"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/expenses", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateExpense(rec, req)
	assertValidationError(t, rec, "date")
}

func TestHandleCreateExpense_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/expenses", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleCreateExpense(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateExpense_ValidBodyReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"description":"Bürobedarf","amount":10.5,"date":"2026-05-20"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/expenses", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateExpense(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateExpense — a partial body (only the fields the Kontierung dialog
// sends) must pass validation and reach the RPC; an explicit negative amount
// in an update is rejected the same way as on create.
// ============================================================================

func TestHandleUpdateExpense_PartialBodyReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"account":"4930"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/finance/expenses/id", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleUpdateExpense(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateExpense_NegativeAmountRejected(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"amount":-5}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/finance/expenses/id", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleUpdateExpense(rec, req)
	assertValidationError(t, rec, "amount")
}

func TestHandleUpdateExpense_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/finance/expenses/id", invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleUpdateExpense(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// ============================================================================
// HandleApproveExpense / HandleRejectExpense share decideExpense; the
// four-eyes rule and the "only a pending expense" guard live in expense.Service
// (already covered by internal/biz/expense's own tests) and are unreachable
// from here without a real RPC — this only pins that both paths clear the
// gateway boundary and reach the RPC with the expected decision status.
// ============================================================================

func TestHandleApproveExpense_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/expenses/id/approve", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleApproveExpense(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRejectExpense_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/expenses/id/reject", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleRejectExpense(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleAttachExpenseReceipt — receiptName is required; the actual receipt
// content is never uploaded through this route at all (internal/biz/expense's
// package comment already carries a lean: marker for this — a receipt is a
// filename, not a stored document, until the presign-upload flow from
// internal/chat/file/minio_store.go is added). No 413/GoBD-archive test is
// meaningful here because there is no file body to reject or archive.
// ============================================================================

func TestHandleAttachExpenseReceipt_MissingReceiptName(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/expenses/id/receipt", strings.NewReader("{}"))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleAttachExpenseReceipt(rec, req)
	assertValidationError(t, rec, "receiptName")
}

func TestHandleAttachExpenseReceipt_ValidBodyReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"receiptName":"beleg-q2.pdf"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/expenses/id/receipt", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleAttachExpenseReceipt(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteExpense — the "no deleting after a decision" guard
// (expense.ErrDecided, FailedPrecondition -> 409) lives in expense.Service and
// is unreachable here without a real RPC; this pins the boundary reaches the
// RPC. The guard itself is the same "decided record stays" rule tested in
// internal/biz/expense/service_test.go, not a period-close lock — this
// codebase has no GoBD period-lock concept for any entity, only the
// per-invoice manual LockInvoice, which expenses do not share.
// ============================================================================

func TestHandleDeleteExpense_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/finance/expenses/id", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleDeleteExpense(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListExpenses — no body, no URL param; the only two reachable states
// are NoTenant (covered above) and reaching the RPC.
// ============================================================================

func TestHandleListExpenses_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/expenses", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListExpenses(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
