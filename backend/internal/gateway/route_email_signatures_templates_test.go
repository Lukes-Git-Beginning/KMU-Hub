package gateway

// route_email_signatures_templates_test.go covers the signature and email
// template handlers in route_email.go (unit
// cov-gateway-email-signatures-templates, second part of the file after
// cov-gateway-email-messages-folders-sync). go tool cover -func on
// route_email.go before writing this file showed the twelve handlers below
// (plus ServiceName, already covered by
// route_email_folders_messages_sync_test.go) at 0%.
//
// Render-injection: internal/email/template's Service.Render only ever
// substitutes a fixed AllowedPlaceholders allow-list via strings.ReplaceAll,
// so an unknown key in the request body can never reach the output -- see
// TestRender_SubstitutesOnlyAllowedPlaceholders in
// internal/email/template/service_test.go, which already pins that case at
// the service layer. This gateway test file only exercises the HTTP
// boundary (decode/validate/dispatch), not template substitution itself --
// the gateway handler forwards body.Values verbatim to the RPC and has no
// rendering logic of its own to test.
//
// HTML sanitizing: neither internal/email/signature nor
// internal/email/template sanitizes HTMLContent/BodyHtml on write or read --
// both round-trip the caller-supplied string unchanged (verified by reading
// service.go in both packages; no sanitiz*/dompurify/bluemonday import in
// either). The desktop app's EmailTemplateDialog.tsx does call
// sanitizeHtml() (DOMPurify wrapper, desktop/src/renderer/src/lib/sanitize.ts)
// for its own preview and body panes, but ComposeInline.tsx/ComposeModal.tsx
// hand the selected template's raw body straight to setBody() when a
// template is actually chosen (handleTemplateSelect, no sanitizeHtml call) --
// only the dialog's own preview is sanitized, not the value that ends up in
// the compose editor. That gap is desktop-only (Frontend/Desktop is locked
// for this run, see ITERATION.md) and isn't something a backend Fix-Unit can
// close -- noted in JOURNAL.md for a future frontend session instead of
// queued here.
//
// Test names are prefixed with "Email" (TestEmailHandleX), matching
// route_email_folders_messages_sync_test.go: route_bexio.go/route_document.go
// declare handler methods with overlapping names on their own route structs
// and Go test function names are package-scoped regardless of receiver type.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// Signature Handlers
// ============================================================================

func TestEmailHandleListSignatures_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListSignatures)
}

func TestEmailHandleListSignatures_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/email/signatures?user_id=u-1", nil)
	routes.HandleListSignatures(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleGetSignature_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetSignature)
}

func TestEmailHandleGetSignature_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/email/signatures/sig-1", nil)
	req = withChiURLParam(req, "id", "sig-1")
	routes.HandleGetSignature(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleCreateSignature_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateSignature)
}

func TestEmailHandleCreateSignature_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/signatures", invalidJSON())
	routes.HandleCreateSignature(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestEmailHandleCreateSignature_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/signatures", jsonBody(t, map[string]interface{}{
		"user_id":      "u-1",
		"name":         "Standard",
		"html_content": "<p>Gruss</p>",
	}))
	routes.HandleCreateSignature(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleUpdateSignature_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdateSignature)
}

func TestEmailHandleUpdateSignature_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/email/signatures/sig-1", invalidJSON())
	req = withChiURLParam(req, "id", "sig-1")
	routes.HandleUpdateSignature(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestEmailHandleUpdateSignature_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/email/signatures/sig-1", jsonBody(t, map[string]interface{}{
		"name":         "Neu",
		"html_content": "<p>Neu</p>",
	}))
	req = withChiURLParam(req, "id", "sig-1")
	routes.HandleUpdateSignature(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleDeleteSignature_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleDeleteSignature)
}

func TestEmailHandleDeleteSignature_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/email/signatures/sig-1", nil)
	req = withChiURLParam(req, "id", "sig-1")
	routes.HandleDeleteSignature(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleSetDefaultSignature_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleSetDefaultSignature)
}

func TestEmailHandleSetDefaultSignature_MissingUserID(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/signatures/sig-1/default", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "sig-1")
	routes.HandleSetDefaultSignature(rec, req)
	assertValidationError(t, rec, "user_id")
}

