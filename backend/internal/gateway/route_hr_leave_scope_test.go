package gateway

// The self-service view lists leave requests with user_id=<self>. The gateway used to read only
// employee_id, so the filter vanished and a caller holding hr:read saw every request of the
// tenant as their own. A real gRPC server on loopback plays the HR service.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

const (
	leaveCallerID = "22222222-2222-2222-2222-222222222222"
	leaveOtherID  = "33333333-3333-3333-3333-333333333333"
)

type leaveScopeStub struct {
	hrv1.UnimplementedHRServiceServer
	listed *hrv1.ListLeaveRequestsReq
}

func (s *leaveScopeStub) ListLeaveRequests(_ context.Context, r *hrv1.ListLeaveRequestsReq) (*hrv1.ListLeaveRequestsResp, error) {
	s.listed = r
	return &hrv1.ListLeaveRequestsResp{}, nil
}

func (s *leaveScopeStub) GetLeaveRequest(_ context.Context, r *hrv1.GetLeaveRequestReq) (*hrv1.GetLeaveRequestResp, error) {
	owner := leaveOtherID
	if r.GetId() == "11111111-1111-1111-1111-111111111111" {
		owner = leaveCallerID
	}
	return &hrv1.GetLeaveRequestResp{LeaveRequest: &hrv1.LeaveRequest{Id: r.GetId(), EmployeeId: owner}}, nil
}

func newLeaveScopeRoutes(t *testing.T) (*HRRoutes, *leaveScopeStub) {
	t.Helper()
	stub := &leaveScopeStub{}
	return NewHRRoutes(registryFor(t, "biz", func(s *grpc.Server) { hrv1.RegisterHRServiceServer(s, stub) }), nil), stub
}

func leaveRequest(target string, scopes map[string]string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req = withAuth(req, leaveCallerID, testTenantID)
	if scopes != nil {
		req = withScopes(req, scopes)
	}
	return req
}

func TestLeaveList_UserIDQueryParameterFiltersLikeEmployeeID(t *testing.T) {
	routes, stub := newLeaveScopeRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleListLeaveRequests(rec, leaveRequest("/api/v1/hr/leave/requests?user_id="+leaveOtherID, nil))

	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.listed)
	assert.Equal(t, leaveOtherID, stub.listed.GetEmployeeId(), "the self-service view's user_id must reach the service")
}

func TestLeaveList_OwnScopeOverridesTheRequestedEmployee(t *testing.T) {
	routes, stub := newLeaveScopeRoutes(t)
	own := map[string]string{"hr:read": "own"}

	for _, query := range []string{"", "?employee_id=" + leaveOtherID, "?user_id=" + leaveOtherID} {
		stub.listed = nil
		rec := httptest.NewRecorder()
		routes.HandleListLeaveRequests(rec, leaveRequest("/api/v1/hr/leave/requests"+query, own))

		assertStatus(t, rec, http.StatusOK)
		require.NotNil(t, stub.listed)
		assert.Equal(t, leaveCallerID, stub.listed.GetEmployeeId(), "query %q must not widen an own-scoped grant", query)
	}
}

func TestLeaveGet_OwnScopeHidesSomeoneElsesRequest(t *testing.T) {
	routes, _ := newLeaveScopeRoutes(t)
	own := map[string]string{"hr:read": "own"}

	rec := httptest.NewRecorder()
	req := withChiURLParam(leaveRequest("/api/v1/hr/leave/requests/x", own), "id", "44444444-4444-4444-4444-444444444444")
	routes.HandleGetLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusNotFound)

	rec = httptest.NewRecorder()
	req = withChiURLParam(leaveRequest("/api/v1/hr/leave/requests/x", own), "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleGetLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusOK)

	rec = httptest.NewRecorder()
	req = withChiURLParam(leaveRequest("/api/v1/hr/leave/requests/x", nil), "id", "44444444-4444-4444-4444-444444444444")
	routes.HandleGetLeaveRequest(rec, req)
	assertStatus(t, rec, http.StatusOK)
}

// hr:self at scope own must not read someone else's balance or time entries: both routes take an
// id from the request, so the guard alone would hand the whole tenant to every employee.

type ownScopeHRStub struct {
	hrv1.UnimplementedHRServiceServer
	balanceFor string
	entriesReq *hrv1.ListWorkTimeEntriesReq
}

func (s *ownScopeHRStub) GetEmployeeLeaveBalance(_ context.Context, r *hrv1.GetEmployeeLeaveBalanceReq) (*hrv1.GetEmployeeLeaveBalanceResp, error) {
	s.balanceFor = r.GetEmployeeId()
	return &hrv1.GetEmployeeLeaveBalanceResp{Balance: &hrv1.LeaveBalance{}}, nil
}

func (s *ownScopeHRStub) ListWorkTimeEntries(_ context.Context, r *hrv1.ListWorkTimeEntriesReq) (*hrv1.ListWorkTimeEntriesResp, error) {
	s.entriesReq = r
	return &hrv1.ListWorkTimeEntriesResp{}, nil
}

func TestHROwnScope_ForeignBalanceAndTimeEntries(t *testing.T) {
	stub := &ownScopeHRStub{}
	router := chi.NewRouter()
	NewHRRoutes(registryFor(t, "biz", func(s *grpc.Server) { hrv1.RegisterHRServiceServer(s, stub) }), nil).
		RegisterRoutes(router, guardTestAuth)

	call := func(path string, scopes map[string]string, perm string) *httptest.ResponseRecorder {
		req := withPermissions(leaveRequest(path, scopes), perm)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	self := map[string]string{"hr:self": "own"}

	// Balance by id: foreign -> 403 and the service is never asked, own -> 200.
	rec := call("/api/v1/hr/leave/balance/"+leaveOtherID, self, "hr:self")
	assertStatus(t, rec, http.StatusForbidden)
	assert.Empty(t, stub.balanceFor)
	rec = call("/api/v1/hr/leave/balance/"+leaveCallerID, self, "hr:self")
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, leaveCallerID, stub.balanceFor)

	// Time entries: foreign employee_id -> 403, none -> pinned to the caller, own -> 200.
	rec = call("/api/v1/hr/time/entries?employee_id="+leaveOtherID, self, "hr:self")
	assertStatus(t, rec, http.StatusForbidden)
	assert.Nil(t, stub.entriesReq)
	rec = call("/api/v1/hr/time/entries", self, "hr:self")
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, leaveCallerID, stub.entriesReq.GetEmployeeId(), "no filter must not mean every employee")
	rec = call("/api/v1/hr/time/entries?employee_id="+leaveCallerID, self, "hr:self")
	assertStatus(t, rec, http.StatusOK)

	// hr:read at scope all keeps the manager/admin view of any employee.
	stub.balanceFor, stub.entriesReq = "", nil
	rec = call("/api/v1/hr/leave/balance/"+leaveOtherID, nil, "hr:read")
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, leaveOtherID, stub.balanceFor)
	rec = call("/api/v1/hr/time/entries?employee_id="+leaveOtherID, nil, "hr:read")
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, leaveOtherID, stub.entriesReq.GetEmployeeId())
	rec = call("/api/v1/hr/time/entries", nil, "hr:read")
	assertStatus(t, rec, http.StatusOK)
	assert.Empty(t, stub.entriesReq.GetEmployeeId())
}
