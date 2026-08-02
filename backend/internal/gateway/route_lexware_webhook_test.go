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

// ============================================================================
// HandleConnect validation tests
// ============================================================================

// TestHandleLexwareConnect_MissingAPIKey verifies that api_key is required.
// Uses registryWithService("biz") so the gRPC client is obtained; getTenantID
// runs next (tenant set in context); decodeAndValidate fires last and rejects the empty body.
func TestHandleLexwareConnect_MissingAPIKey(t *testing.T) {
	routes := &LexwareRoutes{registry: registryWithService("biz")}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/integrations/lexware/connect",
		strings.NewReader(`{}`))
	req = withTenantID(req, testTenantID)
	routes.HandleConnect(rec, req)
	assertValidationError(t, rec, "api_key")
}

// TestHandleLexwareConnect_InvalidJSON verifies 400 on malformed JSON.
func TestHandleLexwareConnect_InvalidJSON(t *testing.T) {
	routes := &LexwareRoutes{registry: registryWithService("biz")}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/integrations/lexware/connect",
		strings.NewReader("{invalid"))
	req = withTenantID(req, testTenantID)
	routes.HandleConnect(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

// --- Payload decoding ---

// The Lexware Office API sends camelCase keys (eventType / resourceId /
// organizationId). The handler previously only read snake_case, so a real
// webhook decoded to an empty event type and was rejected with 400 before it
// ever reached the biz service.
func TestParseLexwareWebhookBody(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
		want    lexwareWebhookEvent
	}{
		{
			name: "lexware camelCase payload",
			body: `{"eventType":"contact.changed","resourceId":"res-1","organizationId":"org-1"}`,
			want: lexwareWebhookEvent{EventType: "contact.changed", ResourceID: "res-1", OrganizationID: "org-1"},
		},
		{
			name: "snake_case payload stays supported",
			body: `{"event_type":"contact.created","resource_id":"res-2","organization_id":"org-2"}`,
			want: lexwareWebhookEvent{EventType: "contact.created", ResourceID: "res-2", OrganizationID: "org-2"},
		},
		{
			name: "camelCase wins over an empty snake_case alias",
			body: `{"eventType":"invoice.status.changed","event_type":"","resourceId":"res-3"}`,
			want: lexwareWebhookEvent{EventType: "invoice.status.changed", ResourceID: "res-3"},
		},
		{
			name:    "malformed json",
			body:    `{invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLexwareWebhookBody([]byte(tt.body))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