func TestEmailHandleSetDefaultSignature_InvalidUserID(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/signatures/sig-1/default", jsonBody(t, map[string]interface{}{
		"user_id": "not-a-uuid",
	}))
	req = withChiURLParam(req, "id", "sig-1")
	routes.HandleSetDefaultSignature(rec, req)
	assertValidationError(t, rec, "user_id")
}

func TestEmailHandleSetDefaultSignature_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/signatures/sig-1/default", jsonBody(t, map[string]interface{}{
		"user_id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
	}))
	req = withChiURLParam(req, "id", "sig-1")
	routes.HandleSetDefaultSignature(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// Template Handlers
// ============================================================================

func TestEmailHandleListEmailTemplates_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListEmailTemplates)
}

func TestEmailHandleListEmailTemplates_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/email/templates", nil)
	routes.HandleListEmailTemplates(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleGetEmailTemplate_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetEmailTemplate)
}

func TestEmailHandleGetEmailTemplate_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/email/templates/tpl-1", nil)
	req = withChiURLParam(req, "id", "tpl-1")
	routes.HandleGetEmailTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleCreateEmailTemplate_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateEmailTemplate)
}

func TestEmailHandleCreateEmailTemplate_MissingName(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/templates", jsonBody(t, map[string]interface{}{
		"subject": "Betreff",
	}))
	routes.HandleCreateEmailTemplate(rec, req)
	assertValidationError(t, rec, "name")
}

func TestEmailHandleCreateEmailTemplate_InvalidVisibility(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/templates", jsonBody(t, map[string]interface{}{
		"name":       "Angebot",
		"visibility": "public",
	}))
	routes.HandleCreateEmailTemplate(rec, req)
	assertValidationError(t, rec, "visibility")
}

func TestEmailHandleCreateEmailTemplate_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/templates", jsonBody(t, map[string]interface{}{
		"name":       "Angebot",
		"visibility": "shared",
	}))
	routes.HandleCreateEmailTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleUpdateEmailTemplate_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdateEmailTemplate)
}

func TestEmailHandleUpdateEmailTemplate_InvalidVisibility(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/email/templates/tpl-1", jsonBody(t, map[string]interface{}{
		"visibility": "public",
	}))
	req = withChiURLParam(req, "id", "tpl-1")
	routes.HandleUpdateEmailTemplate(rec, req)
	assertValidationError(t, rec, "visibility")
}

func TestEmailHandleUpdateEmailTemplate_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/email/templates/tpl-1", jsonBody(t, map[string]interface{}{
		"name": "Neuer Name",
	}))
	req = withChiURLParam(req, "id", "tpl-1")
	routes.HandleUpdateEmailTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleDeleteEmailTemplate_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleDeleteEmailTemplate)
}

func TestEmailHandleDeleteEmailTemplate_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/email/templates/tpl-1", nil)
	req = withChiURLParam(req, "id", "tpl-1")
	routes.HandleDeleteEmailTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleRenderEmailTemplate_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleRenderEmailTemplate)
}

func TestEmailHandleRenderEmailTemplate_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/templates/tpl-1/render", invalidJSON())
	req = withChiURLParam(req, "id", "tpl-1")
	routes.HandleRenderEmailTemplate(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// TestEmailHandleRenderEmailTemplate_UnknownPlaceholderIgnoredAtBoundary
// documents that the gateway handler has no substitution logic of its own --
// an unknown key in "values" is decoded and forwarded to the RPC verbatim
// (the allow-list enforcement lives in internal/email/template.Service.Render,
// already pinned by TestRender_SubstitutesOnlyAllowedPlaceholders). This test
// only proves the handler accepts the body and reaches the RPC layer rather
// than rejecting it at decode time.
func TestEmailHandleRenderEmailTemplate_UnknownPlaceholderIgnoredAtBoundary(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/templates/tpl-1/render", jsonBody(t, map[string]interface{}{
		"values": map[string]string{
			"unknown_key":         "<script>alert(1)</script>",
			"contact_first_name":  "{{sender_name}}",
		},
	}))
	req = withChiURLParam(req, "id", "tpl-1")
	routes.HandleRenderEmailTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleRenderEmailTemplate_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/templates/tpl-1/render", jsonBody(t, map[string]interface{}{
		"values": map[string]string{"contact_first_name": "Anna"},
	}))
	req = withChiURLParam(req, "id", "tpl-1")
	routes.HandleRenderEmailTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
