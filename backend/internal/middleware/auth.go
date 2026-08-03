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
	UserIDKey     userContextKey = "user_id"
	UserRolesKey  userContextKey = "user_roles"
	UserPermsKey  userContextKey = "user_permissions"
	UserScopesKey userContextKey = "user_permission_scopes"
	UserDeniedKey userContextKey = "user_denied_permissions"
	TenantIDKey   userContextKey = "tenant_id"
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
			ctx = context.WithValue(ctx, UserScopesKey, claims.Scopes)
			ctx = context.WithValue(ctx, UserDeniedKey, claims.Denied)
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

// GetDeniedPermissions returns the capability keys a per-user deny override
// took away from the current account.
//
// These keys are already absent from GetUserPermissions — the list is not a
// second allow check. It answers the one question absence cannot: whether a
// coarse legacy key in the same guard is still a legitimate stand-in for the
// fine key next to it (RequirePermissionAny) or whether an administrator
// deliberately closed that door.
//
// Empty for every account without overrides and for every token minted before
// the claim existed, which is the behaviour those tokens already had.
func GetDeniedPermissions(ctx context.Context) []string {
	if denied, ok := ctx.Value(UserDeniedKey).([]string); ok {
		return denied
	}
	return nil
}

// PermissionScope reports how far one permission reaches for the current user:
// auth.ScopeOwn (only rows they own), auth.ScopeTeam or auth.ScopeAll.
//
// It answers "how many rows", never "may this run at all" — that stays with
// RequirePermission/RequirePermissionAny, which have already passed by the time
// a handler asks. A caller without the permission never gets here, so an
// unknown key legitimately means "not narrowed" rather than "denied".
//
// Absence therefore resolves to auth.ScopeAll, which is also what a token
// minted before the scopes claim existed reports for every key. That keeps a
// still-valid old session behaving exactly as it did instead of silently
// emptying its lists.
//
// Handler pattern for a list endpoint (see route_rapporte.go HandleListReports):
//
//	if middleware.PermissionScope(r.Context(), "rapporte:report", "read") == auth.ScopeOwn {
//	    grpcReq.AuthorId = &userID // overrides any client-supplied filter
//	}
//
// The filter belongs in the repository query, not in a post-filter on the
// response: paging and the total count have to agree with what the caller may
// see. auth.ScopeTeam needs a reporting-line resolver and does not exist yet —
// it currently behaves like auth.ScopeAll.
func PermissionScope(ctx context.Context, resource, action string) string {
	scopes, ok := ctx.Value(UserScopesKey).(map[string]string)
	if !ok {
		return auth.ScopeAll
	}
	if scope, ok := scopes[resource+":"+action]; ok {
		return scope
	}
	return auth.ScopeAll
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
