package action

import (
	"fmt"
	"strings"
)

// resolveTemplate performs simple {{key}} replacement in a template string
// using values from the environment map. Supports dot notation for nested keys.
//
// Example: resolveTemplate("Hello {{deal.name}}", env) where env contains
// {"deal.name": "Acme Deal"} returns "Hello Acme Deal".
func resolveTemplate(template string, env map[string]any) string {
	if !strings.Contains(template, "{{") {
		return template
	}

	// Flatten the environment map with dot notation for replacement
	flat := flattenEnv(env)

	// Build replacer pairs
	pairs := make([]string, 0, len(flat)*2)
	for key, val := range flat {
		pairs = append(pairs, "{{"+key+"}}", fmt.Sprintf("%v", val))
	}

	replacer := strings.NewReplacer(pairs...)
	return replacer.Replace(template)
}

// flattenEnv flattens a nested map into dot-notation keys.
// {"deal": {"name": "Acme"}} becomes {"deal.name": "Acme"}.
func flattenEnv(env map[string]any) map[string]any {
	flat := make(map[string]any)
	flattenRecursive("", env, flat)
	return flat
}

func flattenRecursive(prefix string, m map[string]any, flat map[string]any) {
	for key, val := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := val.(type) {
		case map[string]any:
			flattenRecursive(fullKey, v, flat)
		default:
			flat[fullKey] = val
		}
	}
}
