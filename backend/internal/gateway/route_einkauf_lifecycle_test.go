package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Covers the thirteen PO-lifecycle and supplier handlers in route_einkauf.go
// that had no tests: HandleGetPO, HandleListPOs, HandleUpdatePO, HandleDeletePO,
// HandleCancelPO, HandleSubmitPO, HandleListPOLines, HandleUpdatePOLine,
// HandleDeletePOLine, HandleReceiveGoods, HandlePartialReceive, HandleGetSupplier,
// HandleUpdateSupplier. Same pattern as route_vermietung_test.go: the gateway has
// no state-machine or idempotency logic of its own — it forwards any
// syntactically valid request straight to the RPC, so a dummy (unreachable)
// connection surfaces as 503 ("ReachesRPC"). State-transition guards (draft-only
// delete, non-cancellable-after-receive, wrong-status receive) and the
// double-receive idempotency question live in internal/einkauf/service.go and
// are covered there (TestService_ReceiveGoods_WrongStatus and friends), not here.

const (
	testEinkaufPOID       = "550e8400-e29b-41d4-a716-446655440010"
	testEinkaufSupplierID = "550e8400-e29b-41d4-a716-446655440011"
	testEinkaufLineID     = "550e8400-e29b-41d4-a716-446655440012"
)

// --- HandleGetSupplier ---

func TestHandleGetSupplier_ServiceUnavailable(t *testing.T) {
	routes := NewEinkaufRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleGetSupplier)
}

