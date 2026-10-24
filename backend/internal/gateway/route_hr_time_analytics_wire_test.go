package gateway

// GET /hr/time/analytics feeds the Auswertungen view, which divides by worked_days and labels
// the project bars with `name`. A real gRPC server on loopback plays the biz service; the
// assertions run on the JSON the client receives.

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

	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

type timeAnalyticsStub struct {
	hrv1.UnimplementedHRServiceServer
	resp *hrv1.GetTimeAnalyticsResp
}

func (s *timeAnalyticsStub) GetTimeAnalytics(context.Context, *hrv1.GetTimeAnalyticsReq) (*hrv1.GetTimeAnalyticsResp, error) {
	return s.resp, nil
}

func getTimeAnalyticsJSON(t *testing.T, target string, resp *hrv1.GetTimeAnalyticsResp) map[string]any {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	hrv1.RegisterHRServiceServer(srv, &timeAnalyticsStub{resp: resp})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("biz", lis.Addr().String())
	t.Cleanup(reg.Close)

	rec := httptest.NewRecorder()
	req := withAuth(httptest.NewRequest(http.MethodGet, target, nil), "user-123", testTenantID)
	NewHRRoutes(reg, nil).HandleGetTimeAnalytics(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}

func TestHandleGetTimeAnalytics_CarriesFieldsTheViewDividesAndLabelsBy(t *testing.T) {
	body := getTimeAnalyticsJSON(t, "/api/v1/hr/time/analytics?range=month", &hrv1.GetTimeAnalyticsResp{
		TotalMinutes:    960,
		TargetMinutes:   2400,
		AvgDailyMinutes: 320,
		DayTrend: []*hrv1.DayTrendEntry{
			{Date: "2026-10-01", NetMinutes: 480, TargetMinutes: 480},
			{Date: "2026-10-02", NetMinutes: 0, TargetMinutes: 480},
			{Date: "2026-10-03", NetMinutes: 480, TargetMinutes: 480},
		},
		ByProject: []*hrv1.ProjectBreakdown{{ProjectId: "p-1", ProjectName: "Relaunch", Minutes: 960}},
	})

	assert.Equal(t, "month", body["range"])
	assert.Equal(t, "2026-10-01", body["period_start"])
	assert.Equal(t, "2026-10-03", body["period_end"])
	assert.Equal(t, float64(2), body["worked_days"], "only days with booked time count")
	assert.Equal(t, float64(0), body["billable_minutes"])

	trend := body["day_trend"].([]any)
	require.Len(t, trend, 3)
	zeroDay := trend[1].(map[string]any)
	assert.Equal(t, float64(0), zeroDay["net_minutes"], "a zero day stays a 0, not a missing key")
	assert.Equal(t, float64(0), zeroDay["billable_minutes"])

	project := body["by_project"].([]any)[0].(map[string]any)
	assert.Equal(t, "Relaunch", project["name"])
	assert.Equal(t, "Relaunch", project["project_name"])
	assert.Equal(t, float64(960), project["minutes"])
}

func TestHandleGetTimeAnalytics_DefaultsToWeekAndEmptyListsAreNotNull(t *testing.T) {
	body := getTimeAnalyticsJSON(t, "/api/v1/hr/time/analytics?range=bogus", &hrv1.GetTimeAnalyticsResp{})

	assert.Equal(t, "week", body["range"])
	assert.Equal(t, float64(0), body["worked_days"])
	assert.Equal(t, []any{}, body["day_trend"])
	assert.Equal(t, []any{}, body["by_project"])
}
