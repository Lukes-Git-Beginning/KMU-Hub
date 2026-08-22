package gateway

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// testDashboardRoutes builds a DashboardRoutes backed by a mockDashboardRepo,
// mirroring the direct-postgres-pool exception documented on DashboardRoutes
// itself: there is no gRPC client to stub, the service sits on a repository
// interface instead.
func testDashboardRoutes(repo DashboardRepository) *DashboardRoutes {
	return NewDashboardRoutes(NewDashboardService(repo))
}

func TestDashboardRoutes_ServiceName(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	if routes.ServiceName() != "dashboard" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "dashboard")
	}
}

// ============================================================================
// Router-level guard wiring: RequireAuthenticated on the user endpoints,
// RequireRole("admin") on the /defaults/{role} endpoints.
// ============================================================================

func TestDashboardRoutes_Guards(t *testing.T) {
	router := chi.NewRouter()
	testDashboardRoutes(newMockDashboardRepo()).RegisterRoutes(router, guardTestAuth)

	cases := []struct {
		name   string
		method string
		path   string
		auth   bool
		admin  bool
		want   int
	}{
		{"get layout without auth", http.MethodGet, "/api/v1/dashboard/layout", false, false, http.StatusUnauthorized},
		{"get layout with auth", http.MethodGet, "/api/v1/dashboard/layout", true, false, http.StatusOK},
		{"save layout without auth", http.MethodPut, "/api/v1/dashboard/layout", false, false, http.StatusUnauthorized},
		{"reset layout without auth", http.MethodDelete, "/api/v1/dashboard/layout", false, false, http.StatusUnauthorized},

		{"get defaults without auth", http.MethodGet, "/api/v1/dashboard/defaults/admin", false, false, http.StatusUnauthorized},
		{"get defaults authenticated but not admin", http.MethodGet, "/api/v1/dashboard/defaults/admin", true, false, http.StatusForbidden},
		{"get defaults as admin", http.MethodGet, "/api/v1/dashboard/defaults/admin", true, true, http.StatusNotFound},
		{"save defaults authenticated but not admin", http.MethodPut, "/api/v1/dashboard/defaults/admin", true, false, http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.auth {
				req = withAuth(req, uuid.New().String(), testTenantID)
			}
			if tc.admin {
				req = withRoles(req, "admin")
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assertStatus(t, rec, tc.want)
		})
	}
}

// ============================================================================
// HandleGetDashboard
// ============================================================================

func TestHandleGetDashboard_NoTenant(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/layout", nil)
	req = withUserID(req, "user-1")
	routes.HandleGetDashboard(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "missing tenant context")
}

func TestHandleGetDashboard_UserLayoutFound(t *testing.T) {
	repo := newMockDashboardRepo()
	repo.users["user-1"] = &models.UserDashboardLayout{
		ID:            "1",
		UserID:        "user-1",
		Layout:        json.RawMessage(`[{"i":"own"}]`),
		ActiveWidgets: json.RawMessage(`["own"]`),
	}
	routes := testDashboardRoutes(repo)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/layout", nil)
	req = withAuth(req, "user-1", testTenantID)
	routes.HandleGetDashboard(rec, req)
	assertStatus(t, rec, http.StatusOK)

	var got models.DashboardLayoutResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !got.IsCustom {
		t.Error("is_custom = false, want true for a stored user layout")
	}
	if string(got.ActiveWidgets) != `["own"]` {
		t.Errorf("active_widgets = %s, want [\"own\"]", got.ActiveWidgets)
	}
}

func TestHandleGetDashboard_FallsBackToRoleDefault(t *testing.T) {
	repo := newMockDashboardRepo()
	repo.setDefault(testTenantID, &models.DashboardDefault{
		Role:          "manager",
		Layout:        json.RawMessage(`[{"i":"role-default"}]`),
		ActiveWidgets: json.RawMessage(`["role-default"]`),
	})
	routes := testDashboardRoutes(repo)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/layout", nil)
	req = withAuth(req, "user-1", testTenantID)
	req = withRoles(req, "manager")
	routes.HandleGetDashboard(rec, req)
	assertStatus(t, rec, http.StatusOK)

	var got models.DashboardLayoutResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.IsCustom {
		t.Error("is_custom = true, want false for a role default fallback")
	}
	if string(got.ActiveWidgets) != `["role-default"]` {
		t.Errorf("active_widgets = %s, want [\"role-default\"]", got.ActiveWidgets)
	}
}

