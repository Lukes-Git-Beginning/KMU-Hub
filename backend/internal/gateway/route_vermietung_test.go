package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kmuhub/kmuhub/internal/featureflag"
)

func newVermietungRoutes(registry *ServiceRegistry) *VermietungRoutes {
	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_VERMIETUNG_ENABLED" {
			return "true"
		}
		return ""
	})
	return NewVermietungRoutes(registry, flags)
}

// VermietungRoutes.ServiceName() uses "vermietung".

func TestVermietungRoutes_ServiceName(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	if routes.ServiceName() != "vermietung" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "vermietung")
	}
}

// --- HandleCreateObject ---

func TestHandleCreateObject_MissingName(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vermietung/objects", jsonBody(t, map[string]interface{}{
		"daily_rate": 50.0,
		"deposit":    100.0,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateObject(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateObject_ZeroDailyRate(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vermietung/objects", jsonBody(t, map[string]interface{}{
		"name":       "Drill Machine",
		"daily_rate": 0.0, // invalid: must be > 0
		"deposit":    50.0,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateObject(rec, req)
	assertValidationError(t, rec, "daily_rate")
}

func TestHandleCreateObject_InvalidJSON(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vermietung/objects", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateObject(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

// --- HandleCreateRental ---

func TestHandleCreateRental_MissingObjectID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vermietung/rentals", jsonBody(t, map[string]interface{}{
		"renter_name": "Max Mustermann",
		"start_date":  "2026-07-01T00:00:00Z",
		"end_date":    "2026-07-07T00:00:00Z",
		"total_price": 350.0,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateRental(rec, req)
	assertValidationError(t, rec, "object_id")
}

func TestHandleCreateRental_InvalidObjectIDUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vermietung/rentals", jsonBody(t, map[string]interface{}{
		"object_id":   "not-a-uuid",
		"renter_name": "Max Mustermann",
		"start_date":  "2026-07-01T00:00:00Z",
		"end_date":    "2026-07-07T00:00:00Z",
		"total_price": 350.0,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateRental(rec, req)
	assertValidationError(t, rec, "object_id")
}

func TestHandleCreateRental_MissingRenterName(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vermietung/rentals", jsonBody(t, map[string]interface{}{
		"object_id":   "550e8400-e29b-41d4-a716-446655440000",
		"start_date":  "2026-07-01T00:00:00Z",
		"end_date":    "2026-07-07T00:00:00Z",
		"total_price": 350.0,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateRental(rec, req)
	assertValidationError(t, rec, "renter_name")
}

func TestHandleCreateRental_MissingStartDate(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vermietung/rentals", jsonBody(t, map[string]interface{}{
		"object_id":   "550e8400-e29b-41d4-a716-446655440000",
		"renter_name": "Max Mustermann",
		"end_date":    "2026-07-07T00:00:00Z",
		"total_price": 350.0,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateRental(rec, req)
	assertValidationError(t, rec, "start_date")
}

func TestHandleCreateRental_InvalidContactIDUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vermietung/rentals", jsonBody(t, map[string]interface{}{
		"object_id":   "550e8400-e29b-41d4-a716-446655440000",
		"renter_name": "Max Mustermann",
		"start_date":  "2026-07-01T00:00:00Z",
		"end_date":    "2026-07-07T00:00:00Z",
		"total_price": 350.0,
		"contact_id":  "not-a-uuid",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateRental(rec, req)
	assertValidationError(t, rec, "contact_id")
}

// --- HandleCreateInspection ---

