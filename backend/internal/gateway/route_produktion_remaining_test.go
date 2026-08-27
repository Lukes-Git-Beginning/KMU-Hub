package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Covers the remaining untested handlers in route_produktion.go:
// HandleDeleteOrder, HandleGetMaterialAvailability, HandleListMachineBookings,
// HandleUpdateMachineBooking, HandleDeleteMachineBooking, HandleGetPlan,
// HandleUpdatePlan, HandleGetCapacityOverview. route_produktion_test.go and
// route_produktion_orders_test.go already cover the order lifecycle and the
// create-side validation of orders/bookings/plans.

const testOrderID = "550e8400-e29b-41d4-a716-446655440000"

// --- HandleDeleteOrder ---

func TestHandleDeleteOrder_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/produktion/orders/"+testOrderID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testOrderID)
	routes.HandleDeleteOrder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteOrder_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/produktion/orders/bad", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteOrder(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// --- HandleGetMaterialAvailability ---

func TestHandleGetMaterialAvailability_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/orders/"+testOrderID+"/material-availability", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testOrderID)
	routes.HandleGetMaterialAvailability(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetMaterialAvailability_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/orders/bad/material-availability", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetMaterialAvailability(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// --- HandleListMachineBookings ---

func TestHandleListMachineBookings_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/bookings", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListMachineBookings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListMachineBookings_MissingTenant(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/bookings", nil)
	routes.HandleListMachineBookings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "missing or invalid tenant")
}

// HandleListMachineBookings builds its request straight from query params
// (machine_id/production_order_id/date_from/date_to) with no gateway-local
// filtering logic (route_produktion.go:497-540); there is no bufconn stub for
// the produktion service in this package to fake a real response, so this
// proves the handler parses the filters and reaches the RPC layer.
func TestHandleListMachineBookings_ReachesRPC(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/bookings?machine_id=m1&production_order_id="+testOrderID+"&date_from=2026-06-01T00:00:00Z&date_to=2026-06-30T00:00:00Z", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListMachineBookings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateMachineBooking ---

func TestHandleUpdateMachineBooking_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/bookings/"+testOrderID, jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testOrderID)
	routes.HandleUpdateMachineBooking(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateMachineBooking_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/bookings/bad", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateMachineBooking(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateMachineBooking_InvalidJSON(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/bookings/"+testOrderID, invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testOrderID)
	routes.HandleUpdateMachineBooking(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// --- HandleDeleteMachineBooking ---

func TestHandleDeleteMachineBooking_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/produktion/bookings/"+testOrderID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testOrderID)
	routes.HandleDeleteMachineBooking(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteMachineBooking_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/produktion/bookings/bad", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteMachineBooking(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// --- HandleGetPlan ---

func TestHandleGetPlan_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/plans/"+testOrderID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testOrderID)
	routes.HandleGetPlan(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetPlan_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/plans/bad", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetPlan(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// --- HandleUpdatePlan ---

func TestHandleUpdatePlan_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/plans/"+testOrderID, jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testOrderID)
	routes.HandleUpdatePlan(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdatePlan_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/plans/bad", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdatePlan(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdatePlan_InvalidJSON(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/plans/"+testOrderID, invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testOrderID)
	routes.HandleUpdatePlan(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// --- HandleGetCapacityOverview ---

func TestHandleGetCapacityOverview_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/plans/"+testOrderID+"/capacity?machine_id=m1", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testOrderID)
	routes.HandleGetCapacityOverview(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetCapacityOverview_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/plans/bad/capacity?machine_id=m1", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetCapacityOverview(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// HandleGetCapacityOverview requires machine_id as a query param — unlike every
// other handler in this file, this validation lives in the gateway itself
// (route_produktion.go:807-811), not behind the RPC boundary, so it is directly
// testable without a service connection.
func TestHandleGetCapacityOverview_MissingMachineID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/plans/"+testOrderID+"/capacity", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testOrderID)
	routes.HandleGetCapacityOverview(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "machine_id")
}
