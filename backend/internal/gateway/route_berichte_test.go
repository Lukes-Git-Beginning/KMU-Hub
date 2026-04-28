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
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "name is required")
}

func TestHandleCreateDefinition_MissingModule(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/definitions",
		jsonBody(t, map[string]interface{}{"name": "My Report"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateDefinition(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "module is required")
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
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "name is required")
}

func TestHandleCreateSchedule_MissingCron(t *testing.T) {
	routes := NewBerichteRoutes(registryWithService("berichte"), berichteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/schedules",
		jsonBody(t, map[string]interface{}{"name": "Weekly"}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateSchedule(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "cron_expression is required")
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
