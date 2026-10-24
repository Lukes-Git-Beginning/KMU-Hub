package gateway

// A task's due date must be clearable: the Desktop sends `due_date: null` for it. A null used
// to decode into the same nil pointer as an absent key, so the date stayed forever. The tests
// read the request the work service actually receives.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

const testWorkTaskID = "550e8400-e29b-41d4-a716-446655440000"
const testWorkTaskUserID = "33333333-3333-3333-3333-333333333333"

type workTaskWireStub struct {
	workv1.UnimplementedWorkServiceServer
	updated *workv1.UpdateTaskRequest
}

func (s *workTaskWireStub) UpdateTask(_ context.Context, r *workv1.UpdateTaskRequest) (*workv1.UpdateTaskResponse, error) {
	s.updated = r
	return &workv1.UpdateTaskResponse{Task: &workv1.TaskProto{Id: r.GetId()}}, nil
}

func newWorkTaskWireRoutes(t *testing.T) (*WorkRoutes, *workTaskWireStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &workTaskWireStub{}
	srv := grpc.NewServer()
	workv1.RegisterWorkServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("work", lis.Addr().String())
	t.Cleanup(reg.Close)
	return NewWorkRoutes(reg), stub
}

func updateTaskRequestFor(t *testing.T, body map[string]any) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/x", jsonBody(t, body))
	req = withAuth(req, testWorkTaskUserID, testTenantID)
	return withChiURLParam(req, "id", testWorkTaskID)
}

func TestWorkWire_UpdateTaskNullDueDateClearsIt(t *testing.T) {
	routes, stub := newWorkTaskWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleUpdateTask(rec, updateTaskRequestFor(t, map[string]any{"title": "x", "due_date": nil}))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, stub.updated)
	assert.True(t, stub.updated.GetClearDueDate())
	assert.Nil(t, stub.updated.DueDate)
}

func TestWorkWire_UpdateTaskAbsentDueDateKeepsIt(t *testing.T) {
	routes, stub := newWorkTaskWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleUpdateTask(rec, updateTaskRequestFor(t, map[string]any{"title": "x"}))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, stub.updated)
	assert.False(t, stub.updated.GetClearDueDate())
	assert.Nil(t, stub.updated.DueDate)
}

func TestWorkWire_UpdateTaskDueDateStringStillSetsIt(t *testing.T) {
	routes, stub := newWorkTaskWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleUpdateTask(rec, updateTaskRequestFor(t, map[string]any{"due_date": "2026-03-15"}))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, stub.updated)
	assert.False(t, stub.updated.GetClearDueDate())
	require.NotNil(t, stub.updated.DueDate)
	assert.Equal(t, "2026-03-15", stub.updated.DueDate.AsTime().Format("2006-01-02"))
}

func TestWorkWire_UpdateTaskMalformedDueDateIs400(t *testing.T) {
	for _, bad := range []any{"not-a-date", 42, true} {
		routes, stub := newWorkTaskWireRoutes(t)
		rec := httptest.NewRecorder()
		routes.HandleUpdateTask(rec, updateTaskRequestFor(t, map[string]any{"due_date": bad}))
		assert.Equal(t, http.StatusBadRequest, rec.Code, "%v", bad)
		assert.Nil(t, stub.updated, "must not reach the service: %v", bad)
	}
}

func TestWorkWire_UpdateTaskIsClosedReachesTheService(t *testing.T) {
	yes, no := true, false
	for name, tc := range map[string]struct {
		body map[string]any
		want *bool
	}{
		"complete": {map[string]any{"is_closed": true, "completed_at": "2026-10-24T10:00:00Z"}, &yes},
		"reopen":   {map[string]any{"is_closed": false, "completed_at": nil}, &no},
		"absent":   {map[string]any{"title": "x"}, nil},
	} {
		t.Run(name, func(t *testing.T) {
			routes, stub := newWorkTaskWireRoutes(t)

			rec := httptest.NewRecorder()
			routes.HandleUpdateTask(rec, updateTaskRequestFor(t, tc.body))

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.NotNil(t, stub.updated)
			if tc.want == nil {
				assert.Nil(t, stub.updated.IsClosed)
				return
			}
			require.NotNil(t, stub.updated.IsClosed)
			assert.Equal(t, *tc.want, *stub.updated.IsClosed)
		})
	}
}

func TestWorkWire_UpdateTaskIsClosedMustBeABool(t *testing.T) {
	routes, stub := newWorkTaskWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleUpdateTask(rec, updateTaskRequestFor(t, map[string]any{"is_closed": "yes"}))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Nil(t, stub.updated)
}
