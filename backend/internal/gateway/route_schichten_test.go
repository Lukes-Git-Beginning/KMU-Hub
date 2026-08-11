package gateway

import (
	"net/http"
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

// --- HandleCreateSwapRequest ---

const (
	testSwapAssignmentID = "550e8400-e29b-41d4-a716-446655440000"
	testSwapRequesterID  = "660e8400-e29b-41d4-a716-446655440001"
	testSwapPartnerID    = "770e8400-e29b-41d4-a716-446655440002"
	testSwapShiftID      = "880e8400-e29b-41d4-a716-446655440003"
	testSwapRequestID    = "990e8400-e29b-41d4-a716-446655440004"
)

func validCreateSwapRequestBody() map[string]interface{} {
	return map[string]interface{}{
		"assignment_id":            testSwapAssignmentID,
		"requested_by_employee_id": testSwapRequesterID,
		"swap_with_employee_id":    testSwapPartnerID,
		"shift_id":                 testSwapShiftID,
		"reason":                   "Arzttermin",
	}
}

func TestHandleCreateSwapRequest_MissingSwapWithEmployeeID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	body := validCreateSwapRequestBody()
	delete(body, "swap_with_employee_id")
	req := httptest.NewRequest("POST", "/api/v1/schichten/swap-requests", jsonBody(t, body))
	req = withAuth(req, "user-123", testTenantID)
	req.Header.Set("Idempotency-Key", "idem-key-1")
	routes.HandleCreateSwapRequest(rec, req)
	assertValidationError(t, rec, "swap_with_employee_id")
}

func TestHandleCreateSwapRequest_MissingShiftID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	body := validCreateSwapRequestBody()
	delete(body, "shift_id")
	req := httptest.NewRequest("POST", "/api/v1/schichten/swap-requests", jsonBody(t, body))
	req = withAuth(req, "user-123", testTenantID)
	req.Header.Set("Idempotency-Key", "idem-key-1")
	routes.HandleCreateSwapRequest(rec, req)
	assertValidationError(t, rec, "shift_id")
}

func TestHandleCreateSwapRequest_InvalidAssignmentIDUUID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	body := validCreateSwapRequestBody()
	body["assignment_id"] = "not-a-uuid"
	req := httptest.NewRequest("POST", "/api/v1/schichten/swap-requests", jsonBody(t, body))
	req = withAuth(req, "user-123", testTenantID)
	req.Header.Set("Idempotency-Key", "idem-key-1")
	routes.HandleCreateSwapRequest(rec, req)
	assertValidationError(t, rec, "assignment_id")
}

// TestHandleCreateSwapRequest_MissingIdempotencyKey documents that the
// Idempotency-Key header is enforced in the gateway handler itself, after
// decodeAndValidate — a fully valid body without the header is still
// rejected before it reaches the RPC.
func TestHandleCreateSwapRequest_MissingIdempotencyKey(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/swap-requests", jsonBody(t, validCreateSwapRequestBody()))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateSwapRequest(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "Idempotency-Key header is required")
}

func TestHandleCreateSwapRequest_ServiceUnavailable(t *testing.T) {
	routes := NewSchichtenRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleCreateSwapRequest)
}

func TestHandleCreateSwapRequest_ReachesRPC(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/swap-requests", jsonBody(t, validCreateSwapRequestBody()))
	req = withAuth(req, "user-123", testTenantID)
	req.Header.Set("Idempotency-Key", "idem-key-1")
	routes.HandleCreateSwapRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListSwapRequests ---

func TestHandleListSwapRequests_ServiceUnavailable(t *testing.T) {
	routes := NewSchichtenRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleListSwapRequests)
}

func TestHandleListSwapRequests_ReachesRPC(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/schichten/swap-requests?status=pending", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListSwapRequests(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleApproveSwapRequest / HandleRejectSwapRequest ---

func TestHandleApproveSwapRequest_ServiceUnavailable(t *testing.T) {
	routes := NewSchichtenRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleApproveSwapRequest)
}

func TestHandleApproveSwapRequest_InvalidRequestIDUUID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/swap-requests/not-a-uuid/approve", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleApproveSwapRequest(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// TestHandleApproveSwapRequest_AlreadyDecided_ReachesRPC documents that the
// swap-request status guard (rejecting an approve on an already-decided
// request) lives in the schichten service, not the gateway: the handler
// forwards any syntactically valid request ID straight to ApproveSwapRequest.
func TestHandleApproveSwapRequest_AlreadyDecided_ReachesRPC(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/swap-requests/"+testSwapRequestID+"/approve", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSwapRequestID)
	routes.HandleApproveSwapRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRejectSwapRequest_ServiceUnavailable(t *testing.T) {
	routes := NewSchichtenRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleRejectSwapRequest)
}

func TestHandleRejectSwapRequest_InvalidRequestIDUUID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/swap-requests/not-a-uuid/reject", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleRejectSwapRequest(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// TestHandleRejectSwapRequest_AlreadyDecided_ReachesRPC documents the same
// server-side status guard as HandleApproveSwapRequest above, for the reject path.
func TestHandleRejectSwapRequest_AlreadyDecided_ReachesRPC(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/swap-requests/"+testSwapRequestID+"/reject", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSwapRequestID)
	routes.HandleRejectSwapRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleCheckArbzgCompliance ---

func TestHandleCheckArbzgCompliance_ServiceUnavailable(t *testing.T) {
	routes := NewSchichtenRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleCheckArbzgCompliance)
}

func TestHandleCheckArbzgCompliance_MissingEmployeeID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/schichten/compliance?new_shift_start=2026-06-10T08:00:00Z&new_shift_end=2026-06-10T16:00:00Z", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCheckArbzgCompliance(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "employee_id is required")
}

// TestHandleCheckArbzgCompliance_MissingNewShiftStart documents that an
// absent new_shift_start query param is rejected as an invalid RFC3339
// timestamp, not as its own "required" error.
func TestHandleCheckArbzgCompliance_MissingNewShiftStart(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/schichten/compliance?employee_id="+testSwapRequesterID+"&new_shift_end=2026-06-10T16:00:00Z", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCheckArbzgCompliance(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid new_shift_start")
}

func TestHandleCheckArbzgCompliance_InvalidNewShiftStartFormat(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/schichten/compliance?employee_id="+testSwapRequesterID+"&new_shift_start=gestern&new_shift_end=2026-06-10T16:00:00Z", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCheckArbzgCompliance(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid new_shift_start")
}

func TestHandleCheckArbzgCompliance_InvalidNewShiftEndFormat(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/schichten/compliance?employee_id="+testSwapRequesterID+"&new_shift_start=2026-06-10T08:00:00Z&new_shift_end=morgen", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCheckArbzgCompliance(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid new_shift_end")
}

func TestHandleCheckArbzgCompliance_ReachesRPC(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/schichten/compliance?employee_id="+testSwapRequesterID+"&new_shift_start=2026-06-10T08:00:00Z&new_shift_end=2026-06-10T16:00:00Z", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCheckArbzgCompliance(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
