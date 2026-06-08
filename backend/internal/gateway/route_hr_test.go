package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
