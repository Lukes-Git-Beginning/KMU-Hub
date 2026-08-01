package middleware

import (
	"net/http"

	"github.com/kmuhub/kmuhub/internal/server/response"
)

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRoles := GetUserRoles(r.Context())
			for _, ur := range userRoles {
				if roleSet[ur] {
					next.ServeHTTP(w, r)
					return
				}
			}
			response.Error(w, http.StatusForbidden, "insufficient role")
		})
	}
}

func RequirePermission(resource, action string) func(http.Handler) http.Handler {
	required := resource + ":" + action

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			perms := GetUserPermissions(r.Context())
			for _, p := range perms {
				if p == required {
					next.ServeHTTP(w, r)
					return
				}
			}
			response.Error(w, http.StatusForbidden, "insufficient permissions")
		})
	}
}

// RequirePermissionAny passes a request that carries at least ONE of the given
// resource:action pairs. It exists so a route can be tightened from a coarse
// permission ("files:write") to a fine-grained capability
// ("documents:file:upload") without locking anyone out:
//
//	RequirePermissionAny([2]string{"files", "write"}, [2]string{"documents:file", "upload"})
//
// Permissions are baked into the access token at login/refresh and are never
// re-read per request. Replacing a guard's key outright would therefore 403
// every user holding a still-valid token minted before the change — until they
// log in again. Always keep the existing key as one of the pairs; extend, never
// swap.
//
// The first pair is a separate parameter so that a call with no pair at all —
// which would deny everyone and only surface in production — cannot compile.
func RequirePermissionAny(first [2]string, rest ...[2]string) func(http.Handler) http.Handler {
	permSet := make(map[string]bool, len(rest)+1)
	permSet[first[0]+":"+first[1]] = true
	for _, p := range rest {
		permSet[p[0]+":"+p[1]] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, p := range GetUserPermissions(r.Context()) {
				if permSet[p] {
					next.ServeHTTP(w, r)
					return
				}
			}
			response.Error(w, http.StatusForbidden, "insufficient permissions")
		})
	}
}
