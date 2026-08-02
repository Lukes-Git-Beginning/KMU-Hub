package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kmuhub/kmuhub/internal/featureflag"
)

// wasmFlagsOFF returns a featureflag.Registry with plugins.wasm at its
// (production) default of false.
func wasmFlagsOFF() *featureflag.Registry {
	return featureflag.NewRegistry().Load(func(string) string { return "" })
}

// wasmFlagsON returns a featureflag.Registry with plugins.wasm explicitly enabled.
func wasmFlagsON() *featureflag.Registry {
	return featureflag.NewRegistry().Load(func(k string) string {
		if k == "COSMI_WASM_PLUGINS_ENABLED" {
			return "true"
		}
		return ""
	})
}

// TestHandleCreateManifest_WASM_RejectedWhenFlagOff verifies the gateway-side
// defense-in-depth gate: a plugin_type=wasm manifest is rejected with 400
// while plugins.wasm is off, instead of being silently accepted into a
// no_wasm build (or a hook dispatcher that isn't wired up) where it would
// never execute.
func TestHandleCreateManifest_WASM_RejectedWhenFlagOff(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), wasmFlagsOFF())

	rec := httptest.NewRecorder()
	body := `{"slug":"my-plugin","name":"My Plugin","plugin_type":"wasm"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/manifests", strings.NewReader(body))
	routes.HandleCreateManifest(rec, req)

	assertStatus(t, rec, http.StatusBadRequest)
	if !strings.Contains(rec.Body.String(), "plugins.wasm") {
		t.Errorf("expected error body to mention plugins.wasm, got: %s", rec.Body.String())
	}
}

// TestHandleCreateManifest_WASM_PassesGateWhenFlagOn verifies that with
// plugins.wasm enabled, a wasm manifest is not blocked by the gateway gate
// (it proceeds to the gRPC call, which fails with 503 in this unit test
// because no real plugin backend is dialed — anything other than the gate's
// 400 proves the request was not rejected by the flag check).
func TestHandleCreateManifest_WASM_PassesGateWhenFlagOn(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), wasmFlagsON())

	rec := httptest.NewRecorder()
	body := `{"slug":"my-plugin","name":"My Plugin","plugin_type":"wasm"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/manifests", strings.NewReader(body))
	routes.HandleCreateManifest(rec, req)

	if rec.Code == http.StatusBadRequest {
		t.Errorf("wasm manifest should not be gate-rejected when plugins.wasm=true; body = %s", rec.Body.String())
	}
}

// TestHandleCreateManifest_Config_NotGated verifies that config-type
// manifests are unaffected by the plugins.wasm flag.
func TestHandleCreateManifest_Config_NotGated(t *testing.T) {
	routes := NewPluginRoutes(registryWithService("plugin"), wasmFlagsOFF())

	rec := httptest.NewRecorder()
	body := `{"slug":"my-plugin","name":"My Plugin","plugin_type":"config"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/manifests", strings.NewReader(body))
	routes.HandleCreateManifest(rec, req)

	if rec.Code == http.StatusBadRequest {
		t.Errorf("config manifest should not be gated by plugins.wasm; body = %s", rec.Body.String())
	}
}
