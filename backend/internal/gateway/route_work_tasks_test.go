package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// This file covers internal/gateway/route_work_tasks.go — the task,
// dependency, comment, entity-link, activity, file, and custom-field
// handlers. Project and time-entry handlers, plus the subset of task
// handlers already exercised, live in route_work_test.go; nothing here
// duplicates those.

// --- Tasks: HandleCreateTask (additional paths beyond route_work_test.go) ---

func TestHandleCreateTask_NoTenant(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks", jsonBody(t, map[string]interface{}{
		"title":      "My Task",
		"project_id": "550e8400-e29b-41d4-a716-446655440000",
	}))
	req = withUserID(req, "user-123") // no tenant
	routes.HandleCreateTask(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "missing or invalid tenant")
}

func TestHandleCreateTask_InvalidStartDate(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks", jsonBody(t, map[string]interface{}{
		"title":      "My Task",
		"project_id": "550e8400-e29b-41d4-a716-446655440000",
		"start_date": "not-a-date",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid start_date format")
}

func TestHandleCreateTask_InvalidDueDate(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks", jsonBody(t, map[string]interface{}{
		"title":      "My Task",
		"project_id": "550e8400-e29b-41d4-a716-446655440000",
		"due_date":   "not-a-date",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid due_date format")
}

func TestHandleCreateTask_InvalidPriority(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks", jsonBody(t, map[string]interface{}{
		"title":      "My Task",
		"project_id": "550e8400-e29b-41d4-a716-446655440000",
		"priority":   "urgentest",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "priority")
}

func TestHandleCreateTask_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks", jsonBody(t, map[string]interface{}{
		"title":       "My Task",
		"project_id":  "550e8400-e29b-41d4-a716-446655440000",
		"priority":    "high",
		"start_date":  "2026-01-01T00:00:00Z",
		"due_date":    "2026-02-01T00:00:00Z",
		"description": "details",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTask(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Tasks: HandleListTasks ---

func TestHandleListTasks_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListTasks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListTasks_NoTenant(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	routes.HandleListTasks(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "missing or invalid tenant")
}

func TestHandleListTasks_InvalidDueFrom(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tasks?due_from=not-a-date", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListTasks(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid due_from format")
}

func TestHandleListTasks_InvalidDueTo(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tasks?due_to=not-a-date", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListTasks(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid due_to format")
}

// TestHandleListTasks_ScopeOwnWithoutUserRejected proves the scope check runs
// before the request reaches the RPC layer: a caller narrowed to "own" (on
// either permission key of the route's RequirePermissionAny guard) without a
// user id in the token is refused outright, not handed the unfiltered list.
func TestHandleListTasks_ScopeOwnWithoutUserRejected(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	req = withTenantID(req, testTenantID)
	req = withScopes(req, map[string]string{"work:task:read": "own"})
	routes.HandleListTasks(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "missing user in token")
}

// TestHandleListTasks_ScopeOwnValidUserReachesRPC proves a caller narrowed to
// "own" with a valid user id still reaches the RPC layer (their own
// assignee_id filter is not a rejection).
func TestHandleListTasks_ScopeOwnValidUserReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tasks?assignee_id=550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withScopes(req, map[string]string{"tasks:read": "own"})
	routes.HandleListTasks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListTasks_FilterCombinations exercises the filter-parsing branches
// (project_id, assignee_id, status_id, priority, parent_task_id, label_ids,
// due_from/due_to, sort, include_completed) including the empty combination.
// None of these can be observed reaching the proto request without a fake
// WorkServiceClient (none exists for this package, same boundary noted in
// prior gateway coverage units) — each case is proven to parse past every
// filter branch by reaching the RPC layer (503), not panicking or 400ing.
func TestHandleListTasks_FilterCombinations(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"empty", ""},
		{"project_id_only", "?project_id=550e8400-e29b-41d4-a716-446655440000"},
		{"assignee_id_only", "?assignee_id=550e8400-e29b-41d4-a716-446655440000"},
		{"status_id_only", "?status_id=550e8400-e29b-41d4-a716-446655440000"},
		{"priority_only", "?priority=high"},
		{"parent_task_id_only", "?parent_task_id=550e8400-e29b-41d4-a716-446655440000"},
		{"label_ids_only", "?label_ids=aaa,bbb,ccc"},
		{"label_ids_with_blanks", "?label_ids=aaa,,%20bbb%20,"},
		{"due_range", "?due_from=2026-01-01T00:00:00Z&due_to=2026-02-01T00:00:00Z"},
		{"search_and_sort", "?search=foo&sort_by=due_date&sort_desc=true"},
		{"include_completed", "?include_completed=true"},
		{"all_filters_combined", "?project_id=550e8400-e29b-41d4-a716-446655440000" +
			"&assignee_id=550e8400-e29b-41d4-a716-446655440000" +
			"&status_id=550e8400-e29b-41d4-a716-446655440000" +
			"&priority=urgent&parent_task_id=550e8400-e29b-41d4-a716-446655440000" +
			"&label_ids=aaa,bbb&due_from=2026-01-01T00:00:00Z&due_to=2026-02-01T00:00:00Z" +
			"&search=foo&sort_by=title&sort_desc=false&include_completed=true&page=2&page_size=50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routes := NewWorkRoutes(registryWithService("work"))
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/v1/tasks"+tt.query, nil)
			req = withTenantID(req, testTenantID)
			routes.HandleListTasks(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// --- Tasks: HandleUpdateTask ---

func TestHandleUpdateTask_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdateTask)
}

