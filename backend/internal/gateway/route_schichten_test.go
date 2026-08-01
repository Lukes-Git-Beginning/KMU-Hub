package gateway

import (
	"net/http/httptest"
	"testing"

	"github.com/kmuhub/kmuhub/internal/featureflag"
)

func newSchichtenRoutes(registry *ServiceRegistry) *SchichtenRoutes {
	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_SCHICHTEN_ENABLED" {
			return "true"
		}
		return ""
	})
	return NewSchichtenRoutes(registry, flags)
}

// SchichtenRoutes.ServiceName() uses "schichten".

func TestSchichtenRoutes_ServiceName(t *testing.T) {
	routes := NewSchichtenRoutes(emptyRegistry(), nil)
	if routes.ServiceName() != "schichten" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "schichten")
	}
}

// --- HandleCreateShift ---

func TestHandleCreateShift_MissingTitle(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/shifts", jsonBody(t, map[string]interface{}{
		"start_time": "2026-06-10T08:00:00Z",
		"end_time":   "2026-06-10T16:00:00Z",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateShift(rec, req)
	assertValidationError(t, rec, "title")
}

func TestHandleCreateShift_MissingStartTime(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/shifts", jsonBody(t, map[string]interface{}{
		"title":    "Morning Shift",
		"end_time": "2026-06-10T16:00:00Z",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateShift(rec, req)
	assertValidationError(t, rec, "start_time")
}

func TestHandleCreateShift_MissingEndTime(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/shifts", jsonBody(t, map[string]interface{}{
		"title":      "Morning Shift",
		"start_time": "2026-06-10T08:00:00Z",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateShift(rec, req)
	assertValidationError(t, rec, "end_time")
}

func TestHandleCreateShift_InvalidCreatedByUUID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	createdBy := "not-a-uuid"
	req := httptest.NewRequest("POST", "/api/v1/schichten/shifts", jsonBody(t, map[string]interface{}{
		"title":      "Morning Shift",
		"start_time": "2026-06-10T08:00:00Z",
		"end_time":   "2026-06-10T16:00:00Z",
		"created_by": createdBy,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateShift(rec, req)
	assertValidationError(t, rec, "created_by")
}

// --- HandleAssignEmployee ---

func TestHandleAssignEmployee_MissingEmployeeID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/shifts/550e8400-e29b-41d4-a716-446655440000/assignments",
		jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAssignEmployee(rec, req)
	assertValidationError(t, rec, "employee_id")
}

func TestHandleAssignEmployee_InvalidEmployeeIDUUID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/shifts/550e8400-e29b-41d4-a716-446655440000/assignments",
		jsonBody(t, map[string]interface{}{
			"employee_id": "not-a-uuid",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAssignEmployee(rec, req)
	assertValidationError(t, rec, "employee_id")
}

// --- HandleCreateTemplate ---

func TestHandleCreateTemplate_MissingName(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/templates", jsonBody(t, map[string]interface{}{
		"day_of_week":       1,
		"start_hour":        8,
		"start_minute":      0,
		"duration_minutes":  480,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTemplate(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateTemplate_InvalidDayOfWeek(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/templates", jsonBody(t, map[string]interface{}{
		"name":             "Weekend Shift",
		"day_of_week":      7, // invalid: must be 0-6
		"start_hour":       8,
		"start_minute":     0,
		"duration_minutes": 480,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTemplate(rec, req)
	assertValidationError(t, rec, "day_of_week")
}

func TestHandleCreateTemplate_ZeroDurationMinutes(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/templates", jsonBody(t, map[string]interface{}{
		"name":             "Short Shift",
		"day_of_week":      1,
		"start_hour":       8,
		"start_minute":     0,
		"duration_minutes": 0, // invalid: must be > 0
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTemplate(rec, req)
	assertValidationError(t, rec, "duration_minutes")
}

// --- HandlePublishShifts ---

func TestHandlePublishShifts_MissingFrom(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/shifts/publish", jsonBody(t, map[string]interface{}{
		"to": "2026-06-17T00:00:00Z",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandlePublishShifts(rec, req)
	assertValidationError(t, rec, "from")
}
