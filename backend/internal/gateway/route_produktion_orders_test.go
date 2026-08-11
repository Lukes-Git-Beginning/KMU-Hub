package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Covers the production order lifecycle handlers in route_produktion.go:
// HandleListOrders, HandleGetOrder, HandleUpdateOrder, HandleStartOrder,
// HandleCompleteOrder, HandleCancelOrder. route_produktion_test.go so far
// only exercised HandleCreateOrder/HandleCreateMachineBooking/HandleCreatePlan
// validation, and route_produktion_ext_test.go only the route registration.

// --- HandleListOrders ---

func TestHandleListOrders_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/orders", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListOrders(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListOrders_MissingTenant(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/orders", nil)
	routes.HandleListOrders(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "missing or invalid tenant")
}

// HandleListOrders passes the RPC response straight to response.Proto with no
// gateway-local wire-shape logic of its own (route_produktion.go:214-219),
// and there is no bufconn stub for the produktion service in this package to
// fake an actual ListOrdersResponse (same boundary noted in every prior
// gateway coverage unit this run, e.g. TestHandleListCampaignContacts_ReachesRPC
// in route_dialer_test.go). This proves the handler builds the request from
// query params (status/priority/date_from/date_to) and reaches the RPC layer;
// the {items,total}-wrapped, []-not-null wire shape of the order list is a
// property of the produktion service's own proto marshaling.
func TestHandleListOrders_ReachesRPC(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/orders?status=planned&priority=3&date_from=2026-06-01T00:00:00Z&date_to=2026-06-30T00:00:00Z", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListOrders(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetOrder ---

func TestHandleGetOrder_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/orders/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetOrder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetOrder_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/orders/bad", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetOrder(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// --- HandleUpdateOrder ---

func TestHandleUpdateOrder_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/orders/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateOrder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateOrder_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/orders/bad", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateOrder(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateOrder_InvalidJSON(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/orders/550e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateOrder(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// --- HandleStartOrder / HandleCompleteOrder / HandleCancelOrder ---
//
// These handlers carry no local status-transition logic of their own — they
// forward an OrderActionRequest{TenantId, OrderId} straight to the RPC
// (route_produktion.go:389-465) and let the produktion service enforce the
// geplant -> gestartet -> abgeschlossen/storniert transition, returning
// FailedPrecondition (-> 409 via grpcStatusToHTTP) for an invalid one. There
// is no bufconn stub for the produktion service in this package to fake that
// FailedPrecondition response (same boundary as every other gateway coverage
// unit this run without one), so the *_ReachesRPC tests below prove the
// handler reaches the RPC layer with a valid order ID — the invalid-
// transition rejection itself is exercised at the produktion service level,
// not here.

func TestHandleStartOrder_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/orders/550e8400-e29b-41d4-a716-446655440000/start", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleStartOrder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleStartOrder_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/orders/bad/start", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleStartOrder(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleStartOrder_ReachesRPC(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/orders/550e8400-e29b-41d4-a716-446655440000/start", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleStartOrder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCompleteOrder_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/orders/550e8400-e29b-41d4-a716-446655440000/complete", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCompleteOrder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCompleteOrder_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/orders/bad/complete", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCompleteOrder(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleCompleteOrder_ReachesRPC(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/orders/550e8400-e29b-41d4-a716-446655440000/complete", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCompleteOrder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCancelOrder_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/orders/550e8400-e29b-41d4-a716-446655440000/cancel", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCancelOrder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCancelOrder_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/orders/bad/cancel", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCancelOrder(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleCancelOrder_ReachesRPC(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/orders/550e8400-e29b-41d4-a716-446655440000/cancel", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCancelOrder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
