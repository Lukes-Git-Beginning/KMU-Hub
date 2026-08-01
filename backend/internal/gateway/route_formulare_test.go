package gateway

import (
	"github.com/kmuhub/kmuhub/internal/featureflag"
)

func newFormulareRoutes(registry *ServiceRegistry) *FormulareRoutes {
	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_FORMULARE_ENABLED" {
			return "true"
		}
		return ""
	})
	return NewFormulareRoutes(registry, flags)
}
