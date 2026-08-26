package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// ============================================================================
// HandleListEmployees
// ============================================================================

func TestHandleListEmployees_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/employees", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListEmployees(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListEmployees_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/employees", nil)
	req = withUserID(req, "user-123")
	routes.HandleListEmployees(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListEmployees_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/employees?department=IT", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListEmployees(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListEmployees_RequiresPermission belongs to the "can a coworker
// reach a colleague's file" question the unit scope asks about: "hr:read" is
// assigned to admin only (migration 000129), so a caller without it never
// reaches the handler at all, regardless of which employee they ask for.
func TestHandleListEmployees_RequiresPermission(t *testing.T) {
	r := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(r, guardTestAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/employees", nil)
	req = withAuth(req, "user-123", testTenantID)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusForbidden)
}

// ============================================================================
// HandleGetSelfProfile
//
// No tenant or ownership check in this handler: it reads only the caller's own
// user ID from the context and asks GetEmployee for that user's profile. There
// is no {id} to substitute a colleague's — the "self" in the name is the
// entire guard, so there is nothing here to test a MissingTenant case against.
// ============================================================================

func TestHandleGetSelfProfile_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/employees/me", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetSelfProfile(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetSelfProfile_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/employees/me", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetSelfProfile(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetEmployee
//
// Same shape as HandleGetSelfProfile: no local tenant/ownership check, {id}
// goes straight into the RPC's UserId. Cross-employee access is stopped by the
// "hr:read" guard (admin only, see TestHandleGetEmployee_RequiresPermission
// below), not by anything in this handler — consistent with the doc comment
// on the /{id}/documents route in RegisterRoutes.
// ============================================================================

func TestHandleGetEmployee_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetEmployee(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetEmployee_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetEmployee(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleGetEmployee_RequiresPermission is one of the >=3 handlers the
// unit scope asks for: a caller asking for a colleague's Akte via /{id}
// without "hr:read" is rejected before the handler ever runs the RPC, no
// matter whose id is in the path.
func TestHandleGetEmployee_RequiresPermission(t *testing.T) {
	r := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(r, guardTestAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusForbidden)
}

// TestHandleGetEmployee_WithPermissionReachesRPC pins the other side: holding
// "hr:read" clears the guard and the request reaches the (unregistered) RPC.
func TestHandleGetEmployee_WithPermissionReachesRPC(t *testing.T) {
	r := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(r, guardTestAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withPermissions(req, "hr:read")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateEmployee
// ============================================================================

func TestHandleUpdateEmployee_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"department": "Sales"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateEmployee(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateEmployee_InvalidJSON(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateEmployee(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateEmployee_InvalidContractType(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"contract_type": "not-a-real-type"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateEmployee(rec, req)
	assertValidationError(t, rec, "contract_type")
}

func TestHandleUpdateEmployee_InvalidWorkDaysPerWeek(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"work_days_per_week": 8}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateEmployee(rec, req)
	assertValidationError(t, rec, "work_days_per_week")
}

func TestHandleUpdateEmployee_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"department": "Sales", "position_title": "Lead"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateEmployee(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCreateEmployee
// ============================================================================

func TestHandleCreateEmployee_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/employees",
		jsonBody(t, map[string]interface{}{
			"user_id":    "550e8400-e29b-41d4-a716-446655440000",
			"start_date": "2026-01-01",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateEmployee(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateEmployee_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/employees",
		jsonBody(t, map[string]interface{}{
			"user_id":    "550e8400-e29b-41d4-a716-446655440000",
			"start_date": "2026-01-01",
		}))
	req = withUserID(req, "user-123")
	routes.HandleCreateEmployee(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateEmployee_MissingUserID(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/employees",
		jsonBody(t, map[string]interface{}{"start_date": "2026-01-01"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateEmployee(rec, req)
	assertValidationError(t, rec, "user_id")
}

func TestHandleCreateEmployee_MissingStartDate(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/employees",
		jsonBody(t, map[string]interface{}{"user_id": "550e8400-e29b-41d4-a716-446655440000"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateEmployee(rec, req)
	assertValidationError(t, rec, "start_date")
}

func TestHandleCreateEmployee_InvalidPhone(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/employees",
		jsonBody(t, map[string]interface{}{
			"user_id":                 "550e8400-e29b-41d4-a716-446655440000",
			"start_date":              "2026-01-01",
			"emergency_contact_phone": "not-a-phone",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateEmployee(rec, req)
	assertValidationError(t, rec, "emergency_contact_phone")
}

func TestHandleCreateEmployee_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/employees",
		jsonBody(t, map[string]interface{}{
			"user_id":    "550e8400-e29b-41d4-a716-446655440000",
			"start_date": "2026-01-01",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateEmployee(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleRecordSickLeave
// ============================================================================

func TestHandleRecordSickLeave_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/sick",
		jsonBody(t, map[string]interface{}{
			"start_date": "2026-06-01",
			"end_date":   "2026-06-03",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleRecordSickLeave(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRecordSickLeave_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/sick",
		jsonBody(t, map[string]interface{}{
			"start_date": "2026-06-01",
			"end_date":   "2026-06-03",
		}))
	req = withUserID(req, "user-123")
	routes.HandleRecordSickLeave(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleRecordSickLeave_InvalidJSON(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/sick", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleRecordSickLeave(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleRecordSickLeave_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/sick",
		jsonBody(t, map[string]interface{}{
			"start_date": "2026-06-01",
			"end_date":   "2026-06-03",
			"notes":      "Grippe",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleRecordSickLeave(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleRecordSickLeave_RequiresPermission is the third handler proving a
// coworker cannot reach someone else's health data through this route family:
// "/leave/sick" is guarded by "hr:write" and RecordSickLeave always records
// against the CALLER's own user ID (middleware.GetUserID), never a path
// parameter — there is no {id} to substitute a colleague's in the first place.
func TestHandleRecordSickLeave_RequiresPermission(t *testing.T) {
	r := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(r, guardTestAuth)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/leave/sick",
		jsonBody(t, map[string]interface{}{
			"start_date": "2026-06-01",
			"end_date":   "2026-06-03",
		}))
	req = withAuth(req, "user-123", testTenantID)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusForbidden)
}

// ============================================================================
// HandleGetEmployeeLeaveBalance
// ============================================================================

func TestHandleGetEmployeeLeaveBalance_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/balance/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "userId", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetEmployeeLeaveBalance(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetEmployeeLeaveBalance_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/balance/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "userId", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetEmployeeLeaveBalance(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetEmployeeLeaveBalance_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/balance/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "userId", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetEmployeeLeaveBalance(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetAbsenceCalendar
// ============================================================================

func TestHandleGetAbsenceCalendar_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/calendar", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetAbsenceCalendar(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetAbsenceCalendar_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/calendar", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetAbsenceCalendar(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetAbsenceCalendar_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/calendar?start_date=2026-06-01&end_date=2026-06-30&department=IT", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetAbsenceCalendar(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListEmployeeDocuments
//
// No local tenant check (relies on RLS policy hr_document_access via ctx, per
// the doc comment on the RPC server side) — same shape as HandleGetEmployee.
// ============================================================================

func TestHandleListEmployeeDocuments_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000/documents", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListEmployeeDocuments(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListEmployeeDocuments_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000/documents", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListEmployeeDocuments(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListEmployeeDocuments_RequiresPermission is the router-level
// counterpart to HandleGetEmployee's: "/{id}/documents" is guarded by
// "hr:read" (admin only), the same key that stops HandleGetEmployee — so a
// coworker without it cannot list a colleague's personnel documents either.
func TestHandleListEmployeeDocuments_RequiresPermission(t *testing.T) {
	r := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(r, guardTestAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000/documents", nil)
	req = withAuth(req, "user-123", testTenantID)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusForbidden)
}

// ============================================================================
// HandleUploadEmployeeDocument
// ============================================================================

func TestHandleUploadEmployeeDocument_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000/documents",
		jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUploadEmployeeDocument(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUploadEmployeeDocument_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000/documents",
		jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUploadEmployeeDocument(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUploadEmployeeDocument_InvalidCategoryID(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000/documents",
		jsonBody(t, map[string]interface{}{"category_id": "not-a-uuid"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUploadEmployeeDocument(rec, req)
	assertValidationError(t, rec, "category_id")
}

func TestHandleUploadEmployeeDocument_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000/documents",
		jsonBody(t, map[string]interface{}{
			"file_id": "660e8400-e29b-41d4-a716-446655440000",
			"notes":   "Arbeitsvertrag",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUploadEmployeeDocument(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListPersonnelDocuments
// ============================================================================

func TestHandleListPersonnelDocuments_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/personnel-documents", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListPersonnelDocuments(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListPersonnelDocuments_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/personnel-documents", nil)
	req = withUserID(req, "user-123")
	routes.HandleListPersonnelDocuments(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListPersonnelDocuments_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/personnel-documents", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListPersonnelDocuments(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCreatePersonnelDocument
// ============================================================================

func TestHandleCreatePersonnelDocument_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/personnel-documents",
		jsonBody(t, map[string]interface{}{
			"employee_id": "550e8400-e29b-41d4-a716-446655440000",
			"title":       "Arbeitsvertrag",
			"category":    "contract",
			"file_name":   "vertrag.pdf",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreatePersonnelDocument(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreatePersonnelDocument_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/personnel-documents",
		jsonBody(t, map[string]interface{}{
			"employee_id": "550e8400-e29b-41d4-a716-446655440000",
			"title":       "Arbeitsvertrag",
			"category":    "contract",
			"file_name":   "vertrag.pdf",
		}))
	req = withUserID(req, "user-123")
	routes.HandleCreatePersonnelDocument(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreatePersonnelDocument_MissingEmployeeID(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/personnel-documents",
		jsonBody(t, map[string]interface{}{
			"title":     "Arbeitsvertrag",
			"category":  "contract",
			"file_name": "vertrag.pdf",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreatePersonnelDocument(rec, req)
	assertValidationError(t, rec, "employee_id")
}

func TestHandleCreatePersonnelDocument_MissingTitle(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/personnel-documents",
		jsonBody(t, map[string]interface{}{
			"employee_id": "550e8400-e29b-41d4-a716-446655440000",
			"category":    "contract",
			"file_name":   "vertrag.pdf",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreatePersonnelDocument(rec, req)
	assertValidationError(t, rec, "title")
}

func TestHandleCreatePersonnelDocument_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/personnel-documents",
		jsonBody(t, map[string]interface{}{
			"employee_id": "550e8400-e29b-41d4-a716-446655440000",
			"title":       "Arbeitsvertrag",
			"category":    "contract",
			"file_name":   "vertrag.pdf",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreatePersonnelDocument(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
