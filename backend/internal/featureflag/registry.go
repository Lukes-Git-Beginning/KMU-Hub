// Package featureflag provides a typed feature-flag registry with env-driven loading.
// Flags are registered at startup via NewRegistry and resolved once via Load(os.Getenv).
// After Load, call IsEnabled(key) at the call site — no globals, no init() abuse.
//
// AI-First Note: Each registration carries risk/llm_toggle_safe metadata so that
// future LLM-managed toggles can reason about the blast radius of a flag change.
package featureflag

import (
	"log/slog"
	"sort"
)

// Risk classifies the blast-radius of toggling a flag.
type Risk string

const (
	// SafeRisk flags can be toggled without service restarts or data migrations.
	SafeRisk Risk = "safe"
	// BreakingRisk flags may cause API incompatibilities or require migrations.
	BreakingRisk Risk = "breaking"
	// SecurityRisk flags affect authentication, authorization, or encryption.
	SecurityRisk Risk = "security"
)

// Flag is the static definition of a feature flag.
type Flag struct {
	Key            string
	DefaultEnabled bool
	EnvVar         string
	Description    string
	Risk           Risk
	LLMToggleSafe  bool // hint: true means an LLM agent may safely toggle this flag autonomously
}

// Registry holds all known flags and their resolved runtime values.
type Registry struct {
	flags  map[string]Flag
	values map[string]bool
}

// NewRegistry creates a Registry pre-populated with all known flags.
// Call Load to resolve env overrides before calling IsEnabled.
func NewRegistry() *Registry {
	r := &Registry{
		flags:  make(map[string]Flag),
		values: make(map[string]bool),
	}

	// ── Module flags (14 optional industry modules, all off by default) ─────────

	// risk: safe | llm_toggle_safe: true | description: enables the Wiki module
	r.register(Flag{Key: "modules.wiki", DefaultEnabled: false, EnvVar: "COSMI_MODULE_WIKI_ENABLED", Description: "enables the Wiki module", Risk: SafeRisk, LLMToggleSafe: true})

	// risk: safe | llm_toggle_safe: true | description: enables the Berichte (reports) module
	r.register(Flag{Key: "modules.berichte", DefaultEnabled: false, EnvVar: "COSMI_MODULE_BERICHTE_ENABLED", Description: "enables the Berichte (reports) module", Risk: SafeRisk, LLMToggleSafe: true})

	// risk: safe | llm_toggle_safe: true | description: enables the Formulare (forms) module
	r.register(Flag{Key: "modules.formulare", DefaultEnabled: false, EnvVar: "COSMI_MODULE_FORMULARE_ENABLED", Description: "enables the Formulare (forms) module", Risk: SafeRisk, LLMToggleSafe: true})

	// risk: safe | llm_toggle_safe: true | description: enables the Helpdesk module
	r.register(Flag{Key: "modules.helpdesk", DefaultEnabled: false, EnvVar: "COSMI_MODULE_HELPDESK_ENABLED", Description: "enables the Helpdesk module", Risk: SafeRisk, LLMToggleSafe: true})

	// risk: safe | llm_toggle_safe: true | description: enables the Vertraege (contracts) module
	r.register(Flag{Key: "modules.vertraege", DefaultEnabled: false, EnvVar: "COSMI_MODULE_VERTRAEGE_ENABLED", Description: "enables the Vertraege (contracts) module", Risk: SafeRisk, LLMToggleSafe: true})

	// risk: safe | llm_toggle_safe: true | description: enables the Buchhaltung (accounting) module
	r.register(Flag{Key: "modules.buchhaltung", DefaultEnabled: false, EnvVar: "COSMI_MODULE_BUCHHALTUNG_ENABLED", Description: "enables the Buchhaltung (accounting) module", Risk: SafeRisk, LLMToggleSafe: true})

	// risk: safe | llm_toggle_safe: true | description: enables the Video (meetings) module
	r.register(Flag{Key: "modules.video", DefaultEnabled: false, EnvVar: "COSMI_MODULE_VIDEO_ENABLED", Description: "enables the Video (meetings) module", Risk: SafeRisk, LLMToggleSafe: true})

	// risk: safe | llm_toggle_safe: true | description: enables the Rapporte (field reports) module
	r.register(Flag{Key: "modules.rapporte", DefaultEnabled: false, EnvVar: "COSMI_MODULE_RAPPORTE_ENABLED", Description: "enables the Rapporte (field reports) module", Risk: SafeRisk, LLMToggleSafe: true})

	// risk: safe | llm_toggle_safe: true | description: enables the Schichten (shift planning) module
	r.register(Flag{Key: "modules.schichten", DefaultEnabled: false, EnvVar: "COSMI_MODULE_SCHICHTEN_ENABLED", Description: "enables the Schichten (shift planning) module", Risk: SafeRisk, LLMToggleSafe: true})

	// risk: safe | llm_toggle_safe: true | description: enables the Fuhrpark (fleet management) module
	r.register(Flag{Key: "modules.fuhrpark", DefaultEnabled: false, EnvVar: "COSMI_MODULE_FUHRPARK_ENABLED", Description: "enables the Fuhrpark (fleet management) module", Risk: SafeRisk, LLMToggleSafe: true})

	// risk: safe | llm_toggle_safe: true | description: enables the Vermietung (rental) module
	r.register(Flag{Key: "modules.vermietung", DefaultEnabled: false, EnvVar: "COSMI_MODULE_VERMIETUNG_ENABLED", Description: "enables the Vermietung (rental) module", Risk: SafeRisk, LLMToggleSafe: true})

	// risk: safe | llm_toggle_safe: true | description: enables the Inventar (inventory) module
	r.register(Flag{Key: "modules.inventar", DefaultEnabled: false, EnvVar: "COSMI_MODULE_INVENTAR_ENABLED", Description: "enables the Inventar (inventory) module", Risk: SafeRisk, LLMToggleSafe: true})

	// risk: safe | llm_toggle_safe: true | description: enables the Einkauf (purchasing) module
	r.register(Flag{Key: "modules.einkauf", DefaultEnabled: false, EnvVar: "COSMI_MODULE_EINKAUF_ENABLED", Description: "enables the Einkauf (purchasing) module", Risk: SafeRisk, LLMToggleSafe: true})

	// risk: safe | llm_toggle_safe: true | description: enables the Produktion (production) module
	r.register(Flag{Key: "modules.produktion", DefaultEnabled: false, EnvVar: "COSMI_MODULE_PRODUKTION_ENABLED", Description: "enables the Produktion (production) module", Risk: SafeRisk, LLMToggleSafe: true})

	// ── Plugin flags ─────────────────────────────────────────────────────────────

	// risk: breaking | llm_toggle_safe: false | description: enables WASM plugin runtime (off in prod until Phase D)
	r.register(Flag{Key: "plugins.wasm", DefaultEnabled: false, EnvVar: "COSMI_WASM_PLUGINS_ENABLED", Description: "enables WASM plugin runtime — off in production until Phase D", Risk: BreakingRisk, LLMToggleSafe: false})

	// risk: safe | llm_toggle_safe: true | description: enables config-based plugins (JSON/YAML declarative plugins)
	r.register(Flag{Key: "plugins.config", DefaultEnabled: true, EnvVar: "COSMI_CONFIG_PLUGINS_ENABLED", Description: "enables config-based (declarative) plugins", Risk: SafeRisk, LLMToggleSafe: true})

	return r
}

