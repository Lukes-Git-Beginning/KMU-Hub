package gateway

// route_work_labels_test.go covers the 12 functions in route_work_labels.go
// (5 label handlers, 1 task-label handler, 5 custom-field-definition handlers,
// plus the labelIDsFromQuery helper), which had no dedicated test file before
// this unit.
//
// Both /api/v1/work/labels and /api/v1/work/custom-fields sit on a plain
// middleware.RequirePermission("work_labels"|"work_custom_fields", action) —
// unlike most of route_work.go, these two groups were deliberately left off
// the RequirePermissionAny catalogue rollout (see the "no catalogue key" note
// in route_work.go), so the additive-wiring tests in
// route_capability_guard_test.go don't cover them. TestWorkLabelRoutes_Guards
// and TestWorkCustomFieldRoutes_Guards below fill that gap: single-key
// guard, not additive.
//
// There is no bufconn stub for WorkServiceClient in this package (same
// boundary as every other gateway coverage unit this run), so the
// *_ReachesRPC tests only prove a handler reaches the RPC layer with a valid
// request; registryWithService dials "localhost:0", so the RPC itself always
// fails with codes.Unavailable -> 503.
//
// Bug found while researching "Loeschverhalten benutzter Labels" (done_when):
// task_custom_field_values.field_id still has its original FK from migration
// 000026 pointing at custom_field_definitions(id) — the CRM-only table whose
// entity_type CHECK constraint doesn't even list 'task'. work_custom_field_
// definitions (migration 000146) is a distinct table with its own id space.
// Confirmed live against the local DB: inserting a task_custom_field_values
// row with a real work_custom_field_definitions.id raises
// foreign_key_violation on task_custom_field_values_field_id_fkey every
// time. HandleSetTaskCustomFieldValues (route_work_tasks.go) and
// HandleGetTaskCustomFieldValues's underlying repository query
// (work/task/postgres_repository.go:613, joining the same wrong table) are
// therefore both unusable end-to-end. Filed as
// fix-work-task-custom-field-values-wrong-fk at the end of BACKLOG.yml
// instead of building a test around it (see BLOCK B header: "wer beim
// Testen einen Bug findet, legt eine Fix-Unit an, statt den Test um ihn
// herum zu bauen").
//
// Label deletion, by contrast, is not a bug: work_labels(id) -> task_labels
// carries ON DELETE CASCADE (migration 000145), matching the "also cascades
// to task_labels" doc comment on Repository.Delete. Proven against the real
// schema in internal/work/label/postgres_repository_db_test.go
// (TestLabelDelete_CascadesTaskLabels), not re-tested here since this file
// has no DB access.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

const testLabelID = "550e8400-e29b-41d4-a716-446655440001"
const testFieldDefID = "550e8400-e29b-41d4-a716-446655440002"

// --- HandleListLabels ---

func TestHandleListLabels_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/work/labels", nil)
	routes.HandleListLabels(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListLabels_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/work/labels", nil)
	routes.HandleListLabels(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleCreateLabel ---

func TestHandleCreateLabel_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateLabel)
}

func TestHandleCreateLabel_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/work/labels", invalidJSON())
	routes.HandleCreateLabel(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateLabel_MissingName(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/work/labels", jsonBody(t, map[string]interface{}{
		"color": "#6b7280",
	}))
	routes.HandleCreateLabel(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateLabel_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/work/labels", jsonBody(t, map[string]interface{}{
		"name":  "Bug",
		"color": "#ef4444",
	}))
	routes.HandleCreateLabel(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetLabel ---

func TestHandleGetLabel_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/work/labels/"+testLabelID, nil)
	req = withChiURLParam(req, "id", testLabelID)
	routes.HandleGetLabel(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetLabel_InvalidUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/work/labels/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetLabel(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetLabel_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/work/labels/"+testLabelID, nil)
	req = withChiURLParam(req, "id", testLabelID)
	routes.HandleGetLabel(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateLabel ---
//
// Unlike most other id params in this package, HandleUpdateLabel reads the id
// straight off chi.URLParam without validateUUIDParam (route_work_labels.go:97)
// — so an empty id is the only client-side-rejected case, not a malformed one.

func TestHandleUpdateLabel_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/work/labels/"+testLabelID, jsonBody(t, map[string]interface{}{
		"name": "Renamed",
	}))
	req = withChiURLParam(req, "id", testLabelID)
	routes.HandleUpdateLabel(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateLabel_MissingID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/work/labels/", jsonBody(t, map[string]interface{}{
		"name": "Renamed",
	}))
	routes.HandleUpdateLabel(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "missing label id")
}

