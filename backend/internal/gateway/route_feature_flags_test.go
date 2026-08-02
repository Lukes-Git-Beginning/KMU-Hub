package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kmuhub/kmuhub/internal/featureflag"
)

func TestFeatureFlagRoutes_ServiceName(t *testing.T) {
	reg := featureflag.NewRegistry().Load(func(string) string { return "" })
	routes := NewFeatureFlagRoutes(reg)
	if routes.ServiceName() != "feature-flags" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "feature-flags")
	}
}

// TestHandleGetFlags_NoAuth verifies that unauthenticated requests are rejected.
func TestHandleGetFlags_NoAuth(t *testing.T) {
	reg := featureflag.NewRegistry().Load(func(string) string { return "" })
	routes := NewFeatureFlagRoutes(reg)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags", nil)
	withAuthRequired(routes.HandleGetFlags)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// TestHandleGetFlags_ReturnsAllFlags verifies the response shape and flag count.
func TestHandleGetFlags_ReturnsAllFlags(t *testing.T) {
	reg := featureflag.NewRegistry().Load(func(string) string { return "" })
	routes := NewFeatureFlagRoutes(reg)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetFlags(rec, req)

	assertStatus(t, rec, http.StatusOK)

	var resp struct {
		Flags   map[string]bool `json:"flags"`
		Version string          `json:"version"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Version != "v1" {
		t.Errorf("version = %q, want %q", resp.Version, "v1")
	}

	// Expect all 17 flags (14 modules + plugins.wasm/config/api)
	const wantCount = 17
	if len(resp.Flags) != wantCount {
		t.Errorf("flags count = %d, want %d", len(resp.Flags), wantCount)
	}

	// Spot-check key presence and default values
	checkFlag := func(key string, wantEnabled bool) {
		t.Helper()
		v, ok := resp.Flags[key]
		if !ok {
			t.Errorf("flag %q missing from response", key)
			return
		}
		if v != wantEnabled {
			t.Errorf("flag %q = %v, want %v", key, v, wantEnabled)
		}
	}

	checkFlag("plugins.wasm", false)
	checkFlag("plugins.config", true)
	checkFlag("plugins.api", false)
	checkFlag("modules.wiki", false)
	checkFlag("modules.produktion", false)
}

// TestHandleGetFlags_EnvOverride verifies env overrides are reflected in the response.
func TestHandleGetFlags_EnvOverride(t *testing.T) {
	env := map[string]string{
		"COSMI_MODULE_WIKI_ENABLED": "true",
	}
	reg := featureflag.NewRegistry().Load(func(k string) string { return env[k] })
	routes := NewFeatureFlagRoutes(reg)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags", nil)
	req = withUserID(req, "user-456")
	routes.HandleGetFlags(rec, req)

	assertStatus(t, rec, http.StatusOK)

	var resp struct {
		Flags map[string]bool `json:"flags"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Flags["modules.wiki"] {
		t.Error("modules.wiki: expected true after env override, got false")
	}
}
