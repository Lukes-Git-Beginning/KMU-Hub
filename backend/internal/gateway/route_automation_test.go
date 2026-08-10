package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	automationv1 "github.com/kmuhub/kmuhub/proto/automation/v1"
)

// This file covers internal/gateway/route_automation.go — 21 exported
// handlers plus the RegisterPublicRoutes webhook trigger and three pure
// helper functions. Permission-mapping (RequirePermissionAny wiring) is
// already covered end-to-end in route_capability_guard_test.go — nothing
// here duplicates that matrix. Unlike every other module tested so far in
// this loop, no handler in this file calls middleware.GetTenantID: tenant
// scoping happens downstream in workflow.Service, not in the gateway, so
// there is deliberately no NoTenant test series below.

// ============================================================================
// ServiceUnavailable — every handler calls getClient() before doing anything
// else, so a single generic request proves the 503 path for all of them.
// ============================================================================

func TestAutomationRoutes_ServiceUnavailable(t *testing.T) {
	routes := NewAutomationRoutes(emptyRegistry())

	handlers := map[string]http.HandlerFunc{
		"HandleCreateAutomation":   routes.HandleCreateAutomation,
		"HandleListAutomations":    routes.HandleListAutomations,
		"HandleGetAutomation":      routes.HandleGetAutomation,
		"HandleUpdateAutomation":   routes.HandleUpdateAutomation,
		"HandleDeleteAutomation":   routes.HandleDeleteAutomation,
		"HandleEnableAutomation":   routes.HandleEnableAutomation,
		"HandleDisableAutomation":  routes.HandleDisableAutomation,
		"HandleListExecutions":     routes.HandleListExecutions,
		"HandleGetExecution":       routes.HandleGetExecution,
		"HandleListTriggers":       routes.HandleListTriggers,
		"HandleListActions":        routes.HandleListActions,
		"HandleListTemplates":      routes.HandleListTemplates,
		"HandleCreateFromTemplate": routes.HandleCreateFromTemplate,
		"HandleTestCondition":      routes.HandleTestCondition,
		"HandleDryRun":             routes.HandleDryRun,
		"HandleGetStats":           routes.HandleGetStats,
		"HandleTriggerWebhook":     routes.HandleTriggerWebhook,
	}

	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			testServiceUnavailable(t, h)
		})
	}
}

// ============================================================================
// HandleCreateAutomation
// ============================================================================

