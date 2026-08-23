package gateway

import (
	"net/http"
	"net/http/httptest"
	"net/url"
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

// TestDatevHandleOAuthCallback_ExternalErrorParamIsEscaped is the
// mutations-probe target for fix-gateway-datev-upload-error-message-leakage:
// the "error" query param is attacker/DATEV-controlled (this is the public,
// unauthenticated callback route) and reached the redirect Location unescaped
// before the fix. A raw "&" would let the value inject an extra query param,
// and a raw newline would be header/response splitting in front of some
// proxies. Both must come out percent-encoded, and the Location header itself
// must be a single line.
func TestDatevHandleOAuthCallback_ExternalErrorParamIsEscaped(t *testing.T) {
	routes := NewDatevUploadRoutes(registryWithService("biz"), "test-state-secret")
	rec := httptest.NewRecorder()
	malicious := "evil&injected=1\r\nSet-Cookie: pwned=1"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/datev/oauth/callback?error="+url.QueryEscape(malicious), nil)
	routes.HandleOAuthCallback(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	loc := rec.Header().Get("Location")
	if strings.Contains(loc, "\r") || strings.Contains(loc, "\n") {
		t.Fatalf("Location contains a raw CR/LF: %q", loc)
	}
	if strings.Contains(loc, "injected=1") || strings.Contains(loc, "Set-Cookie:") {
		t.Errorf("Location = %q, the malicious param leaked unescaped into the redirect", loc)
	}
	if !strings.Contains(loc, "datev_error=evil%26injected") {
		t.Errorf("Location = %q, want the value percent-encoded (e.g. %%26 for '&')", loc)
	}
}

// TestDatevHandleOAuthCallback_ValidState_RPCFailsRedirectsGenerically pins the
// actual leak fix: whatever the biz service returns (a masked gRPC error, per
// fix-gateway-datev-upload-error-message-leakage), the redirect the browser
// sees carries only the fixed "connection_failed" code, never the RPC error's
// text.
func TestDatevHandleOAuthCallback_ValidState_RPCFailsRedirectsGenerically(t *testing.T) {
	routes := NewDatevUploadRoutes(registryWithService("biz"), "test-state-secret")
	state, err := encodeBexioState("test-state-secret", "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("encodeBexioState: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/datev/oauth/callback?code=abc&state="+state, nil)
	routes.HandleOAuthCallback(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "datev_error=connection_failed") {
		t.Errorf("Location = %q, want to contain %q", loc, "datev_error=connection_failed")
	}
}

// ============================================================================
// ServiceUnavailable / NoTenant — every handler below calls getDatevUploadClient
// first and getTenantID second (HandleGetAuthURL checks the state secret before
// either, but with one configured it falls into the same order). Before this
// unit only the OAuth callback flow had any coverage; the other eight handlers
// — including the two that actually move money-adjacent documents to a third
// party — had none.
// ============================================================================

func TestDatevUploadRoutes_ServiceUnavailable(t *testing.T) {
	routes := NewDatevUploadRoutes(emptyRegistry(), "test-state-secret")

	handlers := map[string]http.HandlerFunc{
		"HandleGetAuthURL":           routes.HandleGetAuthURL,
		"HandleDisconnect":           routes.HandleDisconnect,
		"HandleGetConnectionStatus":  routes.HandleGetConnectionStatus,
		"HandleUploadBuchungsstapel": routes.HandleUploadBuchungsstapel,
		"HandleUploadBeleg":          routes.HandleUploadBeleg,
		"HandleGetUploadConfig":      routes.HandleGetUploadConfig,
		"HandleUpdateUploadConfig":   routes.HandleUpdateUploadConfig,
		"HandleListUploadLogs":       routes.HandleListUploadLogs,
	}

	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			testServiceUnavailable(t, h)
		})
	}
}

func TestDatevUploadRoutes_NoTenant(t *testing.T) {
	routes := NewDatevUploadRoutes(registryWithService("biz"), "test-state-secret")

	handlers := map[string]http.HandlerFunc{
		"HandleGetAuthURL":           routes.HandleGetAuthURL,
		"HandleDisconnect":           routes.HandleDisconnect,
		"HandleGetConnectionStatus":  routes.HandleGetConnectionStatus,
		"HandleUploadBuchungsstapel": routes.HandleUploadBuchungsstapel,
		"HandleUploadBeleg":          routes.HandleUploadBeleg,
		"HandleGetUploadConfig":      routes.HandleGetUploadConfig,
		"HandleUpdateUploadConfig":   routes.HandleUpdateUploadConfig,
		"HandleListUploadLogs":       routes.HandleListUploadLogs,
	}

	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
			// Deliberately no withTenantID — simulates a token without a tid claim.
			h(rec, req)
			assertStatus(t, rec, http.StatusUnauthorized)
			assertErrorContains(t, rec, "missing or invalid tenant")
		})
	}
}

