package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
)

// caldavTestTimeout bounds each protocol probe in handleTestConnection so a
// hung upstream connection can't stall the settings page indefinitely.
const caldavTestTimeout = 5 * time.Second

// CalDAVUserInfo holds user metadata for admin CalDAV management.
type CalDAVUserInfo struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	CalDAVEnabled bool       `json:"caldav_enabled"`
	PasswordCount int        `json:"password_count"`
	LastUsed      *time.Time `json:"last_used"`
}

// CalDAVUserPreferenceService abstracts user-level CalDAV preference operations needed by the
// route handler. This interface breaks the import cycle between gateway and caldav packages.
type CalDAVUserPreferenceService interface {
	GetCalDAVEnabled(ctx context.Context, userID uuid.UUID) (bool, error)
	SetCalDAVEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error
	ListCalDAVUsers(ctx context.Context) ([]CalDAVUserInfo, error)
	RevokeAllUserPasswords(ctx context.Context, userID uuid.UUID) (int64, error)
}

// CalDAVPasswordService abstracts app-specific password operations needed by the route handler.
// This interface breaks the import cycle between gateway and caldav packages.
type CalDAVPasswordService interface {
	Validate(ctx context.Context, username, password string) (uuid.UUID, error)
	Create(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID, label string) (string, *CalDAVPasswordInfo, error)
	List(ctx context.Context, userID uuid.UUID) ([]*CalDAVPasswordInfo, error)
	Revoke(ctx context.Context, id, userID uuid.UUID) error
	IsOrgEnabled(ctx context.Context) (bool, error)
	SetOrgEnabled(ctx context.Context, enabled bool) error
}

