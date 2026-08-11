package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// ============================================================================
// HandleCreateLeaveRequest
// ============================================================================

func TestHandleCreateLeaveRequest_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests",
		jsonBody(t, map[string]interface{}{"start_date": "2026-08-20", "end_date": "2026-08-21"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateLeaveRequest_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests",
		jsonBody(t, map[string]interface{}{"start_date": "2026-08-20", "end_date": "2026-08-21"}))
	req = withUserID(req, "user-123")
	routes.HandleCreateLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateLeaveRequest_InvalidJSON(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateLeaveRequest_InvalidLeaveTypeID(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests",
		jsonBody(t, map[string]interface{}{
			"leave_type_id": "not-a-uuid",
			"start_date":    "2026-08-20",
			"end_date":      "2026-08-21",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateLeaveRequest(rec, req)
	assertValidationError(t, rec, "leave_type_id")
}

func TestHandleCreateLeaveRequest_InvalidHalfDayPeriod(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests",
		jsonBody(t, map[string]interface{}{
			"start_date":            "2026-08-20",
			"end_date":              "2026-08-20",
			"is_half_day_start":     true,
			"half_day_period_start": "noon",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateLeaveRequest(rec, req)
	assertValidationError(t, rec, "half_day_period_start")
}

func TestHandleCreateLeaveRequest_ValidRequestReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests",
		jsonBody(t, map[string]interface{}{
			"leave_type_id":         "550e8400-e29b-41d4-a716-446655440000",
			"start_date":            "2026-08-20",
			"end_date":              "2026-08-20",
			"is_half_day_start":     true,
			"half_day_period_start": "afternoon",
			"reason":                "Arzttermin",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListLeaveRequests
// ============================================================================

func TestHandleListLeaveRequests_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/requests", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListLeaveRequests(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListLeaveRequests_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/requests", nil)
	req = withUserID(req, "user-123")
	routes.HandleListLeaveRequests(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// TestHandleListLeaveRequests_FilterCombinations exercises every query-string
// filter branch (employee_id, status, date range, pagination). None of these
// can be observed reaching the proto request without a fake HRServiceClient
// (no bufconn stub exists for this package, same boundary as every other
// gateway coverage unit this run) — each case is proven to parse past every
// filter branch by reaching the RPC layer (503), not panicking.
func TestHandleListLeaveRequests_FilterCombinations(t *testing.T) {
	cases := []struct {
		name  string
		query string
	}{
		{"empty", ""},
		{"employee_id", "?employee_id=550e8400-e29b-41d4-a716-446655440000"},
		{"status", "?status=approved"},
		{"date_range", "?start_date=2026-08-01&end_date=2026-08-31"},
		{"pagination", "?page=2&page_size=25"},
		{"all_combined", "?employee_id=550e8400-e29b-41d4-a716-446655440000&status=pending&start_date=2026-08-01&end_date=2026-08-31&page=2&page_size=25"},
	}
	routes := NewHRRoutes(registryWithService("biz"), nil)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/requests"+tc.query, nil)
			req = withAuth(req, "user-123", testTenantID)
			routes.HandleListLeaveRequests(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// ============================================================================
// HandleGetLeaveRequest
// ============================================================================

func TestHandleGetLeaveRequest_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/requests/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetLeaveRequest_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/requests/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleApproveLeaveRequest / HandleRejectLeaveRequest / HandleCancelLeaveRequest
//
// Unlike the CRM/Automation handlers, these three read chi.URLParam(r, "id")
// straight into the gRPC request without a local validateUUIDParam check —
// there is no "invalid id" 400 to observe here. A missing or malformed id is
// opaque to the handler and only ever surfaces as a gRPC error from the HR
// service itself, so "missing/invalid id" and "service unavailable" collapse
// into the same observable outcome in a unit test without a live server.
// ============================================================================

func TestHandleApproveLeaveRequest_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests/550e8400-e29b-41d4-a716-446655440000/approve",
		jsonBody(t, map[string]interface{}{"comment": "ok"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleApproveLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleApproveLeaveRequest_InvalidJSON(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests/550e8400-e29b-41d4-a716-446655440000/approve", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleApproveLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleApproveLeaveRequest_MissingIDReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests//approve",
		jsonBody(t, map[string]interface{}{"comment": "ok"}))
	req = withAuth(req, "user-123", testTenantID)
	// deliberately no withChiURLParam("id", ...) — id resolves to the zero value
	routes.HandleApproveLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleApproveLeaveRequest_ValidIDReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests/550e8400-e29b-41d4-a716-446655440000/approve",
		jsonBody(t, map[string]interface{}{"comment": "ok"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleApproveLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRejectLeaveRequest_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests/550e8400-e29b-41d4-a716-446655440000/reject",
		jsonBody(t, map[string]interface{}{"comment": "kein Personal frei"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRejectLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRejectLeaveRequest_InvalidJSON(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests/550e8400-e29b-41d4-a716-446655440000/reject", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRejectLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleRejectLeaveRequest_MissingIDReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests//reject",
		jsonBody(t, map[string]interface{}{"comment": "kein Personal frei"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleRejectLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCancelLeaveRequest_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests/550e8400-e29b-41d4-a716-446655440000/cancel", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCancelLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCancelLeaveRequest_ValidIDReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests/550e8400-e29b-41d4-a716-446655440000/cancel", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCancelLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCancelLeaveRequest_MissingIDReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/requests//cancel", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCancelLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetLeaveBalance
// ============================================================================

func TestHandleGetLeaveBalance_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/balance", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetLeaveBalance(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetLeaveBalance_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/balance", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetLeaveBalance(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetLeaveBalance_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/balance", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetLeaveBalance(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListLeaveTypes
// ============================================================================

func TestHandleListLeaveTypes_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/types", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListLeaveTypes(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListLeaveTypes_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/types", nil)
	req = withUserID(req, "user-123")
	routes.HandleListLeaveTypes(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListLeaveTypes_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/types", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListLeaveTypes(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// hrMarshalSlice — wire shape
//
// HandleListLeaveRequests builds its response envelope by hand
// ({"leave_requests": [...], "total": N}) instead of response.ProtoList, so
// the empty-list-not-null guarantee lives in hrMarshalSlice itself. Cannot be
// observed through the handler without a live gRPC server (see the filter
// tests above) — tested directly instead, same approach as
// TestFilterDocumentsByCategory_EmptyResultIsNotNil in
// route_hr_self_documents_test.go.
// ============================================================================

func TestHrMarshalSlice_EmptyResultIsNotNil(t *testing.T) {
	got, err := hrMarshalSlice([]*hrv1.LeaveRequest(nil))
	if err != nil {
		t.Fatalf("hrMarshalSlice returned error: %v", err)
	}
	if got == nil {
		t.Fatal("hrMarshalSlice result is nil; protojson envelope would emit null instead of []")
	}
	if len(got) != 0 {
		t.Fatalf("got %d elements from a nil input, want 0", len(got))
	}
}

func TestHrMarshalSlice_MarshalsEachElement(t *testing.T) {
	reqs := []*hrv1.LeaveRequest{
		{Id: "leave-1", EmployeeId: "emp-1", Status: hrv1.LeaveRequestStatus_LEAVE_PENDING},
		{Id: "leave-2", EmployeeId: "emp-2", Status: hrv1.LeaveRequestStatus_LEAVE_APPROVED},
	}
	got, err := hrMarshalSlice(reqs)
	if err != nil {
		t.Fatalf("hrMarshalSlice returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d elements, want 2", len(got))
	}
	for i, raw := range got {
		if len(raw) == 0 {
			t.Errorf("element %d marshaled to empty JSON", i)
		}
	}
}
