package gateway

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	calv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
)

// Handler level: a real gRPC server on loopback stands in for the work service,
// so the assertions run against what the handler writes to the wire and what it
// forwards over gRPC.

type calendarStub struct {
	calv1.UnimplementedCalendarServiceServer
	gotRange     *calv1.ListEventsInRangeRequest
	gotRecurring *calv1.UpdateRecurringEventRequest
	gotCreate    *calv1.CreateEventRequest
	gotUpdate    *calv1.UpdateEventRequest
	gotHolidays  *calv1.ListHolidaysRequest
}

func (s *calendarStub) ListEventsInRange(_ context.Context, r *calv1.ListEventsInRangeRequest) (*calv1.ListEventsInRangeResponse, error) {
	s.gotRange = r
	return &calv1.ListEventsInRangeResponse{Events: []*calv1.ExpandedEventProto{{
		Event: &calv1.CalendarEventProto{
			Id: "e-1", Title: "Standup",
			StartTime: timestamppb.New(time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)),
		},
		DisplayColor: "#abcdef",
	}}}, nil
}

func (s *calendarStub) GetEvent(context.Context, *calv1.GetEventRequest) (*calv1.GetEventResponse, error) {
	return &calv1.GetEventResponse{Event: &calv1.ExpandedEventProto{
		Event: &calv1.CalendarEventProto{Id: "e-2", Title: "Einzeln"}, CalendarName: "Team",
	}}, nil
}

func (s *calendarStub) UpdateRecurringEvent(_ context.Context, r *calv1.UpdateRecurringEventRequest) (*calv1.UpdateRecurringEventResponse, error) {
	s.gotRecurring = r
	return &calv1.UpdateRecurringEventResponse{Event: &calv1.CalendarEventProto{Id: r.EventId}}, nil
}

func (s *calendarStub) CreateEvent(_ context.Context, r *calv1.CreateEventRequest) (*calv1.CreateEventResponse, error) {
	s.gotCreate = r
	return &calv1.CreateEventResponse{Event: &calv1.CalendarEventProto{Id: "new"}}, nil
}

func (s *calendarStub) UpdateEvent(_ context.Context, r *calv1.UpdateEventRequest) (*calv1.UpdateEventResponse, error) {
	s.gotUpdate = r
	return &calv1.UpdateEventResponse{Event: &calv1.CalendarEventProto{Id: r.Id}}, nil
}

func (s *calendarStub) ListHolidays(_ context.Context, r *calv1.ListHolidaysRequest) (*calv1.ListHolidaysResponse, error) {
	s.gotHolidays = r
	return &calv1.ListHolidaysResponse{Holidays: []*calv1.PublicHolidayProto{{Id: "h-1", Date: "2026-01-01", Name: "New Year"}}}, nil
}

func newCalendarStubRoutes(t *testing.T) (*CalendarRoutes, *calendarStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &calendarStub{}
	srv := grpc.NewServer()
	calv1.RegisterCalendarServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("work", lis.Addr().String())
	return NewCalendarRoutes(reg), stub
}

func TestHandleListEventsInRange_WritesFlatEventsAndInclusiveEnd(t *testing.T) {
	routes, stub := newCalendarStubRoutes(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events?start=2026-03-02&end=2026-03-08", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListEventsInRange(rec, req)
	assertStatus(t, rec, http.StatusOK)

	var got struct {
		Events []map[string]any `json:"events"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Events, 1)
	assert.Equal(t, "e-1", got.Events[0]["id"])
	assert.Equal(t, "2026-03-02T09:00:00Z", got.Events[0]["start_time"])
	assert.Equal(t, "#abcdef", got.Events[0]["display_color"])
	assert.NotContains(t, got.Events[0], "event")

	require.NotNil(t, stub.gotRange)
	assert.Equal(t, time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC), stub.gotRange.End.AsTime(),
		"the last visible day (Sunday) must be inside the range")
}

func TestHandleGetEvent_WritesFlatEvent(t *testing.T) {
	routes, _ := newCalendarStubRoutes(t)
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetEvent(rec, req)
	assertStatus(t, rec, http.StatusOK)

	var got struct {
		Event map[string]any `json:"event"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "e-2", got.Event["id"])
	assert.Equal(t, "Team", got.Event["calendar_name"])
	assert.NotContains(t, got.Event, "event")
}

func TestHandleListHolidays_YearReachesRPCAsFullYear(t *testing.T) {
	routes, stub := newCalendarStubRoutes(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/holidays?year=2026&country_code=DE&subdivision_code=BY", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListHolidays(rec, req)
	assertStatus(t, rec, http.StatusOK)

	require.NotNil(t, stub.gotHolidays)
	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), stub.gotHolidays.Start.AsTime())
	assert.Equal(t, time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), stub.gotHolidays.End.AsTime())
	assert.Contains(t, rec.Body.String(), `"name":"New Year"`)
}

func TestHandleUpdateRecurringEvent_FlatBodyReachesRPC(t *testing.T) {
	routes, stub := newCalendarStubRoutes(t)
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/events/"+id+"/recurring",
		strings.NewReader(`{"scope":"all","original_date":"2026-03-02","title":"Neu","rrule":"FREQ=WEEKLY"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateRecurringEvent(rec, req)
	assertStatus(t, rec, http.StatusOK)

	require.NotNil(t, stub.gotRecurring)
	require.NotNil(t, stub.gotRecurring.Title)
	assert.Equal(t, "Neu", *stub.gotRecurring.Title)
	assert.Equal(t, "2026-03-02", stub.gotRecurring.OriginalDate)
	assert.Nil(t, stub.gotRecurring.Rrule, "the lossy FE rrule must not overwrite the series rule")
}

func TestHandleCreateAndUpdateEvent_PassReminderAndRRule(t *testing.T) {
	routes, stub := newCalendarStubRoutes(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/events", strings.NewReader(
		`{"calendar_id":"11111111-1111-1111-1111-111111111111","title":"t","start_time":"2026-03-02T09:00:00Z","end_time":"2026-03-02T10:00:00Z","reminder_minutes":[15,60]}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateEvent(rec, req)
	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.gotCreate)
	assert.Equal(t, []int32{15, 60}, stub.gotCreate.ReminderMinutes)

	rec = httptest.NewRecorder()
	id := uuid.New().String()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/calendar/events/"+id, strings.NewReader(`{"rrule":"FREQ=DAILY"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateEvent(rec, req)
	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.gotUpdate.Rrule)
	assert.Equal(t, "FREQ=DAILY", *stub.gotUpdate.Rrule)
}