// CalDAVPasswordInfo holds app-specific password metadata for the route handler.
type CalDAVPasswordInfo struct {
	ID             uuid.UUID  `json:"id"`
	Label          string     `json:"label"`
	PasswordPrefix string     `json:"password_prefix"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}

// CalDAVCtxInjector injects a user ID into a context for CalDAV authentication.
type CalDAVCtxInjector func(ctx context.Context, userID uuid.UUID) context.Context

// CalDAVRoutes handles CalDAV/CardDAV protocol routes (Basic Auth) and
// REST API routes for app-specific password management (JWT auth).
// Similar to WOPIRoutes, these are NOT part of the RouteRegistrar loop
// because CalDAV/CardDAV use HTTP Basic Auth, not JWT.
type CalDAVRoutes struct {
	caldavHandler  http.Handler
	carddavHandler http.Handler
	pwService      CalDAVPasswordService
	userPrefRepo   CalDAVUserPreferenceService
	ctxInjector    CalDAVCtxInjector
	authMiddleware func(http.Handler) http.Handler
	idempotencyMW  func(http.Handler) http.Handler
	selfBaseURL    string
}

// NewCalDAVRoutes creates new CalDAV/CardDAV route handler.
//
// selfBaseURL is this gateway's own loopback address (e.g. "http://127.0.0.1:8080"),
// used by handleTestConnection to make a real request against the CalDAV/CardDAV
// endpoints below. It is deliberately never derived from request headers (Host,
// X-Forwarded-Host, ...) -- doing so would let any authenticated caller turn the
// test endpoint into an SSRF probe against an arbitrary target.
//
// authMiddleware and idempotencyMW are taken separately, not pre-composed the
// way cmd/gateway/main.go hands authWithIdempotency to every other route
// registrar: this file registers three different auth flavours on one router
// (Basic-Auth protocol routes, JWT REST routes, JWT+admin routes) and only the
// mutating JWT routes may carry idempotency. Passing the composed chain is what
// made the REST block silently miss it -- setupCalDAV only ever got the bare
// authMiddleware, so POST /api/v1/caldav/passwords could be double-submitted
// into two valid app passwords of which the caller sees only one.
func NewCalDAVRoutes(
	caldavHandler http.Handler,
	carddavHandler http.Handler,
	pwService CalDAVPasswordService,
	userPrefRepo CalDAVUserPreferenceService,
	ctxInjector CalDAVCtxInjector,
	authMiddleware func(http.Handler) http.Handler,
	idempotencyMW func(http.Handler) http.Handler,
	selfBaseURL string,
) *CalDAVRoutes {
	return &CalDAVRoutes{
		caldavHandler:  caldavHandler,
		carddavHandler: carddavHandler,
		pwService:      pwService,
		userPrefRepo:   userPrefRepo,
		ctxInjector:    ctxInjector,
		authMiddleware: authMiddleware,
		idempotencyMW:  idempotencyMW,
		selfBaseURL:    selfBaseURL,
	}
}

// useAPIChain applies the middleware chain every other /api/v1 route gets from
// cmd/gateway/main.go's authWithIdempotency: authMiddleware runs first because
// it stamps the tenant/user context that middleware.Idempotency reads, then
// idempotencyMW. GET routes pass straight through -- the middleware skips
// non-mutating methods itself (middleware/idempotency.go:65-69).
func (c *CalDAVRoutes) useAPIChain(sub chi.Router) {
	sub.Use(c.authMiddleware)
	sub.Use(c.idempotencyMW)
}

// RegisterRoutes registers CalDAV/CardDAV protocol routes, .well-known discovery,
// and REST API endpoints on the router.
func (c *CalDAVRoutes) RegisterRoutes(r chi.Router) {
	// =========================================================================
	// .well-known auto-discovery (no auth)
	// =========================================================================
	r.Get("/.well-known/caldav", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/caldav/", http.StatusMovedPermanently)
	})
	r.Get("/.well-known/carddav", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/carddav/", http.StatusMovedPermanently)
	})

	// =========================================================================
	// CalDAV protocol routes (HTTP Basic Auth via app-specific passwords)
	// =========================================================================
	r.Route("/caldav", func(sub chi.Router) {
		sub.Use(c.basicAuthMiddleware())
		sub.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
			c.caldavHandler.ServeHTTP(w, r)
		})
	})

	// =========================================================================
	// CardDAV protocol routes (HTTP Basic Auth via app-specific passwords)
	// =========================================================================
	r.Route("/carddav", func(sub chi.Router) {
		sub.Use(c.basicAuthMiddleware())
		sub.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
			c.carddavHandler.ServeHTTP(w, r)
		})
	})

	// =========================================================================
	// App-specific password REST API (JWT auth, under /api/v1/caldav/)
	// =========================================================================
	r.Route("/api/v1/caldav", func(sub chi.Router) {
		c.useAPIChain(sub)

		sub.Get("/passwords", c.handleListPasswords)
		sub.Post("/passwords", c.handleCreatePassword)
		sub.Delete("/passwords/{id}", c.handleRevokePassword)
		sub.Get("/status", c.handleGetStatus)
		sub.Put("/enable", c.handleEnableCalDAV)
		sub.Put("/disable", c.handleDisableCalDAV)
		sub.Post("/test", c.handleTestConnection)
	})

	// =========================================================================
	// Admin CalDAV API (JWT auth + admin role)
	// =========================================================================
	r.Route("/api/v1/admin/caldav", func(sub chi.Router) {
		c.useAPIChain(sub)
		sub.Use(middleware.RequireRole("admin"))

		sub.Get("/settings", c.handleAdminGetSettings)
		sub.Put("/settings", c.handleAdminSetSettings)
		sub.Get("/users", c.handleAdminListUsers)
		sub.Delete("/users/{userId}/passwords", c.handleAdminRevokeUserPasswords)
	})
}

// basicAuthMiddleware validates HTTP Basic Auth credentials against app-specific passwords.
func (c *CalDAVRoutes) basicAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="KMU Hub CalDAV"`)
				response.Error(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			userID, err := c.pwService.Validate(r.Context(), username, password)
			if err != nil {
				slog.Warn("caldav basic auth failed",
					"username_fingerprint", fingerprintCaldavUsername(username),
					"error", err,
				)
				w.Header().Set("WWW-Authenticate", `Basic realm="KMU Hub CalDAV"`)
				response.Error(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			ctx := c.ctxInjector(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// fingerprintCaldavUsername returns a short, non-reversible fingerprint of a
// Basic-Auth username for logging. The route is reachable without prior
// authentication, and CalDAV clients conventionally send an email address
// here (the expected value is a user UUID) -- logging the raw value would
// regularly leak plaintext email addresses into the application log. The
// fingerprint still lets repeated invalid values be spotted for anomaly/
// rate-limit analysis.
func fingerprintCaldavUsername(username string) string {
	sum := sha256.Sum256([]byte(username))
	return hex.EncodeToString(sum[:4])
}

// =========================================================================
// App-specific password handlers (user-facing, JWT auth)
// =========================================================================

// handleListPasswords lists the current user's app-specific passwords.
func (c *CalDAVRoutes) handleListPasswords(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(middleware.GetUserID(r.Context()))
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	passwords, err := c.pwService.List(r.Context(), userID)
	if err != nil {
		slog.Error("failed to list app passwords", "user_id", userID, "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to list passwords")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"passwords": passwords,
	})
}

// handleCreatePassword creates a new app-specific password.
func (c *CalDAVRoutes) handleCreatePassword(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(middleware.GetUserID(r.Context()))
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid tenant")
		return
	}

	req, ok := decodeAndValidate[struct {
		Label string `json:"label" validate:"required"`
	}](w, r)
	if !ok {
		return
	}

	plaintext, pw, err := c.pwService.Create(r.Context(), userID, tenantID, req.Label)
	if err != nil {
		slog.Error("failed to create app password", "user_id", userID, "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to create password")
		return
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"password": plaintext,
		"id":       pw.ID,
		"label":    pw.Label,
		"prefix":   pw.PasswordPrefix,
	})
}

// handleRevokePassword revokes an app-specific password.
func (c *CalDAVRoutes) handleRevokePassword(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(middleware.GetUserID(r.Context()))
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	pwID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid password ID")
		return
	}

	if err := c.pwService.Revoke(r.Context(), pwID, userID); err != nil {
		slog.Error("failed to revoke app password", "password_id", pwID, "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to revoke password")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status": "revoked",
	})
}

