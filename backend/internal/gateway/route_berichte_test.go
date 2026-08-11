package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/featureflag"
)

// ============================================================================
// Helpers for Berichte tests
// ============================================================================

// berichteFlagsOFF returns a featureflag.Registry with modules.berichte disabled.
func berichteFlagsOFF() *featureflag.Registry {
	return featureflag.NewRegistry().Load(func(string) string { return "" })
}

// berichteFlagsON returns a featureflag.Registry with modules.berichte enabled.
func berichteFlagsON() *featureflag.Registry {
	return featureflag.NewRegistry().Load(func(k string) string {
		if k == "COSMI_MODULE_BERICHTE_ENABLED" {
			return "true"
		}
		return ""
	})
}

// ============================================================================
// ServiceName
// ============================================================================

func TestBerichteRoutes_ServiceName(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsOFF())
	if routes.ServiceName() != "berichte" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "berichte")
	}
}

// ============================================================================
// Feature Flag guard
// ============================================================================

// TestBerichteRoutes_FlagOFF verifies that no routes are registered when the flag is off.
func TestBerichteRoutes_FlagOFF(t *testing.T) {
	r := chi.NewRouter()
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsOFF())
	routes.RegisterRoutes(r, func(next http.Handler) http.Handler { return next })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/berichte/definitions", nil)
	r.ServeHTTP(rec, req)

	// With flag off the route is not registered → chi returns 404
	assertStatus(t, rec, http.StatusNotFound)
}

// TestBerichteRoutes_FlagON verifies that routes are registered when the flag is on.
// The route is matched (not 404) — the exact status code depends on middleware:
// • 401 when tenant check fires (no TenantID in context) → route matched
// • 403 when RequirePermission fires (no permission in context) → route matched
// • 503 when gRPC backend is absent
// All are valid "route was registered" signals; we only reject 404.
func TestBerichteRoutes_FlagON(t *testing.T) {
	r := chi.NewRouter()
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	// Use identity auth middleware so RequireAuthenticated doesn't block
	routes.RegisterRoutes(r, func(next http.Handler) http.Handler { return next })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/berichte/definitions", nil)
	req = withUserID(req, "user-123")
	r.ServeHTTP(rec, req)

	// Route is registered — anything other than 404 proves the route matched.
	if rec.Code == http.StatusNotFound {
		t.Errorf("expected route to be registered (not 404), got %d; body = %s",
			rec.Code, rec.Body.String())
	}
}

// ============================================================================
// Definition Handlers
// ============================================================================

func TestHandleListDefinitions_ServiceUnavailable(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	testServiceUnavailable(t, routes.HandleListDefinitions)
}

func TestHandleCreateDefinition_ServiceUnavailable(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	testServiceUnavailable(t, routes.HandleCreateDefinition)
}

