package gateway

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
)

// route_integration_test.go covers registration, the five unauthenticated
// webhook/OAuth paths and that all ten admin routes sit behind
// RequireRole("admin"). It never calls a config or mapping handler by name.
// This file covers the thirteen configuration/mapping handlers themselves,
// following the RPC-boundary pattern already used by route_lexware_test.go:
// this package has no bufconn/fake-client harness for
// NotificationServiceClient, so a handler test can prove the ServiceUnavailable
// path (empty registry) and the RPC-failure path (registered but unreachable
// dummy connection, which surfaces as a gRPC error mapped to 503) — not a
// successful round trip.

const testIntegrationMappingID = "3f1c2a54-9b6d-4e21-8a77-5c0d1e2f3a4b"

func integrationRoutesForTest(reg *ServiceRegistry) *IntegrationRoutes {
	return NewIntegrationRoutes(reg)
}

// --- HandleListConfigs ---

func TestIntegrationHandleListConfigs_ServiceUnavailable(t *testing.T) {
	routes := integrationRoutesForTest(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListConfigs)
}

func TestIntegrationHandleListConfigs_RPCFails(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/configs", nil)
	routes.HandleListConfigs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetConfig ---

func TestIntegrationHandleGetConfig_ServiceUnavailable(t *testing.T) {
	routes := integrationRoutesForTest(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetConfig)
}

func TestIntegrationHandleGetConfig_RPCFails(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/configs/slack", nil)
	req = withChiURLParam(req, "platform", "slack")
	routes.HandleGetConfig(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleCreateConfig ---
// The only config/mapping handler whose validation runs before any RPC —
// Create takes the platform directly in the body instead of resolving it
// through a prior GetIntegrationConfig lookup, so its 400 paths are reachable
// without a working fake client.

func TestIntegrationHandleCreateConfig_ServiceUnavailable(t *testing.T) {
	routes := integrationRoutesForTest(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateConfig)
}

func TestIntegrationHandleCreateConfig_InvalidJSON(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/configs", invalidJSON())
	routes.HandleCreateConfig(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestIntegrationHandleCreateConfig_MissingPlatform(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/configs", jsonBody(t, map[string]interface{}{
		"credentials_vault_key": "vault-key-1",
	}))
	routes.HandleCreateConfig(rec, req)
	assertValidationError(t, rec, "platform")
}

func TestIntegrationHandleCreateConfig_MissingCredentialsVaultKey(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/configs", jsonBody(t, map[string]interface{}{
		"platform": "slack",
	}))
	routes.HandleCreateConfig(rec, req)
	assertValidationError(t, rec, "credentials_vault_key")
}

func TestIntegrationHandleCreateConfig_ValidBody_RPCFails(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/configs", jsonBody(t, map[string]interface{}{
		"platform":              "slack",
		"credentials_vault_key": "vault-key-1",
	}))
	routes.HandleCreateConfig(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateConfig ---
// Resolves the config ID via GetIntegrationConfig before decoding the body, so
// the 400 paths are unreachable without a working fake client — only the two
// RPC-boundary cases are testable here.

func TestIntegrationHandleUpdateConfig_ServiceUnavailable(t *testing.T) {
	routes := integrationRoutesForTest(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdateConfig)
}

func TestIntegrationHandleUpdateConfig_RPCFails(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/configs/slack", jsonBody(t, map[string]interface{}{
		"is_active": true,
	}))
	req = withChiURLParam(req, "platform", "slack")
	routes.HandleUpdateConfig(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteConfig ---
// Also resolves the config ID via GetIntegrationConfig first.

func TestIntegrationHandleDeleteConfig_ServiceUnavailable(t *testing.T) {
	routes := integrationRoutesForTest(emptyRegistry())
	testServiceUnavailable(t, routes.HandleDeleteConfig)
}

func TestIntegrationHandleDeleteConfig_RPCFails(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/integrations/configs/slack", nil)
	req = withChiURLParam(req, "platform", "slack")
	routes.HandleDeleteConfig(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleTestConfig ---
// The message-leak fix for the RPC this calls (TestIntegrationConfig)
// belongs to the server package and is pinned there
// (TestTestIntegrationConfigDoesNotLeakProbeErrorDetail,
// internal/server/notification_integration_test.go) — this package has no
// fake client to reach a real probe failure through the gateway.

func TestIntegrationHandleTestConfig_ServiceUnavailable(t *testing.T) {
	routes := integrationRoutesForTest(emptyRegistry())
	testServiceUnavailable(t, routes.HandleTestConfig)
}

func TestIntegrationHandleTestConfig_RPCFails(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/configs/slack/test", nil)
	req = withChiURLParam(req, "platform", "slack")
	routes.HandleTestConfig(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListMappings ---
// Resolves the config ID via GetIntegrationConfig first.

func TestIntegrationHandleListMappings_ServiceUnavailable(t *testing.T) {
	routes := integrationRoutesForTest(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListMappings)
}

func TestIntegrationHandleListMappings_RPCFails(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/configs/slack/mappings", nil)
	req = withChiURLParam(req, "platform", "slack")
	routes.HandleListMappings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleCreateMapping ---
// Also resolves the config ID via GetIntegrationConfig before decoding the
// body — same unreachable-400 situation as HandleUpdateConfig.

func TestIntegrationHandleCreateMapping_ServiceUnavailable(t *testing.T) {
	routes := integrationRoutesForTest(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateMapping)
}

func TestIntegrationHandleCreateMapping_RPCFails(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/configs/slack/mappings", jsonBody(t, map[string]interface{}{
		"channel_id":   "C123",
		"channel_name": "general",
		"modules":      []string{"crm"},
	}))
	req = withChiURLParam(req, "platform", "slack")
	routes.HandleCreateMapping(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateMapping ---
// No prior RPC — validateUUIDParam and decodeAndValidate both run before any
// gRPC call, so every 400 path is reachable here.

func TestIntegrationHandleUpdateMapping_ServiceUnavailable(t *testing.T) {
	routes := integrationRoutesForTest(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdateMapping)
}

func TestIntegrationHandleUpdateMapping_InvalidID(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/mappings/not-a-uuid", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateMapping(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestIntegrationHandleUpdateMapping_InvalidJSON(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/mappings/"+testIntegrationMappingID, invalidJSON())
	req = withChiURLParam(req, "id", testIntegrationMappingID)
	routes.HandleUpdateMapping(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestIntegrationHandleUpdateMapping_MissingChannelID(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/mappings/"+testIntegrationMappingID, jsonBody(t, map[string]interface{}{
		"channel_name": "general",
		"modules":      []string{"crm"},
	}))
	req = withChiURLParam(req, "id", testIntegrationMappingID)
	routes.HandleUpdateMapping(rec, req)
	assertValidationError(t, rec, "channel_id")
}

func TestIntegrationHandleUpdateMapping_MissingModules(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/mappings/"+testIntegrationMappingID, jsonBody(t, map[string]interface{}{
		"channel_id":   "C123",
		"channel_name": "general",
		"modules":      []string{},
	}))
	req = withChiURLParam(req, "id", testIntegrationMappingID)
	routes.HandleUpdateMapping(rec, req)
	assertValidationError(t, rec, "modules")
}