// handleGetStatus returns CalDAV/CardDAV status for the current user.
func (c *CalDAVRoutes) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(middleware.GetUserID(r.Context()))
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	orgEnabled, err := c.pwService.IsOrgEnabled(r.Context())
	if err != nil {
		slog.Error("failed to check org caldav status", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to get status")
		return
	}

	// Check user-level caldav_enabled
	userCaldavEnabled, prefErr := c.userPrefRepo.GetCalDAVEnabled(r.Context(), userID)
	if prefErr != nil {
		slog.Warn("failed to check user caldav status", "user_id", userID, "error", prefErr)
	}

	// Count active passwords
	passwords, _ := c.pwService.List(r.Context(), userID)
	activeCount := 0
	for _, pw := range passwords {
		if pw.RevokedAt == nil {
			activeCount++
		}
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"org_enabled":    orgEnabled,
		"user_enabled":   userCaldavEnabled,
		"password_count": activeCount,
		"user_id":        userID.String(),
		"caldav_url":     "/caldav/",
		"carddav_url":    "/carddav/",
	})
}

// handleEnableCalDAV enables CalDAV for the current user.
func (c *CalDAVRoutes) handleEnableCalDAV(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(middleware.GetUserID(r.Context()))
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	if err := c.userPrefRepo.SetCalDAVEnabled(r.Context(), userID, true); err != nil {
		slog.Error("failed to enable caldav for user", "user_id", userID, "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to enable CalDAV")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status": "enabled",
	})
}

// handleDisableCalDAV disables CalDAV for the current user.
func (c *CalDAVRoutes) handleDisableCalDAV(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(middleware.GetUserID(r.Context()))
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	if err := c.userPrefRepo.SetCalDAVEnabled(r.Context(), userID, false); err != nil {
		slog.Error("failed to disable caldav for user", "user_id", userID, "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to disable CalDAV")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status": "disabled",
	})
}

// caldavTestResult is the JSON response of handleTestConnection.
type caldavTestResult struct {
	Success          bool   `json:"success"`
	Message          string `json:"message,omitempty"`
	CalDAVReachable  bool   `json:"caldav_reachable"`
	CardDAVReachable bool   `json:"carddav_reachable"`
}

