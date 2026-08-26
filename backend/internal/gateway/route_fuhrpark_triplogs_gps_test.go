package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// HandleListTripLogs
// ============================================================================

func TestHandleListTripLogs_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/trip-logs", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListTripLogs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListTripLogs_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/trip-logs", nil)
	req = withUserID(req, "user-123")
	routes.HandleListTripLogs(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListTripLogs_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/trip-logs?vehicle_id=550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListTripLogs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListVehicleTripLogs
// ============================================================================

func TestHandleListVehicleTripLogs_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/trip-logs", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListVehicleTripLogs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListVehicleTripLogs_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/trip-logs", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListVehicleTripLogs(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListVehicleTripLogs_InvalidVehicleIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/not-a-uuid/trip-logs", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListVehicleTripLogs(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListVehicleTripLogs_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/trip-logs", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListVehicleTripLogs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleExportTripLogs
//
// trip_logs.is_private (000193_create_trip_logs.up.sql) is scanned and
// returned by both ListTripLogs and ListTripLogsForExport
// (internal/fuhrpark/postgres_repository.go) but neither query filters on it —
// a driver's private trip is indistinguishable from a business trip in the
// employer-facing Fahrtenbuch export. Not fixable from this gateway-only
// coverage unit (repo/service change, no test double for it here); documented
// as a finding, filed as its own fix-unit at the end of BACKLOG.yml.
// ============================================================================

func TestHandleExportTripLogs_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/trip-logs/export", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleExportTripLogs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleExportTripLogs_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/trip-logs/export", nil)
	req = withUserID(req, "user-123")
	routes.HandleExportTripLogs(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleExportTripLogs_InvalidFormat(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/trip-logs/export?format=xlsx", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleExportTripLogs(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "format must be csv or pdf")
}

func TestHandleExportTripLogs_DefaultFormatReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/trip-logs/export", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleExportTripLogs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleExportTripLogs_PdfFormatReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/trip-logs/export?format=pdf&vehicle_id=550e8400-e29b-41d4-a716-446655440000&from=2026-01-01&to=2026-12-31", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleExportTripLogs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListFuelLogs
// ============================================================================

func TestHandleListFuelLogs_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/fuel-logs", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListFuelLogs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListFuelLogs_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/fuel-logs", nil)
	req = withUserID(req, "user-123")
	routes.HandleListFuelLogs(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListFuelLogs_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/fuel-logs?vehicle_id=550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListFuelLogs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListVehicleFuelLogs
// ============================================================================

func TestHandleListVehicleFuelLogs_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/fuel-logs", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListVehicleFuelLogs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListVehicleFuelLogs_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/fuel-logs", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListVehicleFuelLogs(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListVehicleFuelLogs_InvalidVehicleIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/not-a-uuid/fuel-logs", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListVehicleFuelLogs(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListVehicleFuelLogs_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/fuel-logs", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListVehicleFuelLogs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListServices
// ============================================================================

func TestHandleListServices_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/services", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListServices(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListServices_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/services", nil)
	req = withUserID(req, "user-123")
	routes.HandleListServices(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListServices_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/services?status=scheduled&vehicle_id=550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListServices(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteService
// ============================================================================

func TestHandleDeleteService_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/services/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteService(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteService_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/services/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteService(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteService_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/services/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteService(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteService_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/services/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteService(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListUpcomingServices
// ============================================================================

func TestHandleListUpcomingServices_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/services/upcoming", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListUpcomingServices(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListUpcomingServices_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/services/upcoming", nil)
	req = withUserID(req, "user-123")
	routes.HandleListUpcomingServices(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListUpcomingServices_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/services/upcoming", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListUpcomingServices(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListVehicleBookings
// ============================================================================

func TestHandleListVehicleBookings_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/bookings", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListVehicleBookings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListVehicleBookings_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/bookings", nil)
	req = withUserID(req, "user-123")
	routes.HandleListVehicleBookings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListVehicleBookings_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/bookings?vehicle_id=550e8400-e29b-41d4-a716-446655440000&status=booked", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListVehicleBookings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleIngestGpsPositions
//
// IngestGpsPositionsRequest only validates len(positions) >= 1
// (route_fuhrpark.go createVehicleDocumentRequest sibling ingestGpsPositionsRequest,
// "required,min=1") — there is no upper bound, and neither the gateway, the
// service (internal/fuhrpark/service.go IngestGpsPositions) nor the repository
// (postgres_repository.go IngestGpsPositions) verify that vehicle_id belongs to
// the caller's tenant before inserting: gps_positions.vehicle_id only has an FK
// to vehicles(id), which holds every tenant's vehicles in one table. A caller
// can plant positions against another tenant's vehicle_id, tagged with their
// own tenant_id. Documented as a finding (LargeBatchReachesRPC proves no local
// rejection at any size), filed as its own fix-unit for the tenant-ownership
// gap; the missing upper bound is a resource question per this unit's scope
// note, not fixed here.
// ============================================================================

func TestHandleIngestGpsPositions_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/gps/ingest?vehicle_id=550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{
			"positions": []map[string]interface{}{{"lat": 52.5, "lng": 13.4, "recorded_at": "2026-08-01T12:00:00Z"}},
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleIngestGpsPositions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleIngestGpsPositions_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/gps/ingest?vehicle_id=550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{
			"positions": []map[string]interface{}{{"lat": 52.5, "lng": 13.4, "recorded_at": "2026-08-01T12:00:00Z"}},
		}))
	req = withUserID(req, "user-123")
	routes.HandleIngestGpsPositions(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleIngestGpsPositions_MissingPositions(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/gps/ingest?vehicle_id=550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"positions": []map[string]interface{}{}}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleIngestGpsPositions(rec, req)
	assertValidationError(t, rec, "positions")
}

// Body validation runs before the vehicle_id query-param check
// (route_fuhrpark.go HandleIngestGpsPositions: decodeAndValidate first, then
// the vehicle_id lookup) — a valid body with no vehicle_id in the query still
// gets rejected, just with the handler's own 400 rather than a validation_failed
// shape.
func TestHandleIngestGpsPositions_MissingVehicleIDQueryParam(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/gps/ingest",
		jsonBody(t, map[string]interface{}{
			"positions": []map[string]interface{}{{"lat": 52.5, "lng": 13.4, "recorded_at": "2026-08-01T12:00:00Z"}},
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleIngestGpsPositions(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "vehicle_id query param required")
}

func TestHandleIngestGpsPositions_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/gps/ingest?vehicle_id=550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{
			"positions": []map[string]interface{}{
				{"lat": 52.5, "lng": 13.4, "speed_kmh": 42.0, "recorded_at": "2026-08-01T12:00:00Z"},
			},
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleIngestGpsPositions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// Proves the batch has no local upper bound: 5000 positions pass validation
// (only "required,min=1" is enforced) and reach the RPC layer unrejected.
func TestHandleIngestGpsPositions_LargeBatchReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	positions := make([]map[string]interface{}, 5000)
	for i := range positions {
		positions[i] = map[string]interface{}{"lat": 52.5, "lng": 13.4, "recorded_at": "2026-08-01T12:00:00Z"}
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/gps/ingest?vehicle_id=550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"positions": positions}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleIngestGpsPositions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetVehicleRoutes
// ============================================================================

func TestHandleGetVehicleRoutes_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/gps/routes", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetVehicleRoutes(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetVehicleRoutes_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/gps/routes", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetVehicleRoutes(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetVehicleRoutes_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/gps/routes", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetVehicleRoutes(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetVehicleRoutes_WithVehicleFilterReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/gps/routes?vehicle_id=550e8400-e29b-41d4-a716-446655440000&date_from=2026-08-01&date_to=2026-08-07", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetVehicleRoutes(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetGpsPositions
// ============================================================================

func TestHandleGetGpsPositions_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/gps/positions?vehicle_id=550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetGpsPositions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetGpsPositions_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/gps/positions?vehicle_id=550e8400-e29b-41d4-a716-446655440000", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetGpsPositions(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetGpsPositions_MissingVehicleIDQueryParam(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/gps/positions", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetGpsPositions(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "vehicle_id query param required")
}

func TestHandleGetGpsPositions_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/gps/positions?vehicle_id=550e8400-e29b-41d4-a716-446655440000&from=2026-08-01T00:00:00Z&to=2026-08-02T00:00:00Z", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetGpsPositions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleExportVehicleReport
// ============================================================================

func TestHandleExportVehicleReport_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/export", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleExportVehicleReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleExportVehicleReport_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/export", nil)
	req = withUserID(req, "user-123")
	routes.HandleExportVehicleReport(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleExportVehicleReport_DefaultFormatReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/export", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleExportVehicleReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleExportVehicleReport_ExplicitFormatReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/export?format=pdf", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleExportVehicleReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
