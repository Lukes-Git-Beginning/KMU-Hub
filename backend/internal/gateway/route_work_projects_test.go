package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// This file covers internal/gateway/route_work_projects.go — project CRUD,
// project members, templates, project statuses, and per-user preferences.
// HandleCreateProject/HandleGetProject/HandleListProjects' baseline paths
// (ServiceUnavailable, NoUserID, InvalidJSON, MissingFields, InvalidUUID)
// already live in route_work_test.go; nothing here duplicates those.

// --- Projects: additional paths beyond route_work_test.go ---

func TestHandleCreateProject_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects", jsonBody(t, map[string]interface{}{
		"name":        "Website Relaunch",
		"project_key": "WEB",
		"description": "Q3 relaunch",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateProject(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetProject_ValidUUIDReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetProject(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListProjects_FilterCombinations exercises templates_only in both
// positions (true/false) alongside include_archived, search, and pagination —
// the query-parsing branches are the only logic in this handler. None of
// these can be observed reaching the proto request without a fake
// WorkServiceClient (no bufconn stub exists for this package, same boundary
// noted in prior gateway coverage units) — each case is proven to parse past
// every filter branch by reaching the RPC layer (503), not panicking.
func TestHandleListProjects_FilterCombinations(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"empty", ""},
		{"templates_only_true", "?templates_only=true"},
		{"templates_only_false", "?templates_only=false"},
		{"include_archived_true", "?include_archived=true"},
		{"search_only", "?search=relaunch"},
		{"pagination", "?page=2&page_size=10"},
		{"all_combined", "?templates_only=true&include_archived=true&search=x&page=1&page_size=5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routes := NewWorkRoutes(registryWithService("work"))
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/v1/projects"+tt.query, nil)
			routes.HandleListProjects(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

func TestHandleUpdateProject_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/projects/123", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateProject(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateProject_InvalidUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/projects/bad", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateProject(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateProject_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateProject(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateProject_EmptyName(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"name": "",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateProject(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "name")
}

func TestHandleUpdateProject_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"name":        "Renamed",
		"description": "updated",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateProject(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleArchiveProject_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/123/archive", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleArchiveProject(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleArchiveProject_InvalidUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/bad/archive", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleArchiveProject(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleArchiveProject_ValidUUIDReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/archive", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleArchiveProject(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteProject: newest handler in the file, no prior coverage ---

func TestHandleDeleteProject_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/projects/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteProject(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteProject_InvalidUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/projects/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteProject(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// TestHandleDeleteProject_UnknownIDReachesRPC covers "unknown project id":
// the gateway cannot distinguish an unknown-but-well-formed id from a real
// one without a fake WorkServiceClient (same boundary as every other
// gateway coverage unit — no bufconn stub exists in this package). A
// structurally valid, non-existent id is proven to pass every gateway-side
// check and reach the RPC layer (503); NotFound mapping itself is the
// service's responsibility and grpcStatusToHTTP's NotFound->404 branch is
// covered generically in helpers_test.go.
func TestHandleDeleteProject_UnknownIDReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/projects/00000000-0000-0000-0000-000000000000", nil)
	req = withChiURLParam(req, "id", "00000000-0000-0000-0000-000000000000")
	routes.HandleDeleteProject(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Project Members ---

func TestHandleAddProjectMember_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/123/members", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddProjectMember(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleAddProjectMember_InvalidProjectUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/bad/members", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleAddProjectMember(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleAddProjectMember_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/members", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddProjectMember(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleAddProjectMember_MissingFields(t *testing.T) {
	tests := []struct {
		name      string
		body      map[string]interface{}
		wantField string
	}{
		{"missing_user_id", map[string]interface{}{"role": "member"}, "user_id"},
		{"missing_role", map[string]interface{}{"user_id": "550e8400-e29b-41d4-a716-446655440000"}, "role"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routes := NewWorkRoutes(registryWithService("work"))
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/members", jsonBody(t, tt.body))
			req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
			routes.HandleAddProjectMember(rec, req)
			assertStatus(t, rec, http.StatusBadRequest)
			assertValidationError(t, rec, tt.wantField)
		})
	}
}

func TestHandleAddProjectMember_InvalidUserIDUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/members", jsonBody(t, map[string]interface{}{
		"user_id": "not-a-uuid",
		"role":    "member",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddProjectMember(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "user_id")
}

func TestHandleAddProjectMember_InvalidRole(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/members", jsonBody(t, map[string]interface{}{
		"user_id": "550e8400-e29b-41d4-a716-446655440000",
		"role":    "superadmin",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddProjectMember(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "role")
}

func TestHandleAddProjectMember_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/members", jsonBody(t, map[string]interface{}{
		"user_id": "550e8400-e29b-41d4-a716-446655440000",
		"role":    "admin",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddProjectMember(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// HandleRemoveProjectMember reads both ids via chi.URLParam directly, with
// no validateUUIDParam call — see the "missing id validation" finding in
// JOURNAL.md. There is no 400 path to exercise; only the reach-RPC path is
// observable at this layer.
func TestHandleRemoveProjectMember_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/projects/123/members/456", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withChiURLParam(req, "userId", "660e8400-e29b-41d4-a716-446655440000")
	routes.HandleRemoveProjectMember(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRemoveProjectMember_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/members/660e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withChiURLParam(req, "userId", "660e8400-e29b-41d4-a716-446655440000")
	routes.HandleRemoveProjectMember(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListProjectMembers_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/123/members", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListProjectMembers(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListProjectMembers_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/members", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListProjectMembers(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateProjectMemberRole_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/projects/123/members/456", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withChiURLParam(req, "userId", "660e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateProjectMemberRole(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateProjectMemberRole_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/members/660e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withChiURLParam(req, "userId", "660e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateProjectMemberRole(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateProjectMemberRole_MissingRole(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/members/660e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withChiURLParam(req, "userId", "660e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateProjectMemberRole(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "role")
}

func TestHandleUpdateProjectMemberRole_InvalidRole(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/members/660e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"role": "root",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withChiURLParam(req, "userId", "660e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateProjectMemberRole(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "role")
}

func TestHandleUpdateProjectMemberRole_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/members/660e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"role": "viewer",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withChiURLParam(req, "userId", "660e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateProjectMemberRole(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Templates ---

func TestHandleSaveProjectAsTemplate_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/123/template", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSaveProjectAsTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSaveProjectAsTemplate_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/template", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSaveProjectAsTemplate(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSaveProjectAsTemplate_MissingTemplateName(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/template", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSaveProjectAsTemplate(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "template_name")
}

func TestHandleSaveProjectAsTemplate_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/template", jsonBody(t, map[string]interface{}{
		"template_name": "Standard Rollout",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSaveProjectAsTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateProjectFromTemplate_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/from-template", jsonBody(t, map[string]interface{}{}))
	routes.HandleCreateProjectFromTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateProjectFromTemplate_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/from-template", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateProjectFromTemplate(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateProjectFromTemplate_MissingFields(t *testing.T) {
	tests := []struct {
		name      string
		body      map[string]interface{}
		wantField string
	}{
		{"missing_template_id", map[string]interface{}{"name": "N", "project_key": "PK"}, "template_id"},
		{"missing_name", map[string]interface{}{"template_id": "550e8400-e29b-41d4-a716-446655440000", "project_key": "PK"}, "name"},
		{"missing_project_key", map[string]interface{}{"template_id": "550e8400-e29b-41d4-a716-446655440000", "name": "N"}, "project_key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routes := NewWorkRoutes(registryWithService("work"))
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/v1/projects/from-template", jsonBody(t, tt.body))
			req = withAuth(req, "user-123", testTenantID)
			routes.HandleCreateProjectFromTemplate(rec, req)
			assertStatus(t, rec, http.StatusBadRequest)
			assertValidationError(t, rec, tt.wantField)
		})
	}
}

func TestHandleCreateProjectFromTemplate_InvalidTemplateIDUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/from-template", jsonBody(t, map[string]interface{}{
		"template_id": "not-a-uuid",
		"name":        "N",
		"project_key": "PK",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateProjectFromTemplate(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "template_id")
}

func TestHandleCreateProjectFromTemplate_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/from-template", jsonBody(t, map[string]interface{}{
		"template_id": "550e8400-e29b-41d4-a716-446655440000",
		"name":        "New Project",
		"project_key": "NP",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateProjectFromTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Status Handlers ---

func TestHandleCreateProjectStatus_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/123/statuses", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateProjectStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateProjectStatus_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/statuses", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateProjectStatus(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateProjectStatus_MissingName(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/statuses", jsonBody(t, map[string]interface{}{
		"is_default": true,
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateProjectStatus(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateProjectStatus_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/statuses", jsonBody(t, map[string]interface{}{
		"name":       "In Review",
		"color":      "#00ff00",
		"is_default": false,
		"is_closed":  false,
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateProjectStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateProjectStatus_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/project-statuses/123", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateProjectStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateProjectStatus_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/project-statuses/550e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateProjectStatus(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateProjectStatus_EmptyName(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/project-statuses/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"name": "",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateProjectStatus(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "name")
}

func TestHandleUpdateProjectStatus_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/project-statuses/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"name":       "Done",
		"color":      "#0000ff",
		"is_default": false,
		"is_closed":  true,
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateProjectStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteProjectStatus_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/project-statuses/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteProjectStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteProjectStatus_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/project-statuses/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteProjectStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleReorderProjectStatuses_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/123/statuses/reorder", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleReorderProjectStatuses(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleReorderProjectStatuses_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/statuses/reorder", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleReorderProjectStatuses(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleReorderProjectStatuses_EmptyIDs(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/statuses/reorder", jsonBody(t, map[string]interface{}{
		"status_ids": []string{},
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleReorderProjectStatuses(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "status_ids")
}

func TestHandleReorderProjectStatuses_InvalidElementUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/statuses/reorder", jsonBody(t, map[string]interface{}{
		"status_ids": []string{"not-a-uuid"},
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleReorderProjectStatuses(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "status_ids[0]")
}

func TestHandleReorderProjectStatuses_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/statuses/reorder", jsonBody(t, map[string]interface{}{
		"status_ids": []string{
			"550e8400-e29b-41d4-a716-446655440000",
			"660e8400-e29b-41d4-a716-446655440000",
		},
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleReorderProjectStatuses(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListProjectStatuses_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/123/statuses", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListProjectStatuses(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListProjectStatuses_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/statuses", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListProjectStatuses(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Preference Handlers ---

func TestHandleGetUserProjectPreference_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/123/preferences", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetUserProjectPreference(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetUserProjectPreference_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/preferences", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetUserProjectPreference(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSetUserProjectPreference_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/projects/123/preferences", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSetUserProjectPreference(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSetUserProjectPreference_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/preferences", invalidJSON())
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSetUserProjectPreference(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// TestHandleSetUserProjectPreference_EmptyBodyReachesRPC covers that every
// field on setPreferenceRequest is optional (no `validate` tags at all) —
// an empty body is a legal "clear all preferences" request, not a 400.
func TestHandleSetUserProjectPreference_EmptyBodyReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/preferences", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSetUserProjectPreference(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSetUserProjectPreference_AllFieldsReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/preferences", jsonBody(t, map[string]interface{}{
		"view_type":      "board",
		"list_group_by":  "status",
		"list_sort_by":   "due_date",
		"list_sort_desc": true,
	}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSetUserProjectPreference(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
