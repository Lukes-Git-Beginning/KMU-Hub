package gateway

// GET /finance/dashboard?date_from&date_to: the desktop's period presets send both as YYYY-MM-DD.
// A real gRPC server on loopback records the request the biz service receives.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

type dashboardRangeStub struct {
	bizv1.UnimplementedFinanceServiceServer
	mu   sync.Mutex
	reqs []*bizv1.GetFinanceDashboardRequest
}

func (s *dashboardRangeStub) GetFinanceDashboard(_ context.Context, req *bizv1.GetFinanceDashboardRequest) (*bizv1.GetFinanceDashboardResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reqs = append(s.reqs, req)
	return &bizv1.GetFinanceDashboardResponse{Dashboard: &bizv1.FinanceDashboard{}}, nil
}

func getDashboardWithQuery(t *testing.T, stub *dashboardRangeStub, query string) *httptest.ResponseRecorder {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	bizv1.RegisterFinanceServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("biz", lis.Addr().String())
	t.Cleanup(reg.Close)

	rec := httptest.NewRecorder()
	NewBizRoutes(reg).HandleGetFinanceDashboard(rec, noBodyRequest(http.MethodGet, "/api/v1/finance/dashboard"+query, ""))
	return rec
}

func TestHandleGetFinanceDashboard_ForwardsTheDateRange(t *testing.T) {
	stub := &dashboardRangeStub{}
	rec := getDashboardWithQuery(t, stub, "?date_from=2026-01-01&date_to=2026-03-31")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	require.Len(t, stub.reqs, 1)
	assert.Equal(t, "2026-01-01", stub.reqs[0].GetDateFrom())
	assert.Equal(t, "2026-03-31", stub.reqs[0].GetDateTo())
	assert.Equal(t, testTenantID.String(), stub.reqs[0].GetTenantId())
}

func TestHandleGetFinanceDashboard_WithoutParametersLeavesTheRangeUnset(t *testing.T) {
	stub := &dashboardRangeStub{}
	rec := getDashboardWithQuery(t, stub, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	require.Len(t, stub.reqs, 1)
	assert.Nil(t, stub.reqs[0].DateFrom)
	assert.Nil(t, stub.reqs[0].DateTo)
}

func TestHandleGetFinanceDashboard_RejectsABadRangeBeforeTheRPC(t *testing.T) {
	for name, query := range map[string]string{
		"garbage date_from":  "?date_from=yesterday",
		"impossible date_to": "?date_to=2026-02-30",
		"inverted range":     "?date_from=2026-04-01&date_to=2026-03-01",
	} {
		t.Run(name, func(t *testing.T) {
			stub := &dashboardRangeStub{}
			rec := getDashboardWithQuery(t, stub, query)
			assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			assert.Empty(t, stub.reqs, "a rejected range must not reach the biz service")
		})
	}
}
