package gateway

// A manager decides leave requests through team:absence:approve at scope team; hr:write keeps the
// tenant-wide reach. The gateway resolves the scope from the token and hands it to the HR service,
// which enforces the reporting line. A real gRPC server on loopback plays the service.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

type leaveDecideStub struct {
	hrv1.UnimplementedHRServiceServer
	approved *hrv1.ApproveLeaveRequestReq
	rejected *hrv1.RejectLeaveRequestReq
	listed   *hrv1.ListLeaveRequestsReq
}

func (s *leaveDecideStub) ApproveLeaveRequest(_ context.Context, r *hrv1.ApproveLeaveRequestReq) (*hrv1.ApproveLeaveRequestResp, error) {
	s.approved = r
	return &hrv1.ApproveLeaveRequestResp{LeaveRequest: &hrv1.LeaveRequest{Id: r.GetId()}}, nil
}

func (s *leaveDecideStub) RejectLeaveRequest(_ context.Context, r *hrv1.RejectLeaveRequestReq) (*hrv1.RejectLeaveRequestResp, error) {
	s.rejected = r
	return &hrv1.RejectLeaveRequestResp{LeaveRequest: &hrv1.LeaveRequest{Id: r.GetId()}}, nil
}

func (s *leaveDecideStub) ListLeaveRequests(_ context.Context, r *hrv1.ListLeaveRequestsReq) (*hrv1.ListLeaveRequestsResp, error) {
	s.listed = r
	return &hrv1.ListLeaveRequestsResp{}, nil
}

func newLeaveDecideRouter(t *testing.T) (*leaveDecideStub, http.Handler) {
	t.Helper()
	stub := &leaveDecideStub{}
	router := chi.NewRouter()
	NewHRRoutes(registryFor(t, "biz", func(s *grpc.Server) { hrv1.RegisterHRServiceServer(s, stub) }), nil).
		RegisterRoutes(router, guardTestAuth)
	return stub, router
}

// Token shapes taken from the preset roles after migration 000335.
var (
	managerToken = tokenShape{perms: []string{"hr:self", "team:absence:approve"}, scopes: map[string]string{"hr:self": auth.ScopeOwn, "team:absence:approve": auth.ScopeTeam}}
	adminToken   = tokenShape{perms: []string{"hr:read", "hr:write", "hr:self", "team:absence:approve"}, scopes: map[string]string{"hr:read": auth.ScopeAll, "hr:write": auth.ScopeAll, "hr:self": auth.ScopeAll, "team:absence:approve": auth.ScopeAll}}
	memberToken  = tokenShape{perms: []string{"hr:self", "team:absence:read"}, scopes: map[string]string{"hr:self": auth.ScopeOwn, "team:absence:read": auth.ScopeAll}}
)

type tokenShape struct {
	perms  []string
	scopes map[string]string
}

func (tk tokenShape) request(method, target string) *http.Request {
	return withPermissions(withScopes(withAuth(httptest.NewRequest(method, target, strings.NewReader("{}")), leaveCallerID, testTenantID), tk.scopes), tk.perms...)
}

const decideTarget = "/api/v1/hr/leave/requests/55555555-5555-5555-5555-555555555555"

func TestLeaveDecide_ManagerReachesApproveAndRejectWithScopeTeam(t *testing.T) {
	stub, router := newLeaveDecideRouter(t)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, managerToken.request(http.MethodPost, decideTarget+"/approve"))
	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.approved)
	assert.Equal(t, auth.ScopeTeam, stub.approved.GetScope(), "a manager must not decide beyond the reporting line")
	assert.Equal(t, leaveCallerID, stub.approved.GetApproverId(), "the approver comes from the token")

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, managerToken.request(http.MethodPost, decideTarget+"/reject"))
	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.rejected)
	assert.Equal(t, auth.ScopeTeam, stub.rejected.GetScope())
}

func TestLeaveDecide_AdminKeepsTheTenantWideReachAndMemberIsRefused(t *testing.T) {
	stub, router := newLeaveDecideRouter(t)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, adminToken.request(http.MethodPost, decideTarget+"/approve"))
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, auth.ScopeAll, stub.approved.GetScope())

	// Old token that carries only the coarse key and no scopes claim.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, tokenShape{perms: []string{"hr:write"}}.request(http.MethodPost, decideTarget+"/approve"))
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, auth.ScopeAll, stub.approved.GetScope())

	stub.approved = nil
	for _, path := range []string{"/approve", "/reject"} {
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, memberToken.request(http.MethodPost, decideTarget+path))
		assertStatus(t, rec, http.StatusForbidden)
	}
	assert.Nil(t, stub.approved, "a refused member never reaches the service")
}

func TestLeaveList_ManagerSeesTheReportingLineAndStillTheirOwnRequests(t *testing.T) {
	stub, router := newLeaveDecideRouter(t)
	list := func(tk tokenShape, query string) {
		stub.listed = nil
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, tk.request(http.MethodGet, "/api/v1/hr/leave/requests"+query))
		assertStatus(t, rec, http.StatusOK)
		require.NotNil(t, stub.listed)
	}

	list(managerToken, "?status=pending")
	assert.True(t, stub.listed.GetReportingLineOnly(), "the approval tab (no employee filter) lists the reporting line")
	assert.Empty(t, stub.listed.GetEmployeeId())

	list(managerToken, "?user_id="+leaveCallerID)
	assert.False(t, stub.listed.GetReportingLineOnly(), "the self-service view asks for the manager's own requests")
	assert.Equal(t, leaveCallerID, stub.listed.GetEmployeeId())

	list(managerToken, "?employee_id="+leaveOtherID)
	assert.True(t, stub.listed.GetReportingLineOnly(), "a foreign employee filter is still bounded by the reporting line")
	assert.Equal(t, leaveOtherID, stub.listed.GetEmployeeId())

	list(memberToken, "?status=pending")
	assert.False(t, stub.listed.GetReportingLineOnly())
	assert.Equal(t, leaveCallerID, stub.listed.GetEmployeeId(), "a member without the approve key stays pinned to their own requests")

	list(adminToken, "")
	assert.False(t, stub.listed.GetReportingLineOnly())
	assert.Empty(t, stub.listed.GetEmployeeId())
}
