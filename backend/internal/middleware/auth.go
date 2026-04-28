package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/server/response"
)

type userContextKey string

const (
	UserIDKey    userContextKey = "user_id"
	UserRolesKey userContextKey = "user_roles"
	UserPermsKey userContextKey = "user_permissions"
	TenantIDKey  userContextKey = "tenant_id"
)

// ErrMissingTenantID is returned by GetTenantID when the JWT has no tid claim or the claim is empty.
// Handlers must treat this as 401 Unauthorized — no default substitution is permitted.
var ErrMissingTenantID = errors.New("missing or invalid tenant_id in token")

func Auth(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				response.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				response.Error(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}

			claims, err := authService.ValidateToken(r.Context(), parts[1])
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserRolesKey, claims.Roles)
			ctx = context.WithValue(ctx, UserPermsKey, claims.Permissions)
			// TenantID is stored as-is (may be empty for legacy tokens without tid claim).
			// GetTenantID enforces non-empty + valid UUID — no placeholder substitution here.
			ctx = context.WithValue(ctx, TenantIDKey, claims.TenantID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(UserIDKey).(string); ok {
		return id
	}
	return ""
}

// GetTenantID extracts and validates the tenant UUID from the request context.
// Returns ErrMissingTenantID if the tid claim is absent or not a valid UUID.
// Callers must respond with 401 on error — never substitute a default tenant.
func GetTenantID(ctx context.Context) (uuid.UUID, error) {
	raw, ok := ctx.Value(TenantIDKey).(string)
	if !ok || raw == "" {
		return uuid.Nil, ErrMissingTenantID
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %s", ErrMissingTenantID, err.Error())
	}
	return id, nil
}

func GetUserRoles(ctx context.Context) []string {
	if roles, ok := ctx.Value(UserRolesKey).([]string); ok {
		return roles
	}
	return nil
}

func GetUserPermissions(ctx context.Context) []string {
	if perms, ok := ctx.Value(UserPermsKey).([]string); ok {
		return perms
	}
	return nil
}

// IsAdmin checks if the current user has the "admin" role.
func IsAdmin(ctx context.Context) bool {
	for _, role := range GetUserRoles(ctx) {
		if role == "admin" {
			return true
		}
	}
	return false
}
