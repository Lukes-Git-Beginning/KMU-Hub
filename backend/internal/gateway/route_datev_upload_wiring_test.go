package gateway_test

import "testing"

// TestDatevUploadRoutes_ReachableFromGatewayRouter asserts the DATEV upload
// endpoints are present in the router assembled the way cmd/gateway/main.go
// assembles it. NewDatevUploadRoutes had no caller at all, so the whole
// /api/v1/finance/datev/* block was a 404 in production while the spec, the
// generated frontend types and the frontend client all described it as live.
//
// TestOpenAPIRouteDrift cannot catch that: it only checks "registered ⊆
// documented", so a route that exists on paper but is never mounted passes.
// This test checks the other direction for the paths that were missing.
func TestDatevUploadRoutes_ReachableFromGatewayRouter(t *testing.T) {
	registered := registeredAPIv1Paths(t, buildGatewayRouter(t))

	want := []string{
		"/api/v1/finance/datev/oauth/authorize",
		"/api/v1/finance/datev/oauth/callback",
		"/api/v1/finance/datev/disconnect",
		"/api/v1/finance/datev/status",
		"/api/v1/finance/datev/upload",
		"/api/v1/finance/datev/upload/beleg/{invoice_id}",
		"/api/v1/finance/datev/config",
		"/api/v1/finance/datev/upload/logs",
	}
	for _, path := range want {
		if !registered[path] {
			t.Errorf("gateway router does not register %s", path)
		}
	}
}