// handleTestConnection handles POST /api/v1/caldav/test. Unlike a "is a value
// configured" check, this makes two real, timeout-bounded, authenticated HTTP
// requests against this server's own /caldav/ and /carddav/ endpoints -- the
// same request path a real CalDAV/CardDAV client (Apple Calendar, Thunderbird,
// ...) would use -- through a freshly minted, immediately-revoked app-specific
// password. A green result therefore proves the full protocol path (Basic
// Auth validation + WebDAV handler) actually works, not just that the
// enabled flags are set in the database.
func (c *CalDAVRoutes) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(middleware.GetUserID(r.Context()))
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid tenant")
		return
	}

	plaintext, pw, err := c.pwService.Create(r.Context(), userID, tenantID, "connection-test")
	if err != nil {
		slog.Error("caldav test: failed to create ephemeral password", "user_id", userID, "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to prepare connection test")
		return
	}
	defer func() {
		// Use a background context: the request context may already be
		// cancelled (client navigated away) by the time the probes return,
		// but the ephemeral password must still be revoked either way.
		if revokeErr := c.pwService.Revoke(context.Background(), pw.ID, userID); revokeErr != nil {
			slog.Warn("caldav test: failed to revoke ephemeral password",
				"password_id", pw.ID, "user_id", userID, "error", revokeErr)
		}
	}()

	caldavOK, caldavMsg := c.probeSelf(r.Context(), "/caldav/", userID.String(), plaintext)
	carddavOK, carddavMsg := c.probeSelf(r.Context(), "/carddav/", userID.String(), plaintext)

	var problems []string
	if !caldavOK {
		problems = append(problems, "CalDAV: "+caldavMsg)
	}
	if !carddavOK {
		problems = append(problems, "CardDAV: "+carddavMsg)
	}

	response.JSON(w, http.StatusOK, caldavTestResult{
		Success:          caldavOK && carddavOK,
		Message:          strings.Join(problems, "; "),
		CalDAVReachable:  caldavOK,
		CardDAVReachable: carddavOK,
	})
}

// probeSelf makes one real HTTP OPTIONS request against this server's own
// DAV endpoint at path (with HTTP Basic Auth) and classifies the outcome so
// the caller can tell a network failure from an auth failure from an
// unexpected protocol response -- never a plain "field is not empty" check.
// The target is always c.selfBaseURL (loopback), never derived from the
// incoming request, to rule out SSRF via a spoofed Host header.
func (c *CalDAVRoutes) probeSelf(ctx context.Context, path, username, password string) (ok bool, message string) {
	ctx, cancel := context.WithTimeout(ctx, caldavTestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodOptions, c.selfBaseURL+path, nil)
	if err != nil {
		return false, "invalid request"
	}
	req.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return false, "timeout"
		}
		return false, "network unreachable"
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return false, "authentication failed"
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return true, "reachable"
	default:
		return false, fmt.Sprintf("unexpected status %d", resp.StatusCode)
	}
}

// =========================================================================
// Admin CalDAV handlers (admin role required)
// =========================================================================

// handleAdminGetSettings returns org-wide CalDAV/CardDAV enabled status.
func (c *CalDAVRoutes) handleAdminGetSettings(w http.ResponseWriter, r *http.Request) {
	enabled, err := c.pwService.IsOrgEnabled(r.Context())
	if err != nil {
		slog.Error("failed to get admin caldav settings", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to get settings")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"enabled": enabled,
	})
}

// handleAdminSetSettings sets org-wide CalDAV/CardDAV enabled/disabled.
func (c *CalDAVRoutes) handleAdminSetSettings(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeAndValidate[struct {
		Enabled bool `json:"enabled"`
	}](w, r)
	if !ok {
		return
	}

	if err := c.pwService.SetOrgEnabled(r.Context(), req.Enabled); err != nil {
		slog.Error("failed to set admin caldav settings", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to update settings")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"enabled": req.Enabled,
	})
}

// handleAdminListUsers lists users with CalDAV enabled (for audit).
func (c *CalDAVRoutes) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := c.userPrefRepo.ListCalDAVUsers(r.Context())
	if err != nil {
		slog.Error("failed to list caldav users", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
	})
}

// handleAdminRevokeUserPasswords revokes all passwords for a specific user.
func (c *CalDAVRoutes) handleAdminRevokeUserPasswords(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	revokedCount, err := c.userPrefRepo.RevokeAllUserPasswords(r.Context(), targetUserID)
	if err != nil {
		slog.Error("failed to revoke all passwords for user",
			"target_user_id", targetUserID, "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to revoke passwords")
		return
	}

	slog.Info("admin revoked all CalDAV passwords for user",
		"target_user_id", targetUserID,
		"revoked_count", revokedCount,
	)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"revoked_count": revokedCount,
	})
}
