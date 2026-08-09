package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kmuhub/kmuhub/internal/featureflag"
)

func settingsRoutesForPreferences(reg *ServiceRegistry) *SettingsRoutes {
	return NewSettingsRoutes(reg, featureflag.NewRegistry().Load(func(string) string { return "" }))
}

func TestHandleGetUserPreferences_ServiceUnavailable(t *testing.T) {
	routes := settingsRoutesForPreferences(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetUserPreferences)
}

func TestHandlePutUserPreferences_ServiceUnavailable(t *testing.T) {
	routes := settingsRoutesForPreferences(emptyRegistry())
	testServiceUnavailable(t, routes.HandlePutUserPreferences)
}

// An unknown preference key must be rejected before the settings service is
// even called — the allowlist is the entire point of this thin wrapper over
// the generic (unrestricted) /settings/{module_id}/user endpoint.
func TestHandlePutUserPreferences_UnknownKeyRejected(t *testing.T) {
	routes := settingsRoutesForPreferences(registryWithService("auth"))

	req := httptest.NewRequest("PUT", "/api/v1/users/preferences", jsonBody(t, map[string]any{
		"preferences": map[string]any{"favoriteColor": "blue"},
	}))
	req = withAuth(req, "22222222-2222-2222-2222-222222222222", testTenantID)

	rec := httptest.NewRecorder()
	routes.HandlePutUserPreferences(rec, req)

	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "favoriteColor")
}

// A known key still passes validation (and only fails downstream because the
// dummy gRPC connection in registryWithService has no real server behind it).
func TestHandlePutUserPreferences_KnownKeyPassesValidation(t *testing.T) {
	routes := settingsRoutesForPreferences(registryWithService("auth"))

	req := httptest.NewRequest("PUT", "/api/v1/users/preferences", jsonBody(t, map[string]any{
		"preferences": map[string]any{"language": "de"},
	}))
	req = withAuth(req, "22222222-2222-2222-2222-222222222222", testTenantID)

	rec := httptest.NewRecorder()
	routes.HandlePutUserPreferences(rec, req)

	// Never a 400: validation passed, the failure (if any) is the unreachable
	// dummy RPC connection, not the request shape.
	if rec.Code == http.StatusBadRequest {
		t.Fatalf("known key must not be rejected: status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandlePutUserPreferences_NoUserID(t *testing.T) {
	routes := settingsRoutesForPreferences(registryWithService("auth"))

	req := httptest.NewRequest("PUT", "/api/v1/users/preferences", jsonBody(t, map[string]any{
		"preferences": map[string]any{"language": "de"},
	}))
	req = withTenantID(req, testTenantID)

	rec := httptest.NewRecorder()
	routes.HandlePutUserPreferences(rec, req)

	assertStatus(t, rec, http.StatusUnauthorized)
}

// A user_id in the request body has no field to land in: putUserPreferencesRequest
// only declares "preferences", so decoding drops any other top-level key instead
// of it silently becoming the write target. This is what keeps HandlePutUserPreferences
// from ever reading anyone but middleware.GetUserID(ctx) as the target user.
func TestPutUserPreferencesRequest_BodyUserIDIgnored(t *testing.T) {
	raw := []byte(`{"user_id":"11111111-1111-1111-1111-111111111111","preferences":{"language":"de"}}`)

	var req putUserPreferencesRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(req.Preferences) != 1 {
		t.Fatalf("expected exactly one preference key, got %+v", req.Preferences)
	}
	if _, ok := req.Preferences["user_id"]; ok {
		t.Fatalf("user_id leaked into Preferences: %+v", req.Preferences)
	}
}
