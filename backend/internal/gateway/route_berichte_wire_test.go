package gateway

// Berichte definition, schedule and run routes as the report builder and the chart cards read
// them (berichte-types.ts): `query_config`/`params` as JSON objects, the run `result` with
// `columns`/`rows`/`series`/`meta` directly readable. A loopback gRPC server plays the service.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	berichtev1 "github.com/kmuhub/kmuhub/proto/berichte/v1"
)

type berichteWireStub struct {
	berichtev1.UnimplementedBerichteServiceServer
	empty         bool
	badPayload    bool
	gotQuery      []byte
	gotParams     []byte
	gotSchedParam []byte
}

var berichteWireTime = timestamppb.New(time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC))

const berichteQueryJSON = `{"sourceId":"crm.contacts","viz":"bar"}`

func wireDefinition() *berichtev1.Definition {
	by := "u1"
	return &berichtev1.Definition{
		Id: "d1", TenantId: "t1", Name: "Umsatz", Module: "finanzen", Kind: "custom",
		QueryConfig: []byte(berichteQueryJSON), DefaultFormat: "pdf", CreatedBy: &by, IsPublished: true,
		CreatedAt: berichteWireTime, UpdatedAt: berichteWireTime,
	}
}

func wireSchedule() *berichtev1.Schedule {
	status := "success"
	return &berichtev1.Schedule{
		Id: "s1", TenantId: "t1", DefinitionId: "d1", Name: "Montag", CronExpression: "0 8 * * 1",
		Recipients: []string{"a@example.com"}, Format: "pdf", Params: []byte(`{"year":2026}`), Active: true,
		LastRunAt: berichteWireTime, LastRunStatus: &status,
		CreatedAt: berichteWireTime, UpdatedAt: berichteWireTime,
	}
}

func (s *berichteWireStub) CreateDefinition(_ context.Context, req *berichtev1.CreateDefinitionRequest) (*berichtev1.DefinitionResponse, error) {
	s.gotQuery = req.GetQueryConfig()
	return &berichtev1.DefinitionResponse{Definition: wireDefinition()}, nil
}

func (s *berichteWireStub) GetDefinition(context.Context, *berichtev1.GetDefinitionRequest) (*berichtev1.DefinitionResponse, error) {
	d := wireDefinition()
	if s.empty {
		d.QueryConfig = nil
	}
	return &berichtev1.DefinitionResponse{Definition: d}, nil
}

func (s *berichteWireStub) UpdateDefinition(_ context.Context, req *berichtev1.UpdateDefinitionRequest) (*berichtev1.DefinitionResponse, error) {
	s.gotQuery = req.GetQueryConfig()
	return &berichtev1.DefinitionResponse{Definition: wireDefinition()}, nil
}

func (s *berichteWireStub) ListDefinitions(context.Context, *berichtev1.ListDefinitionsRequest) (*berichtev1.ListDefinitionsResponse, error) {
	if s.empty {
		return &berichtev1.ListDefinitionsResponse{}, nil
	}
	return &berichtev1.ListDefinitionsResponse{Definitions: []*berichtev1.Definition{wireDefinition()}, Total: 1}, nil
}

func (s *berichteWireStub) RunReport(_ context.Context, req *berichtev1.RunReportRequest) (*berichtev1.RunReportResponse, error) {
	s.gotParams = req.GetParams()
	payload := `{"columns":[{"name":"revenue","label":"Umsatz","kind":"currency"}],` +
		`"rows":[{"revenue":120.5}],` +
		`"series":[{"label":"Umsatz","data_points":[{"label":"Jan","value":120.5}]}],` +
		`"totals":{"revenue":120.5},` +
		`"meta":{"generated_at":"2026-10-23T12:00:00Z","row_count":1,"definition_id":"d1","from_cache":false,"warning":"downstream_not_available"}}`
	if s.empty {
		payload = `{"columns":null,"rows":null,"meta":{}}`
	}
	if s.badPayload {
		payload = `not json`
	}
	dur, rows := int32(42), int32(1)
	return &berichtev1.RunReportResponse{
		Result: &berichtev1.ReportResult{Payload: []byte(payload), RowCount: 1, GeneratedAt: berichteWireTime, FromCache: true},
		Run: &berichtev1.Run{
			Id: "r1", TenantId: "t1", DefinitionId: "d1", Trigger: "manual", Params: []byte(`{"year":2026}`),
			DurationMs: &dur, RowCount: &rows, Status: "success", StartedAt: berichteWireTime,
		},
	}, nil
}