// TestHandleGetDashboard_EmptyTenant_HardcodedFallback is the state of every
// first customer: a freshly provisioned tenant with neither a personal layout
// nor a role default. The handler must not panic or 500, it must fall
// through to the hardcoded layout — see hardcodedDefaultLayout() in
// dashboard_service.go.
func TestHandleGetDashboard_EmptyTenant_HardcodedFallback(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/layout", nil)
	req = withAuth(req, "user-1", testTenantID)
	routes.HandleGetDashboard(rec, req)
	assertStatus(t, rec, http.StatusOK)

	var got models.DashboardLayoutResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.IsCustom {
		t.Error("is_custom = true, want false for the hardcoded fallback")
	}
	if len(got.Layout) == 0 || len(got.ActiveWidgets) == 0 {
		t.Error("hardcoded fallback must not be empty")
	}
}

func TestHandleGetDashboard_UserLayoutRepoError(t *testing.T) {
	repo := newMockDashboardRepo()
	repo.getUserErr = errors.New("connection reset")
	routes := testDashboardRoutes(repo)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/layout", nil)
	req = withAuth(req, "user-1", testTenantID)
	routes.HandleGetDashboard(rec, req)
	assertStatus(t, rec, http.StatusInternalServerError)
}

func TestHandleGetDashboard_RoleDefaultRepoError(t *testing.T) {
	repo := newMockDashboardRepo()
	repo.getDefaultErr = errors.New("connection reset")
	routes := testDashboardRoutes(repo)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/layout", nil)
	req = withAuth(req, "user-1", testTenantID)
	routes.HandleGetDashboard(rec, req)
	assertStatus(t, rec, http.StatusInternalServerError)
}

// ============================================================================
// HandleSaveDashboard
// ============================================================================

func validSaveDashboardBody() map[string]interface{} {
	return map[string]interface{}{
		"layout":         []map[string]interface{}{{"i": "widget-1", "x": 0, "y": 0, "w": 1, "h": 1}},
		"active_widgets": []string{"widget-1"},
	}
}

