package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/server/response"
)

type userContextKey string

const (
	UserIDKey    userContextKey = "user_id"
	UserRolesKey userContextKey = "user_roles"
	UserPermsKey userContextKey = "user_permissions"
)

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
