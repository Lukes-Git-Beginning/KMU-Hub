package gateway

// Covers the eight handlers in route_work_time.go that route_work_time_test.go
// leaves untested (that file only pins the two pure wire-shape conversion
// functions): HandleStartTimer, HandleStopTimer, HandleGetActiveTimer,
// HandleAddManualTimeEntry, HandleUpdateTimeEntry, HandleGetTaskTimeSummary,
// HandleListProjectTimeEntries, HandleListProjectTeamUtilization. Follows the
// established pattern in route_work_tasks_test.go: no live gRPC server, so
// validation-before-RPC and the 503-on-unreachable-service path are what's
// verifiable at this layer.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const workTimeValidUUID = "550e8400-e29b-41d4-a716-446655440000"

// --- HandleStartTimer ---

func TestHandleStartTimer_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/"+workTimeValidUUID+"/timer/start", nil)
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleStartTimer(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleStartTimer_InvalidTaskID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/not-a-uuid/timer/start", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleStartTimer(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleStartTimer_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/"+workTimeValidUUID+"/timer/start", nil)
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleStartTimer(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleStopTimer ---

func TestHandleStopTimer_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/time-entries/timer/stop", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleStopTimer(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleStopTimer_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/time-entries/timer/stop", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleStopTimer(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetActiveTimer ---

func TestHandleGetActiveTimer_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/time-entries/timer/active", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetActiveTimer(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetActiveTimer_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/time-entries/timer/active", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetActiveTimer(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleAddManualTimeEntry ---

func TestHandleAddManualTimeEntry_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/"+workTimeValidUUID+"/time-entries", jsonBody(t, map[string]interface{}{
		"started_at":       "2026-08-01T10:00:00Z",
		"duration_seconds": 3600,
	}))
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleAddManualTimeEntry(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleAddManualTimeEntry_InvalidTaskID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/not-a-uuid/time-entries", jsonBody(t, map[string]interface{}{
		"started_at":       "2026-08-01T10:00:00Z",
		"duration_seconds": 3600,
	}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleAddManualTimeEntry(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleAddManualTimeEntry_MissingStartedAt(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/"+workTimeValidUUID+"/time-entries", jsonBody(t, map[string]interface{}{
		"duration_seconds": 3600,
	}))
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleAddManualTimeEntry(rec, req)
	assertValidationError(t, rec, "started_at")
}

func TestHandleAddManualTimeEntry_ZeroDuration(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/"+workTimeValidUUID+"/time-entries", jsonBody(t, map[string]interface{}{
		"started_at":       "2026-08-01T10:00:00Z",
		"duration_seconds": 0,
	}))
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleAddManualTimeEntry(rec, req)
	assertValidationError(t, rec, "duration_seconds")
}

func TestHandleAddManualTimeEntry_InvalidStartedAtFormat(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/"+workTimeValidUUID+"/time-entries", jsonBody(t, map[string]interface{}{
		"started_at":       "not-a-date",
		"duration_seconds": 3600,
	}))
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleAddManualTimeEntry(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid started_at format")
}

func TestHandleAddManualTimeEntry_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks/"+workTimeValidUUID+"/time-entries", jsonBody(t, map[string]interface{}{
		"started_at":       "2026-08-01T10:00:00Z",
		"duration_seconds": 3600,
		"description":      "worked",
	}))
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleAddManualTimeEntry(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateTimeEntry ---

func TestHandleUpdateTimeEntry_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/time-entries/"+workTimeValidUUID, jsonBody(t, map[string]interface{}{
		"duration_seconds": 1800,
	}))
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateTimeEntry(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateTimeEntry_InvalidEntryID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/time-entries/not-a-uuid", jsonBody(t, map[string]interface{}{
		"duration_seconds": 1800,
	}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateTimeEntry(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateTimeEntry_InvalidStartedAtFormat(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/time-entries/"+workTimeValidUUID, jsonBody(t, map[string]interface{}{
		"started_at": "not-a-date",
	}))
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateTimeEntry(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid started_at format")
}

func TestHandleUpdateTimeEntry_ZeroDurationRejected(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/time-entries/"+workTimeValidUUID, jsonBody(t, map[string]interface{}{
		"duration_seconds": 0,
	}))
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateTimeEntry(rec, req)
	assertValidationError(t, rec, "duration_seconds")
}

func TestHandleUpdateTimeEntry_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/time-entries/"+workTimeValidUUID, jsonBody(t, map[string]interface{}{
		"started_at":       "2026-08-01T10:00:00Z",
		"duration_seconds": 1800,
		"description":      "updated",
	}))
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateTimeEntry(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetTaskTimeSummary ---

func TestHandleGetTaskTimeSummary_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tasks/"+workTimeValidUUID+"/time-summary", nil)
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetTaskTimeSummary(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetTaskTimeSummary_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tasks/"+workTimeValidUUID+"/time-summary", nil)
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetTaskTimeSummary(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListProjectTimeEntries ---

func TestHandleListProjectTimeEntries_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/"+workTimeValidUUID+"/time-entries", nil)
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListProjectTimeEntries(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListProjectTimeEntries_InvalidProjectID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/not-a-uuid/time-entries", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListProjectTimeEntries(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// TestHandleListProjectTimeEntries_BilledTrue_ShortCircuitsWithoutRPC proves
// the documented lean gap (route_work_time.go:255-262): billed=true always
// answers an empty list without ever calling ListProjectTimeEntries on the
// gRPC client, even though a client is still obtained first (so a truly
// unregistered service still 503s regardless of the billed value).
func TestHandleListProjectTimeEntries_BilledTrue_ShortCircuitsWithoutRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/"+workTimeValidUUID+"/time-entries?billed=true", nil)
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListProjectTimeEntries(rec, req)
	assertStatus(t, rec, http.StatusOK)

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	entries, ok := body["entries"].([]interface{})
	if !ok {
		t.Fatalf("entries missing or not an array; got %v", body)
	}
	if len(entries) != 0 {
		t.Fatalf("billed=true expected an empty entries array, got %d", len(entries))
	}
}

func TestHandleListProjectTimeEntries_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/"+workTimeValidUUID+"/time-entries", nil)
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListProjectTimeEntries(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListProjectTeamUtilization ---

func TestHandleListProjectTeamUtilization_ServiceUnavailable(t *testing.T) {
	routes := NewWorkRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/"+workTimeValidUUID+"/team-utilization", nil)
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListProjectTeamUtilization(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListProjectTeamUtilization_InvalidProjectID(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/not-a-uuid/team-utilization", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListProjectTeamUtilization(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListProjectTeamUtilization_ValidRequestReachesRPC(t *testing.T) {
	routes := NewWorkRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects/"+workTimeValidUUID+"/team-utilization", nil)
	req = withChiURLParam(req, "id", workTimeValidUUID)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListProjectTeamUtilization(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
