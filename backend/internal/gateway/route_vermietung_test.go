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

// --- HandleGetObject ---

func TestHandleGetObject_ServiceUnavailable(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleGetObject)
}

func TestHandleGetObject_InvalidIDUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/objects/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetObject(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetObject_ReachesRPC(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/objects/"+testVermietungObjectID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungObjectID)
	routes.HandleGetObject(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateObject ---

func TestHandleUpdateObject_ServiceUnavailable(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleUpdateObject)
}

func TestHandleUpdateObject_InvalidIDUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/vermietung/objects/not-a-uuid", jsonBody(t, map[string]interface{}{
		"name": "Neuer Name",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateObject(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateObject_InvalidJSON(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/vermietung/objects/"+testVermietungObjectID, invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungObjectID)
	routes.HandleUpdateObject(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateObject_ReachesRPC(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	newRate := 99.0
	req := httptest.NewRequest("PATCH", "/api/v1/vermietung/objects/"+testVermietungObjectID, jsonBody(t, map[string]interface{}{
		"name":       "Neuer Name",
		"daily_rate": newRate,
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungObjectID)
	routes.HandleUpdateObject(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteObject ---
//
// The gateway handler is a thin Parse/Call/Respond wrapper with no
// referential-integrity guard of its own — whether deleting an object with a
// running rental is allowed is entirely a service-layer decision. Checked at
// that layer (service.go DeleteObject / mockRepository in service_test.go):
// DeleteObject only verifies the object exists, then soft-deletes it
// unconditionally, with no check for rentals still referencing it. This is a
// VERIFIED finding (see TestService_DeleteObject_ActiveRental_NoReferentialIntegrityGuard
// in internal/vermietung/service_test.go), filed as its own fix-unit (see
// BACKLOG.yml), not fixed here — a coverage unit changes no behaviour.

func TestHandleDeleteObject_ServiceUnavailable(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleDeleteObject)
}

func TestHandleDeleteObject_InvalidIDUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/vermietung/objects/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteObject(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteObject_ReachesRPC(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/vermietung/objects/"+testVermietungObjectID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungObjectID)
	routes.HandleDeleteObject(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetRental ---

func TestHandleGetRental_ServiceUnavailable(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleGetRental)
}

func TestHandleGetRental_InvalidIDUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/rentals/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetRental(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetRental_ReachesRPC(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/rentals/"+testVermietungRentalID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungRentalID)
	routes.HandleGetRental(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListRentals ---

func TestHandleListRentals_ServiceUnavailable(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleListRentals)
}

func TestHandleListRentals_ReachesRPC(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/rentals?object_id="+testVermietungObjectID+
		"&status=active&from=2026-07-01T00:00:00Z&to=2026-07-31T00:00:00Z", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListRentals(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListRentals_UnparsableFromToIgnored documents that unparsable
// from/to query params are silently dropped (the grpc request field stays
// unset) rather than rejected — mirrors HandleGetRentalCalendar's year/month
// handling.
func TestHandleListRentals_UnparsableFromToIgnored(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/rentals?from=gestern&to=morgen", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListRentals(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleSaveRentalSignature ---
//
// Same signature pattern documented at cov-gateway-rapporte-lines-attachments-export
// (route_rapporte_test.go): the handler is a thin Parse/Call/Respond wrapper
// with no read-before-write. Whether a second signature on the same rental is
// allowed is entirely a service/repo decision. Checked at that layer
// (postgres_repository.go SaveSignature, backend/internal/vermietung/postgres_repository.go:504):
// the UPDATE has no "AND signature_data IS NULL" guard and no rental-status
// check, so a rental that is already signed accepts a second SaveSignature
// call and silently overwrites signature_data/signed_at/signed_by — the same
// gap as rapporte's HandleSaveReportSignature. VERIFIED finding, see
// TestSaveRentalSignature_OverwritesExistingSignatureWithoutGuard in
// internal/vermietung/signature_test.go, filed as its own fix-unit (see
// BACKLOG.yml), not fixed here — a coverage unit changes no behaviour.

func TestHandleSaveRentalSignature_InvalidIDUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/vermietung/rentals/not-a-uuid/signature", jsonBody(t, map[string]interface{}{
		"signature_data": "data:image/png;base64,abc123",
		"signed_by":      "Max Mustermann",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleSaveRentalSignature(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleSaveRentalSignature_MissingSignatureData(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/vermietung/rentals/"+testVermietungRentalID+"/signature", jsonBody(t, map[string]interface{}{
		"signed_by": "Max Mustermann",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungRentalID)
	routes.HandleSaveRentalSignature(rec, req)
	assertValidationError(t, rec, "signature_data")
}

func TestHandleSaveRentalSignature_MissingSignedBy(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/vermietung/rentals/"+testVermietungRentalID+"/signature", jsonBody(t, map[string]interface{}{
		"signature_data": "data:image/png;base64,abc123",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungRentalID)
	routes.HandleSaveRentalSignature(rec, req)
	assertValidationError(t, rec, "signed_by")
}

func TestHandleSaveRentalSignature_InvalidJSON(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/vermietung/rentals/"+testVermietungRentalID+"/signature", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungRentalID)
	routes.HandleSaveRentalSignature(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSaveRentalSignature_MissingTenant(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/vermietung/rentals/"+testVermietungRentalID+"/signature", jsonBody(t, map[string]interface{}{
		"signature_data": "data:image/png;base64,abc123",
		"signed_by":      "Max Mustermann",
	}))
	req = withChiURLParam(req, "id", testVermietungRentalID)
	routes.HandleSaveRentalSignature(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleSaveRentalSignature_ReachesRPCOnResign(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/vermietung/rentals/"+testVermietungRentalID+"/signature", jsonBody(t, map[string]interface{}{
		"signature_data": "data:image/png;base64,secondSignatureOverwritingTheFirst",
		"signed_by":      "Erika Musterfrau",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungRentalID)
	routes.HandleSaveRentalSignature(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListInspections ---

func TestHandleListInspections_ServiceUnavailable(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleListInspections)
}

func TestHandleListInspections_InvalidRentalIDUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/rentals/not-a-uuid/inspections", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListInspections(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListInspections_ReachesRPC(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/rentals/"+testVermietungRentalID+"/inspections", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVermietungRentalID)
	routes.HandleListInspections(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetInspection ---

func TestHandleGetInspection_ServiceUnavailable(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleGetInspection)
}

func TestHandleGetInspection_InvalidIDUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/inspections/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetInspection(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetInspection_ReachesRPC(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	inspectionID := "770e8400-e29b-41d4-a716-446655440002"
	req := httptest.NewRequest("GET", "/api/v1/vermietung/inspections/"+inspectionID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", inspectionID)
	routes.HandleGetInspection(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateInspection ---

func TestHandleUpdateInspection_ServiceUnavailable(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleUpdateInspection)
}

func TestHandleUpdateInspection_InvalidIDUUID(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/vermietung/inspections/not-a-uuid", jsonBody(t, map[string]interface{}{
		"notes": "Kratzer am Heck",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateInspection(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateInspection_InvalidJSON(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	inspectionID := "770e8400-e29b-41d4-a716-446655440002"
	req := httptest.NewRequest("PATCH", "/api/v1/vermietung/inspections/"+inspectionID, invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", inspectionID)
	routes.HandleUpdateInspection(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateInspection_ReachesRPCWithChecklistReplace(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	inspectionID := "770e8400-e29b-41d4-a716-446655440002"
	req := httptest.NewRequest("PATCH", "/api/v1/vermietung/inspections/"+inspectionID, jsonBody(t, map[string]interface{}{
		"notes":             "Rueckgabe geprueft",
		"replace_checklist": true,
		"checklist": []map[string]interface{}{
			{"label": "Lack", "condition": "gut"},
		},
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", inspectionID)
	routes.HandleUpdateInspection(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleExportRentalReport ---

func TestHandleExportRentalReport_ServiceUnavailable(t *testing.T) {
	routes := NewVermietungRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleExportRentalReport)
}

func TestHandleExportRentalReport_ReachesRPCWithDefaultFormat(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/export", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleExportRentalReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleExportRentalReport_ReachesRPCWithExplicitFormat(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vermietung/export?format=json", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleExportRentalReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
