package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/server/response"
	"github.com/kmuhub/kmuhub/internal/sysctx"
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

			if !reserveOrReplay(w, r, repo, idempotencyKey, tenantID, userID, requestHash) {
				return
			}

			// Fresh key (or fail-open): capture the response so we can store it.
			crw := newCapturingResponseWriter(w)
			next.ServeHTTP(crw, r)
			completeAsync(r.Context(), repo, tenantID, idempotencyKey, crw)
		})
	}
}

// reserveOrReplay reserves the key for this request. It returns true when the
// caller should run the handler (fresh key, or Reserve failed and the request
// fails open) and false when a response was already written: replay of a
// completed request, 409 for an in-flight key, 422 for a reused key.
func reserveOrReplay(w http.ResponseWriter, r *http.Request, repo idempotency.Repository, key string, tenantID, userID uuid.UUID, requestHash string) bool {
	rec, err := repo.Reserve(r.Context(), key, tenantID, userID, r.Method, r.URL.Path, requestHash)
	if err != nil {
		switch {
		case isInFlight(err):
			w.Header().Set("Retry-After", "2")
			response.Error(w, http.StatusConflict, "request with this Idempotency-Key is already in flight")
			return false
		case isConflict(err):
			response.Error(w, http.StatusUnprocessableEntity, "Idempotency-Key reused with different request payload")
			return false
		default:
			slog.Error("idempotency reserve failed", "key", key, "error", err)
			// Fail open: proceed without deduplication rather than blocking all requests.
			return true
		}
	}

	if rec != nil {
		// Replay: serve cached response.
		slog.Debug("idempotency replay",
			"key", key,
			"status", *rec.ResponseStatus,
		)
		w.Header().Set("Idempotency-Replayed", "true")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(*rec.ResponseStatus)
		if len(rec.ResponseBody) > 0 {
			_, _ = w.Write(rec.ResponseBody)
		}
		return false
	}
	return true
}

// completeAsync stores the captured response asynchronously to avoid blocking
// the client. context.WithoutCancel ensures the goroutine isn't cancelled when
// the request context is done (which happens as soon as ServeHTTP returns).
// tenantID is passed explicitly so the UPDATE is scoped to the correct tenant
// (composite PK: tenant_id, key).
func completeAsync(ctx context.Context, repo idempotency.Repository, tenantID uuid.UUID, key string, crw *capturingResponseWriter) {
	completeCtx := context.WithoutCancel(ctx)
	go func() {
		if err := repo.Complete(completeCtx, tenantID, key, crw.statusCode, crw.body.Bytes()); err != nil {
			if !errors.Is(err, idempotency.ErrKeyMissing) {
				slog.Error("idempotency complete failed", "key", key, "error", err)
			}
		}
	}()
}

// BeginIdempotent applies the optional Idempotency-Key header inside a handler
// that cannot sit behind Idempotency because that middleware buffers the whole
// request body to hash it (multipart uploads). fingerprint stands in for the
// body hash and must identify the logical request without reading the body.
//
// ok=false means a response was already written (replay, in-flight, conflict,
// missing auth context) and the handler must return. Otherwise the handler
// writes to the returned writer and calls done() once it has responded.
// Without the header, or with a nil repo, it is a no-op: no deduplication.
func BeginIdempotent(repo idempotency.Repository, w http.ResponseWriter, r *http.Request, fingerprint string) (rw http.ResponseWriter, done func(), ok bool) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" || repo == nil {
		return w, func() {}, true
	}
	tenantID, err := GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return w, nil, false
	}
	userID, err := uuid.Parse(GetUserID(r.Context()))
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing user context")
		return w, nil, false
	}
	hash := computeRequestHash(tenantID.String(), userID.String(), r.Method, r.URL.Path, []byte(fingerprint))
	if !reserveOrReplay(w, r, repo, key, tenantID, userID, hash) {
		return w, nil, false
	}
	crw := newCapturingResponseWriter(w)
	return crw, func() { completeAsync(r.Context(), repo, tenantID, key, crw) }, true
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
	return errors.Is(err, idempotency.ErrInFlight)
}

func isConflict(err error) bool {
	return errors.Is(err, idempotency.ErrConflict)
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

// idempotencyCleanupLockKey is a stable int64 used as the pg_try_advisory_lock key
// so only one replica runs the cleanup per hour. Value is arbitrary but must be constant.
//
// Session-level, NOT pg_try_advisory_xact_lock -- this comment claimed the latter
// until 2026-08-22 and the difference matters: a transaction lock releases itself at
// COMMIT, a session lock belongs to its connection and has to be released on that
// same connection. CleanupWithLock did not, and the lock leaked; see the comment
// there.
const idempotencyCleanupLockKey int64 = 0x49444D50 // "IDMP" in ASCII

// cleanupCtx returns the ctx the cleanup worker uses for repo calls. Wrapped
// with sysctx.With so the post-RLS DELETE on idempotency_keys is admitted —
// the worker has no per-request tenant.
func cleanupCtx(ctx context.Context) context.Context {
	return sysctx.With(ctx)
}

// IdempotencyCleanupWorker runs a periodic cleanup of expired idempotency keys.
// pg_try_advisory_lock ensures leader election: only one replica deletes per tick.
func IdempotencyCleanupWorker(ctx context.Context, repo idempotency.Repository) {
	ctx = cleanupCtx(ctx)

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	slog.Info("idempotency cleanup worker starting", "interval", "1h", "lock_key", idempotencyCleanupLockKey)

	for {
		select {
		case <-ctx.Done():
			slog.Info("idempotency cleanup worker stopped")
			return
		case <-ticker.C:
			count, err := repo.CleanupWithLock(ctx, idempotencyCleanupLockKey)
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
