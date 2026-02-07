package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RouteRegistrar is the interface that per-service route modules must implement.
// Each backend service (auth, CRM, chat, etc.) provides its own RouteRegistrar
// that registers HTTP routes on the gateway router.
type RouteRegistrar interface {
	// RegisterRoutes adds all HTTP routes for this service to the router.
	// The authMiddleware parameter provides JWT authentication for protected routes.
	RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler)

	// ServiceName returns the name of the backend service this registrar handles.
	// This must match the name used when registering the service in ServiceRegistry.
	ServiceName() string
}
