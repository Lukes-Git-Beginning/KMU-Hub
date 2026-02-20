package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
)

// CalDAVPasswordService abstracts app-specific password operations needed by the route handler.
// This interface breaks the import cycle between gateway and caldav packages.
type CalDAVPasswordService interface {
	Validate(ctx context.Context, username, password string) (uuid.UUID, error)
	Create(ctx context.Context, userID uuid.UUID, label string) (string, *CalDAVPasswordInfo, error)
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
	ctxInjector    CalDAVCtxInjector
	pool           *pgxpool.Pool
	authMiddleware func(http.Handler) http.Handler
}

// NewCalDAVRoutes creates new CalDAV/CardDAV route handler.
func NewCalDAVRoutes(
	caldavHandler http.Handler,
	carddavHandler http.Handler,
	pwService CalDAVPasswordService,
	ctxInjector CalDAVCtxInjector,
	pool *pgxpool.Pool,
	authMiddleware func(http.Handler) http.Handler,
) *CalDAVRoutes {
	return &CalDAVRoutes{
		caldavHandler:  caldavHandler,
		carddavHandler: carddavHandler,
		pwService:      pwService,
		ctxInjector:    ctxInjector,
		pool:           pool,
		authMiddleware: authMiddleware,
	}
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
		sub.Use(c.authMiddleware)

		sub.Get("/passwords", c.handleListPasswords)
		sub.Post("/passwords", c.handleCreatePassword)
		sub.Delete("/passwords/{id}", c.handleRevokePassword)
		sub.Get("/status", c.handleGetStatus)
		sub.Put("/enable", c.handleEnableCalDAV)
		sub.Put("/disable", c.handleDisableCalDAV)
	})

	// =========================================================================
	// Admin CalDAV API (JWT auth + admin role)
	// =========================================================================
	r.Route("/api/v1/admin/caldav", func(sub chi.Router) {
		sub.Use(c.authMiddleware)
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
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			userID, err := c.pwService.Validate(r.Context(), username, password)
			if err != nil {
				slog.Warn("caldav basic auth failed",
					"username", username,
					"error", err,
				)
				w.Header().Set("WWW-Authenticate", `Basic realm="KMU Hub CalDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := c.ctxInjector(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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

	var req struct {
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Label == "" {
		response.Error(w, http.StatusBadRequest, "label is required")
		return
	}

	plaintext, pw, err := c.pwService.Create(r.Context(), userID, req.Label)
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
	var userCaldavEnabled bool
	scanErr := c.pool.QueryRow(r.Context(),
		`SELECT COALESCE(caldav_enabled, false) FROM users WHERE id = $1`,
		userID,
	).Scan(&userCaldavEnabled)
	if scanErr != nil {
		slog.Warn("failed to check user caldav status", "user_id", userID, "error", scanErr)
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

	_, execErr := c.pool.Exec(r.Context(),
		`UPDATE users SET caldav_enabled = true WHERE id = $1`,
		userID,
	)
	if execErr != nil {
		slog.Error("failed to enable caldav for user", "user_id", userID, "error", execErr)
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

	_, execErr := c.pool.Exec(r.Context(),
		`UPDATE users SET caldav_enabled = false WHERE id = $1`,
		userID,
	)
	if execErr != nil {
		slog.Error("failed to disable caldav for user", "user_id", userID, "error", execErr)
		response.Error(w, http.StatusInternalServerError, "failed to disable CalDAV")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status": "disabled",
	})
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
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
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
	rows, err := c.pool.Query(r.Context(),
		`SELECT u.id, u.email, u.first_name, u.last_name, u.caldav_enabled,
		        (SELECT COUNT(*) FROM app_specific_passwords asp
		         WHERE asp.user_id = u.id AND asp.revoked_at IS NULL) AS password_count,
		        (SELECT MAX(asp.last_used_at) FROM app_specific_passwords asp
		         WHERE asp.user_id = u.id) AS last_used
		 FROM users u
		 WHERE u.caldav_enabled = true
		 ORDER BY u.email`,
	)
	if err != nil {
		slog.Error("failed to list caldav users", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	defer rows.Close()

	type caldavUser struct {
		ID            string  `json:"id"`
		Email         string  `json:"email"`
		FirstName     string  `json:"first_name"`
		LastName      string  `json:"last_name"`
		CalDAVEnabled bool    `json:"caldav_enabled"`
		PasswordCount int     `json:"password_count"`
		LastUsed      *string `json:"last_used"`
	}

	var users []caldavUser
	for rows.Next() {
		var u caldavUser
		if err := rows.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName,
			&u.CalDAVEnabled, &u.PasswordCount, &u.LastUsed); err != nil {
			slog.Error("failed to scan caldav user", "error", err)
			continue
		}
		users = append(users, u)
	}
	if users == nil {
		users = []caldavUser{}
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

	tag, execErr := c.pool.Exec(r.Context(),
		`UPDATE app_specific_passwords
		 SET revoked_at = NOW()
		 WHERE user_id = $1 AND revoked_at IS NULL`,
		targetUserID,
	)
	if execErr != nil {
		slog.Error("failed to revoke all passwords for user",
			"target_user_id", targetUserID, "error", execErr)
		response.Error(w, http.StatusInternalServerError, "failed to revoke passwords")
		return
	}

	slog.Info("admin revoked all CalDAV passwords for user",
		"target_user_id", targetUserID,
		"revoked_count", tag.RowsAffected(),
	)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"revoked_count": tag.RowsAffected(),
	})
}