func TestHandleUpdateLabel_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/work/labels/"+testLabelID, invalidJSON())
	req = withChiURLParam(req, "id", testLabelID)
	routes.HandleUpdateLabel(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateLabel_MissingName(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/work/labels/"+testLabelID, jsonBody(t, map[string]interface{}{
		"color": "#6b7280",
	}))
	req = withChiURLParam(req, "id", testLabelID)
	routes.HandleUpdateLabel(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleUpdateLabel_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/work/labels/"+testLabelID, jsonBody(t, map[string]interface{}{
		"name": "Renamed",
	}))
	req = withChiURLParam(req, "id", testLabelID)
	routes.HandleUpdateLabel(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteLabel ---

func TestHandleDeleteLabel_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/work/labels/"+testLabelID, nil)
	req = withChiURLParam(req, "id", testLabelID)
	routes.HandleDeleteLabel(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteLabel_MissingID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/work/labels/", nil)
	routes.HandleDeleteLabel(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "missing label id")
}

func TestHandleDeleteLabel_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/work/labels/"+testLabelID, nil)
	req = withChiURLParam(req, "id", testLabelID)
	routes.HandleDeleteLabel(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleSetTaskLabels (registered under /api/v1/tasks/{id}/labels) ---

func TestHandleSetTaskLabels_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/"+testQuoteID+"/labels", jsonBody(t, map[string]interface{}{
		"label_ids": []string{testLabelID},
	}))
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleSetTaskLabels(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSetTaskLabels_InvalidUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/bad/labels", jsonBody(t, map[string]interface{}{
		"label_ids": []string{testLabelID},
	}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleSetTaskLabels(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleSetTaskLabels_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/"+testQuoteID+"/labels", invalidJSON())
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleSetTaskLabels(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSetTaskLabels_EmptyClearsLabels_ReachesRPC(t *testing.T) {
	// label_ids has no `validate` tag, so an empty/absent list is a valid
	// "clear all labels" request, not a validation error.
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/"+testQuoteID+"/labels", jsonBody(t, map[string]interface{}{
		"label_ids": []string{},
	}))
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleSetTaskLabels(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSetTaskLabels_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/tasks/"+testQuoteID+"/labels", jsonBody(t, map[string]interface{}{
		"label_ids": []string{testLabelID},
	}))
	req = withChiURLParam(req, "id", testQuoteID)
	routes.HandleSetTaskLabels(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListCustomFieldDefinitions ---

func TestHandleListCustomFieldDefinitions_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/work/custom-fields", nil)
	routes.HandleListCustomFieldDefinitions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListCustomFieldDefinitions_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/work/custom-fields", nil)
	routes.HandleListCustomFieldDefinitions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleCreateCustomFieldDefinition ---

func TestHandleCreateCustomFieldDefinition_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateCustomFieldDefinition)
}

func TestHandleCreateCustomFieldDefinition_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/work/custom-fields", invalidJSON())
	routes.HandleCreateCustomFieldDefinition(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateCustomFieldDefinition_MissingFields(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))

	tests := []struct {
		name      string
		body      map[string]interface{}
		wantField string
	}{
		{"no name", map[string]interface{}{"field_type": "text"}, "name"},
		{"no field_type", map[string]interface{}{"name": "Priority"}, "field_type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/v1/work/custom-fields", jsonBody(t, tt.body))
			routes.HandleCreateCustomFieldDefinition(rec, req)
			assertValidationError(t, rec, tt.wantField)
		})
	}
}