// TestHandleUploadBuchungsstapel_MissingDates pins the transport-layer boundary
// check: start_date/end_date are required here even though the RPC field is
// technically optional proto3, because an absent range used to mean "every
// booking the tenant ever made" (see the handler's comment).
func TestHandleUploadBuchungsstapel_MissingDates(t *testing.T) {
	routes := NewDatevUploadRoutes(registryWithService("biz"), "test-state-secret")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/datev/upload", strings.NewReader("{}"))
	req = withTenantID(req, testTenantID)
	routes.HandleUploadBuchungsstapel(rec, req)
	assertValidationError(t, rec, "start_date")
}

// TestHandleUploadBuchungsstapel_ValidBodyReachesRPC proves a well-formed
// request clears the transport-layer checks and reaches the RPC (which then
// fails against the dummy localhost:0 connection as Unavailable -> 503,
// same pattern as the rest of the gateway package's "_ReachesRPC" tests).
func TestHandleUploadBuchungsstapel_ValidBodyReachesRPC(t *testing.T) {
	routes := NewDatevUploadRoutes(registryWithService("biz"), "test-state-secret")
	rec := httptest.NewRecorder()
	body := `{"start_date":"2026-01-01","end_date":"2026-01-31"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/datev/upload", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	routes.HandleUploadBuchungsstapel(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleUploadBeleg_InvalidInvoiceID pins that a malformed invoice_id path
// param is rejected locally (400) rather than reaching the RPC as a downstream
// error, the same guarantee route_biz_billing.go's id-bearing handlers give.
func TestHandleUploadBeleg_InvalidInvoiceID(t *testing.T) {
	routes := NewDatevUploadRoutes(registryWithService("biz"), "test-state-secret")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/datev/upload/beleg/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "invoice_id", "not-a-uuid")
	routes.HandleUploadBeleg(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid invoice_id")
}

func TestHandleUploadBeleg_ValidIDReachesRPC(t *testing.T) {
	routes := NewDatevUploadRoutes(registryWithService("biz"), "test-state-secret")
	rec := httptest.NewRecorder()
	invoiceID := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/datev/upload/beleg/"+invoiceID, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "invoice_id", invoiceID)
	routes.HandleUploadBeleg(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleUpdateUploadConfig_InvalidJSON pins the malformed-body 400 path;
// TestHandleUpdateUploadConfig_ValidBodyReachesRPC pins that a well-formed one
// clears validation (the struct has no `validate` tags — any JSON object with
// the right field types passes) and reaches the RPC.
func TestHandleUpdateUploadConfig_InvalidJSON(t *testing.T) {
	routes := NewDatevUploadRoutes(registryWithService("biz"), "test-state-secret")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/finance/datev/config", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleUpdateUploadConfig(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateUploadConfig_ValidBodyReachesRPC(t *testing.T) {
	routes := NewDatevUploadRoutes(registryWithService("biz"), "test-state-secret")
	rec := httptest.NewRecorder()
	body := `{"client_number":"12345","auto_upload_enabled":true,"upload_after_export":false}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/finance/datev/config", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	routes.HandleUpdateUploadConfig(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestDatevUploadRoutes_ReachRPC covers the four remaining handlers that take
// no body and no URL param, so the only two states possible are "no tenant"
// (covered above) and "reaches the RPC".
func TestDatevUploadRoutes_ReachRPC(t *testing.T) {
	routes := NewDatevUploadRoutes(registryWithService("biz"), "test-state-secret")

	handlers := map[string]struct {
		method string
		path   string
		h      http.HandlerFunc
	}{
		"HandleGetAuthURL":          {http.MethodGet, "/api/v1/finance/datev/oauth/authorize", routes.HandleGetAuthURL},
		"HandleDisconnect":          {http.MethodPost, "/api/v1/finance/datev/disconnect", routes.HandleDisconnect},
		"HandleGetConnectionStatus": {http.MethodGet, "/api/v1/finance/datev/status", routes.HandleGetConnectionStatus},
		"HandleGetUploadConfig":     {http.MethodGet, "/api/v1/finance/datev/config", routes.HandleGetUploadConfig},
		"HandleListUploadLogs":      {http.MethodGet, "/api/v1/finance/datev/upload/logs", routes.HandleListUploadLogs},
	}

	for name, tc := range handlers {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req = withTenantID(req, testTenantID)
			tc.h(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}
