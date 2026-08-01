package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestEmailLabelRoutes_Guards pins the permission wiring of the label
// endpoints onto the module's existing coarse keys (email:read / write /
// delete), which is what keeps this unit free of a new permission seed.
func TestEmailLabelRoutes_Guards(t *testing.T) {
	router := chi.NewRouter()
	NewEmailRoutes(emptyRegistry()).RegisterRoutes(router, guardTestAuth)

	const labelID = "6f1c9f1e-0000-4000-8000-000000000002"
	const messageID = "6f1c9f1e-0000-4000-8000-000000000003"

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		perms  []string
		want   int
	}{
		{"list with read", http.MethodGet, "/api/v1/email/labels", "", []string{"email:read"}, http.StatusBadGateway},
		{"list without read", http.MethodGet, "/api/v1/email/labels", "", nil, http.StatusForbidden},
		{"list with write only", http.MethodGet, "/api/v1/email/labels", "", []string{"email:write"}, http.StatusForbidden},

		{"create with write", http.MethodPost, "/api/v1/email/labels", "{}", []string{"email:write"}, http.StatusBadGateway},
		{"create without write", http.MethodPost, "/api/v1/email/labels", "{}", []string{"email:read"}, http.StatusForbidden},

		{"patch with write", http.MethodPatch, "/api/v1/email/labels/" + labelID, "{}", []string{"email:write"}, http.StatusBadGateway},
		{"patch without write", http.MethodPatch, "/api/v1/email/labels/" + labelID, "{}", nil, http.StatusForbidden},

		{"delete with delete", http.MethodDelete, "/api/v1/email/labels/" + labelID, "", []string{"email:delete"}, http.StatusBadGateway},
		{"delete with write only", http.MethodDelete, "/api/v1/email/labels/" + labelID, "", []string{"email:write"}, http.StatusForbidden},

		{"assign with write", http.MethodPost, "/api/v1/email/messages/" + messageID + "/labels", "{}", []string{"email:write"}, http.StatusBadGateway},
		{"assign without write", http.MethodPost, "/api/v1/email/messages/" + messageID + "/labels", "{}", []string{"email:read"}, http.StatusForbidden},
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
