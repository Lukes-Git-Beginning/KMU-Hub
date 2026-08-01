package gateway

import (
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