func TestHandleCreateCustomFieldDefinition_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/work/custom-fields", jsonBody(t, map[string]interface{}{
		"name":       "Priority",
		"field_type": "select",
		"options":    []string{"High", "Low"},
	}))
	routes.HandleCreateCustomFieldDefinition(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetCustomFieldDefinition ---

func TestHandleGetCustomFieldDefinition_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/work/custom-fields/"+testFieldDefID, nil)
	req = withChiURLParam(req, "id", testFieldDefID)
	routes.HandleGetCustomFieldDefinition(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetCustomFieldDefinition_InvalidUUID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/work/custom-fields/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetCustomFieldDefinition(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetCustomFieldDefinition_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/work/custom-fields/"+testFieldDefID, nil)
	req = withChiURLParam(req, "id", testFieldDefID)
	routes.HandleGetCustomFieldDefinition(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateCustomFieldDefinition ---
//
// Same as HandleUpdateLabel: the id comes straight from chi.URLParam, not
// validateUUIDParam (route_work_labels.go:265), so only a missing id is
// rejected client-side.

func TestHandleUpdateCustomFieldDefinition_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/work/custom-fields/"+testFieldDefID, jsonBody(t, map[string]interface{}{
		"name":       "Priority",
		"field_type": "select",
	}))
	req = withChiURLParam(req, "id", testFieldDefID)
	routes.HandleUpdateCustomFieldDefinition(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateCustomFieldDefinition_MissingID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/work/custom-fields/", jsonBody(t, map[string]interface{}{
		"name":       "Priority",
		"field_type": "select",
	}))
	routes.HandleUpdateCustomFieldDefinition(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "missing definition id")
}

func TestHandleUpdateCustomFieldDefinition_InvalidJSON(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/work/custom-fields/"+testFieldDefID, invalidJSON())
	req = withChiURLParam(req, "id", testFieldDefID)
	routes.HandleUpdateCustomFieldDefinition(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateCustomFieldDefinition_MissingFields(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))

	tests := []struct {
		name      string
		body      map[string]interface{}
		wantField string
	}{
		{"no name", map[string]interface{}{"field_type": "text"}, "name"},
		{"no field_type", map[string]interface{}{"name": "Priority"}, "field_type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("PUT", "/api/v1/work/custom-fields/"+testFieldDefID, jsonBody(t, tt.body))
			req = withChiURLParam(req, "id", testFieldDefID)
			routes.HandleUpdateCustomFieldDefinition(rec, req)
			assertValidationError(t, rec, tt.wantField)
		})
	}
}

