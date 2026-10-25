package gateway

// "Meine Aufgaben -> Neue Aufgabe" creates a task without a project ({title, priority,
// assignee_id}); the work service has always allowed that. The tests read the request the
// work service actually receives.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTaskRequestFor(t *testing.T, body map[string]any) *http.Request {
	t.Helper()
	return withAuth(httptest.NewRequest(http.MethodPost, "/x", jsonBody(t, body)), testWorkTaskUserID, testTenantID)
}

func TestWorkWire_CreateTaskWithoutProjectReachesTheService(t *testing.T) {
	routes, stub := newWorkTaskWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleCreateTask(rec, createTaskRequestFor(t, map[string]any{
		"title": "Standalone", "priority": "high", "assignee_id": testWorkTaskUserID,
	}))

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.NotNil(t, stub.created)
	assert.Empty(t, stub.created.GetProjectId(), "no project must stay no project")
	assert.Equal(t, testWorkTaskUserID, stub.created.GetAssigneeId())
	assert.Equal(t, testWorkTaskUserID, stub.created.GetCreatedBy())
}

func TestWorkWire_CreateTaskWithProjectStillForwardsIt(t *testing.T) {
	routes, stub := newWorkTaskWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleCreateTask(rec, createTaskRequestFor(t, map[string]any{"title": "In project", "project_id": testWorkTaskID}))

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.NotNil(t, stub.created)
	assert.Equal(t, testWorkTaskID, stub.created.GetProjectId())
}

func TestWorkWire_CreateTaskMalformedProjectIDIs400(t *testing.T) {
	routes, stub := newWorkTaskWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleCreateTask(rec, createTaskRequestFor(t, map[string]any{"title": "x", "project_id": "not-a-uuid"}))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Nil(t, stub.created, "must not reach the service")
}
