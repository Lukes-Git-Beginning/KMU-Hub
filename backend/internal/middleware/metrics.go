package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/metrics"
)

// Metrics returns middleware that records HTTP request metrics.
func Metrics(reg *metrics.Registry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			reg.RequestsInFlight.WithLabelValues(r.Method).Inc()
			defer reg.RequestsInFlight.WithLabelValues(r.Method).Dec()

			wrapped := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			duration := time.Since(start).Seconds()
			path := routePattern(r)

			reg.RequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(wrapped.statusCode)).Inc()
			reg.RequestDuration.WithLabelValues(r.Method, path).Observe(duration)
		})
	}
}

// routePattern returns the chi route pattern to avoid label cardinality explosion.
func routePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return "unknown"
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}
