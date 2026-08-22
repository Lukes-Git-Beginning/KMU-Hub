package gateway

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/kmuhub/kmuhub/internal/biz/lexware"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

const testLexwareWebhookSecret = "test-lexware-webhook-secret"

func TestLexwareRoutes_ServiceName(t *testing.T) {
	routes := NewLexwareRoutes(emptyRegistry(), testLexwareWebhookSecret, false)
	if routes.ServiceName() != "biz" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "biz")
	}
}

// --- HandleConnect ---

func TestLexwareHandleConnect_ServiceUnavailable(t *testing.T) {
	routes := NewLexwareRoutes(emptyRegistry(), testLexwareWebhookSecret, false)
	testServiceUnavailable(t, routes.HandleConnect)
}

func TestLexwareHandleConnect_NoTenantID(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/connect", jsonBody(t, map[string]interface{}{
		"api_key": "secret-key",
	}))
	routes.HandleConnect(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestLexwareHandleConnect_InvalidJSON(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/connect", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleConnect(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestLexwareHandleConnect_MissingAPIKey(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/connect", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	routes.HandleConnect(rec, req)
	assertValidationError(t, rec, "api_key")
}

func TestLexwareHandleConnect_RPCFails(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/connect", jsonBody(t, map[string]interface{}{
		"api_key": "secret-key",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleConnect(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDisconnect ---

func TestLexwareHandleDisconnect_ServiceUnavailable(t *testing.T) {
	routes := NewLexwareRoutes(emptyRegistry(), testLexwareWebhookSecret, false)
	testServiceUnavailable(t, routes.HandleDisconnect)
}

func TestLexwareHandleDisconnect_NoTenantID(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/disconnect", nil)
	routes.HandleDisconnect(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestLexwareHandleDisconnect_RPCFails(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/disconnect", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleDisconnect(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetConnectionStatus ---

func TestLexwareHandleGetConnectionStatus_ServiceUnavailable(t *testing.T) {
	routes := NewLexwareRoutes(emptyRegistry(), testLexwareWebhookSecret, false)
	testServiceUnavailable(t, routes.HandleGetConnectionStatus)
}

func TestLexwareHandleGetConnectionStatus_NoTenantID(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/lexware/status", nil)
	routes.HandleGetConnectionStatus(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestLexwareHandleGetConnectionStatus_RPCFails(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/lexware/status", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGetConnectionStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleTestConnection ---

func TestLexwareHandleTestConnection_ServiceUnavailable(t *testing.T) {
	routes := NewLexwareRoutes(emptyRegistry(), testLexwareWebhookSecret, false)
	testServiceUnavailable(t, routes.HandleTestConnection)
}

func TestLexwareHandleTestConnection_NoTenantID(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/test", nil)
	routes.HandleTestConnection(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestLexwareHandleTestConnection_RPCFails(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/test", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleTestConnection(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleTriggerSync ---

func TestLexwareHandleTriggerSync_ServiceUnavailable(t *testing.T) {
	routes := NewLexwareRoutes(emptyRegistry(), testLexwareWebhookSecret, false)
	testServiceUnavailable(t, routes.HandleTriggerSync)
}

func TestLexwareHandleTriggerSync_NoTenantID(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/sync/trigger", nil)
	routes.HandleTriggerSync(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestLexwareHandleTriggerSync_InvalidJSON(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/sync/trigger", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleTriggerSync(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// TestHandleTriggerSync_EmptyBodySkipsDecode locks down the ContentLength==0
// short-circuit: a bodyless POST must NOT be rejected for lacking JSON, it
// must fall through with sync_type="" ("sync everything").
func TestLexwareHandleTriggerSync_EmptyBodySkipsDecode(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/sync/trigger", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleTriggerSync(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestLexwareHandleTriggerSync_ValidBody_RPCFails(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/sync/trigger", jsonBody(t, map[string]interface{}{
		"sync_type": "contacts",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleTriggerSync(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetSyncStatus ---

func TestLexwareHandleGetSyncStatus_ServiceUnavailable(t *testing.T) {
	routes := NewLexwareRoutes(emptyRegistry(), testLexwareWebhookSecret, false)
	testServiceUnavailable(t, routes.HandleGetSyncStatus)
}

func TestLexwareHandleGetSyncStatus_NoTenantID(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/lexware/sync/status", nil)
	routes.HandleGetSyncStatus(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestLexwareHandleGetSyncStatus_RPCFails(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/lexware/sync/status", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleGetSyncStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListSyncLogs ---

func TestLexwareHandleListSyncLogs_ServiceUnavailable(t *testing.T) {
	routes := NewLexwareRoutes(emptyRegistry(), testLexwareWebhookSecret, false)
	testServiceUnavailable(t, routes.HandleListSyncLogs)
}

func TestLexwareHandleListSyncLogs_NoTenantID(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/lexware/sync/logs", nil)
	routes.HandleListSyncLogs(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestLexwareHandleListSyncLogs_RPCFails(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/lexware/sync/logs?limit=5", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListSyncLogs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetFieldMappings ---

func TestLexwareHandleGetFieldMappings_ServiceUnavailable(t *testing.T) {
	routes := NewLexwareRoutes(emptyRegistry(), testLexwareWebhookSecret, false)
	testServiceUnavailable(t, routes.HandleGetFieldMappings)
}

func TestLexwareHandleGetFieldMappings_NoTenantID(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/lexware/mappings/contact", nil)
	req = withChiURLParam(req, "entity_type", "contact")
	routes.HandleGetFieldMappings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestLexwareHandleGetFieldMappings_RPCFails(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/lexware/mappings/contact", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "entity_type", "contact")
	routes.HandleGetFieldMappings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateFieldMappings ---

func TestLexwareHandleUpdateFieldMappings_ServiceUnavailable(t *testing.T) {
	routes := NewLexwareRoutes(emptyRegistry(), testLexwareWebhookSecret, false)
	testServiceUnavailable(t, routes.HandleUpdateFieldMappings)
}

func TestLexwareHandleUpdateFieldMappings_NoTenantID(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/lexware/mappings/contact", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "entity_type", "contact")
	routes.HandleUpdateFieldMappings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestLexwareHandleUpdateFieldMappings_InvalidJSON(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/lexware/mappings/contact", invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "entity_type", "contact")
	routes.HandleUpdateFieldMappings(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestLexwareHandleUpdateFieldMappings_MissingMappings(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/lexware/mappings/contact", jsonBody(t, map[string]interface{}{
		"mappings": []interface{}{},
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "entity_type", "contact")
	routes.HandleUpdateFieldMappings(rec, req)
	assertValidationError(t, rec, "mappings")
}

func TestLexwareHandleUpdateFieldMappings_RPCFails(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/integrations/lexware/mappings/contact", jsonBody(t, map[string]interface{}{
		"mappings": []map[string]interface{}{
			{"kmuhub_field": "name", "lexware_field": "name", "direction": "push", "required": true},
		},
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "entity_type", "contact")
	routes.HandleUpdateFieldMappings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandlePushInvoice ---

func TestLexwareHandlePushInvoice_ServiceUnavailable(t *testing.T) {
	routes := NewLexwareRoutes(emptyRegistry(), testLexwareWebhookSecret, false)
	testServiceUnavailable(t, routes.HandlePushInvoice)
}

func TestLexwareHandlePushInvoice_NoTenantID(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/push/invoice/x", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandlePushInvoice(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestLexwareHandlePushInvoice_RPCFails(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/push/invoice/x", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandlePushInvoice(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandlePushQuote ---

func TestLexwareHandlePushQuote_ServiceUnavailable(t *testing.T) {
	routes := NewLexwareRoutes(emptyRegistry(), testLexwareWebhookSecret, false)
	testServiceUnavailable(t, routes.HandlePushQuote)
}

func TestLexwareHandlePushQuote_NoTenantID(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/push/quote/x", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandlePushQuote(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestLexwareHandlePushQuote_RPCFails(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/push/quote/x", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandlePushQuote(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleWebhookEvent ---
// Public route (no auth middleware) — Lexware calls this directly. The only
// gate is the HMAC signature over the raw body via the X-Signature header.

func TestLexwareHandleWebhookEvent_SecretNotConfigured_Prod_Rejects(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), "", true)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/webhooks", jsonBody(t, map[string]interface{}{
		"eventType": "contact.created", "resourceId": "123",
	}))
	routes.HandleWebhookEvent(rec, req)
	assertStatus(t, rec, http.StatusInternalServerError)
	assertErrorContains(t, rec, "webhook secret not configured")
}

// TestHandleWebhookEvent_SecretNotConfigured_Dev_SkipsValidation locks down
// that a missing secret in dev/staging (isProd=false) does NOT block the
// request — it falls through to dispatch with only a warning logged.
func TestLexwareHandleWebhookEvent_SecretNotConfigured_Dev_SkipsValidation(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), "", false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/webhooks", jsonBody(t, map[string]interface{}{
		"eventType": "contact.created", "resourceId": "123",
	}))
	routes.HandleWebhookEvent(rec, req)
	// No signature was required, so the request reaches the RPC dispatch and
	// fails there — proving the HMAC gate, not the dummy connection, is what's exercised.
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestLexwareHandleWebhookEvent_MissingSignatureHeader(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/webhooks", jsonBody(t, map[string]interface{}{
		"eventType": "contact.created", "resourceId": "123",
	}))
	routes.HandleWebhookEvent(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "missing webhook signature")
}

func TestLexwareHandleWebhookEvent_InvalidSignature(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/webhooks", jsonBody(t, map[string]interface{}{
		"eventType": "contact.created", "resourceId": "123",
	}))
	req.Header.Set(lexwareWebhookSignatureHeader, "hmac-sha256=deadbeef")
	routes.HandleWebhookEvent(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "invalid webhook signature")
}

func TestLexwareHandleWebhookEvent_ValidSignature_InvalidJSONBody(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	body := []byte("{not valid json")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/webhooks", strings.NewReader(string(body)))
	req.Header.Set(lexwareWebhookSignatureHeader, lexware.ComputeHMAC(body, testLexwareWebhookSecret))
	routes.HandleWebhookEvent(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestLexwareHandleWebhookEvent_ValidSignature_ServiceUnavailable(t *testing.T) {
	routes := NewLexwareRoutes(emptyRegistry(), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	body := []byte(`{"eventType":"contact.created","resourceId":"123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/webhooks", strings.NewReader(string(body)))
	req.Header.Set(lexwareWebhookSignatureHeader, lexware.ComputeHMAC(body, testLexwareWebhookSecret))
	routes.HandleWebhookEvent(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestLexwareHandleWebhookEvent_ValidSignature_RPCFails(t *testing.T) {
	routes := NewLexwareRoutes(registryWithService("biz"), testLexwareWebhookSecret, false)
	rec := httptest.NewRecorder()
	body := []byte(`{"eventType":"contact.created","resourceId":"123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/lexware/webhooks", strings.NewReader(string(body)))
	req.Header.Set(lexwareWebhookSignatureHeader, lexware.ComputeHMAC(body, testLexwareWebhookSecret))
	routes.HandleWebhookEvent(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- parseLexwareWebhookBody ---

func TestParseLexwareWebhookBody_CamelCase(t *testing.T) {
	event, err := parseLexwareWebhookBody([]byte(`{"eventType":"contact.created","resourceId":"r1","organizationId":"o1"}`))
	if err != nil {
		t.Fatalf("parseLexwareWebhookBody: %v", err)
	}
	if event.EventType != "contact.created" || event.ResourceID != "r1" || event.OrganizationID != "o1" {
		t.Errorf("event = %+v, want camelCase fields decoded", event)
	}
}

func TestParseLexwareWebhookBody_SnakeCaseFallback(t *testing.T) {
	event, err := parseLexwareWebhookBody([]byte(`{"event_type":"invoice.status.changed","resource_id":"r2","organization_id":"o2"}`))
	if err != nil {
		t.Fatalf("parseLexwareWebhookBody: %v", err)
	}
	if event.EventType != "invoice.status.changed" || event.ResourceID != "r2" || event.OrganizationID != "o2" {
		t.Errorf("event = %+v, want snake_case fields decoded", event)
	}
}

// TestParseLexwareWebhookBody_CamelCaseWinsOverSnakeCase locks down
// firstNonEmpty's precedence: when both spellings are present (should never
// happen from real Lexware traffic, but a replayed fixture might), the
// camelCase value wins.
func TestParseLexwareWebhookBody_CamelCaseWinsOverSnakeCase(t *testing.T) {
	event, err := parseLexwareWebhookBody([]byte(`{"eventType":"camel","event_type":"snake","resourceId":"r","resource_id":"r-snake"}`))
	if err != nil {
		t.Fatalf("parseLexwareWebhookBody: %v", err)
	}
	if event.EventType != "camel" || event.ResourceID != "r" {
		t.Errorf("event = %+v, want camelCase to win", event)
	}
}

func TestParseLexwareWebhookBody_InvalidJSON(t *testing.T) {
	_, err := parseLexwareWebhookBody([]byte("{not json"))
	if err == nil {
		t.Error("expected an error for invalid JSON, got nil")
	}
}

// TestLexwareResponseProtos_NeverExposeCredentials is a structural guard in
// place of a live round trip: this package has no bufconn/fake-client
// harness for LexwareIntegrationService (same boundary noted in every prior
// gateway coverage unit this run), so a successful RPC response can't be
// exercised here to prove the connected API key never leaks into the JSON
// body. This asserts the invariant statically instead — none of the wire
// response messages this file maps into JSON may ever carry an API key or
// webhook secret field. If one is ever added to the proto "to pass it
// through", this test fails instead of the field silently reaching the client.
func TestLexwareResponseProtos_NeverExposeCredentials(t *testing.T) {
	responses := []interface{}{
		bizv1.ConnectLexwareResponse{},
		bizv1.DisconnectLexwareResponse{},
		bizv1.GetLexwareConnectionStatusResponse{},
		bizv1.TestLexwareConnectionResponse{},
		bizv1.TriggerLexwareSyncResponse{},
		bizv1.GetLexwareSyncStatusResponse{},
		bizv1.ListLexwareSyncLogsResponse{},
		bizv1.LexwareSyncLogEntry{},
		bizv1.GetLexwareFieldMappingsResponse{},
		bizv1.LexwareFieldMappingEntry{},
		bizv1.UpdateLexwareFieldMappingsResponse{},
		bizv1.PushInvoiceToLexwareResponse{},
		bizv1.PushQuoteToLexwareResponse{},
		bizv1.HandleLexwareWebhookEventResponse{},
	}
	for _, resp := range responses {
		typ := reflect.TypeOf(resp)
		for i := 0; i < typ.NumField(); i++ {
			name := strings.ToLower(typ.Field(i).Name)
			if strings.Contains(name, "apikey") || strings.Contains(name, "secret") || strings.Contains(name, "password") {
				t.Errorf("%s.%s: response proto exposes a credential field — the gateway must never surface it to the client", typ.Name(), typ.Field(i).Name)
			}
		}
	}
}
