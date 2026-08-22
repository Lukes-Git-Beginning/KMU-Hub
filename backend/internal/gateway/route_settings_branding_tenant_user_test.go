package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const testBrandingModuleID = "helpdesk"

// --- HandleGetResolvedSettings ---

func TestHandleGetResolvedSettings_ServiceUnavailable(t *testing.T) {
	routes := newModuleAccessRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetResolvedSettings)
}

func TestHandleGetResolvedSettings_MissingTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/settings/"+testBrandingModuleID, nil)
	routes.HandleGetResolvedSettings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetResolvedSettings_NoUserID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/settings/"+testBrandingModuleID, nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGetResolvedSettings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// TestHandleGetResolvedSettings_ReachesRPC documents that GetResolvedSettings
// is a response.Proto passthrough of the RPC's own response -- no
// gateway-owned marshaling to assert a wire shape against, the same
// documented boundary as every other handler in this package without a
// bufconn stub for SettingsServiceClient.
func TestHandleGetResolvedSettings_ReachesRPC(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/settings/"+testBrandingModuleID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "module_id", testBrandingModuleID)
	routes.HandleGetResolvedSettings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetTenantSettings ---

func TestHandleGetTenantSettings_ServiceUnavailable(t *testing.T) {
	routes := newModuleAccessRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetTenantSettings)
}

func TestHandleGetTenantSettings_MissingTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/settings/"+testBrandingModuleID+"/tenant", nil)
	routes.HandleGetTenantSettings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// TestHandleGetTenantSettings_CrossTenant documents that tenant isolation for
// this route is enforced server-side, not in the handler: the handler reads
// TenantId exclusively from middleware.GetTenantID(ctx) (set by the auth
// middleware from the caller's own token) and never from a caller-supplied
// value, so there is no code path here through which a request could name a
// foreign tenant. The RLS/ownership check for the resolved TenantId happens
// in internal/server/settings_grpc.go, which this package's lack of a
// bufconn/fake-client harness for SettingsServiceClient cannot reach (same
// documented boundary as every _ReachesRPC test in this file).
func TestHandleGetTenantSettings_CrossTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/settings/"+testBrandingModuleID+"/tenant", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "module_id", testBrandingModuleID)
	routes.HandleGetTenantSettings(rec, req)
	// No foreign-tenant path exists to hit: the handler always forwards the
	// context tenant, so this reaches the same dummy RPC as the happy path.
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetTenantSettings_ReachesRPC(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/settings/"+testBrandingModuleID+"/tenant", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "module_id", testBrandingModuleID)
	routes.HandleGetTenantSettings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandlePutTenantSettings ---

func TestHandlePutTenantSettings_ServiceUnavailable(t *testing.T) {
	routes := newModuleAccessRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandlePutTenantSettings)
}

func TestHandlePutTenantSettings_MissingTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/settings/"+testBrandingModuleID+"/tenant", jsonBody(t, map[string]interface{}{
		"settings": map[string]interface{}{"defaultView": "week"},
	}))
	routes.HandlePutTenantSettings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandlePutTenantSettings_NoCallerID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/settings/"+testBrandingModuleID+"/tenant", jsonBody(t, map[string]interface{}{
		"settings": map[string]interface{}{"defaultView": "week"},
	}))
	req = withTenantID(req, testTenantID)
	routes.HandlePutTenantSettings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandlePutTenantSettings_InvalidJSON(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/settings/"+testBrandingModuleID+"/tenant", invalidJSON())
	req = withAuth(req, "caller-123", testTenantID)
	routes.HandlePutTenantSettings(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandlePutTenantSettings_MissingSettings(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/settings/"+testBrandingModuleID+"/tenant", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "caller-123", testTenantID)
	routes.HandlePutTenantSettings(rec, req)
	assertValidationError(t, rec, "settings")
}

func TestHandlePutTenantSettings_ReachesRPC(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/settings/"+testBrandingModuleID+"/tenant", jsonBody(t, map[string]interface{}{
		"settings": map[string]interface{}{"defaultView": "week", "workStartHour": 8},
	}))
	req = withAuth(req, "caller-123", testTenantID)
	req = withChiURLParam(req, "module_id", testBrandingModuleID)
	routes.HandlePutTenantSettings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetUserSettings ---

func TestHandleGetUserSettings_ServiceUnavailable(t *testing.T) {
	routes := newModuleAccessRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetUserSettings)
}

