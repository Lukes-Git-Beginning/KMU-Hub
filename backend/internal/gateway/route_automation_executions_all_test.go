package gateway

// GET /api/v1/automations/executions is what the page-level log tab calls when no automation
// is picked (automation-client.ts `listAllExecutions`); it used to fall into GET /{id} with
// id="executions". The tests drive the real router and read the request the automation
// service actually receives over a gRPC hop.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	automationv1 "github.com/kmuhub/kmuhub/proto/automation/v1"
)

type automationExecStub struct {
	automationWireStub
	mu    sync.Mutex
	lists []*automationv1.ListExecutionsRequest
	gets  []string // "execution:<id>" / "automation:<id>"
}

func (s *automationExecStub) ListExecutions(ctx context.Context, r *automationv1.ListExecutionsRequest) (*automationv1.ListExecutionsResponse, error) {
	s.mu.Lock()
	s.lists = append(s.lists, r)
	s.mu.Unlock()
	return s.automationWireStub.ListExecutions(ctx, r)
}

func (s *automationExecStub) GetExecution(ctx context.Context, r *automationv1.GetExecutionRequest) (*automationv1.GetExecutionResponse, error) {
	s.mu.Lock()
	s.gets = append(s.gets, "execution:"+r.GetExecutionId())
	s.mu.Unlock()
	return s.automationWireStub.GetExecution(ctx, r)
}

func (s *automationExecStub) GetAutomation(ctx context.Context, r *automationv1.GetAutomationRequest) (*automationv1.GetAutomationResponse, error) {
	s.mu.Lock()
	s.gets = append(s.gets, "automation:"+r.GetAutomationId())
	s.mu.Unlock()
	return s.automationWireStub.GetAutomation(ctx, r)
}

func newAutomationExecRouter(t *testing.T, stub *automationExecStub) http.Handler {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	automationv1.RegisterAutomationServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("automation", lis.Addr().String())
	t.Cleanup(reg.Close)

	r := chi.NewRouter()
	NewAutomationRoutes(reg).RegisterRoutes(r, func(next http.Handler) http.Handler { return next })
	return r
}

func automationExecGet(router http.Handler, path string, perms ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req = withPermissions(withAuth(req, testHRUserID, testTenantID), perms...)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestAutomationWire_ListAllExecutions(t *testing.T) {
	stub := &automationExecStub{}
	router := newAutomationExecRouter(t, stub)

	rec := automationExecGet(router, "/api/v1/automations/executions?status=failed&limit=25&offset=50", "automations:read")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	require.Len(t, stub.lists, 1)
	got := stub.lists[0]
	assert.Nil(t, got.AutomationId, "no automation picked: the RPC lists the whole tenant")
	assert.Equal(t, automationv1.ExecutionStatus_EXECUTION_STATUS_FAILED, got.GetStatus())
	assert.Equal(t, int32(25), got.GetLimit())
	assert.Equal(t, int32(50), got.GetOffset())
	assert.Empty(t, stub.gets, "must not fall through to GET /{id}")

	body := decodeBody(t, rec)
	items, ok := body["executions"].([]any)
	require.True(t, ok, "wrapped as {executions,total_count}: %v", body)
	assert.Len(t, items, 1)
	assert.EqualValues(t, 1, body["total_count"])
}

func TestAutomationWire_ListAllExecutionsEmptyIsAnArray(t *testing.T) {
	stub := &automationExecStub{automationWireStub: automationWireStub{empty: true}}
	router := newAutomationExecRouter(t, stub)

	rec := automationExecGet(router, "/api/v1/automations/executions", "automations:read")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body := decodeBody(t, rec)
	items, ok := body["executions"].([]any)
	require.True(t, ok, "empty list must be [] not null: %v", body)
	assert.Empty(t, items)
}

func TestAutomationWire_ListAllExecutionsNeedsTheExecutionsGuard(t *testing.T) {
	stub := &automationExecStub{}
	router := newAutomationExecRouter(t, stub)

	rec := automationExecGet(router, "/api/v1/automations/executions", "wiki:articles:read")
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Empty(t, stub.lists)

	// Additive: the fine key works as well as the legacy one.
	rec = automationExecGet(router, "/api/v1/automations/executions", "automatisierung:executions:read")
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Len(t, stub.lists, 1)
}

func TestAutomationWire_NeighbourRoutesStayUntouched(t *testing.T) {
	stub := &automationExecStub{}
	router := newAutomationExecRouter(t, stub)

	// A real automation ID still reaches GetAutomation ...
	rec := automationExecGet(router, "/api/v1/automations/"+testSchemaID, "automations:read")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	// ... one execution still reaches GetExecution ...
	rec = automationExecGet(router, "/api/v1/automations/executions/"+testSchemaID, "automations:read")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, []string{"automation:" + testSchemaID, "execution:" + testSchemaID}, stub.gets)

	// ... and the per-automation list still scopes to that automation.
	rec = automationExecGet(router, "/api/v1/automations/"+testSchemaID+"/executions?status=completed", "automations:read")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Len(t, stub.lists, 1)
	assert.Equal(t, testSchemaID, stub.lists[0].GetAutomationId())
	assert.Equal(t, automationv1.ExecutionStatus_EXECUTION_STATUS_COMPLETED, stub.lists[0].GetStatus())
}
