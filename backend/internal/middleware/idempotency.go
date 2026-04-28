package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/server/response"
)

// IdempotencyMode controls whether missing Idempotency-Key headers are blocked
// or merely warned.
type IdempotencyMode int

const (
	// WarnMode logs a warning but does not block the request.
	// Use during rollout until the frontend always sends the header.
	WarnMode IdempotencyMode = iota
	// HardMode rejects mutations without an Idempotency-Key with 400.
	HardMode
)

// idempotencyWhitelist contains paths for which idempotency is intentionally
// skipped (token rotation, login — these are not idempotent by design).
var idempotencyWhitelist = []string{
	"/auth/login",
	"/auth/refresh",
	"/auth/2fa",
}

// mutationMethods are the HTTP methods that mutate state.
var mutationMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// Idempotency returns a middleware that enforces per-request deduplication
// for mutation endpoints using the Idempotency-Key request header.
//
// Behaviour:
//   - GET / HEAD / OPTIONS are skipped entirely.
//   - Whitelisted auth paths are skipped.
//   - Missing Idempotency-Key: WarnMode logs + passes through; HardMode → 400.
//   - Replay (completed, same hash) → cached response returned, handler skipped.
//   - In-flight (reserved, not completed) → 409 + Retry-After: 2.
//   - Conflict (different request body for same key) → 422.
//   - Fresh key → reserve, run handler, complete.
func Idempotency(repo idempotency.Repository, mode IdempotencyMode) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip non-mutating methods.
			if !mutationMethods[r.Method] {
				next.ServeHTTP(w, r)
				return
			}

			// Skip whitelisted paths.
			for _, prefix := range idempotencyWhitelist {
				if strings.Contains(r.URL.Path, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}

			idempotencyKey := r.Header.Get("Idempotency-Key")
			if idempotencyKey == "" {
				if mode == HardMode {
					response.Error(w, http.StatusBadRequest, "Idempotency-Key header required for mutation requests")
					return
				}
				// WarnMode: log and continue without deduplication.
				slog.Warn("idempotency_key_missing",
					"method", r.Method,
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr,
				)
				next.ServeHTTP(w, r)
				return
			}

			// Extract tenant + user from JWT context (set by Auth middleware).
			tenantID, err := GetTenantID(r.Context())
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "missing tenant context")
				return
			}

			userIDStr := GetUserID(r.Context())
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "missing user context")
				return
			}

			// Read body for hashing (re-set body so handler can read it again).
			var bodyBytes []byte
			if r.Body != nil {
				bodyBytes, err = io.ReadAll(r.Body)
				if err != nil {
					response.Error(w, http.StatusBadRequest, "failed to read request body")
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}

			requestHash := computeRequestHash(tenantID.String(), userIDStr, r.Method, r.URL.Path, bodyBytes)

			rec, err := repo.Reserve(r.Context(), idempotencyKey, tenantID, userID, r.Method, r.URL.Path, requestHash)
			if err != nil {
				switch {
				case isInFlight(err):
					w.Header().Set("Retry-After", "2")
					response.Error(w, http.StatusConflict, "request with this Idempotency-Key is already in flight")
					return
				case isConflict(err):
					response.Error(w, http.StatusUnprocessableEntity, "Idempotency-Key reused with different request payload")
					return
				default:
					slog.Error("idempotency reserve failed", "key", idempotencyKey, "error", err)
					// Fail open: proceed without deduplication rather than blocking all requests.
					next.ServeHTTP(w, r)
					return
				}
			}

			if rec != nil {
				// Replay: serve cached response.
				slog.Debug("idempotency replay",
					"key", idempotencyKey,
					"status", *rec.ResponseStatus,
				)
				w.Header().Set("Idempotency-Replayed", "true")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(*rec.ResponseStatus)
				if len(rec.ResponseBody) > 0 {
					_, _ = w.Write(rec.ResponseBody)
				}
				return
			}

			// Fresh key: capture the response so we can store it.
			crw := newCapturingResponseWriter(w)
			next.ServeHTTP(crw, r)

			// Store the response asynchronously to avoid blocking the client.
			go func() {
				ctx := r.Context()
				if err := repo.Complete(ctx, idempotencyKey, crw.statusCode, crw.body.Bytes()); err != nil {
					slog.Error("idempotency complete failed", "key", idempotencyKey, "error", err)
				}
			}()
		})
	}
}

// computeRequestHash returns a SHA-256 hex digest of the concatenated
// tenant|user|method|path|body to detect payload changes for the same key.
func computeRequestHash(tenantID, userID, method, path string, body []byte) string {
	h := sha256.New()
	_, _ = io.WriteString(h, tenantID)
	_, _ = io.WriteString(h, "|")
	_, _ = io.WriteString(h, userID)
	_, _ = io.WriteString(h, "|")
	_, _ = io.WriteString(h, method)
	_, _ = io.WriteString(h, "|")
	_, _ = io.WriteString(h, path)
	_, _ = io.WriteString(h, "|")
	_, _ = h.Write(body)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func isInFlight(err error) bool {
	return err != nil && err.Error() == idempotency.ErrInFlight.Error()
}

func isConflict(err error) bool {
	return err != nil && err.Error() == idempotency.ErrConflict.Error()
}

// capturingResponseWriter buffers the response so it can be stored.
type capturingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func newCapturingResponseWriter(w http.ResponseWriter) *capturingResponseWriter {
	return &capturingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (c *capturingResponseWriter) WriteHeader(code int) {
	c.statusCode = code
	c.ResponseWriter.WriteHeader(code)
}

func (c *capturingResponseWriter) Write(b []byte) (int, error) {
	c.body.Write(b)
	return c.ResponseWriter.Write(b)
}

// IdempotencyCleanupWorker runs a periodic cleanup of expired idempotency keys.
// It uses pg_try_advisory_xact_lock for leader-election (one replica runs per tick).
func IdempotencyCleanupWorker(ctx context.Context, repo idempotency.Repository) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	slog.Info("idempotency cleanup worker starting", "interval", "1h")

	for {
		select {
		case <-ctx.Done():
			slog.Info("idempotency cleanup worker stopped")
			return
		case <-ticker.C:
			count, err := repo.Cleanup(ctx)
			if err != nil {
				slog.Error("idempotency cleanup failed", "error", err)
				continue
			}
			if count > 0 {
				slog.Info("idempotency keys cleaned up", "deleted", count)
			}
		}
	}
}

// jsonResponse is a helper used in tests — kept unexported.
var _ = json.Marshal
