package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// HandleListVehicles
// ============================================================================

func TestHandleListVehicles_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListVehicles(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListVehicles_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles", nil)
	req = withUserID(req, "user-123")
	routes.HandleListVehicles(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListVehicles_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles?status=active&fuel_type=diesel", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListVehicles(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetVehicle
// ============================================================================

func TestHandleGetVehicle_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetVehicle(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetVehicle_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetVehicle(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetVehicle_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetVehicle(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetVehicle_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetVehicle(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateVehicle
// ============================================================================

func TestHandleUpdateVehicle_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"notes": "updated"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateVehicle(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateVehicle_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"notes": "updated"}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateVehicle(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateVehicle_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/vehicles/not-a-uuid",
		jsonBody(t, map[string]interface{}{"notes": "updated"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateVehicle(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateVehicle_InvalidFuelType(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"fuel_type": "hydrogen"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateVehicle(rec, req)
	assertValidationError(t, rec, "fuel_type")
}

func TestHandleUpdateVehicle_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"notes": "updated"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateVehicle(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteVehicle
// ============================================================================

func TestHandleDeleteVehicle_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteVehicle(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteVehicle_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteVehicle(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteVehicle_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/vehicles/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteVehicle(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteVehicle_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteVehicle(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetVehicleHistory
// ============================================================================

func TestHandleGetVehicleHistory_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/history", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetVehicleHistory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetVehicleHistory_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/history", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetVehicleHistory(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetVehicleHistory_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/not-a-uuid/history", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetVehicleHistory(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetVehicleHistory_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/history?page=2&page_size=10", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetVehicleHistory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListVehicleServices
// ============================================================================

func TestHandleListVehicleServices_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/services", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListVehicleServices(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListVehicleServices_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/not-a-uuid/services", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListVehicleServices(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListVehicleServices_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/services", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListVehicleServices(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListVehicleServices_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/services", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListVehicleServices(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateService
// ============================================================================

func TestHandleUpdateService_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/services/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"status": "completed"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateService(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateService_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/services/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"status": "completed"}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateService(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateService_InvalidServiceIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/services/not-a-uuid",
		jsonBody(t, map[string]interface{}{"status": "completed"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateService(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateService_InvalidJSON(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/services/550e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateService(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// updateServiceRequest.Status carries no validate tag (route_fuhrpark.go:227)
// — unlike e.g. createVehicleRequest.FuelType, an arbitrary status string is
// not rejected at the gateway. A bogus status transition (e.g. "completed"
// applied to an already-cancelled service) is business logic that only the
// fuhrpark service can reject, and there is no bufconn stub for it in this
// package (same boundary noted in every prior gateway coverage unit this
// run) — so this proves the request parses past every field and still
// reaches the RPC layer, not that an invalid transition is rejected locally.
func TestHandleUpdateService_ReachesRPCWithArbitraryStatus(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/services/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"status": "not-a-real-status"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateService(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCompleteService
// ============================================================================

func TestHandleCompleteService_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/services/550e8400-e29b-41d4-a716-446655440000/complete",
		jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCompleteService(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCompleteService_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/services/550e8400-e29b-41d4-a716-446655440000/complete",
		jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCompleteService(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCompleteService_InvalidServiceIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/services/not-a-uuid/complete",
		jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCompleteService(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleCompleteService_InvalidJSON(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/services/550e8400-e29b-41d4-a716-446655440000/complete", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCompleteService(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCompleteService_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/services/550e8400-e29b-41d4-a716-446655440000/complete",
		jsonBody(t, map[string]interface{}{"cost_cents": 4999, "mileage_km": 12000}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCompleteService(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCheckTuevDue
// ============================================================================

func TestHandleCheckTuevDue_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/tuev-due", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCheckTuevDue(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCheckTuevDue_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/tuev-due", nil)
	req = withUserID(req, "user-123")
	routes.HandleCheckTuevDue(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// HandleCheckTuevDue passes the RPC response straight to response.Proto with
// no intermediate marshal step of its own (route_fuhrpark.go:968-975) — unlike
// e.g. hrMarshalSlice in route_hr.go, there is no gateway-local wire-shape
// logic here to unit-test, and no bufconn stub for the fuhrpark service in
// this package to fake an actual response (same boundary noted in every
// prior gateway coverage unit this run). This proves the handler reaches the
// RPC layer with a well-formed request; the []-not-null wire shape of the
// due list is a property of the fuhrpark service's own proto marshaling.
func TestHandleCheckTuevDue_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/tuev-due", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCheckTuevDue(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