func TestHandleUpdateTask_InvalidUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/bad", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateTask_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateTask_EmptyTitle(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"title": "",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "title")
}

func TestHandleUpdateTask_InvalidPriority(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"priority": "urgentest",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "priority")
}

func TestHandleUpdateTask_InvalidStatusIDUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"status_id": "not-a-uuid",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "status_id")
}

func TestHandleUpdateTask_InvalidAssigneeIDUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"assignee_id": "not-a-uuid",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "assignee_id")
}

func TestHandleUpdateTask_InvalidStartDate(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"start_date": "not-a-date",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid start_date format")
}

func TestHandleUpdateTask_InvalidDueDate(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"due_date": "not-a-date",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid due_date format")
}

func TestHandleUpdateTask_EmptyBodyReachesRPC(t *testing.T) {
	// An empty JSON object is valid — every field is *optional. Proves the
	// no-op update path (no field set) still reaches the RPC call.
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTask(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Tasks: HandleDeleteTask ---
//
// HandleDeleteTask reads the id straight off chi.URLParam without running it
// through validateUUIDParam first (unlike HandleGetTask/HandleUpdateTask in
// this same file). An invalid id therefore reaches the RPC layer instead of
// being rejected locally with 400 — a stylistic inconsistency, not a security
// gap (the service still validates), documented here rather than fixed: see
// journal entry for this unit.

func TestHandleDeleteTask_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteTask(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteTask_InvalidIDReachesRPCNotLocalValidation(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/tasks/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteTask(rec, req)
	// Documents current behaviour: no local UUID check, so this reaches the
	// (unreachable) RPC layer and comes back 503, not 400.
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Tasks: HandleMoveTask ---

func TestHandleMoveTask_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleMoveTask)
}

func TestHandleMoveTask_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/move", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleMoveTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleMoveTask_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/move", jsonBody(t, map[string]interface{}{
		"status_id":  "660e8400-e29b-41d4-a716-446655440000",
		"sort_order": 3.5,
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleMoveTask(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Tasks: HandleListSubtasks ---

func TestHandleListSubtasks_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/subtasks", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListSubtasks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListSubtasks_RecursiveFlag(t *testing.T) {
	for _, recursive := range []string{"true", "false", ""} {
		t.Run("recursive="+recursive, func(t *testing.T) {
			routes := NewWorkRoutes(registryWithService("work"))
			rec := httptest.NewRecorder()
			url := "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/subtasks"
			if recursive != "" {
				url += "?recursive=" + recursive
			}
			req := httptest.NewRequest("GET", url, nil)
			req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
			routes.HandleListSubtasks(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// --- Dependency Handlers ---

func TestHandleCreateTaskDependency_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateTaskDependency)
}

func TestHandleCreateTaskDependency_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/dependencies", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateTaskDependency(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateTaskDependency_MissingTargetTaskID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/dependencies", jsonBody(t, map[string]interface{}{
		"dependency_type": "blocks",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateTaskDependency(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "target_task_id")
}

func TestHandleCreateTaskDependency_InvalidTargetTaskIDUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/dependencies", jsonBody(t, map[string]interface{}{
		"target_task_id": "not-a-uuid",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateTaskDependency(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "target_task_id")
}

func TestHandleCreateTaskDependency_InvalidDependencyType(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/dependencies", jsonBody(t, map[string]interface{}{
		"target_task_id":  "660e8400-e29b-41d4-a716-446655440000",
		"dependency_type": "invalidates",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateTaskDependency(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "dependency_type")
}

func TestHandleDeleteTaskDependency_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleDeleteTaskDependency)
}

func TestHandleListTaskDependencies_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListTaskDependencies)
}

// --- Comment Handlers ---

func TestHandleCreateTaskComment_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateTaskComment)
}

func TestHandleCreateTaskComment_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/comments", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateTaskComment(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateTaskComment_MissingContent(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/comments", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateTaskComment(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "content")
}

func TestHandleCreateTaskComment_InvalidQuotedCommentIDUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/comments", jsonBody(t, map[string]interface{}{
		"content":           "hi",
		"quoted_comment_id": "not-a-uuid",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateTaskComment(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "quoted_comment_id")
}

func TestHandleUpdateTaskComment_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdateTaskComment)
}

func TestHandleUpdateTaskComment_MissingContent(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/task-comments/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTaskComment(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "content")
}

func TestHandleDeleteTaskComment_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/task-comments/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteTaskComment(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteTaskComment_IsAdminForwarded(t *testing.T) {
	// IsAdmin comes from the auth context, not the request body — proves the
	// admin path parses and reaches the RPC layer same as the non-admin path.
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/task-comments/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withRoles(req, "admin")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteTaskComment(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListTaskComments_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListTaskComments)
}

// --- Entity Link Handlers ---

func TestHandleLinkEntityToTask_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleLinkEntityToTask)
}

