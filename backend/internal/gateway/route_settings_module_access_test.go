package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newModuleAccessRoutes(registry *ServiceRegistry) *SettingsRoutes {
	return NewSettingsRoutes(registry, noFlags())
}

const (
	testModuleAccessUserID = "550e8400-e29b-41d4-a716-446655440000"
	testModuleAccessModule = "helpdesk"
)

// --- HandleListModuleLeads ---

func TestHandleListModuleLeads_ServiceUnavailable(t *testing.T) {
	routes := newModuleAccessRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListModuleLeads)
}

func TestHandleListModuleLeads_MissingTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tenant/module-leads", nil)
	routes.HandleListModuleLeads(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// TestHandleListModuleLeads_ReachesRPC documents that ListModuleLeads is a
// response.Proto passthrough of the RPC's own response -- no gateway-owned
// marshaling to assert a wire shape against, the same documented boundary as
// every other list handler in this run without a bufconn stub for
// SettingsServiceClient.
func TestHandleListModuleLeads_ReachesRPC(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tenant/module-leads?user_id="+testModuleAccessUserID+"&module_id="+testModuleAccessModule, nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListModuleLeads(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetMyModuleLeads ---

func TestHandleGetMyModuleLeads_ServiceUnavailable(t *testing.T) {
	routes := newModuleAccessRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetMyModuleLeads)
}

func TestHandleGetMyModuleLeads_MissingTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tenant/module-leads/me", nil)
	routes.HandleGetMyModuleLeads(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetMyModuleLeads_NoUserID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tenant/module-leads/me", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGetMyModuleLeads(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetMyModuleLeads_ReachesRPC(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tenant/module-leads/me", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetMyModuleLeads(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGrantModuleLead ---

func TestHandleGrantModuleLead_ServiceUnavailable(t *testing.T) {
	routes := newModuleAccessRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGrantModuleLead)
}

func TestHandleGrantModuleLead_MissingTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tenant/module-leads/"+testModuleAccessUserID+"/"+testModuleAccessModule, nil)
	routes.HandleGrantModuleLead(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGrantModuleLead_NoCallerID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tenant/module-leads/"+testModuleAccessUserID+"/"+testModuleAccessModule, nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGrantModuleLead(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGrantModuleLead_InvalidIDUUID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tenant/module-leads/not-a-uuid/"+testModuleAccessModule, nil)
	req = withAuth(req, "caller-123", testTenantID)
	req = withChiURLParam(req, "user_id", "not-a-uuid")
	req = withChiURLParam(req, "module_id", testModuleAccessModule)
	routes.HandleGrantModuleLead(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGrantModuleLead_MissingModuleID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tenant/module-leads/"+testModuleAccessUserID+"/", nil)
	req = withAuth(req, "caller-123", testTenantID)
	req = withChiURLParam(req, "user_id", testModuleAccessUserID)
	routes.HandleGrantModuleLead(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "module_id")
}

func TestHandleGrantModuleLead_ReachesRPC(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tenant/module-leads/"+testModuleAccessUserID+"/"+testModuleAccessModule, nil)
	req = withAuth(req, "caller-123", testTenantID)
	req = withChiURLParam(req, "user_id", testModuleAccessUserID)
	req = withChiURLParam(req, "module_id", testModuleAccessModule)
	routes.HandleGrantModuleLead(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleRevokeModuleLead ---

func TestHandleRevokeModuleLead_ServiceUnavailable(t *testing.T) {
	routes := newModuleAccessRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleRevokeModuleLead)
}

func TestHandleRevokeModuleLead_MissingTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/tenant/module-leads/"+testModuleAccessUserID+"/"+testModuleAccessModule, nil)
	routes.HandleRevokeModuleLead(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleRevokeModuleLead_InvalidIDUUID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/tenant/module-leads/not-a-uuid/"+testModuleAccessModule, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "user_id", "not-a-uuid")
	req = withChiURLParam(req, "module_id", testModuleAccessModule)
	routes.HandleRevokeModuleLead(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleRevokeModuleLead_MissingModuleID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/tenant/module-leads/"+testModuleAccessUserID+"/", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "user_id", testModuleAccessUserID)
	routes.HandleRevokeModuleLead(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "module_id")
}

func TestHandleRevokeModuleLead_ReachesRPC(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/tenant/module-leads/"+testModuleAccessUserID+"/"+testModuleAccessModule, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "user_id", testModuleAccessUserID)
	req = withChiURLParam(req, "module_id", testModuleAccessModule)
	routes.HandleRevokeModuleLead(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListModuleGrants ---

func TestHandleListModuleGrants_ServiceUnavailable(t *testing.T) {
	routes := newModuleAccessRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListModuleGrants)
}

func TestHandleListModuleGrants_MissingTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tenant/module-grants", nil)
	routes.HandleListModuleGrants(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListModuleGrants_ReachesRPC(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tenant/module-grants?user_id="+testModuleAccessUserID, nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListModuleGrants(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGrantModuleAccess ---

func TestHandleGrantModuleAccess_ServiceUnavailable(t *testing.T) {
	routes := newModuleAccessRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGrantModuleAccess)
}

func TestHandleGrantModuleAccess_MissingTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tenant/module-grants/"+testModuleAccessUserID+"/"+testModuleAccessModule, nil)
	routes.HandleGrantModuleAccess(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGrantModuleAccess_NoCallerID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tenant/module-grants/"+testModuleAccessUserID+"/"+testModuleAccessModule, nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGrantModuleAccess(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGrantModuleAccess_InvalidIDUUID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tenant/module-grants/not-a-uuid/"+testModuleAccessModule, nil)
	req = withAuth(req, "caller-123", testTenantID)
	req = withChiURLParam(req, "user_id", "not-a-uuid")
	req = withChiURLParam(req, "module_id", testModuleAccessModule)
	routes.HandleGrantModuleAccess(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGrantModuleAccess_MissingModuleID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tenant/module-grants/"+testModuleAccessUserID+"/", nil)
	req = withAuth(req, "caller-123", testTenantID)
	req = withChiURLParam(req, "user_id", testModuleAccessUserID)
	routes.HandleGrantModuleAccess(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "module_id")
}

func TestHandleGrantModuleAccess_ReachesRPC(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tenant/module-grants/"+testModuleAccessUserID+"/"+testModuleAccessModule, nil)
	req = withAuth(req, "caller-123", testTenantID)
	req = withChiURLParam(req, "user_id", testModuleAccessUserID)
	req = withChiURLParam(req, "module_id", testModuleAccessModule)
	routes.HandleGrantModuleAccess(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleRevokeModuleAccess ---

func TestHandleRevokeModuleAccess_ServiceUnavailable(t *testing.T) {
	routes := newModuleAccessRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleRevokeModuleAccess)
}

func TestHandleRevokeModuleAccess_MissingTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/tenant/module-grants/"+testModuleAccessUserID+"/"+testModuleAccessModule, nil)
	routes.HandleRevokeModuleAccess(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleRevokeModuleAccess_InvalidIDUUID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/tenant/module-grants/not-a-uuid/"+testModuleAccessModule, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "user_id", "not-a-uuid")
	req = withChiURLParam(req, "module_id", testModuleAccessModule)
	routes.HandleRevokeModuleAccess(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleRevokeModuleAccess_MissingModuleID(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/tenant/module-grants/"+testModuleAccessUserID+"/", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "user_id", testModuleAccessUserID)
	routes.HandleRevokeModuleAccess(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "module_id")
}

func TestHandleRevokeModuleAccess_ReachesRPC(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/tenant/module-grants/"+testModuleAccessUserID+"/"+testModuleAccessModule, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "user_id", testModuleAccessUserID)
	req = withChiURLParam(req, "module_id", testModuleAccessModule)
	routes.HandleRevokeModuleAccess(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleBulkRevokeModuleAccess ---

func TestHandleBulkRevokeModuleAccess_ServiceUnavailable(t *testing.T) {
	routes := newModuleAccessRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleBulkRevokeModuleAccess)
}

func TestHandleBulkRevokeModuleAccess_MissingTenant(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tenant/module-grants/bulk-revoke", jsonBody(t, map[string]interface{}{
		"pairs": []map[string]string{{"userId": testModuleAccessUserID, "moduleId": testModuleAccessModule}},
	}))
	routes.HandleBulkRevokeModuleAccess(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleBulkRevokeModuleAccess_InvalidJSON(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tenant/module-grants/bulk-revoke", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleBulkRevokeModuleAccess(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// A request body without a "pairs" key decodes to a nil slice, which fails
// the field's `validate:"required"` tag -- the same rejection an explicit
// `"pairs": null` produces, and the only way this handler catches a caller
// that forgot the list entirely (an explicit `"pairs": []` passes required,
// since go-playground/validator treats a non-nil empty slice as present).
func TestHandleBulkRevokeModuleAccess_MissingPairs(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tenant/module-grants/bulk-revoke", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	routes.HandleBulkRevokeModuleAccess(rec, req)
	assertValidationError(t, rec, "pairs")
}

func TestHandleBulkRevokeModuleAccess_ReachesRPC(t *testing.T) {
	routes := newModuleAccessRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tenant/module-grants/bulk-revoke", jsonBody(t, map[string]interface{}{
		"pairs": []map[string]string{{"userId": testModuleAccessUserID, "moduleId": testModuleAccessModule}},
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleBulkRevokeModuleAccess(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
