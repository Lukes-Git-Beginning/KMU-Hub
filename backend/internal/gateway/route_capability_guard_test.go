package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

// guardTestAuth stands in for the JWT auth middleware. The guards under test
// read the permission list straight from the request context, which each test
// case fills in itself.
func guardTestAuth(next http.Handler) http.Handler { return next }

// withPermissions puts a permission list into the context the way the auth
// middleware does after decoding an access token.
func withPermissions(r *http.Request, perms ...string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), middleware.UserPermsKey, perms))
}

// TestCapabilityGuards_AdditiveWiring checks the guard wiring of the modules
// that were tightened onto capability-catalogue keys: every route must accept
// BOTH its legacy coarse key and its catalogue key, and must still reject a
// token that carries neither.
//
// The registry is empty, so a request that passes the guard dies further down
// with 503 (service not registered). The expected status is spelled out rather
// than "anything but 403" on purpose: a mistyped path would return 404, which
// would satisfy a loose assertion without any guard ever running.
func TestCapabilityGuards_AdditiveWiring(t *testing.T) {
	wikiRouter := chi.NewRouter()
	newWikiRoutes(emptyRegistry()).RegisterRoutes(wikiRouter, guardTestAuth)

	hrRouter := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(hrRouter, guardTestAuth)

	const articleID = "11111111-1111-1111-1111-111111111111"

	const (
		allowed = http.StatusServiceUnavailable // guard passed, empty registry stops it
		denied  = http.StatusForbidden
	)

	tests := []struct {
		name       string
		router     chi.Router
		method     string
		path       string
		perms      []string
		wantStatus int
	}{
		// --- wiki: read ---
		{"wiki list, legacy key only", wikiRouter, "GET", "/api/v1/wiki/articles/", []string{"wiki:articles:read"}, allowed},
		{"wiki list, catalogue key only", wikiRouter, "GET", "/api/v1/wiki/articles/", []string{"wiki:article:read"}, allowed},
		{"wiki list, neither key", wikiRouter, "GET", "/api/v1/wiki/articles/", []string{"wiki:categories:read"}, denied},
		{"wiki list, no permissions at all", wikiRouter, "GET", "/api/v1/wiki/articles/", nil, denied},

		// --- wiki: write actions are distinct capabilities ---
		{"wiki create, legacy key only", wikiRouter, "POST", "/api/v1/wiki/articles/", []string{"wiki:articles:write"}, allowed},
		{"wiki create, catalogue key only", wikiRouter, "POST", "/api/v1/wiki/articles/", []string{"wiki:article:create"}, allowed},
		{"wiki create, read key does not grant write", wikiRouter, "POST", "/api/v1/wiki/articles/", []string{"wiki:article:read"}, denied},
		{"wiki edit, catalogue key only", wikiRouter, "PATCH", "/api/v1/wiki/articles/" + articleID + "/", []string{"wiki:article:edit"}, allowed},
		{"wiki edit, delete key does not grant edit", wikiRouter, "PATCH", "/api/v1/wiki/articles/" + articleID + "/", []string{"wiki:article:delete"}, denied},
		{"wiki delete, catalogue key only", wikiRouter, "DELETE", "/api/v1/wiki/articles/" + articleID + "/", []string{"wiki:article:delete"}, allowed},

		// --- wiki: share tokens and categories ---
		{"wiki share create, catalogue key only", wikiRouter, "POST", "/api/v1/wiki/articles/" + articleID + "/share", []string{"wiki:share_token:create"}, allowed},
		{"wiki share revoke, catalogue key only", wikiRouter, "DELETE", "/api/v1/wiki/share/" + articleID, []string{"wiki:share_token:create"}, allowed},
		{"wiki category manage, legacy key only", wikiRouter, "POST", "/api/v1/wiki/categories/", []string{"wiki:categories:write"}, allowed},
		{"wiki category manage, catalogue key only", wikiRouter, "POST", "/api/v1/wiki/categories/", []string{"wiki:category:manage"}, allowed},
		{"wiki category manage, article key does not grant it", wikiRouter, "POST", "/api/v1/wiki/categories/", []string{"wiki:article:edit"}, denied},

		// --- zeiterfassung: supervisory routes ---
		{"week approve, legacy key only", hrRouter, "POST", "/api/v1/hr/time/weeks/approve", []string{"hr:write"}, allowed},
		{"week approve, catalogue key only", hrRouter, "POST", "/api/v1/hr/time/weeks/approve", []string{"zeiterfassung:week:approve"}, allowed},
		{"week reject, catalogue key only", hrRouter, "POST", "/api/v1/hr/time/weeks/reject", []string{"zeiterfassung:week:approve"}, allowed},
		{"week approve, team key does not grant it", hrRouter, "POST", "/api/v1/hr/time/weeks/approve", []string{"zeiterfassung:team:view"}, denied},
		{"team view, catalogue key only", hrRouter, "GET", "/api/v1/hr/time/team", []string{"zeiterfassung:team:view"}, allowed},
		{"correction approve, catalogue key only", hrRouter, "POST", "/api/v1/hr/time/corrections/" + articleID + "/approve", []string{"zeiterfassung:corrections:approve"}, allowed},

		// --- zeiterfassung: personal daily use stays on the coarse key ---
		// The catalogue deliberately carries no capability for clocking in, so
		// a supervisory key must NOT open it.
		{"clock-in still requires hr:write", hrRouter, "POST", "/api/v1/hr/time/clock-in", []string{"zeiterfassung:week:approve"}, denied},
		{"clock-in with hr:write", hrRouter, "POST", "/api/v1/hr/time/clock-in", []string{"hr:write"}, allowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req = withAuth(req, "user-123", testTenantID)
			req = withPermissions(req, tt.perms...)

			rec := httptest.NewRecorder()
			tt.router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
