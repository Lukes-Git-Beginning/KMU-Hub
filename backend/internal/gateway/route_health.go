package gateway

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/server/response"
)

// HealthRoutes handles the health check endpoint for the gateway.
type HealthRoutes struct {
	checkers []health.Checker
	registry *ServiceRegistry
}

// NewHealthRoutes creates a new HealthRoutes.
func NewHealthRoutes(checkers []health.Checker, registry *ServiceRegistry) *HealthRoutes {
	return &HealthRoutes{
		checkers: checkers,
		registry: registry,
	}
}

// ServiceName returns the service name.
func (h *HealthRoutes) ServiceName() string { return "health" }

// RegisterRoutes registers the /health endpoint.
func (h *HealthRoutes) RegisterRoutes(r chi.Router, _ func(http.Handler) http.Handler) {
	r.Get("/health", h.HandleHealth)
}

// HandleHealth runs all health checkers and reports registry status.
func (h *HealthRoutes) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	overall, checks := health.CheckAll(ctx, h.checkers)

	statusCode := http.StatusOK
	if overall == health.StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	// Add registry info: which services are registered
	registeredServices := h.registry.RegisteredServices()

	response.JSON(w, statusCode, map[string]interface{}{
		"status":              string(overall),
		"checks":              checks,
		"registered_services": registeredServices,
	})
}
