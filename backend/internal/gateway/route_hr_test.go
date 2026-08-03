package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// ============================================================================
// HandleSubmitCorrection — validation tests (S4.1)
// ============================================================================

func TestHandleSubmitCorrection_MissingClockIn(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/corrections",
		jsonBody(t, map[string]interface{}{
			"corrected_clock_out": "2026-06-08T18:00:00Z",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleSubmitCorrection(rec, req)
	assertValidationError(t, rec, "corrected_clock_in")
}

func TestHandleSubmitCorrection_MissingClockOut(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/corrections",
		jsonBody(t, map[string]interface{}{
			"corrected_clock_in": "2026-06-08T09:00:00Z",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleSubmitCorrection(rec, req)
	assertValidationError(t, rec, "corrected_clock_out")
}

func TestHandleSubmitCorrection_InvalidJSON(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/corrections", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleSubmitCorrection(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// ============================================================================
// HandleUpdateSelfProfile — DACH validators
// ============================================================================

func TestHandleUpdateSelfProfile_InvalidPostalCode(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/employees/me",
		jsonBody(t, map[string]interface{}{
			"address_postal_code": "INVALID",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateSelfProfile(rec, req)
	assertValidationError(t, rec, "address_postal_code")
}

func TestHandleUpdateSelfProfile_InvalidPhone(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/employees/me",
		jsonBody(t, map[string]interface{}{
			"emergency_contact_phone": "not-a-phone",
		}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateSelfProfile(rec, req)
	assertValidationError(t, rec, "emergency_contact_phone")
}

// ============================================================================
// Document category routes (fix-hr-document-paths) — router-level on purpose:
// a handler unit test calling HandleListEmployeeDocumentCategories directly
// can never catch a mistyped path or a guard that locks non-admin roles out.
// ============================================================================

const testEmployeeDocsCategoriesPath = "/api/v1/hr/employees/550e8400-e29b-41d4-a716-446655440000/documents/categories"

// TestHandleListEmployeeDocumentCategories_MatchesFrontendPath locks in the
// path hr-client.ts actually calls. The registry is empty, so a request that
// resolves to the handler dies at the gRPC client with 503; 404 would mean
// the path itself is unmatched — the defect fix-hr-document-paths guards
// against.
func TestHandleListEmployeeDocumentCategories_MatchesFrontendPath(t *testing.T) {
	r := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(r, guardTestAuth)

	req := httptest.NewRequest(http.MethodGet, testEmployeeDocsCategoriesPath, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withPermissions(req, "team:documents:view")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListEmployeeDocumentCategories_LegacyPermissionStillWorks pins
// that extending the guard to RequirePermissionAny did not drop the
// pre-existing "hr:read" key (migration 000129 grants it to admin only) —
// additive, never replace, per the zeiterfassung guards above in
// route_hr.go.
func TestHandleListEmployeeDocumentCategories_LegacyPermissionStillWorks(t *testing.T) {
	r := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(r, guardTestAuth)

	req := httptest.NewRequest(http.MethodGet, testEmployeeDocsCategoriesPath, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withPermissions(req, "hr:read")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListEmployeeDocumentCategories_RequiresPermission verifies a
// caller with neither key is rejected rather than silently falling through.
func TestHandleListEmployeeDocumentCategories_RequiresPermission(t *testing.T) {
	r := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(r, guardTestAuth)

	req := httptest.NewRequest(http.MethodGet, testEmployeeDocsCategoriesPath, nil)
	req = withAuth(req, "user-123", testTenantID)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusForbidden)
}

// TestOldDocumentCategoriesRouteRemoved pins that the plain, never-called
// "/hr/document-categories" route (superseded by the per-employee path above)
// is gone rather than left as a second, unfiltered way to read categories.
func TestOldDocumentCategoriesRouteRemoved(t *testing.T) {
	r := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(r, guardTestAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/document-categories", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withPermissions(req, "hr:read")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusNotFound)
}
