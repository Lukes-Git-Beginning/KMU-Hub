package gateway

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

func TestBexioRoutes_ServiceName(t *testing.T) {
	routes := NewBexioRoutes(emptyRegistry(), testStateSecret)
	if routes.ServiceName() != "biz" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "biz")
	}
}

// --- HandleOAuthCallback ---
// Public route (no auth middleware) — Bexio redirects here after consent.

func TestHandleOAuthCallback_ServiceUnavailable(t *testing.T) {
	routes := NewBexioRoutes(emptyRegistry(), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/oauth/callback?code=abc&state=xyz", nil)
	routes.HandleOAuthCallback(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleOAuthCallback_MissingCode_NoErrorParam(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/oauth/callback", nil)
	routes.HandleOAuthCallback(rec, req)
	assertStatus(t, rec, http.StatusFound)
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "bexio_error=missing") {
		t.Errorf("Location = %q, want to contain %q", loc, "bexio_error=missing")
	}
}

func TestHandleOAuthCallback_MissingCode_WithErrorParam(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/oauth/callback?error=access_denied", nil)
	routes.HandleOAuthCallback(rec, req)
	assertStatus(t, rec, http.StatusFound)
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "bexio_error=access_denied") {
		t.Errorf("Location = %q, want to contain %q", loc, "bexio_error=access_denied")
	}
}

// TestHandleOAuthCallback_MissingCode_ErrorParamIsEscaped is the mutations-probe
// target for fix-gateway-bexio-error-message-leakage: the "error" query param
// is attacker/Bexio-controlled (this is the public, unauthenticated callback
// route) and reaches the redirect Location unescaped before the fix. A raw "&"
// would let the value inject an extra query param, and a raw newline would be
// header/response splitting in front of some proxies. Both must come out
// percent-encoded, and the Location header itself must be a single line.
func TestHandleOAuthCallback_MissingCode_ErrorParamIsEscaped(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	malicious := "evil&injected=1\r\nSet-Cookie: pwned=1"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/oauth/callback?error="+url.QueryEscape(malicious), nil)
	routes.HandleOAuthCallback(rec, req)
	assertStatus(t, rec, http.StatusFound)
	loc := rec.Header().Get("Location")
	if strings.Contains(loc, "\r") || strings.Contains(loc, "\n") {
		t.Fatalf("Location contains a raw CR/LF: %q", loc)
	}
	if strings.Contains(loc, "injected=1") || strings.Contains(loc, "Set-Cookie:") {
		t.Errorf("Location = %q, the malicious param leaked unescaped into the redirect", loc)
	}
	if !strings.Contains(loc, "bexio_error=evil%26injected") {
		t.Errorf("Location = %q, want the value percent-encoded (e.g. %%26 for '&')", loc)
	}
}

func TestHandleOAuthCallback_StateSecretNotConfigured(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/oauth/callback?code=abc&state=xyz", nil)
	routes.HandleOAuthCallback(rec, req)
	assertStatus(t, rec, http.StatusInternalServerError)
	assertErrorContains(t, rec, "OAuth state validation not configured")
}

func TestHandleOAuthCallback_InvalidState_RejectedGenerically(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/oauth/callback?code=abc&state=not-a-valid-token", nil)
	routes.HandleOAuthCallback(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid or expired OAuth state")
}

// TestHandleOAuthCallback_InvalidState_SameResponseAsExpiredState locks down
// that a malformed token and a validly-signed-but-expired token are
// indistinguishable to the caller: neither response may leak *which* check
// failed, or an attacker probing the endpoint gets a signal to work with.
func TestHandleOAuthCallback_InvalidState_SameResponseAsExpiredState(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)

	garbageRec := httptest.NewRecorder()
	garbageReq := httptest.NewRequest(http.MethodGet, "/x?code=abc&state=garbage", nil)
	routes.HandleOAuthCallback(garbageRec, garbageReq)

	expiredPayload := testTenantID.String() + "|deadbeefnonce|1"
	sig := computeBexioHMAC(testStateSecret, expiredPayload)
	expiredToken := base64.RawURLEncoding.EncodeToString([]byte(expiredPayload)) + "." + sig
	expiredRec := httptest.NewRecorder()
	expiredReq := httptest.NewRequest(http.MethodGet, "/x?code=abc&state="+expiredToken, nil)
	routes.HandleOAuthCallback(expiredRec, expiredReq)

	if garbageRec.Code != expiredRec.Code {
		t.Errorf("status differs: garbage=%d expired=%d", garbageRec.Code, expiredRec.Code)
	}
	if garbageRec.Body.String() != expiredRec.Body.String() {
		t.Errorf("body differs: garbage=%q expired=%q", garbageRec.Body.String(), expiredRec.Body.String())
	}
}

func TestHandleOAuthCallback_ValidState_RPCFailsRedirectsGenerically(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	state, err := encodeBexioState(testStateSecret, testTenantID.String())
	if err != nil {
		t.Fatalf("encodeBexioState: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/oauth/callback?code=abc&state="+state, nil)
	routes.HandleOAuthCallback(rec, req)
	assertStatus(t, rec, http.StatusFound)
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "bexio_error=connection_failed") {
		t.Errorf("Location = %q, want to contain %q", loc, "bexio_error=connection_failed")
	}
}

// --- HandleGetAuthURL ---

func TestHandleGetAuthURL_ServiceUnavailable(t *testing.T) {
	routes := NewBexioRoutes(emptyRegistry(), testStateSecret)
	testServiceUnavailable(t, routes.HandleGetAuthURL)
}

func TestHandleGetAuthURL_NoTenantID(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/oauth/authorize", nil)
	routes.HandleGetAuthURL(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetAuthURL_StateSecretNotConfigured(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/oauth/authorize", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGetAuthURL(rec, req)
	assertStatus(t, rec, http.StatusInternalServerError)
	assertErrorContains(t, rec, "OAuth state secret not configured")
}

func TestHandleGetAuthURL_RPCFails(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/oauth/authorize", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGetAuthURL(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDisconnect ---

func TestHandleDisconnect_ServiceUnavailable(t *testing.T) {
	routes := NewBexioRoutes(emptyRegistry(), testStateSecret)
	testServiceUnavailable(t, routes.HandleDisconnect)
}

func TestHandleDisconnect_NoTenantID(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/bexio/disconnect", nil)
	routes.HandleDisconnect(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDisconnect_RPCFails(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/bexio/disconnect", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleDisconnect(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetConnectionStatus ---

func TestHandleGetConnectionStatus_ServiceUnavailable(t *testing.T) {
	routes := NewBexioRoutes(emptyRegistry(), testStateSecret)
	testServiceUnavailable(t, routes.HandleGetConnectionStatus)
}

func TestHandleGetConnectionStatus_NoTenantID(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/status", nil)
	routes.HandleGetConnectionStatus(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetConnectionStatus_RPCFails(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/status", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGetConnectionStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleTriggerSync ---

func TestHandleTriggerSync_ServiceUnavailable(t *testing.T) {
	routes := NewBexioRoutes(emptyRegistry(), testStateSecret)
	testServiceUnavailable(t, routes.HandleTriggerSync)
}

func TestHandleTriggerSync_NoTenantID(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/bexio/sync/trigger", nil)
	routes.HandleTriggerSync(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleTriggerSync_InvalidJSON(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/bexio/sync/trigger", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleTriggerSync(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// TestHandleTriggerSync_EmptyBodySkipsDecode locks down the ContentLength==0
// short-circuit: a bodyless POST must NOT be rejected for lacking JSON, it
// must fall through with sync_type="" ("sync everything").
func TestHandleTriggerSync_EmptyBodySkipsDecode(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/bexio/sync/trigger", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleTriggerSync(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleTriggerSync_ValidBody_RPCFails(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/bexio/sync/trigger", jsonBody(t, map[string]interface{}{
		"sync_type": "contacts",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleTriggerSync(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetSyncStatus ---

func TestHandleGetSyncStatus_ServiceUnavailable(t *testing.T) {
	routes := NewBexioRoutes(emptyRegistry(), testStateSecret)
	testServiceUnavailable(t, routes.HandleGetSyncStatus)
}

func TestHandleGetSyncStatus_NoTenantID(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/sync/status", nil)
	routes.HandleGetSyncStatus(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetSyncStatus_RPCFails(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/sync/status", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGetSyncStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateSyncConfig ---

func TestHandleUpdateSyncConfig_ServiceUnavailable(t *testing.T) {
	routes := NewBexioRoutes(emptyRegistry(), testStateSecret)
	testServiceUnavailable(t, routes.HandleUpdateSyncConfig)
}

func TestHandleUpdateSyncConfig_NoTenantID(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/bexio/sync/config", jsonBody(t, map[string]interface{}{}))
	routes.HandleUpdateSyncConfig(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateSyncConfig_InvalidJSON(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/bexio/sync/config", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleUpdateSyncConfig(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateSyncConfig_RPCFails(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/bexio/sync/config", jsonBody(t, map[string]interface{}{
		"contact_sync_enabled":          true,
		"contact_sync_interval_minutes": 15,
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleUpdateSyncConfig(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListSyncLogs ---

func TestHandleListSyncLogs_ServiceUnavailable(t *testing.T) {
	routes := NewBexioRoutes(emptyRegistry(), testStateSecret)
	testServiceUnavailable(t, routes.HandleListSyncLogs)
}

func TestHandleListSyncLogs_NoTenantID(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/sync/logs", nil)
	routes.HandleListSyncLogs(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListSyncLogs_RPCFails(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/sync/logs?limit=5", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListSyncLogs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetFieldMappings ---

func TestHandleGetFieldMappings_ServiceUnavailable(t *testing.T) {
	routes := NewBexioRoutes(emptyRegistry(), testStateSecret)
	testServiceUnavailable(t, routes.HandleGetFieldMappings)
}

func TestHandleGetFieldMappings_NoTenantID(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/mappings/contact", nil)
	req = withChiURLParam(req, "entity_type", "contact")
	routes.HandleGetFieldMappings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetFieldMappings_RPCFails(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/bexio/mappings/contact", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "entity_type", "contact")
	routes.HandleGetFieldMappings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateFieldMappings ---

func TestHandleUpdateFieldMappings_ServiceUnavailable(t *testing.T) {
	routes := NewBexioRoutes(emptyRegistry(), testStateSecret)
	testServiceUnavailable(t, routes.HandleUpdateFieldMappings)
}

func TestHandleUpdateFieldMappings_NoTenantID(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/bexio/mappings/contact", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "entity_type", "contact")
	routes.HandleUpdateFieldMappings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateFieldMappings_InvalidJSON(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/bexio/mappings/contact", invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "entity_type", "contact")
	routes.HandleUpdateFieldMappings(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateFieldMappings_MissingMappings(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/bexio/mappings/contact", jsonBody(t, map[string]interface{}{
		"mappings": []interface{}{},
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "entity_type", "contact")
	routes.HandleUpdateFieldMappings(rec, req)
	assertValidationError(t, rec, "mappings")
}

func TestHandleUpdateFieldMappings_RPCFails(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/bexio/mappings/contact", jsonBody(t, map[string]interface{}{
		"mappings": []map[string]interface{}{
			{"kmuhub_field": "name", "bexio_field": "name", "direction": "push", "required": true},
		},
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "entity_type", "contact")
	routes.HandleUpdateFieldMappings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandlePushInvoice ---

func TestHandlePushInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewBexioRoutes(emptyRegistry(), testStateSecret)
	testServiceUnavailable(t, routes.HandlePushInvoice)
}

func TestHandlePushInvoice_NoTenantID(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/bexio/push/invoice/x", nil)
	req = withChiURLParam(req, "invoice_id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandlePushInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandlePushInvoice_RPCFails(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/bexio/push/invoice/x", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "invoice_id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandlePushInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandlePushQuote ---

func TestHandlePushQuote_ServiceUnavailable(t *testing.T) {
	routes := NewBexioRoutes(emptyRegistry(), testStateSecret)
	testServiceUnavailable(t, routes.HandlePushQuote)
}

func TestHandlePushQuote_NoTenantID(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/bexio/push/quote/x", nil)
	req = withChiURLParam(req, "quote_id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandlePushQuote(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandlePushQuote_RPCFails(t *testing.T) {
	routes := NewBexioRoutes(registryWithService("biz"), testStateSecret)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/bexio/push/quote/x", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "quote_id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandlePushQuote(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestBexioResponseProtos_NeverExposeOAuthTokens is a structural guard in
// place of a live round trip: this package has no bufconn/fake-client
// harness for BexioIntegrationService (same boundary noted in every prior
// gateway coverage unit this run), so a successful RPC response can't be
// exercised here to prove no access/refresh token leaks into the JSON body.
// This asserts the invariant statically instead — none of the wire response
// messages this file maps into JSON may ever carry an OAuth token or client
// secret field. If one is ever added to the proto "to pass it through", this
// test fails instead of the field silently reaching the client.
func TestBexioResponseProtos_NeverExposeOAuthTokens(t *testing.T) {
	responses := []interface{}{
		bizv1.GetBexioAuthURLResponse{},
		bizv1.HandleBexioOAuthCallbackResponse{},
		bizv1.DisconnectBexioResponse{},
		bizv1.GetBexioConnectionStatusResponse{},
		bizv1.TriggerBexioSyncResponse{},
		bizv1.GetBexioSyncStatusResponse{},
		bizv1.UpdateBexioSyncConfigResponse{},
		bizv1.ListBexioSyncLogsResponse{},
		bizv1.BexioSyncLogEntry{},
		bizv1.GetBexioFieldMappingsResponse{},
		bizv1.BexioFieldMappingEntry{},
		bizv1.UpdateBexioFieldMappingsResponse{},
		bizv1.PushInvoiceToBexioResponse{},
		bizv1.PushQuoteToBexioResponse{},
	}
	for _, resp := range responses {
		typ := reflect.TypeOf(resp)
		for i := 0; i < typ.NumField(); i++ {
			name := strings.ToLower(typ.Field(i).Name)
			if strings.Contains(name, "token") || strings.Contains(name, "clientsecret") {
				t.Errorf("%s.%s: response proto exposes an OAuth token/secret field — the gateway must never surface it to the client", typ.Name(), typ.Field(i).Name)
			}
		}
	}
}
