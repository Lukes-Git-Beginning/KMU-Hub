package middleware

import (
	"net/http"
	"slices"
	"strings"

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

// RequirePermission passes a request that carries the given resource:action.
//
// It reads no deny list: a key a per-user deny override took away is already
// missing from the token's allow set (Service.ResolveTokenPermissions), and a
// single-key guard has no second key that could stand in for it. The deny list
// only matters where one does — see RequirePermissionAny.
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
//
// # Per-user deny overrides
//
// The coarse key is a stand-in for the fine one, not a right of its own: both
// name the same door, and that is exactly what makes a deny override hard here.
// A deny drops the fine key from the token's allow set, but the coarse key sits
// right next to it and would keep the door open — in 154 of the 164 call sites
// of this guard. So a coarse key stops counting as soon as ANY fine key of the
// SAME guard is explicitly denied.
//
// Fine keys are judged individually. Two fine keys in one guard are two
// different rights that happen to open the same route — denying
// "berichte:export:run" must not also take "berichte:reports:read" away.
//
// "Coarse" means a resource without a colon ("files"), "fine" means the
// three-segment catalogue form ("documents:file"). That is the same line
// PostgresRepository.GetEffectivePermissions draws with `resource LIKE '%:%'`.
func RequirePermissionAny(first [2]string, rest ...[2]string) func(http.Handler) http.Handler {
	fineSet := make(map[string]bool, len(rest)+1)
	coarseSet := make(map[string]bool, len(rest)+1)
	fineKeys := make([]string, 0, len(rest)+1)

	for _, p := range append([][2]string{first}, rest...) {
		key := p[0] + ":" + p[1]
		if strings.Contains(p[0], ":") {
			fineSet[key] = true
			fineKeys = append(fineKeys, key)
			continue
		}
		coarseSet[key] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			coarseCounts := len(coarseSet) > 0
			if coarseCounts && len(fineKeys) > 0 {
				for _, denied := range GetDeniedPermissions(r.Context()) {
					if slices.Contains(fineKeys, denied) {
						coarseCounts = false
						break
					}
				}
			}

			for _, p := range GetUserPermissions(r.Context()) {
				if fineSet[p] || (coarseCounts && coarseSet[p]) {
					next.ServeHTTP(w, r)
					return
				}
			}
			response.Error(w, http.StatusForbidden, "insufficient permissions")
		})
	}
}