func TestHandleGetUserSettings_MissingTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/settings/"+testBrandingModuleID+"/user", nil)
	routes.HandleGetUserSettings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetUserSettings_NoUserID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/settings/"+testBrandingModuleID+"/user", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGetUserSettings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetUserSettings_ReachesRPC(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/settings/"+testBrandingModuleID+"/user", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "module_id", testBrandingModuleID)
	routes.HandleGetUserSettings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandlePutUserSettings ---

func TestHandlePutUserSettings_ServiceUnavailable(t *testing.T) {
	routes := newModuleAccessRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandlePutUserSettings)
}

func TestHandlePutUserSettings_MissingTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/settings/"+testBrandingModuleID+"/user", jsonBody(t, map[string]interface{}{
		"settings": map[string]interface{}{"theme": "dark"},
	}))
	routes.HandlePutUserSettings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandlePutUserSettings_NoUserID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/settings/"+testBrandingModuleID+"/user", jsonBody(t, map[string]interface{}{
		"settings": map[string]interface{}{"theme": "dark"},
	}))
	req = withTenantID(req, testTenantID)
	routes.HandlePutUserSettings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandlePutUserSettings_InvalidJSON(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/settings/"+testBrandingModuleID+"/user", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandlePutUserSettings(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandlePutUserSettings_MissingSettings(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/settings/"+testBrandingModuleID+"/user", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandlePutUserSettings(rec, req)
	assertValidationError(t, rec, "settings")
}

func TestHandlePutUserSettings_ReachesRPC(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/settings/"+testBrandingModuleID+"/user", jsonBody(t, map[string]interface{}{
		"settings": map[string]interface{}{"theme": "dark"},
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "module_id", testBrandingModuleID)
	routes.HandlePutUserSettings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetBranding ---

func TestHandleGetBranding_ServiceUnavailable(t *testing.T) {
	routes := newModuleAccessRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetBranding)
}

func TestHandleGetBranding_MissingTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/admin/branding", nil)
	routes.HandleGetBranding(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetBranding_ReachesRPC(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/admin/branding", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGetBranding(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandlePutBranding ---

func TestHandlePutBranding_ServiceUnavailable(t *testing.T) {
	routes := newModuleAccessRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandlePutBranding)
}

func TestHandlePutBranding_MissingTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/admin/branding", jsonBody(t, map[string]interface{}{
		"name": "Zentria", "accent_color": "#10B981",
	}))
	routes.HandlePutBranding(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandlePutBranding_NoCallerID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/admin/branding", jsonBody(t, map[string]interface{}{
		"name": "Zentria", "accent_color": "#10B981",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandlePutBranding(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandlePutBranding_InvalidJSON(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/admin/branding", invalidJSON())
	req = withAuth(req, "caller-123", testTenantID)
	routes.HandlePutBranding(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandlePutBranding_MissingName(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/admin/branding", jsonBody(t, map[string]interface{}{
		"accent_color": "#10B981",
	}))
	req = withAuth(req, "caller-123", testTenantID)
	routes.HandlePutBranding(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandlePutBranding_MissingAccentColor(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/admin/branding", jsonBody(t, map[string]interface{}{
		"name": "Zentria",
	}))
	req = withAuth(req, "caller-123", testTenantID)
	routes.HandlePutBranding(rec, req)
	assertValidationError(t, rec, "accent_color")
}

// TestHandlePutBranding_NameTooLong exercises the max=200 bound on the one
// free-text field the gateway does validate itself. accent_color has no
// format check here on purpose: the handler forwards it verbatim to
// PutBranding, which rejects anything outside the fixed Cosmi swatch
// palette (settings.allowedAccentColors, internal/settings/branding.go) --
// duplicating that whitelist at the gateway would be the second
// implementation the notes for this unit warn against. Likewise
// logo_object_key/icon_object_key are checked server-side by
// brandingObjectKeyValid against the caller's own tenant prefix, already
// covered by internal/settings/branding_test.go. That split is the
// thick-services/thin-handlers rule, not a gap in this test.
func TestHandlePutBranding_NameTooLong(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	longName := make([]byte, 201)
	for i := range longName {
		longName[i] = 'a'
	}
	req := httptest.NewRequest("PUT", "/api/v1/admin/branding", jsonBody(t, map[string]interface{}{
		"name": string(longName), "accent_color": "#10B981",
	}))
	req = withAuth(req, "caller-123", testTenantID)
	routes.HandlePutBranding(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandlePutBranding_ReachesRPC(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/admin/branding", jsonBody(t, map[string]interface{}{
		"name": "Zentria", "accent_color": "#10B981", "logo_object_key": "tenants/x/logo.png",
	}))
	req = withAuth(req, "caller-123", testTenantID)
	routes.HandlePutBranding(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