func TestIntegrationHandleUpdateMapping_ValidBody_RPCFails(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/mappings/"+testIntegrationMappingID, jsonBody(t, map[string]interface{}{
		"channel_id":   "C123",
		"channel_name": "general",
		"modules":      []string{"crm"},
		"is_active":    true,
	}))
	req = withChiURLParam(req, "id", testIntegrationMappingID)
	routes.HandleUpdateMapping(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteMapping ---

func TestIntegrationHandleDeleteMapping_ServiceUnavailable(t *testing.T) {
	routes := integrationRoutesForTest(emptyRegistry())
	testServiceUnavailable(t, routes.HandleDeleteMapping)
}

func TestIntegrationHandleDeleteMapping_InvalidID(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/integrations/mappings/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteMapping(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestIntegrationHandleDeleteMapping_RPCFails(t *testing.T) {
	routes := integrationRoutesForTest(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/integrations/mappings/"+testIntegrationMappingID, nil)
	req = withChiURLParam(req, "id", testIntegrationMappingID)
	routes.HandleDeleteMapping(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestIntegrationConfigResponseProtos_NeverExposeCredentials is a structural
// guard in place of a live round trip (same boundary limitation noted above):
// none of the wire response messages this file maps into JSON may ever carry
// the credentials vault key. IntegrationConfigInfo already documents the
// omission with a comment ("credentials_vault_key intentionally omitted --
// secrets stay server-side"); this turns that comment into a test that fails
// the moment someone adds the field back "to pass it through", instead of the
// field silently reaching an admin's browser.
func TestIntegrationConfigResponseProtos_NeverExposeCredentials(t *testing.T) {
	responses := []interface{}{
		notificationv1.IntegrationConfigInfo{},
		notificationv1.ListIntegrationConfigsResponse{},
		notificationv1.GetIntegrationConfigResponse{},
		notificationv1.CreateIntegrationConfigResponse{},
		notificationv1.UpdateIntegrationConfigResponse{},
		notificationv1.TestIntegrationConfigResponse{},
		notificationv1.ChannelMappingInfo{},
		notificationv1.ListChannelMappingsResponse{},
		notificationv1.CreateChannelMappingResponse{},
		notificationv1.UpdateChannelMappingResponse{},
	}
	for _, resp := range responses {
		typ := reflect.TypeOf(resp)
		for i := 0; i < typ.NumField(); i++ {
			name := strings.ToLower(typ.Field(i).Name)
			if strings.Contains(name, "credential") || strings.Contains(name, "vaultkey") || strings.Contains(name, "secret") {
				t.Errorf("%s.%s: response proto exposes a credential field — the gateway must never surface it to the client", typ.Name(), typ.Field(i).Name)
			}
		}
	}
}
