package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWorkRoutes_ServiceName(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	if routes.ServiceName() != "work" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "work")
	}
}

// --- Projects ---

func TestHandleCreateProject_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateProject)
}

func TestHandleCreateProject_NoUserID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects", jsonBody(t, map[string]interface{}{}))
	withAuthRequired(routes.HandleCreateProject)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "not authenticated")
}

func TestHandleCreateProject_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/projects", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleCreateProject(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateProject_MissingFields(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))

	tests := []struct {
		name      string
		body      map[string]interface{}
		wantField string
	}{
		{"empty", map[string]interface{}{}, "name"},
		{"no_key", map[string]interface{}{"name": "Test"}, "project_key"},
		{"no_name", map[string]interface{}{"project_key": "TST"}, "name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/v1/projects", jsonBody(t, tt.body))
			req = withUserID(req, "user-123")
			routes.HandleCreateProject(rec, req)
			assertStatus(t, rec, http.StatusBadRequest)
			assertValidationError(t, rec, tt.wantField)
		})
	}
}

func TestHandleGetProject_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetProject(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetProject_InvalidUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetProject(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListProjects_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	routes.HandleListProjects(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Tasks ---

func TestHandleCreateTask_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateTask)
}

func TestHandleCreateTask_NoUserID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks", jsonBody(t, map[string]interface{}{}))
	withAuthRequired(routes.HandleCreateTask)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateTask_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleCreateTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateTask_MissingFields(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks", jsonBody(t, map[string]interface{}{
		"project_id": "550e8400-e29b-41d4-a716-446655440000",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "title")
}

func TestHandleCreateTask_MissingProjectID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks", jsonBody(t, map[string]interface{}{
		"title": "My Task",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "project_id")
}

func TestHandleCreateTask_InvalidProjectIDUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks", jsonBody(t, map[string]interface{}{
		"title":      "My Task",
		"project_id": "not-a-uuid",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "project_id")
}

func TestHandleGetTask_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tasks/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetTask(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetTask_InvalidUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tasks/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleMoveTask_MissingStatusID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/move", jsonBody(t, map[string]interface{}{
		"sort_order": 1.0,
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleMoveTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "status_id")
}

func TestHandleMoveTask_InvalidStatusIDUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/move", jsonBody(t, map[string]interface{}{
		"status_id":  "not-a-uuid",
		"sort_order": 1.0,
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleMoveTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "status_id")
}

// --- Time Entries ---

func TestHandleDeleteTimeEntry_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/time-entries/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteTimeEntry(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
