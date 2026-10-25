package gateway

// "Jetzt ausfuehren" on a report schedule: the gateway finds the schedule through ListSchedules
// (there is no GetSchedule RPC) and runs its definition with the schedule's own params. The tests
// read the requests the berichte service actually receives.

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	berichtev1 "github.com/kmuhub/kmuhub/proto/berichte/v1"
)

const (
	testScheduleRunID   = "550e8400-e29b-41d4-a716-446655440000"
	testScheduleRunDef  = "660e8400-e29b-41d4-a716-446655440000"
	testScheduleRunUser = "33333333-3333-3333-3333-333333333333"
)

type scheduleRunStub struct {
	berichtev1.UnimplementedBerichteServiceServer
	schedules []*berichtev1.Schedule
	listCalls []int32
	runs      []*berichtev1.RunReportRequest
}

func (s *scheduleRunStub) ListSchedules(_ context.Context, r *berichtev1.ListSchedulesRequest) (*berichtev1.ListSchedulesResponse, error) {
	s.listCalls = append(s.listCalls, r.GetPage())
	from := int(r.GetPage()-1) * int(r.GetPageSize())
	to := from + int(r.GetPageSize())
	if from > len(s.schedules) {
		from = len(s.schedules)
	}
	if to > len(s.schedules) {
		to = len(s.schedules)
	}
	return &berichtev1.ListSchedulesResponse{Schedules: s.schedules[from:to], Total: int32(len(s.schedules))}, nil
}

func (s *scheduleRunStub) RunReport(_ context.Context, r *berichtev1.RunReportRequest) (*berichtev1.RunReportResponse, error) {
	s.runs = append(s.runs, r)
	return &berichtev1.RunReportResponse{}, nil
}

func newScheduleRunRouter(t *testing.T, stub *scheduleRunStub) http.Handler {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	berichtev1.RegisterBerichteServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("berichte", lis.Addr().String())
	t.Cleanup(reg.Close)

	r := chi.NewRouter()
	NewBerichteRoutes(reg, berichteFlagsON()).RegisterRoutes(r, func(next http.Handler) http.Handler { return next })
	return r
}

func scheduleRunRequest(id string, perms ...string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/berichte/schedules/"+id+"/run", nil)
	return withPermissions(withAuth(req, testScheduleRunUser, testTenantID), perms...)
}

func fillerSchedules(n int) []*berichtev1.Schedule {
	out := make([]*berichtev1.Schedule, n)
	for i := range out {
		out[i] = &berichtev1.Schedule{Id: fmt.Sprintf("00000000-0000-4000-8000-%012d", i), DefinitionId: testScheduleRunDef}
	}
	return out
}

func TestBerichteWire_RunScheduleRunsTheDefinitionWithTheSchedulesParams(t *testing.T) {
	stub := &scheduleRunStub{schedules: []*berichtev1.Schedule{{
		Id:           testScheduleRunID,
		DefinitionId: testScheduleRunDef,
		Name:         "Monatsbericht",
		Params:       []byte(`{"year":2026}`),
		Active:       false, // an inactive schedule may still run by hand
	}}}
	router := newScheduleRunRouter(t, stub)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, scheduleRunRequest(testScheduleRunID, "berichte:schedule:manage"))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Len(t, stub.runs, 1)
	assert.Equal(t, testScheduleRunDef, stub.runs[0].GetDefinitionId())
	assert.JSONEq(t, `{"year":2026}`, string(stub.runs[0].GetParams()))
	assert.Equal(t, "manual", stub.runs[0].GetTrigger())
	assert.Equal(t, testTenantID.String(), stub.runs[0].GetTenantId())

	var body struct {
		Schedule struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"schedule"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, testScheduleRunID, body.Schedule.ID)
	assert.Equal(t, "Monatsbericht", body.Schedule.Name)
}

func TestBerichteWire_RunScheduleFindsAScheduleOnLaterPages(t *testing.T) {
	schedules := append(fillerSchedules(scheduleLookupPageSize),
		&berichtev1.Schedule{Id: testScheduleRunID, DefinitionId: testScheduleRunDef, Params: []byte(`{}`)})
	stub := &scheduleRunStub{schedules: schedules}
	router := newScheduleRunRouter(t, stub)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, scheduleRunRequest(testScheduleRunID, "berichte:schedule:manage"))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, []int32{1, 2}, stub.listCalls)
	assert.Len(t, stub.runs, 1)
}

func TestBerichteWire_RunScheduleUnknownIDIs404WithoutRunning(t *testing.T) {
	stub := &scheduleRunStub{schedules: fillerSchedules(3)}
	router := newScheduleRunRouter(t, stub)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, scheduleRunRequest(testScheduleRunID, "berichte:schedule:manage"))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, stub.runs)
	assert.Equal(t, []int32{1}, stub.listCalls, "a short first page ends the walk")
}

func TestBerichteWire_RunScheduleInvalidUUIDIs400(t *testing.T) {
	stub := &scheduleRunStub{}
	router := newScheduleRunRouter(t, stub)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, scheduleRunRequest("not-a-uuid", "berichte:schedule:manage"))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, stub.listCalls)
}

func TestBerichteWire_RunScheduleNeedsTheScheduleKey(t *testing.T) {
	stub := &scheduleRunStub{schedules: []*berichtev1.Schedule{{Id: testScheduleRunID, DefinitionId: testScheduleRunDef}}}
	router := newScheduleRunRouter(t, stub)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, scheduleRunRequest(testScheduleRunID, "berichte:reports:read"))
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Empty(t, stub.runs)

	// the legacy coarse key keeps old tokens working (additive guard)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, scheduleRunRequest(testScheduleRunID, "berichte:reports:write"))
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}
