package featureflag

import (
	"testing"
)

// TestUnknownKeyReturnsFalse verifies that IsEnabled returns false for unregistered keys.
func TestUnknownKeyReturnsFalse(t *testing.T) {
	r := NewRegistry().Load(func(string) string { return "" })
	if r.IsEnabled("does.not.exist") {
		t.Error("expected false for unknown flag key, got true")
	}
}

// TestDefaultWithNoEnv verifies that flags keep their default when no env var is set.
func TestDefaultWithNoEnv(t *testing.T) {
	r := NewRegistry().Load(func(string) string { return "" })

	// modules.wiki defaults to false
	if r.IsEnabled("modules.wiki") {
		t.Error("modules.wiki: expected default false, got true")
	}
	// plugins.config defaults to true
	if !r.IsEnabled("plugins.config") {
		t.Error("plugins.config: expected default true, got false")
	}
}

// TestEnvTrueOverridesDefault verifies that env="true" enables a flag regardless of default.
func TestEnvTrueOverridesDefault(t *testing.T) {
	env := map[string]string{
		"COSMI_MODULE_WIKI_ENABLED": "true",
	}
	r := NewRegistry().Load(func(k string) string { return env[k] })
	if !r.IsEnabled("modules.wiki") {
		t.Error("modules.wiki: expected true after env override, got false")
	}
}

// TestEnvFalseOverridesDefault verifies that env="false" disables a flag regardless of default.
func TestEnvFalseOverridesDefault(t *testing.T) {
	env := map[string]string{
		"COSMI_CONFIG_PLUGINS_ENABLED": "false",
	}
	r := NewRegistry().Load(func(k string) string { return env[k] })
	if r.IsEnabled("plugins.config") {
		t.Error("plugins.config: expected false after env override, got true")
	}
}

// TestEnvNumericValues verifies that "1" and "0" are accepted as boolean equivalents.
func TestEnvNumericValues(t *testing.T) {
	env := map[string]string{
		"COSMI_MODULE_WIKI_ENABLED":    "1",
		"COSMI_CONFIG_PLUGINS_ENABLED": "0",
	}
	r := NewRegistry().Load(func(k string) string { return env[k] })
	if !r.IsEnabled("modules.wiki") {
		t.Error("modules.wiki: expected true for env '1', got false")
	}
	if r.IsEnabled("plugins.config") {
		t.Error("plugins.config: expected false for env '0', got true")
	}
}

// TestGarbageEnvFallsBackToDefault verifies that an unrecognised env value uses the flag default.
func TestGarbageEnvFallsBackToDefault(t *testing.T) {
	env := map[string]string{
		"COSMI_MODULE_WIKI_ENABLED": "yes_please",
	}
	r := NewRegistry().Load(func(k string) string { return env[k] })
	// default for modules.wiki is false
	if r.IsEnabled("modules.wiki") {
		t.Error("modules.wiki: expected default false for garbage env, got true")
	}
}

// TestAllContainsAllFlags verifies All() returns every registered flag with its resolved value.
func TestAllContainsAllFlags(t *testing.T) {
	r := NewRegistry().Load(func(string) string { return "" })
	all := r.All()

	expectedKeys := []string{
		"modules.wiki", "modules.berichte", "modules.formulare", "modules.helpdesk",
		"modules.vertraege", "modules.buchhaltung", "modules.video", "modules.rapporte",
		"modules.schichten", "modules.fuhrpark", "modules.vermietung", "modules.inventar",
		"modules.einkauf", "modules.produktion",
		"plugins.wasm", "plugins.config", "plugins.api",
		"integrations.bexio",
	}

	if len(all) != len(expectedKeys) {
		t.Errorf("All() returned %d entries, want %d", len(all), len(expectedKeys))
	}

	for _, key := range expectedKeys {
		if _, ok := all[key]; !ok {
			t.Errorf("All() missing key %q", key)
		}
	}
}

// TestKeysSorted verifies Keys() returns deterministic sorted output.
func TestKeysSorted(t *testing.T) {
	r := NewRegistry().Load(func(string) string { return "" })
	keys := r.Keys()

	if len(keys) == 0 {
		t.Fatal("Keys() returned empty slice")
	}

	for i := 1; i < len(keys); i++ {
		if keys[i] < keys[i-1] {
			t.Errorf("Keys() not sorted: %q comes after %q", keys[i-1], keys[i])
		}
	}
}

// TestWASMDefaultOff verifies plugins.wasm is off by default (Sprint 0 requirement).
func TestWASMDefaultOff(t *testing.T) {
	r := NewRegistry().Load(func(string) string { return "" })
	if r.IsEnabled("plugins.wasm") {
		t.Error("plugins.wasm must default to false (WASM off in production)")
	}
}