func TestHandleCreateInspection_MissingKind(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vermietung/rentals/550e8400-e29b-41d4-a716-446655440000/inspections",
		jsonBody(t, map[string]interface{}{
			"notes": "All good",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateInspection(rec, req)
	assertValidationError(t, rec, "kind")
}

func TestHandleCreateInspection_InvalidKind(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vermietung/rentals/550e8400-e29b-41d4-a716-446655440000/inspections",
		jsonBody(t, map[string]interface{}{
			"kind":  "inspection", // not in allowed set: handover|return
			"notes": "checking",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateInspection(rec, req)
	assertValidationError(t, rec, "kind")
}

func TestHandleCreateInspection_InvalidPerformedByUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	performedBy := "not-a-uuid"
	req := httptest.NewRequest("POST", "/api/v1/vermietung/rentals/550e8400-e29b-41d4-a716-446655440000/inspections",
		jsonBody(t, map[string]interface{}{
			"kind":         "handover",
			"notes":        "item OK",
			"performed_by": performedBy,
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateInspection(rec, req)
	assertValidationError(t, rec, "performed_by")
}

// --- HandleCheckAvailability ---

const (
	testVermietungObjectID = "550e8400-e29b-41d4-a716-446655440000"
	testVermietungRentalID = "660e8400-e29b-41d4-a716-446655440001"
)

func TestHandleCheckAvailability_ServiceUnavailable(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleCheckAvailability)
}

func TestHandleCheckAvailability_InvalidObjectIDUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/objects/not-a-uuid/availability?start_date=2026-07-01T00:00:00Z&end_date=2026-07-07T00:00:00Z", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCheckAvailability(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleCheckAvailability_MissingDates(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/objects/"+testVermietungObjectID+"/availability", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungObjectID)
	routes.HandleCheckAvailability(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "start_date and end_date are required")
}

func TestHandleCheckAvailability_InvalidStartDateFormat(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/objects/"+testVermietungObjectID+"/availability?start_date=gestern&end_date=2026-07-07T00:00:00Z", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungObjectID)
	routes.HandleCheckAvailability(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid start_date format")
}

func TestHandleCheckAvailability_InvalidEndDateFormat(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/objects/"+testVermietungObjectID+"/availability?start_date=2026-07-01T00:00:00Z&end_date=morgen", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungObjectID)
	routes.HandleCheckAvailability(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid end_date format")
}

// TestHandleCheckAvailability_OverlappingRange_ReachesRPC documents that the
// overlap check itself lives in the vermietung service, not the gateway: the
// handler forwards a self-overlapping range (start after end) and the
// exclude_rental_id query param straight to CheckAvailability without
// rejecting it locally.
func TestHandleCheckAvailability_OverlappingRange_ReachesRPC(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/objects/"+testVermietungObjectID+
		"/availability?start_date=2026-07-10T00:00:00Z&end_date=2026-07-01T00:00:00Z&exclude_rental_id="+testVermietungRentalID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungObjectID)
	routes.HandleCheckAvailability(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleStartRental / HandleEndRental ---

func TestHandleStartRental_ServiceUnavailable(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleStartRental)
}

func TestHandleStartRental_InvalidRentalIDUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vermietung/rentals/not-a-uuid/start", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleStartRental(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid id")
}

// TestHandleStartRental_InvalidStatusTransition_ReachesRPC documents that the
// status-transition guard (e.g. starting an already active or cancelled
// rental) lives in the vermietung service: the gateway forwards any
// syntactically valid rental ID straight to StartRental.
func TestHandleStartRental_InvalidStatusTransition_ReachesRPC(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vermietung/rentals/"+testVermietungRentalID+"/start", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungRentalID)
	routes.HandleStartRental(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleEndRental_ServiceUnavailable(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleEndRental)
}

func TestHandleEndRental_InvalidRentalIDUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vermietung/rentals/not-a-uuid/end", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleEndRental(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleEndRental_InvalidStatusTransition_ReachesRPC(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vermietung/rentals/"+testVermietungRentalID+"/end", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungRentalID)
	routes.HandleEndRental(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateRental ---

func TestHandleUpdateRental_ServiceUnavailable(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleUpdateRental)
}

func TestHandleUpdateRental_InvalidRentalIDUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/vermietung/rentals/not-a-uuid", jsonBody(t, map[string]interface{}{
		"renter_name": "Neuer Name",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateRental(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateRental_InvalidJSON(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/vermietung/rentals/"+testVermietungRentalID, invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungRentalID)
	routes.HandleUpdateRental(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateRental_InvalidStartDateFormat(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/vermietung/rentals/"+testVermietungRentalID, jsonBody(t, map[string]interface{}{
		"start_date": "gestern",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungRentalID)
	routes.HandleUpdateRental(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid start_date format")
}

func TestHandleUpdateRental_InvalidEndDateFormat(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/vermietung/rentals/"+testVermietungRentalID, jsonBody(t, map[string]interface{}{
		"end_date": "morgen",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungRentalID)
	routes.HandleUpdateRental(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid end_date format")
}

func TestHandleUpdateRental_ReachesRPC(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/vermietung/rentals/"+testVermietungRentalID, jsonBody(t, map[string]interface{}{
		"renter_name": "Neuer Name",
		"start_date":  "2026-07-01T00:00:00Z",
		"end_date":    "2026-07-07T00:00:00Z",
		"total_price": 400.0,
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungRentalID)
	routes.HandleUpdateRental(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteRental ---

func TestHandleDeleteRental_ServiceUnavailable(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleDeleteRental)
}

func TestHandleDeleteRental_InvalidRentalIDUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/vermietung/rentals/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteRental(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteRental_ReachesRPC(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/vermietung/rentals/"+testVermietungRentalID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungRentalID)
	routes.HandleDeleteRental(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetRentalCalendar ---

func TestHandleGetRentalCalendar_ServiceUnavailable(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleGetRentalCalendar)
}

func TestHandleGetRentalCalendar_ReachesRPC(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/calendar?year=2026&month=7", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetRentalCalendar(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleGetRentalCalendar_NonNumericYearMonthIgnored documents that an
// unparsable year/month query param is silently ignored (falls back to 0)
// rather than rejected — the handler still reaches the RPC.
func TestHandleGetRentalCalendar_NonNumericYearMonthIgnored(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/calendar?year=abc&month=xyz", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetRentalCalendar(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