func (s *berichteWireStub) ListSchedules(context.Context, *berichtev1.ListSchedulesRequest) (*berichtev1.ListSchedulesResponse, error) {
	if s.empty {
		return &berichtev1.ListSchedulesResponse{}, nil
	}
	return &berichtev1.ListSchedulesResponse{Schedules: []*berichtev1.Schedule{wireSchedule()}, Total: 1}, nil
}

func (s *berichteWireStub) CreateSchedule(_ context.Context, req *berichtev1.CreateScheduleRequest) (*berichtev1.ScheduleResponse, error) {
	s.gotSchedParam = req.GetParams()
	return &berichtev1.ScheduleResponse{Schedule: wireSchedule()}, nil
}

func (s *berichteWireStub) UpdateSchedule(_ context.Context, req *berichtev1.UpdateScheduleRequest) (*berichtev1.ScheduleResponse, error) {
	s.gotSchedParam = req.GetParams()
	return &berichtev1.ScheduleResponse{Schedule: wireSchedule()}, nil
}

func (s *berichteWireStub) ToggleSchedule(context.Context, *berichtev1.ToggleScheduleRequest) (*berichtev1.ScheduleResponse, error) {
	sched := wireSchedule()
	sched.Active = false
	if s.empty {
		sched.Recipients = nil
		sched.Params = nil
		sched.LastRunAt = nil
		sched.LastRunStatus = nil
	}
	return &berichtev1.ScheduleResponse{Schedule: sched}, nil
}

func newBerichteWireRoutes(t *testing.T, stub *berichteWireStub) *BerichteRoutes {
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
	return NewBerichteRoutes(reg, berichteFlagsON())
}

