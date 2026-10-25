package gateway

// The desktop reads a project's saved view flat (`prefData.view_type`), not under a
// `preference` key, so the saved view was never restored. The tests go through the real
// handlers and an in-process work service.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

func (s *workTaskWireStub) GetUserProjectPreference(_ context.Context, r *workv1.GetUserProjectPreferenceRequest) (*workv1.GetUserProjectPreferenceResponse, error) {
	if s.pref == nil {
		return &workv1.GetUserProjectPreferenceResponse{Preference: &workv1.UserProjectPreferenceProto{
			UserId: r.GetUserId(), ProjectId: r.GetProjectId(), ViewType: "list", ListGroupBy: "status", ListSortBy: "sort_order",
		}}, nil
	}
	return &workv1.GetUserProjectPreferenceResponse{Preference: s.pref}, nil
}

func (s *workTaskWireStub) SetUserProjectPreference(_ context.Context, r *workv1.SetUserProjectPreferenceRequest) (*workv1.SetUserProjectPreferenceResponse, error) {
	s.pref = &workv1.UserProjectPreferenceProto{
		UserId: r.GetUserId(), ProjectId: r.GetProjectId(), ViewType: r.GetViewType(),
		ListGroupBy: r.GetListGroupBy(), ListSortBy: r.GetListSortBy(), ListSortDesc: r.GetListSortDesc(),
	}
	return &workv1.SetUserProjectPreferenceResponse{Preference: s.pref}, nil
}

func preferenceRequestFor(t *testing.T, method string, body map[string]any) *http.Request {
	t.Helper()
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, "/x", nil)
	} else {
		req = httptest.NewRequest(method, "/x", jsonBody(t, body))
	}
	return withChiURLParam(withAuth(req, testWorkTaskUserID, testTenantID), "id", testWorkTaskID)
}

func TestWorkWire_ProjectPreferenceSavedThenReadIsFlat(t *testing.T) {
	routes, _ := newWorkTaskWireRoutes(t)

	put := httptest.NewRecorder()
	routes.HandleSetUserProjectPreference(put, preferenceRequestFor(t, http.MethodPut,
		map[string]any{"view_type": "kanban", "list_group_by": "assignee", "list_sort_by": "due_date", "list_sort_desc": true}))
	require.Equal(t, http.StatusOK, put.Code, put.Body.String())

	get := httptest.NewRecorder()
	routes.HandleGetUserProjectPreference(get, preferenceRequestFor(t, http.MethodGet, nil))
	require.Equal(t, http.StatusOK, get.Code, get.Body.String())

	got := decodeObject(t, get)
	assert.NotContains(t, got, "preference", "the FE reads the fields at the top level")
	assert.Equal(t, "kanban", got["view_type"])
	assert.Equal(t, "assignee", got["list_group_by"])
	assert.Equal(t, "due_date", got["list_sort_by"])
	assert.Equal(t, true, got["list_sort_desc"])
	assert.Equal(t, testWorkTaskID, got["project_id"])

	assert.Equal(t, decodeObject(t, put), got, "PUT answers with the same flat shape")
}

func TestWorkWire_ProjectPreferenceDefaultsAreFlatAndKeepFalse(t *testing.T) {
	routes, _ := newWorkTaskWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleGetUserProjectPreference(rec, preferenceRequestFor(t, http.MethodGet, nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	got := decodeObject(t, rec)
	assert.NotContains(t, got, "preference")
	assert.Equal(t, "list", got["view_type"])
	assert.Equal(t, "status", got["list_group_by"])
	assert.Equal(t, "sort_order", got["list_sort_by"])
	assert.Equal(t, false, got["list_sort_desc"], "a false must be present, not omitted")
}
