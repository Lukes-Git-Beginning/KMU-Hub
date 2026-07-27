package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// integrationTestMappingID is any well-formed UUID — the mapping routes are
// rejected by the role guard long before the id is looked at.
const integrationTestMappingID = "3f1c2a54-9b6d-4e21-8a77-5c0d1e2f3a4b"

// integrationRouter registers the integration routes on a fresh chi router the
// way cmd/gateway/main.go does — registrar only, no webhook setters, which is
// the production wiring: the three inbound-webhook handlers need a direct
// integration.Repository and arrive unauthenticated, so they stay unset.
func integrationRouter(t *testing.T) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	NewIntegrationRoutes(emptyRegistry()).RegisterRoutes(r, func(next http.Handler) http.Handler { return next })
	return r
}

// TestIntegrationRoutes_Registered walks the router and asserts every
// integration endpoint is mounted. NewIntegrationRoutes had no caller anywhere
// in the repo, so all fifteen /api/v1/integrations/* paths were a 404 against a
// real gateway while openapi.yaml, integration-client.ts and the Slack/Teams
// setup wizards all described them as live. Walking the router is the check;
// "the handler exists" is not.
func TestIntegrationRoutes_Registered(t *testing.T) {
	r := integrationRouter(t)

	registered := make(map[string]bool)
	err := chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		// chi.Walk appends a trailing slash for handlers mounted at the root of
		// an r.Route(...) sub-router; the spec documents them without one.
		registered[method+" "+strings.TrimSuffix(route, "/")] = true
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk failed: %v", err)
	}

	want := []string{
		"GET /api/v1/integrations/configs",
		"POST /api/v1/integrations/configs",
		"GET /api/v1/integrations/configs/{platform}",
		"PUT /api/v1/integrations/configs/{platform}",
		"DELETE /api/v1/integrations/configs/{platform}",
		"POST /api/v1/integrations/configs/{platform}/test",
		"GET /api/v1/integrations/configs/{platform}/mappings",
		"POST /api/v1/integrations/configs/{platform}/mappings",
		"PUT /api/v1/integrations/mappings/{id}",
		"DELETE /api/v1/integrations/mappings/{id}",
		"POST /api/v1/integrations/link",
		"DELETE /api/v1/integrations/link/{platform}",
		"GET /api/v1/integrations/link/{platform}/status",
		"POST /api/v1/integrations/teams/webhook",
		"POST /api/v1/integrations/slack/interact",
		"POST /api/v1/integrations/slack/commands",
		"GET /api/v1/integrations/slack/oauth/install",
		"GET /api/v1/integrations/slack/oauth/callback",
	}
	for _, w := range want {
		if !registered[w] {
			t.Errorf("route not registered: %s", w)
		}
	}
}

// TestIntegrationRoutes_WebhooksRefuseCleanly pins that all five unauthenticated
// webhook paths answer with a status instead of panicking. A nil-pointer panic
// on any of them would be a remotely triggerable crash, so this is a guard, not
// a formality.
//
// The three inbound webhooks proxy to the notification service; the registry in
// this test has none, so they surface 503 rather than reaching a handler. The
// two OAuth paths answer 501 by decision — see respondOAuthNotAvailable.
func TestIntegrationRoutes_WebhooksRefuseCleanly(t *testing.T) {
	r := integrationRouter(t)

	cases := []struct {
		method     string
		path       string
		wantStatus int
	}{
		{http.MethodPost, "/api/v1/integrations/teams/webhook", http.StatusServiceUnavailable},
		{http.MethodPost, "/api/v1/integrations/slack/interact", http.StatusServiceUnavailable},
		{http.MethodPost, "/api/v1/integrations/slack/commands", http.StatusServiceUnavailable},
		{http.MethodGet, "/api/v1/integrations/slack/oauth/install", http.StatusNotImplemented},
		{http.MethodGet, "/api/v1/integrations/slack/oauth/callback", http.StatusNotImplemented},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, strings.NewReader("")))

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d", rec.Code, tc.wantStatus)
			}
			if strings.TrimSpace(rec.Body.String()) == "" {
				t.Error("body is empty; want a message naming the reason")
			}
		})
	}
}

// TestIntegrationRoutes_AdminRoutesRequireRole pins that the config and mapping
// routes sit behind RequireRole("admin"). They are registered inside an
// r.Group with the role middleware; a future reshuffle that moves a handler out
// of that group would expose tenant integration credentials metadata to every
// authenticated user. The router here uses a pass-through auth middleware, so a
// request without roles in its context reaches RequireRole and is rejected.
func TestIntegrationRoutes_AdminRoutesRequireRole(t *testing.T) {
	r := integrationRouter(t)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/integrations/configs"},
		{http.MethodPost, "/api/v1/integrations/configs"},
		{http.MethodGet, "/api/v1/integrations/configs/slack"},
		{http.MethodPut, "/api/v1/integrations/configs/slack"},
		{http.MethodDelete, "/api/v1/integrations/configs/slack"},
		{http.MethodPost, "/api/v1/integrations/configs/slack/test"},
		{http.MethodGet, "/api/v1/integrations/configs/slack/mappings"},
		{http.MethodPost, "/api/v1/integrations/configs/slack/mappings"},
		{http.MethodPut, "/api/v1/integrations/mappings/" + integrationTestMappingID},
		{http.MethodDelete, "/api/v1/integrations/mappings/" + integrationTestMappingID},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}")))

			if rec.Code != http.StatusForbidden {
				t.Errorf("status = %d; want %d — route is not behind RequireRole(admin)", rec.Code, http.StatusForbidden)
			}
		})
	}
}
