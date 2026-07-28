package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// datevRouter registers the DATEV upload routes on a fresh chi router the way
// cmd/gateway/main.go does, with the given OAuth state secret.
func datevRouter(t *testing.T, stateSecret string) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	NewDatevUploadRoutes(emptyRegistry(), stateSecret).RegisterRoutes(r, func(next http.Handler) http.Handler { return next })
	return r
}

// TestDatevUploadRoutes_Registered walks the router and asserts the DATEV
// endpoints are actually mounted. NewDatevUploadRoutes had no caller anywhere in
// the repo, so every /api/v1/finance/datev/* path was a 404 against a real
// gateway even though handlers, RPCs and openapi.yaml entries were complete.
// Walking the router is the check; "the code exists" is not.
func TestDatevUploadRoutes_Registered(t *testing.T) {
	r := datevRouter(t, "test-state-secret")

	registered := make(map[string]bool)
	err := chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		// chi.Walk appends a trailing slash for handlers mounted at the root of
		// an r.Route(...) sub-router; the spec documents them without one.
		registered[method+" "+strings.TrimSuffix(route, "/")] = true
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk failed: %v", err)
	}

	want := []string{
		"GET /api/v1/finance/datev/oauth/callback",
		"GET /api/v1/finance/datev/oauth/authorize",
		"POST /api/v1/finance/datev/disconnect",
		"GET /api/v1/finance/datev/status",
		"POST /api/v1/finance/datev/upload",
		"POST /api/v1/finance/datev/upload/beleg/{invoice_id}",
		"GET /api/v1/finance/datev/config",
		"PUT /api/v1/finance/datev/config",
		"GET /api/v1/finance/datev/upload/logs",
	}
	for _, w := range want {
		if !registered[w] {
			t.Errorf("route not registered: %s", w)
		}
	}
}

// TestDatevUploadRoutes_EmptyStateSecretRejects pins that an unconfigured state
// secret never yields a usable OAuth flow. The empty secret is the default
// (BEXIO_STATE_SECRET has `default=`), and an empty HMAC key must not mean
// "every state is authentic": both handlers refuse before reaching the backend.
func TestDatevUploadRoutes_EmptyStateSecretRejects(t *testing.T) {
	r := datevRouter(t, "")

	t.Run("authorize refuses to issue a URL", func(t *testing.T) {
		// Called directly: the mounted route sits behind RequireRole("admin"), so a
		// request through the router is rejected as 403 before the handler runs —
		// which would test the middleware, not the guard this case is about.
		rec := httptest.NewRecorder()
		NewDatevUploadRoutes(emptyRegistry(), "").
			HandleGetAuthURL(rec, httptest.NewRequest(http.MethodGet, "/api/v1/finance/datev/oauth/authorize", nil))
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("authorize is admin-only through the router", func(t *testing.T) {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/finance/datev/oauth/authorize", nil))
		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want %d (RequireRole admin)", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("callback refuses the redirect", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/datev/oauth/callback?code=abc&state=anything", nil)
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
		}
		if loc := rec.Header().Get("Location"); !strings.Contains(loc, "datev_error=state_not_configured") {
			t.Errorf("Location = %q, want a state_not_configured error redirect", loc)
		}
	})
}

// TestDatevUploadRoutes_ForgedStateRejected pins that a state token signed with
// a different secret does not pass verification — the CSRF guard is what keeps
// the public callback route from writing tokens against an attacker's tenant_id.
func TestDatevUploadRoutes_ForgedStateRejected(t *testing.T) {
	forged, err := encodeBexioState("attacker-secret", "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("encodeBexioState failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/datev/oauth/callback?code=abc&state="+forged, nil)
	datevRouter(t, "real-secret").ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "datev_error=invalid_state") {
		t.Errorf("Location = %q, want an invalid_state error redirect", loc)
	}
}