func TestHandleSaveDashboard_NoTenant(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dashboard/layout", jsonBody(t, validSaveDashboardBody()))
	req = withUserID(req, "user-1")
	routes.HandleSaveDashboard(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleSaveDashboard_InvalidJSON(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dashboard/layout", invalidJSON())
	req = withAuth(req, "user-1", testTenantID)
	routes.HandleSaveDashboard(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSaveDashboard_MissingLayout(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	body := validSaveDashboardBody()
	delete(body, "layout")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dashboard/layout", jsonBody(t, body))
	req = withAuth(req, "user-1", testTenantID)
	routes.HandleSaveDashboard(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "required")
}

func TestHandleSaveDashboard_MissingActiveWidgets(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	body := validSaveDashboardBody()
	delete(body, "active_widgets")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dashboard/layout", jsonBody(t, body))
	req = withAuth(req, "user-1", testTenantID)
	routes.HandleSaveDashboard(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "required")
}

// The handler's own `!json.Valid(req.Layout)` / `!json.Valid(req.ActiveWidgets)`
// checks (route_dashboard.go:102-105, :195-198) are defensive and not
// reachable through the HTTP layer: decodeAndValidate's json.NewDecoder only
// ever populates a json.RawMessage field with a syntactically valid JSON
// span in the first place, so a body that would fail json.Valid already
// fails decode earlier with "invalid request body" — covered above by
// TestHandleSaveDashboard_InvalidJSON / TestHandleSaveDefaults_InvalidJSON.

func TestHandleSaveDashboard_Success(t *testing.T) {
	repo := newMockDashboardRepo()
	routes := testDashboardRoutes(repo)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dashboard/layout", jsonBody(t, validSaveDashboardBody()))
	req = withAuth(req, "user-1", testTenantID)
	routes.HandleSaveDashboard(rec, req)
	assertStatus(t, rec, http.StatusOK)
	if repo.upsertUserCalls != 1 {
		t.Errorf("upsertUserCalls = %d, want 1", repo.upsertUserCalls)
	}
	stored, ok := repo.users["user-1"]
	if !ok {
		t.Fatal("layout was not persisted for user-1")
	}
	if string(stored.ActiveWidgets) != `["widget-1"]` {
		t.Errorf("stored active_widgets = %s, want [\"widget-1\"]", stored.ActiveWidgets)
	}
}

func TestHandleSaveDashboard_RepoError(t *testing.T) {
	repo := newMockDashboardRepo()
	repo.upsertUserErr = errors.New("connection reset")
	routes := testDashboardRoutes(repo)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dashboard/layout", jsonBody(t, validSaveDashboardBody()))
	req = withAuth(req, "user-1", testTenantID)
	routes.HandleSaveDashboard(rec, req)
	assertStatus(t, rec, http.StatusInternalServerError)
}

// ============================================================================
// HandleResetToDefaults
// ============================================================================

func TestHandleResetToDefaults_NoTenant(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/dashboard/layout", nil)
	req = withUserID(req, "user-1")
	routes.HandleResetToDefaults(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleResetToDefaults_Success(t *testing.T) {
	repo := newMockDashboardRepo()
	repo.users["user-1"] = &models.UserDashboardLayout{ID: "1", UserID: "user-1"}
	routes := testDashboardRoutes(repo)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/dashboard/layout", nil)
	req = withAuth(req, "user-1", testTenantID)
	routes.HandleResetToDefaults(rec, req)
	assertStatus(t, rec, http.StatusOK)
	if _, ok := repo.users["user-1"]; ok {
		t.Error("user layout still present after reset")
	}
}

// TestHandleResetToDefaults_NoOverrideIsIdempotent belongs to the same class
// of case as A10's "second run changes nothing": deleting an override that
// was never set is not an error, ResetToDefaults treats ErrDashboardNotFound
// as success (dashboard_service.go:117).
func TestHandleResetToDefaults_NoOverrideIsIdempotent(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/dashboard/layout", nil)
	req = withAuth(req, "user-1", testTenantID)
	routes.HandleResetToDefaults(rec, req)
	assertStatus(t, rec, http.StatusOK)
}

func TestHandleResetToDefaults_RepoError(t *testing.T) {
	repo := newMockDashboardRepo()
	repo.deleteUserErr = errors.New("connection reset")
	routes := testDashboardRoutes(repo)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/dashboard/layout", nil)
	req = withAuth(req, "user-1", testTenantID)
	routes.HandleResetToDefaults(rec, req)
	assertStatus(t, rec, http.StatusInternalServerError)
}

// ============================================================================
// HandleGetDefaults
// ============================================================================

func TestHandleGetDefaults_NoTenant(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/defaults/admin", nil)
	req = withUserID(req, "user-1")
	req = withChiURLParam(req, "role", "admin")
	routes.HandleGetDefaults(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetDefaults_InvalidRole(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/defaults/superuser", nil)
	req = withAuth(req, "user-1", testTenantID)
	req = withChiURLParam(req, "role", "superuser")
	routes.HandleGetDefaults(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid role")
}

func TestHandleGetDefaults_NotFound(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/defaults/admin", nil)
	req = withAuth(req, "user-1", testTenantID)
	req = withChiURLParam(req, "role", "admin")
	routes.HandleGetDefaults(rec, req)
	assertStatus(t, rec, http.StatusNotFound)
	assertErrorContains(t, rec, "admin")
}

func TestHandleGetDefaults_Found(t *testing.T) {
	repo := newMockDashboardRepo()
	repo.setDefault(testTenantID, &models.DashboardDefault{
		Role:          "manager",
		Layout:        json.RawMessage(`[{"i":"m"}]`),
		ActiveWidgets: json.RawMessage(`["m"]`),
	})
	routes := testDashboardRoutes(repo)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/defaults/manager", nil)
	req = withAuth(req, "user-1", testTenantID)
	req = withChiURLParam(req, "role", "manager")
	routes.HandleGetDefaults(rec, req)
	assertStatus(t, rec, http.StatusOK)
}

func TestHandleGetDefaults_RepoError(t *testing.T) {
	repo := newMockDashboardRepo()
	repo.getDefaultErr = errors.New("connection reset")
	routes := testDashboardRoutes(repo)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/defaults/admin", nil)
	req = withAuth(req, "user-1", testTenantID)
	req = withChiURLParam(req, "role", "admin")
	routes.HandleGetDefaults(rec, req)
	assertStatus(t, rec, http.StatusInternalServerError)
}

// ============================================================================
// HandleSaveDefaults
// ============================================================================

func validSaveDefaultsBody() map[string]interface{} {
	return map[string]interface{}{
		"layout":         []map[string]interface{}{{"i": "widget-1", "x": 0, "y": 0, "w": 1, "h": 1}},
		"active_widgets": []string{"widget-1"},
	}
}

func TestHandleSaveDefaults_NoTenant(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dashboard/defaults/admin", jsonBody(t, validSaveDefaultsBody()))
	req = withUserID(req, "user-1")
	req = withChiURLParam(req, "role", "admin")
	routes.HandleSaveDefaults(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleSaveDefaults_InvalidRole(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dashboard/defaults/superuser", jsonBody(t, validSaveDefaultsBody()))
	req = withAuth(req, "user-1", testTenantID)
	req = withChiURLParam(req, "role", "superuser")
	routes.HandleSaveDefaults(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid role")
}

func TestHandleSaveDefaults_InvalidJSON(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dashboard/defaults/admin", invalidJSON())
	req = withAuth(req, "user-1", testTenantID)
	req = withChiURLParam(req, "role", "admin")
	routes.HandleSaveDefaults(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSaveDefaults_MissingLayout(t *testing.T) {
	routes := testDashboardRoutes(newMockDashboardRepo())
	body := validSaveDefaultsBody()
	delete(body, "layout")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dashboard/defaults/admin", jsonBody(t, body))
	req = withAuth(req, "user-1", testTenantID)
	req = withChiURLParam(req, "role", "admin")
	routes.HandleSaveDefaults(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "required")
}

func TestHandleSaveDefaults_Success(t *testing.T) {
	repo := newMockDashboardRepo()
	routes := testDashboardRoutes(repo)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dashboard/defaults/admin", jsonBody(t, validSaveDefaultsBody()))
	req = withAuth(req, "user-1", testTenantID)
	req = withChiURLParam(req, "role", "admin")
	routes.HandleSaveDefaults(rec, req)
	assertStatus(t, rec, http.StatusOK)
	if repo.upsertDefaultCalls != 1 {
		t.Errorf("upsertDefaultCalls = %d, want 1", repo.upsertDefaultCalls)
	}
	stored, ok := repo.defaults[defaultKey(testTenantID, "admin")]
	if !ok {
		t.Fatal("default was not persisted for tenant/role")
	}
	if string(stored.ActiveWidgets) != `["widget-1"]` {
		t.Errorf("stored active_widgets = %s, want [\"widget-1\"]", stored.ActiveWidgets)
	}
}

func TestHandleSaveDefaults_RepoError(t *testing.T) {
	repo := newMockDashboardRepo()
	repo.upsertDefaultErr = errors.New("connection reset")
	routes := testDashboardRoutes(repo)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dashboard/defaults/admin", jsonBody(t, validSaveDefaultsBody()))
	req = withAuth(req, "user-1", testTenantID)
	req = withChiURLParam(req, "role", "admin")
	routes.HandleSaveDefaults(rec, req)
	assertStatus(t, rec, http.StatusInternalServerError)
}

// ============================================================================
// primaryRole / isValidRole helpers
// ============================================================================

func TestPrimaryRole(t *testing.T) {
	cases := []struct {
		name  string
		roles []string
		want  string
	}{
		{"admin wins over manager and member", []string{"member", "admin", "manager"}, "admin"},
		{"manager wins over member", []string{"member", "manager"}, "manager"},
		{"member is the default", []string{"member"}, "member"},
		{"empty defaults to member", nil, "member"},
		{"unknown role defaults to member", []string{"guest"}, "member"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := primaryRole(tc.roles); got != tc.want {
				t.Errorf("primaryRole(%v) = %q, want %q", tc.roles, got, tc.want)
			}
		})
	}
}

func TestIsValidRole(t *testing.T) {
	for _, r := range []string{"admin", "manager", "member"} {
		if !isValidRole(r) {
			t.Errorf("isValidRole(%q) = false, want true", r)
		}
	}
	for _, r := range []string{"superuser", "", "Admin"} {
		if isValidRole(r) {
			t.Errorf("isValidRole(%q) = true, want false", r)
		}
	}
}