func TestHandleLinkEntityToTask_MissingEntityType(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/links", jsonBody(t, map[string]interface{}{
		"entity_id": "660e8400-e29b-41d4-a716-446655440000",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleLinkEntityToTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "entity_type")
}

func TestHandleLinkEntityToTask_MissingEntityID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/links", jsonBody(t, map[string]interface{}{
		"entity_type": "contact",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleLinkEntityToTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "entity_id")
}

func TestHandleLinkEntityToTask_InvalidEntityIDUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/links", jsonBody(t, map[string]interface{}{
		"entity_type": "contact",
		"entity_id":   "not-a-uuid",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleLinkEntityToTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "entity_id")
}

func TestHandleUnlinkEntityFromTask_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUnlinkEntityFromTask)
}

func TestHandleListTaskEntityLinks_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListTaskEntityLinks)
}

func TestHandleListEntityTasks_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/entity-tasks?entity_type=contact&entity_id=660e8400-e29b-41d4-a716-446655440000", nil)
	routes.HandleListEntityTasks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListEntityTasks_MissingParams(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"both_missing", ""},
		{"missing_entity_type", "?entity_id=660e8400-e29b-41d4-a716-446655440000"},
		{"missing_entity_id", "?entity_type=contact"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routes := NewWorkRoutes(registryWithService("work"))
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/v1/entity-tasks"+tt.query, nil)
			routes.HandleListEntityTasks(rec, req)
			assertStatus(t, rec, http.StatusBadRequest)
			assertErrorContains(t, rec, "entity_type and entity_id are required")
		})
	}
}

func TestHandleListEntityTasks_ValidParamsReachRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/entity-tasks?entity_type=contact&entity_id=660e8400-e29b-41d4-a716-446655440000&page=2&page_size=10", nil)
	routes.HandleListEntityTasks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Activity Handler ---

func TestHandleListTaskActivities_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListTaskActivities)
}

// --- File Handlers ---

func TestHandleAttachFileToTask_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleAttachFileToTask)
}

func TestHandleAttachFileToTask_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/files", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAttachFileToTask(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleAttachFileToTask_MissingFields(t *testing.T) {
	tests := []struct {
		name      string
		body      map[string]interface{}
		wantField string
	}{
		{"missing_filename", map[string]interface{}{"mime_type": "application/pdf", "file_size": 10, "storage_key": "k"}, "filename"},
		{"missing_mime_type", map[string]interface{}{"filename": "a.pdf", "file_size": 10, "storage_key": "k"}, "mime_type"},
		{"missing_storage_key", map[string]interface{}{"filename": "a.pdf", "mime_type": "application/pdf", "file_size": 10}, "storage_key"},
		{"zero_file_size", map[string]interface{}{"filename": "a.pdf", "mime_type": "application/pdf", "file_size": 0, "storage_key": "k"}, "file_size"},
		{"negative_file_size", map[string]interface{}{"filename": "a.pdf", "mime_type": "application/pdf", "file_size": -5, "storage_key": "k"}, "file_size"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routes := NewWorkRoutes(registryWithService("work"))
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/files", jsonBody(t, tt.body))
			req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
			routes.HandleAttachFileToTask(rec, req)
			assertStatus(t, rec, http.StatusBadRequest)
			assertValidationError(t, rec, tt.wantField)
		})
	}
}

func TestHandleRemoveTaskFile_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleRemoveTaskFile)
}

func TestHandleListTaskFiles_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListTaskFiles)
}

// --- Custom Fields Handlers ---

func TestHandleSetTaskCustomFieldValues_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleSetTaskCustomFieldValues)
}

func TestHandleSetTaskCustomFieldValues_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/custom-fields", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSetTaskCustomFieldValues(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleSetTaskCustomFieldValues_EmptyValues(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/custom-fields", jsonBody(t, map[string]interface{}{
		"values": []interface{}{},
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSetTaskCustomFieldValues(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "values")
}

func TestHandleSetTaskCustomFieldValues_ValidReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000/custom-fields", jsonBody(t, map[string]interface{}{
		"values": []interface{}{
			map[string]interface{}{"field_id": "f1", "value": "v1"},
		},
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSetTaskCustomFieldValues(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetTaskCustomFieldValues_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetTaskCustomFieldValues)
}

// --- Search Handler ---

func TestHandleSearchTasks_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tasks/search?q=foo", nil)
	routes.HandleSearchTasks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSearchTasks_MissingQuery(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tasks/search", nil)
	routes.HandleSearchTasks(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "query parameter 'q' is required")
}

func TestHandleSearchTasks_EmptyQueryRejected(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tasks/search?q=", nil)
	routes.HandleSearchTasks(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "query parameter 'q' is required")
}

func TestHandleSearchTasks_ValidQueryReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tasks/search?q=foo&project_ids=a&project_ids=b&assignee_ids=c&include_completed=true&page=2&page_size=5", nil)
	routes.HandleSearchTasks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
