package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// route_hr_analytics_settings_test.go covers the third and final HR gateway
// unit: the analytics/summary read paths and the tenant-wide HR settings.
// These handlers only reach the HR gRPC service (no fake HRServiceClient
// exists in this package, see route_hr_leave_test.go), so "reaches RPC" here
// means the handler parses the request and gets as far as a transport error
// against the unstarted test client — the same convention as the rest of
// route_hr*_test.go. Aggregation correctness (tenant scope, empty-period
// behaviour) is proven at the service layer in
// internal/biz/hr/timetracking/service_test.go instead, since a gateway test
// cannot observe response bodies without a live/faked RPC.

// ============================================================================
// HandleDailySummary
// ============================================================================

func TestHandleDailySummary_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/summary/daily", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleDailySummary(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDailySummary_ReachesRPC_DefaultDate(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/summary/daily", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleDailySummary(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDailySummary_ReachesRPC_ExplicitDate(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/summary/daily?date=2026-08-20", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleDailySummary(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleWeeklySummary
// ============================================================================

func TestHandleWeeklySummary_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/summary/weekly", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleWeeklySummary(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleWeeklySummary_ReachesRPC_DefaultWeekStart(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/summary/weekly", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleWeeklySummary(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleWeeklySummary_ReachesRPC_ExplicitWeekStart(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/summary/weekly?week_start=2026-08-17", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleWeeklySummary(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetTimeAnalytics
// ============================================================================

func TestHandleGetTimeAnalytics_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/analytics", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetTimeAnalytics(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetTimeAnalytics_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/analytics", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetTimeAnalytics(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetTimeAnalytics_ReachesRPC_DefaultRange(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/analytics", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetTimeAnalytics(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetTimeAnalytics_ReachesRPC_MonthRange(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/analytics?range=month", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetTimeAnalytics(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetHRSettings
// ============================================================================

func TestHandleGetHRSettings_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/settings", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetHRSettings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetHRSettings_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/settings", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetHRSettings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetHRSettings_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/settings", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetHRSettings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateHRSettings
// ============================================================================

func TestHandleUpdateHRSettings_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/settings", jsonBody(t, map[string]interface{}{
		"au_threshold_days": 6,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateHRSettings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateHRSettings_MissingTenant(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/settings", jsonBody(t, map[string]interface{}{
		"au_threshold_days": 6,
	}))
	req = withUserID(req, "user-123")
	routes.HandleUpdateHRSettings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateHRSettings_InvalidWorkHoursPerDay(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/settings", jsonBody(t, map[string]interface{}{
		"work_hours_per_day": 25,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateHRSettings(rec, req)
	assertValidationError(t, rec, "work_hours_per_day")
}

func TestHandleUpdateHRSettings_NegativeAUThresholdDays(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/settings", jsonBody(t, map[string]interface{}{
		"au_threshold_days": -1,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateHRSettings(rec, req)
	assertValidationError(t, rec, "au_threshold_days")
}

func TestHandleUpdateHRSettings_ReachesRPC(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hr/settings", jsonBody(t, map[string]interface{}{
		"au_threshold_days":         6,
		"default_annual_leave_days": 28,
		"work_hours_per_day":        8,
		"max_daily_hours":           10,
		"break_after_hours":         6,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateHRSettings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