func TestHandleGetSupplier_InvalidIDUUID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/einkauf/suppliers/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetSupplier(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetSupplier_ReachesRPC(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/einkauf/suppliers/"+testEinkaufSupplierID, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufSupplierID)
	routes.HandleGetSupplier(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateSupplier ---

func TestHandleUpdateSupplier_ServiceUnavailable(t *testing.T) {
	routes := NewEinkaufRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleUpdateSupplier)
}

func TestHandleUpdateSupplier_InvalidIDUUID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/einkauf/suppliers/not-a-uuid", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateSupplier(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateSupplier_InvalidEmail(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/einkauf/suppliers/"+testEinkaufSupplierID, jsonBody(t, map[string]interface{}{
		"email": "not-an-email",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufSupplierID)
	routes.HandleUpdateSupplier(rec, req)
	assertValidationError(t, rec, "email")
}

func TestHandleUpdateSupplier_InvalidJSON(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/einkauf/suppliers/"+testEinkaufSupplierID, invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufSupplierID)
	routes.HandleUpdateSupplier(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateSupplier_ReachesRPC(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/einkauf/suppliers/"+testEinkaufSupplierID, jsonBody(t, map[string]interface{}{
		"name": "New Name GmbH",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufSupplierID)
	routes.HandleUpdateSupplier(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetPO ---

func TestHandleGetPO_ServiceUnavailable(t *testing.T) {
	routes := NewEinkaufRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleGetPO)
}

func TestHandleGetPO_InvalidIDUUID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/einkauf/pos/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetPO(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetPO_ReachesRPC(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/einkauf/pos/"+testEinkaufPOID, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	routes.HandleGetPO(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListPOs ---

func TestHandleListPOs_ServiceUnavailable(t *testing.T) {
	routes := NewEinkaufRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleListPOs)
}

func TestHandleListPOs_ReachesRPC(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/einkauf/pos?status=draft&supplier_id="+testEinkaufSupplierID, nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListPOs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdatePO ---

func TestHandleUpdatePO_ServiceUnavailable(t *testing.T) {
	routes := NewEinkaufRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleUpdatePO)
}

func TestHandleUpdatePO_InvalidIDUUID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/einkauf/pos/not-a-uuid", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdatePO(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdatePO_InvalidJSON(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/einkauf/pos/"+testEinkaufPOID, invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	routes.HandleUpdatePO(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// TestHandleUpdatePO_ClosedOrCancelledRejected_ReachesRPC documents that the
// gateway forwards an UpdatePO for any syntactically valid PO ID straight to
// the RPC — the "no edits after closed/cancelled" guard (ErrPONotDraft) lives
// in internal/einkauf/service.go (TestService_UpdatePO_ClosedRejected,
// TestService_UpdatePO_CancelledRejected), not in the gateway.
func TestHandleUpdatePO_ClosedOrCancelledRejected_ReachesRPC(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/einkauf/pos/"+testEinkaufPOID, jsonBody(t, map[string]interface{}{
		"notes": "updated",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	routes.HandleUpdatePO(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeletePO ---

func TestHandleDeletePO_ServiceUnavailable(t *testing.T) {
	routes := NewEinkaufRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleDeletePO)
}

func TestHandleDeletePO_InvalidIDUUID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/einkauf/pos/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeletePO(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// TestHandleDeletePO_NonDraftRejected_ReachesRPC documents that "delete only
// allowed when draft" (ErrPONotDraft) is enforced in the service
// (TestService_DeletePO_NonDraftRejected), not in the gateway.
func TestHandleDeletePO_NonDraftRejected_ReachesRPC(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/einkauf/pos/"+testEinkaufPOID, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	routes.HandleDeletePO(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleSubmitPO ---

func TestHandleSubmitPO_ServiceUnavailable(t *testing.T) {
	routes := NewEinkaufRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleSubmitPO)
}

func TestHandleSubmitPO_InvalidIDUUID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/pos/not-a-uuid/submit", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleSubmitPO(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// TestHandleSubmitPO_NonDraftOrNoLinesRejected_ReachesRPC documents that
// "only draft is submittable" and "at least one line required" (ErrPONotDraft,
// ErrPONotSubmittable) are enforced in the service
// (TestService_SubmitPO_NonDraftRejected, TestService_SubmitPO_NoLines).
func TestHandleSubmitPO_NonDraftOrNoLinesRejected_ReachesRPC(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/pos/"+testEinkaufPOID+"/submit", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	routes.HandleSubmitPO(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleCancelPO ---

func TestHandleCancelPO_ServiceUnavailable(t *testing.T) {
	routes := NewEinkaufRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleCancelPO)
}

func TestHandleCancelPO_InvalidIDUUID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/pos/not-a-uuid/cancel", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCancelPO(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// TestHandleCancelPO_AfterReceivingRejected_ReachesRPC documents that "not
// cancellable once receiving has started" (ErrPONotCancellable) is enforced in
// the service (TestService_CancelPO_NotCancellableRejected), not the gateway.
func TestHandleCancelPO_AfterReceivingRejected_ReachesRPC(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/pos/"+testEinkaufPOID+"/cancel", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	routes.HandleCancelPO(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleReceiveGoods ---

func TestHandleReceiveGoods_ServiceUnavailable(t *testing.T) {
	routes := NewEinkaufRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleReceiveGoods)
}

func TestHandleReceiveGoods_InvalidIDUUID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/pos/not-a-uuid/receive", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleReceiveGoods(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// TestHandleReceiveGoods_DoubleReceive_DoesNotDoubleBook documents the
// idempotency question from cov-gateway-einkauf-purchase-order-lifecycle: does
// a repeated goods receipt double-book stock/liability? The gateway itself has
// no state — it forwards every syntactically valid request straight to
// ReceiveGoods, so the guard against a double receipt has to live in the
// service, which is where it is proven: ReceiveGoods only accepts
// submitted/sent/partially_received (service.go:612), and transitions the PO
// to "received" — so calling it again on an already-received PO is rejected
// with ErrPONotReceivable before any line or stock update runs
// (TestService_ReceiveGoods_DoubleReceive_SecondCallRejected in
// internal/einkauf/service_test.go). At the gateway layer, both calls simply
// reach the RPC identically.
func TestHandleReceiveGoods_ReachesRPC(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/pos/"+testEinkaufPOID+"/receive", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	routes.HandleReceiveGoods(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandlePartialReceive ---

func TestHandlePartialReceive_ServiceUnavailable(t *testing.T) {
	routes := NewEinkaufRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandlePartialReceive)
}

func TestHandlePartialReceive_InvalidIDUUID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/pos/not-a-uuid/partial-receive",
		jsonBody(t, map[string]interface{}{"items": []map[string]string{{"line_id": testEinkaufLineID, "received_quantity": "5"}}}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandlePartialReceive(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandlePartialReceive_MissingItems(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/pos/"+testEinkaufPOID+"/partial-receive",
		jsonBody(t, map[string]interface{}{"items": []map[string]string{}}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	routes.HandlePartialReceive(rec, req)
	assertValidationError(t, rec, "items")
}

func TestHandlePartialReceive_InvalidJSON(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/pos/"+testEinkaufPOID+"/partial-receive", invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	routes.HandlePartialReceive(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// TestHandlePartialReceive_ExceedsOrderedOrWrongStatus_ReachesRPC documents
// that "received quantity exceeds ordered" (ErrExceedsOrderedQty) and "wrong
// status" (ErrPONotReceivable) are enforced in the service
// (TestService_PartialReceive_ExceedsOrdered, and the same status guard as
// ReceiveGoods), not the gateway — a duplicate partial-receive call with an
// identical cumulative quantity is a no-op delta there too (service.go:695-703).
func TestHandlePartialReceive_ReachesRPC(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/pos/"+testEinkaufPOID+"/partial-receive",
		jsonBody(t, map[string]interface{}{"items": []map[string]string{{"line_id": testEinkaufLineID, "received_quantity": "5"}}}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	routes.HandlePartialReceive(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListPOLines ---

func TestHandleListPOLines_ServiceUnavailable(t *testing.T) {
	routes := NewEinkaufRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleListPOLines)
}

func TestHandleListPOLines_InvalidIDUUID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/einkauf/pos/not-a-uuid/lines", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListPOLines(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListPOLines_ReachesRPC(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/einkauf/pos/"+testEinkaufPOID+"/lines", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	routes.HandleListPOLines(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdatePOLine ---

func TestHandleUpdatePOLine_ServiceUnavailable(t *testing.T) {
	routes := NewEinkaufRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleUpdatePOLine)
}

func TestHandleUpdatePOLine_InvalidPOIDUUID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/einkauf/pos/not-a-uuid/lines/"+testEinkaufLineID, jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withChiURLParam(req, "lineId", testEinkaufLineID)
	routes.HandleUpdatePOLine(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdatePOLine_InvalidLineIDUUID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/einkauf/pos/"+testEinkaufPOID+"/lines/not-a-uuid", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	req = withChiURLParam(req, "lineId", "not-a-uuid")
	routes.HandleUpdatePOLine(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid lineId")
}

func TestHandleUpdatePOLine_InvalidQuantity(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/einkauf/pos/"+testEinkaufPOID+"/lines/"+testEinkaufLineID,
		jsonBody(t, map[string]interface{}{"quantity": "0"}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	req = withChiURLParam(req, "lineId", testEinkaufLineID)
	routes.HandleUpdatePOLine(rec, req)
	assertValidationError(t, rec, "quantity")
}

func TestHandleUpdatePOLine_InvalidJSON(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/einkauf/pos/"+testEinkaufPOID+"/lines/"+testEinkaufLineID, invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	req = withChiURLParam(req, "lineId", testEinkaufLineID)
	routes.HandleUpdatePOLine(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdatePOLine_ReachesRPC(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/einkauf/pos/"+testEinkaufPOID+"/lines/"+testEinkaufLineID,
		jsonBody(t, map[string]interface{}{"quantity": "3"}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	req = withChiURLParam(req, "lineId", testEinkaufLineID)
	routes.HandleUpdatePOLine(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeletePOLine ---

func TestHandleDeletePOLine_ServiceUnavailable(t *testing.T) {
	routes := NewEinkaufRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleDeletePOLine)
}

func TestHandleDeletePOLine_InvalidPOIDUUID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/einkauf/pos/not-a-uuid/lines/"+testEinkaufLineID, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withChiURLParam(req, "lineId", testEinkaufLineID)
	routes.HandleDeletePOLine(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeletePOLine_InvalidLineIDUUID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/einkauf/pos/"+testEinkaufPOID+"/lines/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	req = withChiURLParam(req, "lineId", "not-a-uuid")
	routes.HandleDeletePOLine(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid lineId")
}

func TestHandleDeletePOLine_ReachesRPC(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/einkauf/pos/"+testEinkaufPOID+"/lines/"+testEinkaufLineID, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", testEinkaufPOID)
	req = withChiURLParam(req, "lineId", testEinkaufLineID)
	routes.HandleDeletePOLine(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