func TestHandleCreateDefinition_InvalidJSON(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/definitions", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateDefinition(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateDefinition_MissingName(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/definitions",
		jsonBody(t, map[string]interface{}{"module": "crm"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateDefinition(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateDefinition_MissingModule(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/definitions",
		jsonBody(t, map[string]interface{}{"name": "My Report"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateDefinition(rec, req)
	assertValidationError(t, rec, "module")
}

func TestHandleGetDefinition_ServiceUnavailable(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/berichte/definitions/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withTenantID(req, testTenantID)
	routes.HandleGetDefinition(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetDefinition_InvalidUUID(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/berichte/definitions/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withTenantID(req, testTenantID)
	routes.HandleGetDefinition(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateDefinition_InvalidUUID(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/berichte/definitions/bad",
		jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateDefinition(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateDefinition_InvalidJSON(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/berichte/definitions/550e8400-e29b-41d4-a716-446655440000",
		invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateDefinition(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleDeleteDefinition_InvalidUUID(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/berichte/definitions/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withTenantID(req, testTenantID)
	routes.HandleDeleteDefinition(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// ============================================================================
// Run & Export Handlers
// ============================================================================

func TestHandleRunReport_ServiceUnavailable(t *testing.T) {
	// testServiceUnavailable sends POST with "{}" body — HandleRunReport reads body first,
	// then calls getClient(). So we use the direct form with a valid body + UUID.
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/definitions/550e8400-e29b-41d4-a716-446655440000/run",
		jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withTenantID(req, testTenantID)
	routes.HandleRunReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRunReport_InvalidUUID(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/definitions/bad/run",
		jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withTenantID(req, testTenantID)
	routes.HandleRunReport(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleRunReport_InvalidJSON(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/definitions/550e8400-e29b-41d4-a716-446655440000/run",
		invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withTenantID(req, testTenantID)
	routes.HandleRunReport(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleExportReport_ServiceUnavailable(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/definitions/550e8400-e29b-41d4-a716-446655440000/export",
		jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withTenantID(req, testTenantID)
	routes.HandleExportReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleExportReport_InvalidUUID(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/definitions/bad/export",
		jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withTenantID(req, testTenantID)
	routes.HandleExportReport(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleExportReport_InvalidJSON(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/definitions/550e8400-e29b-41d4-a716-446655440000/export",
		invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withTenantID(req, testTenantID)
	routes.HandleExportReport(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleExportReport_ServiceUnavailableAfterUUID(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/definitions/550e8400-e29b-41d4-a716-446655440000/export",
		jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withTenantID(req, testTenantID)
	routes.HandleExportReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleInvalidateCache_ServiceUnavailableAfterUUID(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/berichte/definitions/550e8400-e29b-41d4-a716-446655440000/cache", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withTenantID(req, testTenantID)
	routes.HandleInvalidateCache(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleInvalidateCache_InvalidUUID(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/berichte/definitions/bad/cache", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withTenantID(req, testTenantID)
	routes.HandleInvalidateCache(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// ============================================================================
// Schedule Handlers
// ============================================================================

func TestHandleListSchedules_ServiceUnavailable(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	testServiceUnavailable(t, routes.HandleListSchedules)
}

func TestHandleCreateSchedule_ServiceUnavailable(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	testServiceUnavailable(t, routes.HandleCreateSchedule)
}

func TestHandleCreateSchedule_InvalidJSON(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/schedules", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateSchedule(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateSchedule_MissingName(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/schedules",
		jsonBody(t, map[string]interface{}{"cron_expression": "0 8 * * 1"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateSchedule(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateSchedule_MissingCron(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/schedules",
		jsonBody(t, map[string]interface{}{"name": "Weekly"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateSchedule(rec, req)
	assertValidationError(t, rec, "cron_expression")
}

func TestHandleUpdateSchedule_InvalidUUID(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/berichte/schedules/bad",
		jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withTenantID(req, testTenantID)
	routes.HandleUpdateSchedule(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteSchedule_InvalidUUID(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/berichte/schedules/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withTenantID(req, testTenantID)
	routes.HandleDeleteSchedule(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleToggleSchedule_InvalidUUID(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/schedules/bad/toggle",
		jsonBody(t, map[string]interface{}{"active": true}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withTenantID(req, testTenantID)
	routes.HandleToggleSchedule(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleToggleSchedule_InvalidJSON(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/schedules/550e8400-e29b-41d4-a716-446655440000/toggle",
		invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withTenantID(req, testTenantID)
	routes.HandleToggleSchedule(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// ============================================================================
// KPI Handler
// ============================================================================

func TestHandleGetDashboardKPIs_ServiceUnavailable(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	testServiceUnavailable(t, routes.HandleGetDashboardKPIs)
}

// ============================================================================
// formatFilename helper
// ============================================================================

func TestFormatFilename_Safe(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string // must be wrapped in quotes and sanitized
	}{
		{"normal", "report.pdf", `"report.pdf"`},
		{"with slash", "path/to/report.pdf", `"pathtoreport.pdf"`},
		{"with backslash", `path\to\report.pdf`, `"pathtoreport.pdf"`},
		{"with newline", "report\ninjection.pdf", `"reportinjection.pdf"`},
		{"with CR", "report\rinjection.pdf", `"reportinjection.pdf"`},
		{"with double quote", `report"injection.pdf`, `"reportinjection.pdf"`},
		{"empty", "", `"report"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatFilename(tt.input)
			if got != tt.want {
				t.Errorf("formatFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ============================================================================
// NoUserID auth guard tests
// ============================================================================

func TestHandleCreateDefinition_NoUserID(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/definitions",
		jsonBody(t, map[string]interface{}{"name": "x", "module": "crm"}))
	withAuthRequired(routes.HandleCreateDefinition)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "not authenticated")
}

func TestHandleCreateSchedule_NoUserID(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/schedules",
		jsonBody(t, map[string]interface{}{"name": "x", "cron_expression": "0 8 * * 1"}))
	withAuthRequired(routes.HandleCreateSchedule)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "not authenticated")
}

// ============================================================================
// Export: Content-Disposition header format test (no gRPC call needed — mocked below)
// ============================================================================

// TestFormatFilename_ContentDispositionInjection verifies that filename values
// cannot inject additional Content-Disposition directives.
func TestFormatFilename_ContentDispositionInjection(t *testing.T) {
	malicious := "report.pdf\"; filename=\"evil.exe"
	result := formatFilename(malicious)

	// Must not contain unescaped double-quotes in the middle
	// (outer quotes are part of the format but inner ones must be stripped)
	inner := strings.TrimPrefix(strings.TrimSuffix(result, `"`), `"`)
	if strings.Contains(inner, `"`) {
		t.Errorf("formatFilename did not strip embedded quotes: %q", result)
	}
}

// ============================================================================
// Share links
// ============================================================================

// passthroughLimiter stands in for the public rate limiter. The real one is
// wired in main.go; here we only need the route tree it wraps.
func passthroughLimiter(next http.Handler) http.Handler { return next }

// TestBerichtePublicShareRoute_Registered pins that the public read is mounted
// under /api/v1/public/ as a POST. The verb is not cosmetic: a GET would put
// the password in the URL and let a prefetch bump the view counter.
func TestBerichtePublicShareRoute_Registered(t *testing.T) {
	r := chi.NewRouter()
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	routes.RegisterPublicRoutes(r, passthroughLimiter)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/berichte/reports/sometoken", nil)
	r.ServeHTTP(rec, req)
	if rec.Code == http.StatusNotFound {
		t.Fatalf("public share route not registered; got 404, body = %s", rec.Body.String())
	}

	// The same path as a GET must not answer at all.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/public/berichte/reports/sometoken", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound {
		t.Fatalf("GET on the public share route answered %d; it must not be a GET", rec.Code)
	}
}

// TestBerichtePublicShareRoute_FlagOFF pins that the module flag also gates the
// public path — an unauthenticated route surviving a disabled module would be
// the worst one to forget.
func TestBerichtePublicShareRoute_FlagOFF(t *testing.T) {
	r := chi.NewRouter()
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsOFF())
	routes.RegisterPublicRoutes(r, passthroughLimiter)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/berichte/reports/sometoken", nil)
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusNotFound)
}

// TestHandleGetSharedReport_NoAuthNeeded pins that the public handler never
// reads a tenant or user from the request: it must work with a context that
// carries neither, because a visitor has neither.
func TestHandleGetSharedReport_NoAuthNeeded(t *testing.T) {
	r := chi.NewRouter()
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	routes.RegisterPublicRoutes(r, passthroughLimiter)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/berichte/reports/sometoken", nil)
	r.ServeHTTP(rec, req)

	// No berichte backend is reachable in this test, so the honest answer is
	// 503 — but never 401/403, which would mean the handler asked for auth.
	if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
		t.Fatalf("public share handler demanded auth: %d, body = %s", rec.Code, rec.Body.String())
	}
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleGetSharedReport_MalformedTokenLooksLikeUnknown pins that a token
// which cannot exist gets the same 404 an unknown one gets, and never reaches
// the backend.
func TestHandleGetSharedReport_MalformedTokenLooksLikeUnknown(t *testing.T) {
	r := chi.NewRouter()
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	routes.RegisterPublicRoutes(r, passthroughLimiter)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/public/berichte/reports/"+strings.Repeat("x", 200), nil)
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusNotFound)
}

// TestHandleGetSharedReport_BodyIsOptional pins that a link without a password
// is readable with no request body at all — an empty body must not be a 400.
func TestHandleGetSharedReport_BodyIsOptional(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())

	for name, body := range map[string]string{
		"absent": "",
		"blank":  "   ",
		"empty":  "{}",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost,
			"/api/v1/public/berichte/reports/sometoken", strings.NewReader(body))
		req = withChiURLParam(req, "token", "sometoken")
		routes.HandleGetSharedReport(rec, req)
		if rec.Code == http.StatusBadRequest {
			t.Fatalf("%s body rejected as malformed: %s", name, rec.Body.String())
		}
	}

	// Actually malformed JSON is still a 400.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/public/berichte/reports/sometoken", strings.NewReader("{not json"))
	req = withChiURLParam(req, "token", "sometoken")
	routes.HandleGetSharedReport(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListReportShares_ServiceUnavailable(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	testServiceUnavailable(t, routes.HandleListReportShares)
}

func TestHandleCreateReportShare_ServiceUnavailable(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	testServiceUnavailable(t, routes.HandleCreateReportShare)
}

func TestHandleRevokeReportShare_ServiceUnavailable(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	testServiceUnavailable(t, routes.HandleRevokeReportShare)
}

// TestShareAdminHandlers_RequireTenant pins that the three authenticated share
// handlers refuse a request without a tenant instead of falling through to a
// tenant-less gRPC call.
func TestShareAdminHandlers_RequireTenant(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	handlers := map[string]http.HandlerFunc{
		"list":   routes.HandleListReportShares,
		"create": routes.HandleCreateReportShare,
		"revoke": routes.HandleRevokeReportShare,
	}
	for name, h := range handlers {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/documents/x/shares", nil)
		req = withUserID(req, "user-123") // user but no tenant
		h(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: got %d, want 401 without a tenant", name, rec.Code)
		}
	}
}

// ============================================================================
// Document Handlers (multi-page authoring)
// ============================================================================

func TestHandleListDocuments_ServiceUnavailable(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/berichte/documents", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListDocuments(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListDocuments_MissingTenant(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/berichte/documents", nil)
	routes.HandleListDocuments(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetDocument_ServiceUnavailable(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/berichte/documents/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withTenantID(req, testTenantID)
	routes.HandleGetDocument(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetDocument_InvalidUUID(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/berichte/documents/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withTenantID(req, testTenantID)
	routes.HandleGetDocument(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleExportDocumentPDF_ServiceUnavailable(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/berichte/documents/550e8400-e29b-41d4-a716-446655440000/export/pdf", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withTenantID(req, testTenantID)
	routes.HandleExportDocumentPDF(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleExportDocumentPDF_InvalidUUID(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/berichte/documents/bad/export/pdf", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withTenantID(req, testTenantID)
	routes.HandleExportDocumentPDF(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleCreateDocument_ServiceUnavailable(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	testServiceUnavailable(t, routes.HandleCreateDocument)
}

func TestHandleCreateDocument_InvalidJSON(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/documents", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateDocument(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// TestHandleCreateDocument_InvalidModule covers the request's only structural
// validation rule. createReportDocumentRequest.Title has no `validate:"required"`
// tag (unlike createDefinitionRequest.Name) — an empty title is accepted here and
// left to the berichte service to reject, so it is not a gateway-level error path.
// Module is the one field decodeAndValidate actually rejects at the boundary.
func TestHandleCreateDocument_InvalidModule(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/documents",
		jsonBody(t, map[string]interface{}{"title": "Q1 Report", "module": "not-a-real-module"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateDocument(rec, req)
	assertValidationError(t, rec, "module")
}

func TestHandleUpdateDocument_InvalidUUID(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/berichte/documents/bad",
		jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateDocument(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateDocument_InvalidJSON(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/berichte/documents/550e8400-e29b-41d4-a716-446655440000",
		invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateDocument(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateDocument_InvalidStatus(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/berichte/documents/550e8400-e29b-41d4-a716-446655440000",
		jsonBody(t, map[string]interface{}{"status": "not-a-real-status"}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateDocument(rec, req)
	assertValidationError(t, rec, "status")
}

func TestHandleDeleteDocument_ServiceUnavailable(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/berichte/documents/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withTenantID(req, testTenantID)
	routes.HandleDeleteDocument(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteDocument_InvalidUUID(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/berichte/documents/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withTenantID(req, testTenantID)
	routes.HandleDeleteDocument(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}
