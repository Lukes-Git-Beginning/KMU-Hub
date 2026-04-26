package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kmuhub/kmuhub/internal/health"
	"github.com/kmuhub/kmuhub/internal/server/response"
)

// HealthHandler returns an http.HandlerFunc that runs the given health checkers.
// Used by services that need a standalone health endpoint (e.g. auth, crm, chat services).
// Responds to GET with a JSON body and to HEAD with status code only — Compose
// healthchecks use HEAD via `wget --spider`.
func HealthHandler(checkers []health.Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		overall, checks := health.CheckAll(ctx, checkers)

		statusCode := http.StatusOK
		if overall == health.StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		}

		if r.Method == http.MethodHead {
			w.WriteHeader(statusCode)
			return
		}

		response.JSON(w, statusCode, map[string]any{
			"status": string(overall),
			"checks": checks,
		})
	}
}

// RegisterHealth registers the health endpoint on both GET and HEAD methods.
// Compose healthchecks use HEAD via `wget --spider`; humans and browsers use GET.
// Use this helper instead of registering a single Get(...) so both paths stay consistent.
func RegisterHealth(r chi.Router, path string, checkers []health.Checker) {
	handler := HealthHandler(checkers)
	r.Get(path, handler)
	r.Head(path, handler)
}
