package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// HandleCreateFuelLog
// ============================================================================

func TestHandleCreateFuelLog_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/fuel-logs",
		jsonBody(t, map[string]interface{}{"date": "2026-08-01", "liters": 40.5}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateFuelLog(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateFuelLog_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/fuel-logs",
		jsonBody(t, map[string]interface{}{"date": "2026-08-01", "liters": 40.5}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateFuelLog(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateFuelLog_MissingLiters(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/fuel-logs",
		jsonBody(t, map[string]interface{}{"date": "2026-08-01"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateFuelLog(rec, req)
	assertValidationError(t, rec, "liters")
}

func TestHandleCreateFuelLog_MissingDate(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/fuel-logs",
		jsonBody(t, map[string]interface{}{"liters": 40.5}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateFuelLog(rec, req)
	assertValidationError(t, rec, "date")
}

func TestHandleCreateFuelLog_InvalidVehicleIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/not-a-uuid/fuel-logs",
		jsonBody(t, map[string]interface{}{"date": "2026-08-01", "liters": 40.5}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCreateFuelLog(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// FuelType defaults to "diesel" when omitted (route_fuhrpark.go:1057-1060) —
// this proves the default is applied and the request still reaches the RPC
// layer rather than being rejected for a missing fuel_type.
func TestHandleCreateFuelLog_DefaultFuelTypeReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/fuel-logs",
		jsonBody(t, map[string]interface{}{"date": "2026-08-01", "liters": 40.5}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateFuelLog(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateFuelLog
// ============================================================================

func TestHandleUpdateFuelLog_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/fuel-logs/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"notes": "updated"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateFuelLog(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateFuelLog_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/fuel-logs/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"notes": "updated"}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateFuelLog(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateFuelLog_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/fuel-logs/not-a-uuid",
		jsonBody(t, map[string]interface{}{"notes": "updated"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateFuelLog(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateFuelLog_InvalidLiters(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/fuel-logs/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"liters": -5.0}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateFuelLog(rec, req)
	assertValidationError(t, rec, "liters")
}

func TestHandleUpdateFuelLog_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/fuel-logs/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"mileage_km": 12345}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateFuelLog(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteFuelLog
// ============================================================================

func TestHandleDeleteFuelLog_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/fuel-logs/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteFuelLog(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteFuelLog_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/fuel-logs/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteFuelLog(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteFuelLog_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/fuel-logs/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteFuelLog(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteFuelLog_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/fuel-logs/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteFuelLog(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCreateTripLog
// ============================================================================

func TestHandleCreateTripLog_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/trip-logs",
		jsonBody(t, map[string]interface{}{
			"date": "2026-08-01", "start_location": "Berlin", "end_location": "Hamburg",
			"purpose": "delivery", "driver_name": "Max Muster",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateTripLog(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateTripLog_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/trip-logs",
		jsonBody(t, map[string]interface{}{
			"date": "2026-08-01", "start_location": "Berlin", "end_location": "Hamburg",
			"purpose": "delivery", "driver_name": "Max Muster",
		}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateTripLog(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateTripLog_MissingDate(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/trip-logs",
		jsonBody(t, map[string]interface{}{
			"start_location": "Berlin", "end_location": "Hamburg",
			"purpose": "delivery", "driver_name": "Max Muster",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateTripLog(rec, req)
	assertValidationError(t, rec, "date")
}

func TestHandleCreateTripLog_MissingDriverName(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/trip-logs",
		jsonBody(t, map[string]interface{}{
			"date": "2026-08-01", "start_location": "Berlin", "end_location": "Hamburg",
			"purpose": "delivery",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateTripLog(rec, req)
	assertValidationError(t, rec, "driver_name")
}

func TestHandleCreateTripLog_InvalidVehicleIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/not-a-uuid/trip-logs",
		jsonBody(t, map[string]interface{}{
			"date": "2026-08-01", "start_location": "Berlin", "end_location": "Hamburg",
			"purpose": "delivery", "driver_name": "Max Muster",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCreateTripLog(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleCreateTripLog_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/trip-logs",
		jsonBody(t, map[string]interface{}{
			"date": "2026-08-01", "start_location": "Berlin", "end_location": "Hamburg",
			"purpose": "delivery", "driver_name": "Max Muster",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateTripLog(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateTripLog
// ============================================================================

func TestHandleUpdateTripLog_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/trip-logs/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"notes": "updated"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTripLog(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateTripLog_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/trip-logs/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"notes": "updated"}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTripLog(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateTripLog_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/trip-logs/not-a-uuid",
		jsonBody(t, map[string]interface{}{"notes": "updated"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateTripLog(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateTripLog_InvalidStartKm(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/trip-logs/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"start_km": -10}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTripLog(rec, req)
	assertValidationError(t, rec, "start_km")
}

func TestHandleUpdateTripLog_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/trip-logs/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"end_km": 500}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTripLog(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteTripLog
// ============================================================================

func TestHandleDeleteTripLog_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/trip-logs/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteTripLog(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteTripLog_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/trip-logs/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteTripLog(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteTripLog_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/trip-logs/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteTripLog(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteTripLog_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/trip-logs/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteTripLog(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCreateVehicleBooking
// ============================================================================

func TestHandleCreateVehicleBooking_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/bookings",
		jsonBody(t, map[string]interface{}{
			"vehicle_id": "550e8400-e29b-41d4-a716-446655440000",
			"user_id":    "660e8400-e29b-41d4-a716-446655440000",
			"starts_at":  "2026-08-01T08:00:00Z",
			"ends_at":    "2026-08-01T17:00:00Z",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateVehicleBooking(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateVehicleBooking_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/bookings",
		jsonBody(t, map[string]interface{}{
			"vehicle_id": "550e8400-e29b-41d4-a716-446655440000",
			"user_id":    "660e8400-e29b-41d4-a716-446655440000",
			"starts_at":  "2026-08-01T08:00:00Z",
			"ends_at":    "2026-08-01T17:00:00Z",
		}))
	req = withUserID(req, "user-123")
	routes.HandleCreateVehicleBooking(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateVehicleBooking_MissingEndsAt(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/bookings",
		jsonBody(t, map[string]interface{}{
			"vehicle_id": "550e8400-e29b-41d4-a716-446655440000",
			"user_id":    "660e8400-e29b-41d4-a716-446655440000",
			"starts_at":  "2026-08-01T08:00:00Z",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateVehicleBooking(rec, req)
	assertValidationError(t, rec, "ends_at")
}

func TestHandleCreateVehicleBooking_InvalidVehicleID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/bookings",
		jsonBody(t, map[string]interface{}{
			"vehicle_id": "not-a-uuid",
			"user_id":    "660e8400-e29b-41d4-a716-446655440000",
			"starts_at":  "2026-08-01T08:00:00Z",
			"ends_at":    "2026-08-01T17:00:00Z",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateVehicleBooking(rec, req)
	assertValidationError(t, rec, "vehicle_id")
}

// The request body carries no validation for starts_at coming after ends_at
// (updateVehicleBookingRequest/createVehicleBookingRequest, route_fuhrpark.go:
// 1736-1749, only require the two fields non-empty) — a reversed booking
// period is business logic the fuhrpark service must reject, and there is no
// bufconn stub for it in this package (same boundary as every prior gateway
// coverage unit this run). This proves a start-after-end request still parses
// and reaches the RPC layer instead of being rejected locally.
func TestHandleCreateVehicleBooking_ReversedPeriodReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/bookings",
		jsonBody(t, map[string]interface{}{
			"vehicle_id": "550e8400-e29b-41d4-a716-446655440000",
			"user_id":    "660e8400-e29b-41d4-a716-446655440000",
			"starts_at":  "2026-08-01T17:00:00Z",
			"ends_at":    "2026-08-01T08:00:00Z",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateVehicleBooking(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateVehicleBooking
// ============================================================================

func TestHandleUpdateVehicleBooking_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/bookings/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"purpose": "updated"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateVehicleBooking(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateVehicleBooking_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/bookings/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"purpose": "updated"}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateVehicleBooking(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateVehicleBooking_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/bookings/not-a-uuid",
		jsonBody(t, map[string]interface{}{"purpose": "updated"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateVehicleBooking(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateVehicleBooking_InvalidStatus(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/bookings/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"status": "not-a-real-status"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateVehicleBooking(rec, req)
	assertValidationError(t, rec, "status")
}

// Same boundary as HandleCreateVehicleBooking above: a reversed booking
// period passes gateway validation untouched and reaches the RPC layer.
func TestHandleUpdateVehicleBooking_ReversedPeriodReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/bookings/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{
			"starts_at": "2026-08-01T17:00:00Z",
			"ends_at":   "2026-08-01T08:00:00Z",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateVehicleBooking(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteVehicleBooking
// ============================================================================

func TestHandleDeleteVehicleBooking_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/bookings/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteVehicleBooking(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteVehicleBooking_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/bookings/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteVehicleBooking(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteVehicleBooking_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/bookings/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteVehicleBooking(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteVehicleBooking_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/bookings/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteVehicleBooking(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