// register adds a flag definition and initialises its value to the default.
func (r *Registry) register(f Flag) {
	r.flags[f.Key] = f
	r.values[f.Key] = f.DefaultEnabled
}

// Load resolves each flag against the environment using getEnv (typically os.Getenv).
// Recognised values: "true", "1" → true; "false", "0" → false; empty/missing → default.
// Any other value falls back to the flag default and emits a warn log.
// Returns the receiver for chaining: featureflag.NewRegistry().Load(os.Getenv).
func (r *Registry) Load(getEnv func(string) string) *Registry {
	for key, f := range r.flags {
		raw := getEnv(f.EnvVar)
		switch raw {
		case "":
			// missing env var — keep default, no noise
		case "true", "1":
			r.values[key] = true
		case "false", "0":
			r.values[key] = false
		default:
			slog.Warn("feature flag env var has unrecognised value, using default",
				"flag", key,
				"env_var", f.EnvVar,
				"value", raw,
				"default", f.DefaultEnabled,
			)
		}
	}
	return r
}

// IsEnabled reports whether the flag identified by key is active.
// Unknown keys return false and emit a warn log.
func (r *Registry) IsEnabled(key string) bool {
	v, ok := r.values[key]
	if !ok {
		slog.Warn("feature flag lookup for unknown key", "key", key)
		return false
	}
	return v
}

// All returns a snapshot of all resolved flag values keyed by flag key.
// Useful for exposing the full registry via an API endpoint.
func (r *Registry) All() map[string]bool {
	out := make(map[string]bool, len(r.values))
	for k, v := range r.values {
		out[k] = v
	}
	return out
}

// Keys returns all registered flag keys in deterministic (sorted) order.
func (r *Registry) Keys() []string {
	keys := make([]string, 0, len(r.flags))
	for k := range r.flags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
