package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kmuhub/kmuhub/internal/biz/lexware"
)

// newLexwareWebhookReq is a test helper that creates a POST request to the
// Lexware webhook endpoint with an optional HMAC signature header.
func newLexwareWebhookReq(t *testing.T, body string, secret string) *http.Request {
	t.Helper()
	r := httptest.NewRequest("POST", "/api/v1/integrations/lexware/webhooks", strings.NewReader(body))
	if secret != "" {
		sig := lexware.ComputeHMAC([]byte(body), secret)
		r.Header.Set("X-Signature", sig)
	}
	return r
}

const lexwareTestPayload = `{"event_type":"contact.created","resource_id":"res-1","organization_id":"org-1"}`
const lexwareTestSecret = "test-lexware-webhook-secret-xyz"

// --- No secret configured (dev mode) ---

func TestHandleLexwareWebhook_NoSecret_NonProd_Passes(t *testing.T) {
	// Empty secret + isProd=false → skip HMAC check, warn only.
	// gRPC call will fail (no service registered) but we only test the
	// HMAC layer here — the handler returns 503 when the biz service is unavailable,
	// which proves it got past the signature gate.
	routes := &LexwareRoutes{
		registry:      emptyRegistry(),
		webhookSecret: "",
		isProd:        false,
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", strings.NewReader(lexwareTestPayload))
	// No X-Signature header.
	routes.HandleWebhookEvent(rec, req)
	// Should reach the gRPC call and return 503 (service unavailable), not 401/500.
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleLexwareWebhook_NoSecret_Prod_Returns500(t *testing.T) {
	// Empty secret + isProd=true → must reject with 500 (misconfiguration).
	routes := &LexwareRoutes{
		registry:      emptyRegistry(),
		webhookSecret: "",
		isProd:        true,
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", strings.NewReader(lexwareTestPayload))
	routes.HandleWebhookEvent(rec, req)
	assertStatus(t, rec, http.StatusInternalServerError)
}

// --- Valid signature ---

func TestHandleLexwareWebhook_ValidSignature_Passes(t *testing.T) {
	routes := &LexwareRoutes{
		registry:      emptyRegistry(),
		webhookSecret: lexwareTestSecret,
		isProd:        false,
	}
	rec := httptest.NewRecorder()
	req := newLexwareWebhookReq(t, lexwareTestPayload, lexwareTestSecret)
	routes.HandleWebhookEvent(rec, req)
	// Gets past HMAC gate — gRPC fails with 503.
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Invalid signature ---

func TestHandleLexwareWebhook_InvalidSignature_Returns401(t *testing.T) {
	routes := &LexwareRoutes{
		registry:      emptyRegistry(),
		webhookSecret: lexwareTestSecret,
		isProd:        false,
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", strings.NewReader(lexwareTestPayload))
	req.Header.Set("X-Signature", "hmac-sha256=deadbeef00000000000000000000000000000000000000000000000000000000")
	routes.HandleWebhookEvent(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "invalid webhook signature")
}

// --- Missing header ---

func TestHandleLexwareWebhook_MissingSignatureHeader_Returns401(t *testing.T) {
	routes := &LexwareRoutes{
		registry:      emptyRegistry(),
		webhookSecret: lexwareTestSecret,
		isProd:        false,
	}
	rec := httptest.NewRecorder()
	// No X-Signature header at all.
	req := httptest.NewRequest("POST", "/", strings.NewReader(lexwareTestPayload))
	routes.HandleWebhookEvent(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "missing webhook signature")
}
