package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// DialerRoutes handles HTTP routes for the Dialer backend service.
type DialerRoutes struct {
	registry *ServiceRegistry
}

// NewDialerRoutes creates a new DialerRoutes with the given service registry.
func NewDialerRoutes(registry *ServiceRegistry) *DialerRoutes {
	return &DialerRoutes{registry: registry}
}

// ServiceName returns the backend service name.
func (dr *DialerRoutes) ServiceName() string { return "dialer" }

// RegisterRoutes registers all Dialer HTTP routes.
// TODO(Phase 1C): Add call management, contact dialing, and call log routes here.
func (dr *DialerRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Routes will be added in Sub-Phase 1C once the dialerv1 proto package is available.
}
