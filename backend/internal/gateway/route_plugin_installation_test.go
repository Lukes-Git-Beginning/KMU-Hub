package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Covers the installation lifecycle handlers in route_plugin.go that
// route_plugin_test.go does not yet exercise: HandleInstallPlugin,
// HandleGetInstallation, HandleEnablePlugin, HandleDisablePlugin,
// HandleUninstallPlugin and HandleApprovePermissions.
//
// Unlike most other route files in this run, NONE of these six handlers call
// validateUUIDParam on "installation_id" (grep confirms zero hits in
// route_plugin.go) — chi.URLParam is passed straight into the proto request.
// An unparseable installation id therefore does not get a local 400; it
// reaches the RPC layer exactly like a valid one (same *_ReachesRPC boundary
// used throughout this run: registryWithService dials localhost:0, so a
// request that clears local validation still ends in a connection-refused
// 503, not a 200). The *_InvalidUUID tests below document that passthrough
// instead of asserting a 400 that the handler does not produce — inventing
// one would be a fabricated fix hidden inside a coverage unit. This gap is a
// candidate for the same class of fix as fix-gateway-id-validation-
// consistency (Iteration 6), which only swept chi.URLParam(r, "id") and
// therefore never counted the "installation_id"/"manifest_id"/"rule_id"
// sites in this file; noted in the Iteration 68 journal entry for Lauf 9.
const pluginTestInstallationID = "770e8400-e29b-41d4-a716-446655440000"
const pluginTestManifestID = "880e8400-e29b-41d4-a716-446655440000"

// --- HandleInstallPlugin ---