func TestHandleUpdateCustomFieldDefinition_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/work/custom-fields/"+testFieldDefID, jsonBody(t, map[string]interface{}{
		"name":       "Priority",
		"field_type": "select",
	}))
	req = withChiURLParam(req, "id", testFieldDefID)
	routes.HandleUpdateCustomFieldDefinition(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteCustomFieldDefinition ---

func TestHandleDeleteCustomFieldDefinition_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/work/custom-fields/"+testFieldDefID, nil)
	req = withChiURLParam(req, "id", testFieldDefID)
	routes.HandleDeleteCustomFieldDefinition(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteCustomFieldDefinition_MissingID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/work/custom-fields/", nil)
	routes.HandleDeleteCustomFieldDefinition(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "missing definition id")
}

func TestHandleDeleteCustomFieldDefinition_ReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/work/custom-fields/"+testFieldDefID, nil)
	req = withChiURLParam(req, "id", testFieldDefID)
	routes.HandleDeleteCustomFieldDefinition(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- labelIDsFromQuery ---

func TestLabelIDsFromQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"absent param", "", nil},
		{"single id", "label_ids=a", []string{"a"}},
		{"comma separated", "label_ids=a,b,c", []string{"a", "b", "c"}},
		{"trims whitespace", "label_ids=" + "a" + "%2C+b" + "%2C+c", []string{"a", "b", "c"}},
		{"drops empty segments", "label_ids=a,,b", []string{"a", "b"}},
		{"only commas", "label_ids=,,", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/tasks?"+tt.query, nil)
			got := labelIDsFromQuery(req, "label_ids")
			if len(got) != len(tt.want) {
				t.Fatalf("labelIDsFromQuery() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("labelIDsFromQuery() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// --- Router-level permission guards ---
//
// work_labels/work_custom_fields use a plain middleware.RequirePermission,
// not RequirePermissionAny — there is no legacy/catalogue key duality to
// prove here, just: right key opens the door, any other key (or none) 403s.

func TestWorkLabelRoutes_Guards(t *testing.T) {
	router := chi.NewRouter()
	NewWorkRoutes(emptyRegistry()).RegisterRoutes(router, guardTestAuth)

	const allowed = http.StatusServiceUnavailable // guard passed, empty registry stops it
	const denied = http.StatusForbidden

	cases := []struct {
		name   string
		method string
		path   string
		perms  []string
		want   int
	}{
		{"list, read perm", "GET", "/api/v1/work/labels/", []string{"work_labels:read"}, allowed},
		{"list, no perms", "GET", "/api/v1/work/labels/", nil, denied},
		{"list, write perm does not grant read", "GET", "/api/v1/work/labels/", []string{"work_labels:write"}, denied},
		{"create, write perm", "POST", "/api/v1/work/labels/", []string{"work_labels:write"}, allowed},
		{"create, read perm does not grant write", "POST", "/api/v1/work/labels/", []string{"work_labels:read"}, denied},
		{"get, read perm", "GET", "/api/v1/work/labels/" + testLabelID, []string{"work_labels:read"}, allowed},
		{"update, write perm", "PUT", "/api/v1/work/labels/" + testLabelID, []string{"work_labels:write"}, allowed},
		{"delete, delete perm", "DELETE", "/api/v1/work/labels/" + testLabelID, []string{"work_labels:delete"}, allowed},
		{"delete, write perm does not grant delete", "DELETE", "/api/v1/work/labels/" + testLabelID, []string{"work_labels:write"}, denied},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, jsonBody(t, map[string]interface{}{"name": "x"}))
			req = withPermissions(req, tc.perms...)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assertStatus(t, rec, tc.want)
		})
	}
}

func TestWorkCustomFieldRoutes_Guards(t *testing.T) {
	router := chi.NewRouter()
	NewWorkRoutes(emptyRegistry()).RegisterRoutes(router, guardTestAuth)

	const allowed = http.StatusServiceUnavailable
	const denied = http.StatusForbidden

	cases := []struct {
		name   string
		method string
		path   string
		perms  []string
		want   int
	}{
		{"list, read perm", "GET", "/api/v1/work/custom-fields/", []string{"work_custom_fields:read"}, allowed},
		{"list, no perms", "GET", "/api/v1/work/custom-fields/", nil, denied},
		{"create, write perm", "POST", "/api/v1/work/custom-fields/", []string{"work_custom_fields:write"}, allowed},
		{"create, read perm does not grant write", "POST", "/api/v1/work/custom-fields/", []string{"work_custom_fields:read"}, denied},
		{"delete, delete perm", "DELETE", "/api/v1/work/custom-fields/" + testFieldDefID, []string{"work_custom_fields:delete"}, allowed},
		{"delete, write perm does not grant delete", "DELETE", "/api/v1/work/custom-fields/" + testFieldDefID, []string{"work_custom_fields:write"}, denied},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, jsonBody(t, map[string]interface{}{"name": "x", "field_type": "text"}))
			req = withPermissions(req, tc.perms...)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assertStatus(t, rec, tc.want)
		})
	}
}
