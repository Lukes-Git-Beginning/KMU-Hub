package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/server/response"
)

// FeatureFlagRoutes exposes the feature-flag registry over HTTP.
// GET /api/v1/feature-flags returns all resolved flag values for the authenticated user.
type FeatureFlagRoutes struct {
	registry *featureflag.Registry
}

// NewFeatureFlagRoutes creates a FeatureFlagRoutes backed by the given registry.
func NewFeatureFlagRoutes(registry *featureflag.Registry) *FeatureFlagRoutes {
	return &FeatureFlagRoutes{registry: registry}
}

// ServiceName returns the logical service name used in startup logs.
func (f *FeatureFlagRoutes) ServiceName() string { return "feature-flags" }

// RegisterRoutes mounts GET /api/v1/feature-flags behind auth middleware.
func (f *FeatureFlagRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/feature-flags", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)
		r.Get("/", f.HandleGetFlags)
	})
}

// featureFlagsResponse is the JSON envelope returned by GET /api/v1/feature-flags.
type featureFlagsResponse struct {
	Flags   map[string]bool `json:"flags"`
	Version string          `json:"version"`
}

// HandleGetFlags returns all registered feature flags with their resolved values.
func (f *FeatureFlagRoutes) HandleGetFlags(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, featureFlagsResponse{
		Flags:   f.registry.All(),
		Version: "v1",
	})
}
