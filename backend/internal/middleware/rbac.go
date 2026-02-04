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
