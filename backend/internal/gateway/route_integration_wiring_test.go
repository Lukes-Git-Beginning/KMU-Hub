package gateway_test

import "testing"

// TestIntegrationRoutes_ReachableFromGatewayRouter asserts the integration
// endpoints are present in the router assembled the way cmd/gateway/main.go
// assembles it. NewIntegrationRoutes had no caller at all, so the whole
// /api/v1/integrations/* block was a 404 in production while the spec, the
// generated frontend types, integration-client.ts and both setup wizards
// described it as live.
//
// TestOpenAPIRouteDrift cannot catch that: it only checks "registered ⊆
// documented", so a route that exists on paper but is never mounted passes.
// This test checks the other direction for the paths that were missing.
//
// The block is mounted at /api/v1/integrations while route_bexio.go and
// route_lexware.go mount deeper subtrees (/api/v1/integrations/bexio,
// /api/v1/integrations/lexware) on the same router, so this also pins that the
// three coexist — chi refuses some overlapping Mount combinations outright.
func TestIntegrationRoutes_ReachableFromGatewayRouter(t *testing.T) {
	registered := registeredAPIv1Paths(t, buildGatewayRouter(t))

	want := []string{
		"/api/v1/integrations/configs",
		"/api/v1/integrations/configs/{platform}",
		"/api/v1/integrations/configs/{platform}/test",
		"/api/v1/integrations/configs/{platform}/mappings",
		"/api/v1/integrations/mappings/{id}",
		"/api/v1/integrations/link",
		"/api/v1/integrations/link/{platform}",
		"/api/v1/integrations/link/{platform}/status",
		"/api/v1/integrations/teams/webhook",
		"/api/v1/integrations/slack/interact",
		"/api/v1/integrations/slack/commands",
		"/api/v1/integrations/slack/oauth/install",
		"/api/v1/integrations/slack/oauth/callback",

		// Neighbouring subtrees under the same prefix — these were reachable
		// before this block was mounted and must stay reachable after.
		"/api/v1/integrations/bexio/status",
		"/api/v1/integrations/lexware/status",
	}
	for _, path := range want {
		if !registered[path] {
			t.Errorf("gateway router does not register %s", path)
		}
	}
}
