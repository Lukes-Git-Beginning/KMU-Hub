package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// route_hr_time_tracking_test.go covers the second HR gateway unit: the
// time-tracking approval flow, balances, and the categories/templates/projects
// catalogues. Permission-guard wiring for HandleApproveCorrection and
// HandleGetTeamTime is already pinned by TestCapabilityGuards_AdditiveWiring
// in route_capability_guard_test.go (catalogue keys "zeiterfassung:corrections:approve"
// and "zeiterfassung:team:view") — not duplicated here.

// ============================================================================
// HandleCreateManualEntry
//
// The Idempotency-Key/replay path is covered end-to-end against a real DB in
// route_hr_manual_entry_idempotency_db_test.go. This section covers the
// handler's own validation and error paths that db test does not exercise.
// ============================================================================

func TestHandleCreateManualEntry_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/entries", jsonBody(t, map[string]interface{}{
		"clock_in":  "2026-08-20T08:00:00Z",
		"clock_out": "2026-08-20T16:00:00Z",
	}))
	req.Header.Set("Idempotency-Key", "key-1")
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateManualEntry(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateManualEntry_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/entries", jsonBody(t, map[string]interface{}{
		"clock_in":  "2026-08-20T08:00:00Z",
		"clock_out": "2026-08-20T16:00:00Z",
	}))
	req.Header.Set("Idempotency-Key", "key-1")
	req = withUserID(req, "user-123")
	routes.HandleCreateManualEntry(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateManualEntry_MissingIdempotencyKey(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/entries", jsonBody(t, map[string]interface{}{
		"clock_in":  "2026-08-20T08:00:00Z",
		"clock_out": "2026-08-20T16:00:00Z",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateManualEntry(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "Idempotency-Key")
}

func TestHandleCreateManualEntry_InvalidClockInFormat(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/entries", jsonBody(t, map[string]interface{}{
		"clock_in":  "not-a-timestamp",
		"clock_out": "2026-08-20T16:00:00Z",
	}))
	req.Header.Set("Idempotency-Key", "key-1")
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateManualEntry(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "clock_in")
}

func TestHandleCreateManualEntry_MissingRequiredFields(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/entries", jsonBody(t, map[string]interface{}{}))
	req.Header.Set("Idempotency-Key", "key-1")
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateManualEntry(rec, req)
	assertValidationError(t, rec, "clock_in")
}

// ============================================================================
// HandleApproveCorrection
// ============================================================================