func TestHandleInstallPlugin_ServiceUnavailable(t *testing.T) {
	routes := NewPluginRoutes(emptyRegistry(), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/plugins/installations", jsonBody(t, map[string]interface{}{
		"manifest_id": pluginTestManifestID,
	}))
	routes.HandleInstallPlugin(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleInstallPlugin_InvalidJSON(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/plugins/installations", invalidJSON())
	routes.HandleInstallPlugin(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleInstallPlugin_ReachesRPC(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/plugins/installations", jsonBody(t, map[string]interface{}{
		"manifest_id": pluginTestManifestID,
	}))
	routes.HandleInstallPlugin(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetInstallation ---

func TestHandleGetInstallation_ServiceUnavailable(t *testing.T) {
	routes := NewPluginRoutes(emptyRegistry(), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/plugins/installations/"+pluginTestInstallationID, nil)
	req = withChiURLParam(req, "installation_id", pluginTestInstallationID)
	routes.HandleGetInstallation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetInstallation_InvalidUUID_ReachesRPC(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/plugins/installations/not-a-uuid", nil)
	req = withChiURLParam(req, "installation_id", "not-a-uuid")
	routes.HandleGetInstallation(rec, req)
	// No local UUID guard exists for this handler (see file-level comment) —
	// an unparseable id reaches the RPC layer the same as a valid one.
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetInstallation_ReachesRPC(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/plugins/installations/"+pluginTestInstallationID, nil)
	req = withChiURLParam(req, "installation_id", pluginTestInstallationID)
	routes.HandleGetInstallation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleEnablePlugin ---

func TestHandleEnablePlugin_ServiceUnavailable(t *testing.T) {
	routes := NewPluginRoutes(emptyRegistry(), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/plugins/installations/"+pluginTestInstallationID+"/enable", nil)
	req = withChiURLParam(req, "installation_id", pluginTestInstallationID)
	routes.HandleEnablePlugin(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleEnablePlugin_InvalidUUID_ReachesRPC(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/plugins/installations/not-a-uuid/enable", nil)
	req = withChiURLParam(req, "installation_id", "not-a-uuid")
	routes.HandleEnablePlugin(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleEnablePlugin_ReachesRPC(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/plugins/installations/"+pluginTestInstallationID+"/enable", nil)
	req = withChiURLParam(req, "installation_id", pluginTestInstallationID)
	routes.HandleEnablePlugin(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDisablePlugin ---

func TestHandleDisablePlugin_ServiceUnavailable(t *testing.T) {
	routes := NewPluginRoutes(emptyRegistry(), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/plugins/installations/"+pluginTestInstallationID+"/disable", nil)
	req = withChiURLParam(req, "installation_id", pluginTestInstallationID)
	routes.HandleDisablePlugin(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDisablePlugin_InvalidUUID_ReachesRPC(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/plugins/installations/not-a-uuid/disable", nil)
	req = withChiURLParam(req, "installation_id", "not-a-uuid")
	routes.HandleDisablePlugin(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDisablePlugin_ReachesRPC(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/plugins/installations/"+pluginTestInstallationID+"/disable", nil)
	req = withChiURLParam(req, "installation_id", pluginTestInstallationID)
	routes.HandleDisablePlugin(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUninstallPlugin ---

func TestHandleUninstallPlugin_ServiceUnavailable(t *testing.T) {
	routes := NewPluginRoutes(emptyRegistry(), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/plugins/installations/"+pluginTestInstallationID, nil)
	req = withChiURLParam(req, "installation_id", pluginTestInstallationID)
	routes.HandleUninstallPlugin(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUninstallPlugin_ReachesRPC(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/plugins/installations/"+pluginTestInstallationID, nil)
	req = withChiURLParam(req, "installation_id", pluginTestInstallationID)
	routes.HandleUninstallPlugin(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleApprovePermissions ---
//
// Unlike the other installation handlers, this one runs real gateway-local
// validation (decodeAndValidate[approvePermissionsHTTPReq], route_plugin.go
// :318-321): permissions is required+min=1, granted_by is required+uuid.

func TestHandleApprovePermissions_ServiceUnavailable(t *testing.T) {
	routes := NewPluginRoutes(emptyRegistry(), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/plugins/installations/"+pluginTestInstallationID+"/permissions", jsonBody(t, map[string]interface{}{
		"permissions": []string{"crm:read"},
		"granted_by":  testTenantID.String(),
	}))
	req = withChiURLParam(req, "installation_id", pluginTestInstallationID)
	routes.HandleApprovePermissions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleApprovePermissions_MissingPermissions(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/plugins/installations/"+pluginTestInstallationID+"/permissions", jsonBody(t, map[string]interface{}{
		"permissions": []string{},
		"granted_by":  testTenantID.String(),
	}))
	req = withChiURLParam(req, "installation_id", pluginTestInstallationID)
	routes.HandleApprovePermissions(rec, req)
	assertValidationError(t, rec, "permissions")
}

func TestHandleApprovePermissions_InvalidGrantedBy(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/plugins/installations/"+pluginTestInstallationID+"/permissions", jsonBody(t, map[string]interface{}{
		"permissions": []string{"crm:read"},
		"granted_by":  "not-a-uuid",
	}))
	req = withChiURLParam(req, "installation_id", pluginTestInstallationID)
	routes.HandleApprovePermissions(rec, req)
	assertValidationError(t, rec, "granted_by")
}

func TestHandleApprovePermissions_MissingGrantedBy(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/plugins/installations/"+pluginTestInstallationID+"/permissions", jsonBody(t, map[string]interface{}{
		"permissions": []string{"crm:read"},
	}))
	req = withChiURLParam(req, "installation_id", pluginTestInstallationID)
	routes.HandleApprovePermissions(rec, req)
	assertValidationError(t, rec, "granted_by")
}

func TestHandleApprovePermissions_InvalidJSON(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/plugins/installations/"+pluginTestInstallationID+"/permissions", invalidJSON())
	req = withChiURLParam(req, "installation_id", pluginTestInstallationID)
	routes.HandleApprovePermissions(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleApprovePermissions_ReachesRPC(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), noFlags())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/plugins/installations/"+pluginTestInstallationID+"/permissions", jsonBody(t, map[string]interface{}{
		"permissions": []string{"crm:read"},
		"granted_by":  testTenantID.String(),
	}))
	req = withChiURLParam(req, "installation_id", pluginTestInstallationID)
	routes.HandleApprovePermissions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
