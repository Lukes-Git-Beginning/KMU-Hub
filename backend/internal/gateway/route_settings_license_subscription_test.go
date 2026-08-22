package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kmuhub/kmuhub/internal/featureflag"
)

// flagsWithHelpdeskEnabled returns a registry where the helpdesk module's
// deployment-wide flag is on and every other flagged module stays off — the
// same shape as TestToTenantModuleJSON_FlagMasksStoredActivation, reused here
// so HandleSetTenantModuleActive can exercise both the available and the
// unavailable branch of moduleAvailable without a second fixture.
func flagsWithHelpdeskEnabled() *featureflag.Registry {
	return featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_HELPDESK_ENABLED" {
			return "true"
		}
		return ""
	})
}

// --- HandleGetTenantLicense ---

func TestHandleGetTenantLicense_ServiceUnavailable(t *testing.T) {
	routes := NewSettingsRoutes(emptyRegistry(), noFlags())
	testServiceUnavailable(t, routes.HandleGetTenantLicense)
}

func TestHandleGetTenantLicense_MissingTenant(t *testing.T) {
	routes := NewSettingsRoutes(registryWithService("auth"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/admin/license", nil)
	routes.HandleGetTenantLicense(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetTenantLicense_ReachesRPC(t *testing.T) {
	routes := NewSettingsRoutes(registryWithService("auth"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/admin/license", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGetTenantLicense(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleSetTenantModuleActive ---

func TestHandleSetTenantModuleActive_ServiceUnavailable(t *testing.T) {
	routes := NewSettingsRoutes(emptyRegistry(), noFlags())
	testServiceUnavailable(t, routes.HandleSetTenantModuleActive)
}

func TestHandleSetTenantModuleActive_MissingTenant(t *testing.T) {
	routes := NewSettingsRoutes(registryWithService("auth"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/admin/license", jsonBody(t, map[string]interface{}{
		"moduleId": "crm", "active": true,
	}))
	routes.HandleSetTenantModuleActive(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleSetTenantModuleActive_NoCallerID(t *testing.T) {
	routes := NewSettingsRoutes(registryWithService("auth"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/admin/license", jsonBody(t, map[string]interface{}{
		"moduleId": "crm", "active": true,
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleSetTenantModuleActive(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleSetTenantModuleActive_InvalidJSON(t *testing.T) {
	routes := NewSettingsRoutes(registryWithService("auth"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/admin/license", invalidJSON())
	req = withAuth(req, "caller-123", testTenantID)
	routes.HandleSetTenantModuleActive(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSetTenantModuleActive_MissingActive(t *testing.T) {
	routes := NewSettingsRoutes(registryWithService("auth"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/admin/license", jsonBody(t, map[string]interface{}{
		"moduleId": "crm",
	}))
	req = withAuth(req, "caller-123", testTenantID)
	routes.HandleSetTenantModuleActive(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "active is required")
}

func TestHandleSetTenantModuleActive_UnknownModule(t *testing.T) {
	routes := NewSettingsRoutes(registryWithService("auth"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/admin/license", jsonBody(t, map[string]interface{}{
		"moduleId": "not-a-module", "active": true,
	}))
	req = withAuth(req, "caller-123", testTenantID)
	routes.HandleSetTenantModuleActive(rec, req)
	assertStatus(t, rec, http.StatusNotFound)
	assertErrorContains(t, rec, "unknown module")
}

// TestHandleSetTenantModuleActive_ModuleNotAvailable exercises the deploy-time
// gate: activating a module whose deployment-wide flag is off would store a
// row the very next GET masks back to inactive (toTenantModuleJSON), so the
// handler refuses it up front instead of accepting a write nobody will see.
func TestHandleSetTenantModuleActive_ModuleNotAvailable(t *testing.T) {
	routes := NewSettingsRoutes(registryWithService("auth"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/admin/license", jsonBody(t, map[string]interface{}{
		"moduleId": "helpdesk", "active": true,
	}))
	req = withAuth(req, "caller-123", testTenantID)
	routes.HandleSetTenantModuleActive(rec, req)
	assertStatus(t, rec, http.StatusConflict)
	assertErrorContains(t, rec, "not available")
}

// TestHandleSetTenantModuleActive_DeactivateUnavailableModule documents the
// asymmetry deliberately left in the handler: switching an unavailable module
// OFF is not blocked by moduleAvailable, only switching it on is — turning
// something off can never surprise the licence tab the way turning it on can.
func TestHandleSetTenantModuleActive_DeactivateUnavailableModule(t *testing.T) {
	routes := NewSettingsRoutes(registryWithService("auth"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/admin/license", jsonBody(t, map[string]interface{}{
		"moduleId": "helpdesk", "active": false,
	}))
	req = withAuth(req, "caller-123", testTenantID)
	routes.HandleSetTenantModuleActive(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSetTenantModuleActive_ReachesRPC(t *testing.T) {
	routes := NewSettingsRoutes(registryWithService("auth"), flagsWithHelpdeskEnabled())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/admin/license", jsonBody(t, map[string]interface{}{
		"moduleId": "helpdesk", "active": true,
	}))
	req = withAuth(req, "caller-123", testTenantID)
	routes.HandleSetTenantModuleActive(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetTenantSubscription ---

func TestHandleGetTenantSubscription_ServiceUnavailable(t *testing.T) {
	routes := NewSettingsRoutes(emptyRegistry(), noFlags())
	testServiceUnavailable(t, routes.HandleGetTenantSubscription)
}

func TestHandleGetTenantSubscription_MissingTenant(t *testing.T) {
	routes := NewSettingsRoutes(registryWithService("auth"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/admin/subscription", nil)
	routes.HandleGetTenantSubscription(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetTenantSubscription_ReachesRPC(t *testing.T) {
	routes := NewSettingsRoutes(registryWithService("auth"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/admin/subscription", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGetTenantSubscription(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