func TestHandleApproveCorrection_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/corrections/550e8400-e29b-41d4-a716-446655440000/approve", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleApproveCorrection(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleApproveCorrection_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/corrections/550e8400-e29b-41d4-a716-446655440000/approve", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleApproveCorrection(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListWorkTimeEntries
// ============================================================================

func TestHandleListWorkTimeEntries_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/entries", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListWorkTimeEntries(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListWorkTimeEntries_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/entries", nil)
	req = withUserID(req, "user-123")
	routes.HandleListWorkTimeEntries(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListWorkTimeEntries_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/entries?employee_id=550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListWorkTimeEntries(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetWorkTimeStatus
//
// Unlike most handlers, this one never propagates a downstream RPC error to
// the caller: it composes a best-effort status from two RPCs and silently
// falls back to zero-value defaults if either fails, always answering 200.
// Only a client construction failure (service not registered) reaches the
// caller as an error.
// ============================================================================

func TestHandleGetWorkTimeStatus_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/status", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetWorkTimeStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetWorkTimeStatus_RPCFailureStillReturnsDefaults(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/status", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetWorkTimeStatus(rec, req)
	assertStatus(t, rec, http.StatusOK)
}

// ============================================================================
// HandleGetMyWeekStatus
// ============================================================================

func TestHandleGetMyWeekStatus_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/weeks/status", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetMyWeekStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetMyWeekStatus_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/weeks/status", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetMyWeekStatus(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetMyWeekStatus_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/weeks/status?week_start=2026-08-17", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetMyWeekStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetTimeBalance
// ============================================================================

func TestHandleGetTimeBalance_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/balance", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetTimeBalance(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetTimeBalance_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/balance", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetTimeBalance(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetTimeBalance_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/balance", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetTimeBalance(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetTeamTime
// ============================================================================

func TestHandleGetTeamTime_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/team", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetTeamTime(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetTeamTime_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/team", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetTeamTime(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// TestHandleGetTeamTime_ReachesRPC also documents the finding behind
// fix-hr-team-time-not-scoped-to-manager (BACKLOG.yml): the request carries
// only tenant_id and week_start, never the caller's own id — GetTeamTimeReq
// has no field to scope by. A "manager" role holder sees every employee of
// the tenant, not a subset tied to them.
func TestHandleGetTeamTime_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/team?week_start=2026-08-17", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetTeamTime(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCreateTimeCategory
// ============================================================================

func TestHandleCreateTimeCategory_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/categories", jsonBody(t, map[string]interface{}{"name": "Meetings"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTimeCategory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateTimeCategory_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/categories", jsonBody(t, map[string]interface{}{"name": "Meetings"}))
	req = withUserID(req, "user-123")
	routes.HandleCreateTimeCategory(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateTimeCategory_MissingName(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/categories", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTimeCategory(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateTimeCategory_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/categories", jsonBody(t, map[string]interface{}{"name": "Meetings"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTimeCategory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateTimeCategory
// ============================================================================

func TestHandleUpdateTimeCategory_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/time/categories/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{"name": "Meetings"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTimeCategory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateTimeCategory_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/time/categories/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{"name": "Meetings"}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTimeCategory(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateTimeCategory_MissingName(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/time/categories/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTimeCategory(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleUpdateTimeCategory_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/time/categories/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{"name": "Meetings"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTimeCategory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteTimeCategory
// ============================================================================

func TestHandleDeleteTimeCategory_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/hr/time/categories/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteTimeCategory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteTimeCategory_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/hr/time/categories/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteTimeCategory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListTimeCategories
// ============================================================================

func TestHandleListTimeCategories_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/categories", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListTimeCategories(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListTimeCategories_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/categories", nil)
	req = withUserID(req, "user-123")
	routes.HandleListTimeCategories(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListTimeCategories_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/categories", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListTimeCategories(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCreateTimeProject
// ============================================================================

func TestHandleCreateTimeProject_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/projects", jsonBody(t, map[string]interface{}{"name": "Website Relaunch"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTimeProject(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateTimeProject_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/projects", jsonBody(t, map[string]interface{}{"name": "Website Relaunch"}))
	req = withUserID(req, "user-123")
	routes.HandleCreateTimeProject(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateTimeProject_MissingName(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/projects", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTimeProject(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateTimeProject_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/projects", jsonBody(t, map[string]interface{}{"name": "Website Relaunch"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTimeProject(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListTimeProjects
// ============================================================================

func TestHandleListTimeProjects_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/projects", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListTimeProjects(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListTimeProjects_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/projects", nil)
	req = withUserID(req, "user-123")
	routes.HandleListTimeProjects(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListTimeProjects_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/projects", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListTimeProjects(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCreateTimeTemplate
// ============================================================================

func TestHandleCreateTimeTemplate_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/templates", jsonBody(t, map[string]interface{}{"name": "Standup"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTimeTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateTimeTemplate_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/templates", jsonBody(t, map[string]interface{}{"name": "Standup"}))
	req = withUserID(req, "user-123")
	routes.HandleCreateTimeTemplate(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateTimeTemplate_MissingName(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/templates", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTimeTemplate(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateTimeTemplate_InvalidCategoryID(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/templates", jsonBody(t, map[string]interface{}{
		"name":        "Standup",
		"category_id": "not-a-uuid",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTimeTemplate(rec, req)
	assertValidationError(t, rec, "category_id")
}

func TestHandleCreateTimeTemplate_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/templates", jsonBody(t, map[string]interface{}{"name": "Standup"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTimeTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteTimeTemplate
// ============================================================================

func TestHandleDeleteTimeTemplate_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/hr/time/templates/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteTimeTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteTimeTemplate_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/hr/time/templates/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteTimeTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListTimeTemplates
// ============================================================================

func TestHandleListTimeTemplates_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/templates", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListTimeTemplates(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListTimeTemplates_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/templates", nil)
	req = withUserID(req, "user-123")
	routes.HandleListTimeTemplates(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListTimeTemplates_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/templates", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListTimeTemplates(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListTimeTemplates_RegisteredRoute is a sanity check that the
// route table under test actually wires this handler under the guard the
// scope names, so the ServiceUnavailable/ReachesRPC pair above exercises the
// same path production traffic takes.
func TestHandleListTimeTemplates_RegisteredRoute(t *testing.T) {
	r := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(r, guardTestAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/templates", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withPermissions(req, "hr:time_template:read")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
