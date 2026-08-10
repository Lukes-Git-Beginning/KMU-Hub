package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// HandleClockIn / HandleClockOut
// ============================================================================

func TestHandleClockIn_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/clock-in", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleClockIn(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleClockIn_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/clock-in", nil)
	req = withUserID(req, "user-123")
	routes.HandleClockIn(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleClockIn_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/clock-in", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleClockIn(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleClockOut_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/clock-out", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleClockOut(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleClockOut_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/clock-out", nil)
	req = withUserID(req, "user-123")
	routes.HandleClockOut(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleClockOut_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/clock-out", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleClockOut(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleBreakStart / HandleBreakEnd
//
// Unlike ClockIn/ClockOut, these two never call getTenantID — they build the
// gRPC request from the user ID alone (route_hr.go:631-669). There is no
// "missing tenant" 401 branch to observe here; a tenant-less request still
// reaches the RPC layer and only 503s because no server is listening. Latent
// gap noted in the journal, not fixed here per the coverage-unit rule
// (no behaviour changes).
// ============================================================================

func TestHandleBreakStart_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/break/start", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleBreakStart(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleBreakStart_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/break/start", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleBreakStart(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleBreakEnd_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/break/end", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleBreakEnd(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleBreakEnd_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/break/end", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleBreakEnd(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetActiveShift
//
// The nil-Entry / nil-ActiveBreak wire-shape branch (route_hr.go:688-717) is
// only reachable once client.GetActiveShift returns a real response — there
// is no bufconn stub for the HR gRPC service in this package (same boundary
// documented in route_hr_leave_test.go for hrMarshalSlice), so it cannot be
// exercised via the handler in a unit test. Proven to reach the RPC layer
// instead, same approach as the ReachesRPC tests above.
// ============================================================================

func TestHandleGetActiveShift_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/active", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetActiveShift(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetActiveShift_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/active", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetActiveShift(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleSubmitWeek
// ============================================================================

func TestHandleSubmitWeek_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/submit",
		jsonBody(t, map[string]interface{}{"week_start": "2026-08-03"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleSubmitWeek(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSubmitWeek_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/submit",
		jsonBody(t, map[string]interface{}{"week_start": "2026-08-03"}))
	req = withUserID(req, "user-123")
	routes.HandleSubmitWeek(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleSubmitWeek_MissingWeekStart(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/submit",
		jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleSubmitWeek(rec, req)
	assertValidationError(t, rec, "week_start")
}

func TestHandleSubmitWeek_InvalidJSON(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/submit", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleSubmitWeek(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSubmitWeek_ValidRequestReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/submit",
		jsonBody(t, map[string]interface{}{"week_start": "2026-08-03"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleSubmitWeek(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleApproveWeek
// ============================================================================

func TestHandleApproveWeek_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/approve",
		jsonBody(t, map[string]interface{}{
			"employee_id": "550e8400-e29b-41d4-a716-446655440000",
			"week_start":  "2026-08-03",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleApproveWeek(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleApproveWeek_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/approve",
		jsonBody(t, map[string]interface{}{
			"employee_id": "550e8400-e29b-41d4-a716-446655440000",
			"week_start":  "2026-08-03",
		}))
	req = withUserID(req, "user-123")
	routes.HandleApproveWeek(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleApproveWeek_InvalidJSON(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/approve", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleApproveWeek(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleApproveWeek_InvalidEmployeeID(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/approve",
		jsonBody(t, map[string]interface{}{
			"employee_id": "not-a-uuid",
			"week_start":  "2026-08-03",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleApproveWeek(rec, req)
	assertValidationError(t, rec, "employee_id")
}

func TestHandleApproveWeek_MissingWeekStart(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/approve",
		jsonBody(t, map[string]interface{}{
			"employee_id": "550e8400-e29b-41d4-a716-446655440000",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleApproveWeek(rec, req)
	assertValidationError(t, rec, "week_start")
}

func TestHandleApproveWeek_ValidRequestReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/approve",
		jsonBody(t, map[string]interface{}{
			"employee_id": "550e8400-e29b-41d4-a716-446655440000",
			"week_start":  "2026-08-03",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleApproveWeek(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleRejectWeek
// ============================================================================

func TestHandleRejectWeek_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/reject",
		jsonBody(t, map[string]interface{}{
			"employee_id": "550e8400-e29b-41d4-a716-446655440000",
			"week_start":  "2026-08-03",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleRejectWeek(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRejectWeek_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/reject",
		jsonBody(t, map[string]interface{}{
			"employee_id": "550e8400-e29b-41d4-a716-446655440000",
			"week_start":  "2026-08-03",
		}))
	req = withUserID(req, "user-123")
	routes.HandleRejectWeek(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleRejectWeek_InvalidJSON(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/reject", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleRejectWeek(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleRejectWeek_InvalidEmployeeID(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/reject",
		jsonBody(t, map[string]interface{}{
			"employee_id": "not-a-uuid",
			"week_start":  "2026-08-03",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleRejectWeek(rec, req)
	assertValidationError(t, rec, "employee_id")
}

func TestHandleRejectWeek_ValidRequestReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/reject",
		jsonBody(t, map[string]interface{}{
			"employee_id":      "550e8400-e29b-41d4-a716-446655440000",
			"week_start":       "2026-08-03",
			"rejection_reason": "Ueberstunden ungeklaert",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleRejectWeek(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleReopenWeek
// ============================================================================

func TestHandleReopenWeek_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/reopen",
		jsonBody(t, map[string]interface{}{
			"employee_id": "550e8400-e29b-41d4-a716-446655440000",
			"week_start":  "2026-08-03",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleReopenWeek(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleReopenWeek_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/reopen",
		jsonBody(t, map[string]interface{}{
			"employee_id": "550e8400-e29b-41d4-a716-446655440000",
			"week_start":  "2026-08-03",
		}))
	req = withUserID(req, "user-123")
	routes.HandleReopenWeek(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleReopenWeek_InvalidJSON(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/reopen", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleReopenWeek(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleReopenWeek_InvalidEmployeeID(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/reopen",
		jsonBody(t, map[string]interface{}{
			"employee_id": "not-a-uuid",
			"week_start":  "2026-08-03",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleReopenWeek(rec, req)
	assertValidationError(t, rec, "employee_id")
}

func TestHandleReopenWeek_ValidRequestReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/weeks/reopen",
		jsonBody(t, map[string]interface{}{
			"employee_id": "550e8400-e29b-41d4-a716-446655440000",
			"week_start":  "2026-08-03",
			"reason":      "Korrektur noetig",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleReopenWeek(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
