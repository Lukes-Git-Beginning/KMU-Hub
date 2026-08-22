package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestEmailRuleRoutes_Guards pins the permission wiring of the rule endpoints
// onto the module's existing coarse keys (email:read / write / delete), which
// is what keeps this unit free of a new permission seed.
//
// The registry is empty, so a request that clears the guard dies one layer down
// in getEmailClient with 502. The expected status is spelled out rather than
// "anything but 403": a mistyped path answers 404, which a loose assertion
// would accept without any guard ever having run.
func TestEmailRuleRoutes_Guards(t *testing.T) {
	router := chi.NewRouter()
	NewEmailRoutes(emptyRegistry()).RegisterRoutes(router, guardTestAuth)

	const ruleID = "6f1c9f1e-0000-4000-8000-000000000001"

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		perms  []string
		want   int
	}{
		{"list with read", http.MethodGet, "/api/v1/email/rules", "", []string{"email:read"}, http.StatusServiceUnavailable},
		{"list without read", http.MethodGet, "/api/v1/email/rules", "", nil, http.StatusForbidden},
		{"list with write only", http.MethodGet, "/api/v1/email/rules", "", []string{"email:write"}, http.StatusForbidden},

		{"create with write", http.MethodPost, "/api/v1/email/rules", "{}", []string{"email:write"}, http.StatusServiceUnavailable},
		{"create without write", http.MethodPost, "/api/v1/email/rules", "{}", []string{"email:read"}, http.StatusForbidden},

		{"apply with write", http.MethodPost, "/api/v1/email/rules/apply", "", []string{"email:write"}, http.StatusServiceUnavailable},
		{"apply without write", http.MethodPost, "/api/v1/email/rules/apply", "", []string{"email:read"}, http.StatusForbidden},

		{"patch with write", http.MethodPatch, "/api/v1/email/rules/" + ruleID, "{}", []string{"email:write"}, http.StatusServiceUnavailable},
		{"patch without write", http.MethodPatch, "/api/v1/email/rules/" + ruleID, "{}", nil, http.StatusForbidden},

		{"delete with delete", http.MethodDelete, "/api/v1/email/rules/" + ruleID, "", []string{"email:delete"}, http.StatusServiceUnavailable},
		{"delete with write only", http.MethodDelete, "/api/v1/email/rules/" + ruleID, "", []string{"email:write"}, http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req = withPermissions(req, tc.perms...)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Fatalf("%s %s: status = %d, want %d (body %q)",
					tc.method, tc.path, rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

// TestEmailRuleRoutes_ApplyIsNotSwallowedByTheIdRoute guards the chi ordering:
// /rules/apply must reach the apply handler, not the {id} PATCH/DELETE pair.
// Registering it after the parameterised routes would make "apply" a rule id.
func TestEmailRuleRoutes_ApplyIsNotSwallowedByTheIdRoute(t *testing.T) {
	router := chi.NewRouter()
	NewEmailRoutes(emptyRegistry()).RegisterRoutes(router, guardTestAuth)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/rules/apply", nil)
	req = withPermissions(req, "email:write")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// 405 would mean the path matched a route registered for other verbs only;
	// 404 would mean no match at all.
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST /rules/apply: status = %d, want %d (body %q)",
			rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}