func berichteWireCall(t *testing.T, h http.HandlerFunc, method string, body any, want int) map[string]any {
	t.Helper()
	req := httptest.NewRequest(method, "/x", nil)
	if body != nil {
		req = httptest.NewRequest(method, "/x", jsonBody(t, body))
	}
	req = withAuth(req, testHRUserID, testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	rec := httptest.NewRecorder()
	h(rec, req)
	assertStatus(t, rec, want)
	return decodeBody(t, rec)
}

func assertWireDefinition(t *testing.T, got map[string]any) {
	t.Helper()
	assert.Equal(t, "d1", got["id"])
	cfg, ok := got["query_config"].(map[string]any)
	require.True(t, ok, "query_config must be a JSON object, got %T", got["query_config"])
	assert.Equal(t, "crm.contacts", cfg["sourceId"])
	assert.Equal(t, "bar", cfg["viz"])
	assert.Equal(t, "u1", got["created_by"])
	assert.Equal(t, true, got["is_published"])
}

func TestBerichteWire_DefinitionsCarryQueryConfigAsObject(t *testing.T) {
	br := newBerichteWireRoutes(t, &berichteWireStub{})

	list := berichteWireCall(t, br.HandleListDefinitions, http.MethodGet, nil, http.StatusOK)
	items, ok := list["definitions"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	assert.EqualValues(t, 1, list["total"])
	assertWireDefinition(t, items[0].(map[string]any))

	for name, h := range map[string]http.HandlerFunc{"get": br.HandleGetDefinition, "update": br.HandleUpdateDefinition} {
		got := berichteWireCall(t, h, http.MethodPatch, map[string]any{"name": "x"}, http.StatusOK)
		def, ok := got["definition"].(map[string]any)
		require.True(t, ok, name)
		assertWireDefinition(t, def)
	}

	created := berichteWireCall(t, br.HandleCreateDefinition, http.MethodPost,
		map[string]any{"name": "Umsatz", "module": "finanzen"}, http.StatusCreated)
	assertWireDefinition(t, created["definition"].(map[string]any))
}

func TestBerichteWire_EmptyListsAreArraysAndEmptyConfigIsObject(t *testing.T) {
	br := newBerichteWireRoutes(t, &berichteWireStub{empty: true})

	defs := berichteWireCall(t, br.HandleListDefinitions, http.MethodGet, nil, http.StatusOK)
	assert.Equal(t, []any{}, defs["definitions"])
	assert.EqualValues(t, 0, defs["total"])

	scheds := berichteWireCall(t, br.HandleListSchedules, http.MethodGet, nil, http.StatusOK)
	assert.Equal(t, []any{}, scheds["schedules"])

	got := berichteWireCall(t, br.HandleGetDefinition, http.MethodGet, nil, http.StatusOK)
	assert.Equal(t, map[string]any{}, got["definition"].(map[string]any)["query_config"])

	toggled := berichteWireCall(t, br.HandleToggleSchedule, http.MethodPatch, map[string]any{"active": false}, http.StatusOK)
	sched := toggled["schedule"].(map[string]any)
	assert.Equal(t, []any{}, sched["recipients"])
	assert.Equal(t, map[string]any{}, sched["params"])
	assert.Nil(t, sched["last_run_at"])
	assert.Contains(t, sched, "last_run_at")
	assert.Contains(t, sched, "last_run_status")
	assert.Equal(t, false, sched["active"])
}

func TestBerichteWire_CreateAndUpdateReadQueryConfigAsObject(t *testing.T) {
	stub := &berichteWireStub{}
	br := newBerichteWireRoutes(t, stub)

	cfg := map[string]any{"sourceId": "crm.contacts", "viz": "bar"}
	berichteWireCall(t, br.HandleCreateDefinition, http.MethodPost,
		map[string]any{"name": "Umsatz", "module": "finanzen", "query_config": cfg}, http.StatusCreated)
	assert.JSONEq(t, berichteQueryJSON, string(stub.gotQuery))

	stub.gotQuery = nil
	berichteWireCall(t, br.HandleUpdateDefinition, http.MethodPatch, map[string]any{"query_config": cfg}, http.StatusOK)
	assert.JSONEq(t, berichteQueryJSON, string(stub.gotQuery))
}

func TestBerichteWire_Base64QueryConfigIsStillRead(t *testing.T) {
	stub := &berichteWireStub{}
	br := newBerichteWireRoutes(t, stub)

	// "e30=" is base64 for "{}": the body shape that predates the object form.
	berichteWireCall(t, br.HandleCreateDefinition, http.MethodPost,
		map[string]any{"name": "Umsatz", "module": "finanzen", "query_config": "e30="}, http.StatusCreated)
	assert.Equal(t, "{}", string(stub.gotQuery))
}

func TestBerichteWire_RunResultIsDirectlyReadable(t *testing.T) {
	stub := &berichteWireStub{}
	br := newBerichteWireRoutes(t, stub)

	got := berichteWireCall(t, br.HandleRunReport, http.MethodPost,
		map[string]any{"params": map[string]any{"year": 2026}}, http.StatusOK)
	assert.JSONEq(t, `{"year":2026}`, string(stub.gotParams))

	result, ok := got["result"].(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, result, "payload")

	cols := result["columns"].([]any)
	require.Len(t, cols, 1)
	assert.Equal(t, map[string]any{"key": "revenue", "label": "Umsatz", "type": "currency"}, cols[0])

	rows := result["rows"].([]any)
	require.Len(t, rows, 1)
	assert.EqualValues(t, 120.5, rows[0].(map[string]any)["revenue"])

	series := result["series"].([]any)
	require.Len(t, series, 1)
	s0 := series[0].(map[string]any)
	assert.Equal(t, "Umsatz", s0["id"])
	assert.Equal(t, []any{map[string]any{"label": "Jan", "value": 120.5}}, s0["data"])

	assert.Equal(t, map[string]any{"revenue": 120.5}, result["totals"])

	meta := result["meta"].(map[string]any)
	assert.EqualValues(t, 1, meta["row_count"])
	assert.Equal(t, true, meta["from_cache"], "the proto flag wins over the cached payload")
	assert.Equal(t, "d1", meta["definition_id"])
	assert.Equal(t, "2026-10-23T12:00:00Z", meta["generated_at"])
	assert.Equal(t, "downstream_not_available", meta["warning"])

	run := got["run"].(map[string]any)
	assert.Equal(t, "manual", run["trigger"])
	assert.Equal(t, map[string]any{"year": 2026.0}, run["params"])
	assert.EqualValues(t, 42, run["duration_ms"])
	assert.Nil(t, run["schedule_id"])
	assert.Contains(t, run, "schedule_id")
	assert.Contains(t, run, "completed_at")
	assert.Nil(t, run["error"])
}

func TestBerichteWire_RunWithoutRowsKeepsEmptyArrays(t *testing.T) {
	br := newBerichteWireRoutes(t, &berichteWireStub{empty: true})

	got := berichteWireCall(t, br.HandleRunReport, http.MethodPost, map[string]any{}, http.StatusOK)
	result := got["result"].(map[string]any)
	assert.Equal(t, []any{}, result["columns"])
	assert.Equal(t, []any{}, result["rows"])
	assert.Equal(t, []any{}, result["series"])
	assert.Equal(t, "d1", result["meta"].(map[string]any)["definition_id"])
}

func TestBerichteWire_RunWithInvalidPayloadIs500(t *testing.T) {
	br := newBerichteWireRoutes(t, &berichteWireStub{badPayload: true})

	got := berichteWireCall(t, br.HandleRunReport, http.MethodPost, map[string]any{}, http.StatusInternalServerError)
	assert.NotContains(t, got, "result")
}

func TestBerichteWire_SchedulesCarryParamsAsObject(t *testing.T) {
	stub := &berichteWireStub{}
	br := newBerichteWireRoutes(t, stub)

	list := berichteWireCall(t, br.HandleListSchedules, http.MethodGet, nil, http.StatusOK)
	items := list["schedules"].([]any)
	require.Len(t, items, 1)
	s0 := items[0].(map[string]any)
	assert.Equal(t, map[string]any{"year": 2026.0}, s0["params"])
	assert.Equal(t, []any{"a@example.com"}, s0["recipients"])
	assert.Equal(t, "success", s0["last_run_status"])
	assert.Equal(t, "2026-10-23T12:00:00Z", s0["last_run_at"])
	assert.Nil(t, s0["last_run_error"])
	assert.Contains(t, s0, "last_run_error")

	created := berichteWireCall(t, br.HandleCreateSchedule, http.MethodPost, map[string]any{
		"definition_id": testSchemaID, "name": "Montag", "cron_expression": "0 8 * * 1",
		"params": map[string]any{"year": 2026}, "active": true,
	}, http.StatusCreated)
	assert.JSONEq(t, `{"year":2026}`, string(stub.gotSchedParam))
	assert.Equal(t, map[string]any{"year": 2026.0}, created["schedule"].(map[string]any)["params"])

	stub.gotSchedParam = nil
	berichteWireCall(t, br.HandleUpdateSchedule, http.MethodPatch,
		map[string]any{"params": map[string]any{"year": 2026}}, http.StatusOK)
	assert.JSONEq(t, `{"year":2026}`, string(stub.gotSchedParam))
}