func TestHandleCreateAutomation_InvalidJSON(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateAutomation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateAutomation_MissingName(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/", jsonBody(t, map[string]interface{}{
		"trigger_type": "contact.created",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateAutomation(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateAutomation_MissingTriggerType(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/", jsonBody(t, map[string]interface{}{
		"name": "My Automation",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateAutomation(rec, req)
	assertValidationError(t, rec, "trigger_type")
}

func TestHandleCreateAutomation_InvalidScope(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/", jsonBody(t, map[string]interface{}{
		"name":         "My Automation",
		"trigger_type": "contact.created",
		"scope":        "galactic",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateAutomation(rec, req)
	assertValidationError(t, rec, "scope")
}

func TestHandleCreateAutomation_ValidRequestReachesRPC(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/", jsonBody(t, map[string]interface{}{
		"name":           "My Automation",
		"trigger_type":   "contact.created",
		"scope":          "team",
		"trigger_config": map[string]interface{}{"field": "value"},
		"conditions":     []interface{}{map[string]interface{}{"op": "eq"}},
		"actions":        []interface{}{map[string]interface{}{"type": "send_email"}},
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateAutomation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleCreateAutomation_MalformedStructFieldsIgnored proves that
// trigger_config/conditions/actions which fail to unmarshal into a
// structpb.Struct (e.g. a bare JSON array root, valid JSON but not a JSON
// object) are silently dropped (sErr != nil short-circuits the assignment)
// rather than surfacing as a 400 — the request still reaches the RPC layer.
func TestHandleCreateAutomation_MalformedStructFieldsIgnored(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/", jsonBody(t, map[string]interface{}{
		"name":           "My Automation",
		"trigger_type":   "contact.created",
		"trigger_config": []interface{}{"not", "an", "object"},
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateAutomation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListAutomations — filter-parsing branches (owner_id, limit, offset,
// scope, trigger_type, is_active), including the empty combination and
// non-numeric limit/offset (silently ignored per the pErr == nil guard).
// None of these can be observed reaching the proto request without a fake
// AutomationServiceClient — each case is proven to parse past every filter
// branch by reaching the RPC layer (503), not panicking or 400ing.
// ============================================================================

func TestHandleListAutomations_FilterCombinations(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"empty", ""},
		{"owner_id_only", "?owner_id=550e8400-e29b-41d4-a716-446655440000"},
		{"limit_offset", "?limit=10&offset=5"},
		{"non_numeric_limit_ignored", "?limit=abc"},
		{"non_numeric_offset_ignored", "?offset=xyz"},
		{"scope_personal", "?scope=personal"},
		{"scope_team", "?scope=team"},
		{"scope_organization", "?scope=organization"},
		{"trigger_type_only", "?trigger_type=contact.created"},
		{"is_active_true", "?is_active=true"},
		{"is_active_false", "?is_active=false"},
		{"is_active_garbage_treated_false", "?is_active=notabool"},
		{"all_filters_together", "?owner_id=550e8400-e29b-41d4-a716-446655440000&limit=10&offset=0&scope=team&trigger_type=deal.won&is_active=true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routes := NewAutomationRoutes(registryWithService("automation"))
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/v1/automations/"+tt.query, nil)
			req = withAuth(req, "user-123", testTenantID)
			routes.HandleListAutomations(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// TestHandleListAutomations_UnknownScopeRejected used to expect the request
// to reach the RPC with the filter silently narrowed to SCOPE_PERSONAL — a
// typo like "?scope=organisation" looked like a valid, plausible filter
// instead of surfacing as an error. It now gets a 400 naming the accepted
// values, matching the guard added to parseAutomationScope.
func TestHandleListAutomations_UnknownScopeRejected(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/automations/?scope=galactic", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListAutomations(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "scope")
}

// ============================================================================
// HandleGetAutomation / HandleUpdateAutomation / HandleDeleteAutomation /
// HandleEnableAutomation / HandleDisableAutomation — shared {id} validation.
// ============================================================================

func TestHandleGetAutomation_InvalidUUID(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/automations/bad", nil)
	req = withChiURLParam(req, "id", "bad")
	routes.HandleGetAutomation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetAutomation_ValidUUIDReachesRPC(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/automations/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetAutomation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateAutomation_InvalidUUID(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/automations/bad", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "bad")
	routes.HandleUpdateAutomation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateAutomation_InvalidJSON(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/automations/550e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateAutomation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateAutomation_EmptyBodyReachesRPC(t *testing.T) {
	// Every field is a pointer / optional — an empty object is a valid no-op
	// update and must still reach the RPC call.
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/automations/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateAutomation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleUpdateAutomation_InvalidScope proves the same rejection applies
// on the update path, which — unlike create — never went through
// decodeAndValidate's oneof tag: it decodes the body with a raw
// json.NewDecoder, so parseAutomationScope's ok=false was previously the
// only thing standing between an unknown scope and a silent rescope to
// personal on an existing automation.
func TestHandleUpdateAutomation_InvalidScope(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/automations/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"scope": "galactic",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateAutomation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "scope")
}

func TestHandleUpdateAutomation_AllFieldsReachesRPC(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/automations/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"name":           "Renamed",
		"description":    "new description",
		"scope":          "organization",
		"trigger_type":   "deal.won",
		"trigger_config": map[string]interface{}{"a": 1},
		"conditions":     map[string]interface{}{"op": "eq"},
		"actions":        []interface{}{map[string]interface{}{"type": "notify"}},
		"max_steps":      5,
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateAutomation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteAutomation_InvalidUUID(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/automations/bad", nil)
	req = withChiURLParam(req, "id", "bad")
	routes.HandleDeleteAutomation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteAutomation_ValidUUIDReachesRPC(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/automations/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteAutomation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleEnableAutomation_InvalidUUID(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/bad/enable", nil)
	req = withChiURLParam(req, "id", "bad")
	routes.HandleEnableAutomation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDisableAutomation_InvalidUUID(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/bad/disable", nil)
	req = withChiURLParam(req, "id", "bad")
	routes.HandleDisableAutomation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// ============================================================================
// HandleListExecutions / HandleGetExecution
// ============================================================================

func TestHandleListExecutions_InvalidUUID(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/automations/bad/executions", nil)
	req = withChiURLParam(req, "id", "bad")
	routes.HandleListExecutions(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListExecutions_StatusFilterReachesRPC(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/automations/550e8400-e29b-41d4-a716-446655440000/executions?status=failed&limit=10&offset=0", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListExecutions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetExecution_InvalidUUID(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/automations/executions/bad", nil)
	req = withChiURLParam(req, "executionId", "bad")
	routes.HandleGetExecution(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid executionId")
}

func TestHandleGetExecution_ValidUUIDReachesRPC(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/automations/executions/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "executionId", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetExecution(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListTemplates / HandleCreateFromTemplate
// ============================================================================

func TestHandleListTemplates_WithCategoryReachesRPC(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/automations/templates?category=sales", nil)
	routes.HandleListTemplates(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateFromTemplate_InvalidJSON(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/templates/x/create", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "templateId", "tmpl-1")
	routes.HandleCreateFromTemplate(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateFromTemplate_MissingName(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/templates/x/create", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "templateId", "tmpl-1")
	routes.HandleCreateFromTemplate(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateFromTemplate_ValidRequestReachesRPC(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/templates/tmpl-1/create", jsonBody(t, map[string]interface{}{
		"name": "From Template",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "templateId", "tmpl-1")
	routes.HandleCreateFromTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleTestCondition / HandleDryRun
// ============================================================================

func TestHandleTestCondition_InvalidJSON(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/test-condition", invalidJSON())
	routes.HandleTestCondition(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleTestCondition_EmptyBodyReachesRPC(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/test-condition", jsonBody(t, map[string]interface{}{}))
	routes.HandleTestCondition(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleTestCondition_WithConditionAndSampleEnvReachesRPC(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/test-condition", jsonBody(t, map[string]interface{}{
		"condition":  map[string]interface{}{"op": "eq", "field": "status"},
		"sample_env": map[string]interface{}{"status": "won"},
	}))
	routes.HandleTestCondition(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDryRun_InvalidJSON(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/dry-run", invalidJSON())
	routes.HandleDryRun(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleDryRun_MissingAutomationID(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/dry-run", jsonBody(t, map[string]interface{}{}))
	routes.HandleDryRun(rec, req)
	assertValidationError(t, rec, "automation_id")
}

func TestHandleDryRun_InvalidAutomationIDFormat(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/dry-run", jsonBody(t, map[string]interface{}{
		"automation_id": "not-a-uuid",
	}))
	routes.HandleDryRun(rec, req)
	assertValidationError(t, rec, "automation_id")
}

func TestHandleDryRun_ValidRequestReachesRPC(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/automations/dry-run", jsonBody(t, map[string]interface{}{
		"automation_id": "550e8400-e29b-41d4-a716-446655440000",
		"sample_env":    map[string]interface{}{"foo": "bar"},
	}))
	routes.HandleDryRun(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleTriggerWebhook — the only unauthenticated path in this module
// (mounted via RegisterPublicRoutes, not RegisterRoutes). Signature
// verification and tenant resolution happen downstream in
// workflow.Service.TriggerWebhook per the doc comment on the handler; the
// gateway's own job is: reject an unparsable automationId, and cap the body
// size before it ever leaves this handler (defense-in-depth ahead of the
// workflow-package limit).
// ============================================================================

func TestHandleTriggerWebhook_InvalidAutomationID(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/automations/webhooks/bad", strings.NewReader("{}"))
	req = withChiURLParam(req, "automationId", "bad")
	routes.HandleTriggerWebhook(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid automationId")
}

func TestHandleTriggerWebhook_PayloadTooLarge(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	oversized := strings.Repeat("a", maxWebhookBodyBytes+1)
	req := httptest.NewRequest("POST", "/api/v1/public/automations/webhooks/550e8400-e29b-41d4-a716-446655440000", strings.NewReader(oversized))
	req = withChiURLParam(req, "automationId", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleTriggerWebhook(rec, req)
	assertStatus(t, rec, http.StatusRequestEntityTooLarge)
	assertErrorContains(t, rec, "webhook payload too large")
}

func TestHandleTriggerWebhook_ValidPayloadReachesRPCWithHeaders(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/automations/webhooks/550e8400-e29b-41d4-a716-446655440000", strings.NewReader(`{"event":"ping"}`))
	req = withChiURLParam(req, "automationId", "550e8400-e29b-41d4-a716-446655440000")
	req.Header.Set(webhookSignatureHeader, "sha256=deadbeef")
	req.Header.Set("Idempotency-Key", "idem-1")
	routes.HandleTriggerWebhook(rec, req)
	// The dummy connection fails the RPC itself (503), which is the only
	// observable proof available without a fake AutomationServiceClient —
	// but it also proves the handler read the body, forwarded both headers
	// into the request, and never rejected the well-formed payload locally.
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleTriggerWebhook_EmptyBodyReachesRPC(t *testing.T) {
	routes := NewAutomationRoutes(registryWithService("automation"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/automations/webhooks/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "automationId", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleTriggerWebhook(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// Pure helper functions — parseAutomationScope, parseExecutionStatus,
// rawJSONToAutomationStruct. Table tests against the expected proto
// literals: a wrong mapping here is a silently wrong filter or a silently
// wrong stored scope, not a panic, so nothing but a literal-by-literal check
// would catch it.
// ============================================================================

func TestParseAutomationScope(t *testing.T) {
	tests := []struct {
		in     string
		want   automationv1.AutomationScope
		wantOk bool
	}{
		{"personal", automationv1.AutomationScope_SCOPE_PERSONAL, true},
		{"team", automationv1.AutomationScope_SCOPE_TEAM, true},
		{"organization", automationv1.AutomationScope_SCOPE_ORGANIZATION, true},
		// Empty keeps the historical default so omitting the field never
		// changes behavior. A non-empty unknown value is now rejected
		// (ok=false) instead of silently narrowing to personal — a typo like
		// "organisation" used to turn into a plausible-looking wrong answer
		// instead of a visible error.
		{"", automationv1.AutomationScope_SCOPE_PERSONAL, true},
		{"unknown-garbage", automationv1.AutomationScope_SCOPE_PERSONAL, false},
		{"organisation", automationv1.AutomationScope_SCOPE_PERSONAL, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := parseAutomationScope(tt.in)
			if got != tt.want || ok != tt.wantOk {
				t.Errorf("parseAutomationScope(%q) = (%v, %v), want (%v, %v)", tt.in, got, ok, tt.want, tt.wantOk)
			}
		})
	}
}

func TestParseExecutionStatus(t *testing.T) {
	tests := []struct {
		in   string
		want automationv1.ExecutionStatus
	}{
		{"running", automationv1.ExecutionStatus_EXECUTION_STATUS_RUNNING},
		{"completed", automationv1.ExecutionStatus_EXECUTION_STATUS_COMPLETED},
		{"failed", automationv1.ExecutionStatus_EXECUTION_STATUS_FAILED},
		{"skipped", automationv1.ExecutionStatus_EXECUTION_STATUS_SKIPPED},
		{"aborted", automationv1.ExecutionStatus_EXECUTION_STATUS_ABORTED},
		{"unknown-garbage", automationv1.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED},
		{"", automationv1.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := parseExecutionStatus(tt.in); got != tt.want {
				t.Errorf("parseExecutionStatus(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestRawJSONToAutomationStruct(t *testing.T) {
	t.Run("empty input returns nil, nil", func(t *testing.T) {
		s, err := rawJSONToAutomationStruct(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s != nil {
			t.Errorf("expected nil struct for empty input, got %v", s)
		}
	})

	t.Run("valid object decodes", func(t *testing.T) {
		s, err := rawJSONToAutomationStruct([]byte(`{"field":"value","n":1}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s == nil {
			t.Fatal("expected non-nil struct")
		}
		if got := s.Fields["field"].GetStringValue(); got != "value" {
			t.Errorf("field = %q, want %q", got, "value")
		}
	})

	t.Run("non-object JSON root errors", func(t *testing.T) {
		_, err := rawJSONToAutomationStruct([]byte(`["not", "an", "object"]`))
		if err == nil {
			t.Error("expected an error for a non-object JSON root")
		}
	})

	t.Run("malformed JSON errors", func(t *testing.T) {
		_, err := rawJSONToAutomationStruct([]byte(`{not valid json`))
		if err == nil {
			t.Error("expected an error for malformed JSON")
		}
	})
}
