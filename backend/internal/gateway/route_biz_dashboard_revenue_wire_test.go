package gateway

// GET /finance/dashboard feeds the revenue widgets, which read revenue_this_month,
// revenue_last_month and monthly_revenue[].revenue as numbers and divide by them. A real gRPC
// server on loopback plays the biz service; the assertions run on the JSON the client receives.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

type dashboardRevenueStub struct {
	bizv1.UnimplementedFinanceServiceServer
	dashboard *bizv1.FinanceDashboard
}

func (s *dashboardRevenueStub) GetFinanceDashboard(context.Context, *bizv1.GetFinanceDashboardRequest) (*bizv1.GetFinanceDashboardResponse, error) {
	return &bizv1.GetFinanceDashboardResponse{Dashboard: s.dashboard}, nil
}

func getDashboardJSON(t *testing.T, d *bizv1.FinanceDashboard) map[string]any {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	bizv1.RegisterFinanceServiceServer(srv, &dashboardRevenueStub{dashboard: d})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("biz", lis.Addr().String())
	t.Cleanup(reg.Close)

	rec := httptest.NewRecorder()
	NewBizRoutes(reg).HandleGetFinanceDashboard(rec, noBodyRequest(http.MethodGet, "/api/v1/finance/dashboard", ""))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}

func TestHandleGetFinanceDashboard_RevenueFieldsAreNumbersEvenAtZero(t *testing.T) {
	body := getDashboardJSON(t, &bizv1.FinanceDashboard{
		RevenueThisMonth: 47500.5,
		RevenueLastMonth: 0,
		MonthlyRevenue: []*bizv1.MonthlyRevenue{
			{Month: "2026-02-01", Revenue: 0},
			{Month: "2026-03-01", Revenue: 47500.5},
		},
	})

	assert.Equal(t, 47500.5, body["revenue_this_month"])
	assert.Equal(t, float64(0), body["revenue_last_month"], "a zero month must be a 0, not a missing key")

	series, ok := body["monthly_revenue"].([]any)
	require.True(t, ok, "monthly_revenue must be a list: %v", body["monthly_revenue"])
	require.Len(t, series, 2)
	first := series[0].(map[string]any)
	assert.Equal(t, "2026-02-01", first["month"])
	assert.Equal(t, float64(0), first["revenue"], "revenue 0 must stay in the chart point")
}

func TestHandleGetFinanceDashboard_EmptySeriesIsAListNotNull(t *testing.T) {
	body := getDashboardJSON(t, &bizv1.FinanceDashboard{})

	assert.Equal(t, []any{}, body["monthly_revenue"])
	assert.Equal(t, []any{}, body["recent_invoices"])
}
