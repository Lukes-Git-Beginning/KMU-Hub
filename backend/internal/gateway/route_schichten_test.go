package gateway

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	schichtenv1 "github.com/kmuhub/kmuhub/proto/schichten/v1"
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

// --- HandleGetShift ---

func TestHandleGetShift_ServiceUnavailable(t *testing.T) {
	routes := NewSchichtenRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleGetShift)
}

func TestHandleGetShift_InvalidIDUUID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/schichten/shifts/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetShift(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetShift_ReachesRPC(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/schichten/shifts/"+testSwapShiftID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSwapShiftID)
	routes.HandleGetShift(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateShift ---

func TestHandleUpdateShift_ServiceUnavailable(t *testing.T) {
	routes := NewSchichtenRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleUpdateShift)
}

func TestHandleUpdateShift_InvalidIDUUID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/schichten/shifts/not-a-uuid", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateShift(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateShift_InvalidStartTime(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/schichten/shifts/"+testSwapShiftID, jsonBody(t, map[string]interface{}{
		"start_time": "gestern",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSwapShiftID)
	routes.HandleUpdateShift(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid start_time")
}

func TestHandleUpdateShift_InvalidEndTime(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/schichten/shifts/"+testSwapShiftID, jsonBody(t, map[string]interface{}{
		"end_time": "morgen",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSwapShiftID)
	routes.HandleUpdateShift(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid end_time")
}

func TestHandleUpdateShift_ReachesRPC(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/schichten/shifts/"+testSwapShiftID, jsonBody(t, map[string]interface{}{
		"title": "Renamed Shift",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSwapShiftID)
	routes.HandleUpdateShift(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteShift ---

func TestHandleDeleteShift_ServiceUnavailable(t *testing.T) {
	routes := NewSchichtenRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleDeleteShift)
}

func TestHandleDeleteShift_InvalidIDUUID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/schichten/shifts/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteShift(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteShift_ReachesRPC(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/schichten/shifts/"+testSwapShiftID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSwapShiftID)
	routes.HandleDeleteShift(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListAssignments ---

func TestHandleListAssignments_ServiceUnavailable(t *testing.T) {
	routes := NewSchichtenRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleListAssignments)
}

func TestHandleListAssignments_InvalidIDUUID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/schichten/shifts/not-a-uuid/assignments", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListAssignments(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListAssignments_ReachesRPC(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/schichten/shifts/"+testSwapShiftID+"/assignments", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSwapShiftID)
	routes.HandleListAssignments(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUnassignEmployee ---

func TestHandleUnassignEmployee_ServiceUnavailable(t *testing.T) {
	routes := NewSchichtenRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleUnassignEmployee)
}

func TestHandleUnassignEmployee_InvalidShiftIDUUID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/schichten/shifts/not-a-uuid/assignments/"+testSwapRequesterID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withChiURLParam(req, "employee_id", testSwapRequesterID)
	routes.HandleUnassignEmployee(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUnassignEmployee_InvalidEmployeeIDUUID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/schichten/shifts/"+testSwapShiftID+"/assignments/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSwapShiftID)
	req = withChiURLParam(req, "employee_id", "not-a-uuid")
	routes.HandleUnassignEmployee(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid employee_id")
}

func TestHandleUnassignEmployee_ReachesRPC(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/schichten/shifts/"+testSwapShiftID+"/assignments/"+testSwapRequesterID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSwapShiftID)
	req = withChiURLParam(req, "employee_id", testSwapRequesterID)
	routes.HandleUnassignEmployee(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateTemplate ---

func TestHandleSchichtenUpdateTemplate_ServiceUnavailable(t *testing.T) {
	routes := NewSchichtenRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleUpdateTemplate)
}

func TestHandleSchichtenUpdateTemplate_InvalidIDUUID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/schichten/templates/not-a-uuid", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateTemplate(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleSchichtenUpdateTemplate_ReachesRPC(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/schichten/templates/"+testSwapShiftID, jsonBody(t, map[string]interface{}{
		"name": "Renamed Template",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSwapShiftID)
	routes.HandleUpdateTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteTemplate ---

func TestHandleSchichtenDeleteTemplate_ServiceUnavailable(t *testing.T) {
	routes := NewSchichtenRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleDeleteTemplate)
}

func TestHandleSchichtenDeleteTemplate_InvalidIDUUID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/schichten/templates/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteTemplate(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleSchichtenDeleteTemplate_ReachesRPC(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/schichten/templates/"+testSwapShiftID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSwapShiftID)
	routes.HandleDeleteTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleApplyTemplate ---

func TestHandleApplyTemplate_ServiceUnavailable(t *testing.T) {
	routes := NewSchichtenRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleApplyTemplate)
}

func TestHandleApplyTemplate_InvalidIDUUID(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/templates/not-a-uuid/apply", jsonBody(t, map[string]interface{}{
		"range_start": "2026-06-01T00:00:00Z",
		"range_end":   "2026-06-07T00:00:00Z",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleApplyTemplate(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleApplyTemplate_MissingRangeStart(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/templates/"+testSwapShiftID+"/apply", jsonBody(t, map[string]interface{}{
		"range_end": "2026-06-07T00:00:00Z",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSwapShiftID)
	routes.HandleApplyTemplate(rec, req)
	assertValidationError(t, rec, "range_start")
}

func TestHandleApplyTemplate_InvalidRangeEndFormat(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/templates/"+testSwapShiftID+"/apply", jsonBody(t, map[string]interface{}{
		"range_start": "2026-06-01T00:00:00Z",
		"range_end":   "naechste-woche",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSwapShiftID)
	routes.HandleApplyTemplate(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid range_end")
}

func TestHandleApplyTemplate_ReachesRPC(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/schichten/templates/"+testSwapShiftID+"/apply", jsonBody(t, map[string]interface{}{
		"range_start": "2026-06-01T00:00:00Z",
		"range_end":   "2026-06-07T00:00:00Z",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSwapShiftID)
	routes.HandleApplyTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetShiftStats ---

func TestHandleGetShiftStats_ServiceUnavailable(t *testing.T) {
	routes := NewSchichtenRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleGetShiftStats)
}

func TestHandleGetShiftStats_ReachesRPC(t *testing.T) {
	routes := NewSchichtenRoutes(registryWithService("schichten"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/schichten/stats", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetShiftStats(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// statsCaptureServer records the last GetShiftStatsRequest it received so a
// test can observe what the gateway handler put on the wire.
type statsCaptureServer struct {
	schichtenv1.UnimplementedSchichtenServiceServer
	mu  sync.Mutex
	got *schichtenv1.GetShiftStatsRequest
}

func (s *statsCaptureServer) GetShiftStats(_ context.Context, req *schichtenv1.GetShiftStatsRequest) (*schichtenv1.ShiftStatsResponse, error) {
	s.mu.Lock()
	s.got = req
	s.mu.Unlock()
	return &schichtenv1.ShiftStatsResponse{}, nil
}

// take returns the last recorded request (nil if none) and clears it.
func (s *statsCaptureServer) take() *schichtenv1.GetShiftStatsRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.got
	s.got = nil
	return r
}

// statsRoutesAgainstFake wires SchichtenRoutes to a real in-process gRPC server.
func statsRoutesAgainstFake(t *testing.T) (*SchichtenRoutes, *statsCaptureServer) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	fake := &statsCaptureServer{}
	srv := grpc.NewServer()
	schichtenv1.RegisterSchichtenServiceServer(srv, fake)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("schichten", lis.Addr().String())
	return NewSchichtenRoutes(reg, nil), fake
}

func TestHandleGetShiftStats_ForwardsFromToToRPC(t *testing.T) {
	routes, fake := statsRoutesAgainstFake(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/schichten/stats?from=2026-01-01T00:00:00Z&to=2026-01-31T00:00:00Z", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetShiftStats(rec, req)
	assertStatus(t, rec, http.StatusOK)

	got := fake.take()
	if got == nil {
		t.Fatal("RPC was not called")
	}
	if got, want := got.GetFrom().AsTime(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("from = %v, want %v", got, want)
	}
	if got, want := got.GetTo().AsTime(), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("to = %v, want %v", got, want)
	}
}

func TestHandleGetShiftStats_NoRangeStaysUnbounded(t *testing.T) {
	routes, fake := statsRoutesAgainstFake(t)
	for _, target := range []string{"/api/v1/schichten/stats", "/api/v1/schichten/stats?from=&to="} {
		rec := httptest.NewRecorder()
		req := withAuth(httptest.NewRequest("GET", target, nil), "user-123", testTenantID)
		routes.HandleGetShiftStats(rec, req)
		assertStatus(t, rec, http.StatusOK)
		if got := fake.take(); got == nil || got.From != nil || got.To != nil {
			t.Errorf("%s: want RPC with nil from/to, got %v", target, got)
		}
	}
}

func TestHandleGetShiftStats_InvalidRange_BadRequest(t *testing.T) {
	routes, fake := statsRoutesAgainstFake(t)
	for _, target := range []string{
		"/api/v1/schichten/stats?from=yesterday",
		"/api/v1/schichten/stats?to=2026-01-31",
	} {
		rec := httptest.NewRecorder()
		req := withAuth(httptest.NewRequest("GET", target, nil), "user-123", testTenantID)
		routes.HandleGetShiftStats(rec, req)
		assertStatus(t, rec, http.StatusBadRequest)
	}
	if got := fake.take(); got != nil {
		t.Errorf("RPC must not be called on invalid input, got %v", got)
	}
}
