package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	calv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
)

// The desktop reads ExpandedEvent flat (event.id, event.start_time) -- these
// tests decode the real wire JSON, not the proto type.

func TestFlattenExpandedEvents_FieldsOnElementLevel(t *testing.T) {
	start := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	events := []*calv1.ExpandedEventProto{{
		Event: &calv1.CalendarEventProto{
			Id: "e-1", CalendarId: "c-1", Title: "Standup",
			StartTime: timestamppb.New(start), EndTime: timestamppb.New(start.Add(time.Hour)),
		},
		CalendarName: "Team", DisplayColor: "#123456", AttendeeCount: 3,
	}}

	out, err := flattenExpandedEvents(events)
	require.NoError(t, err)
	body, err := json.Marshal(map[string]any{"events": out})
	require.NoError(t, err)

	var got struct {
		Events []map[string]any `json:"events"`
	}
	require.NoError(t, json.Unmarshal(body, &got))
	require.Len(t, got.Events, 1)
	e := got.Events[0]
	assert.Equal(t, "e-1", e["id"])
	assert.Equal(t, "Standup", e["title"])
	assert.Equal(t, "2026-03-02T09:00:00Z", e["start_time"])
	assert.Equal(t, "Team", e["calendar_name"])
	assert.Equal(t, "#123456", e["display_color"])
	assert.EqualValues(t, 3, e["attendee_count"])
	assert.NotContains(t, e, "event", "the nested wrapper must be gone")
}

func TestFlattenExpandedEvents_EmptyIsArrayNotNull(t *testing.T) {
	out, err := flattenExpandedEvents(nil)
	require.NoError(t, err)
	body, err := json.Marshal(map[string]any{"events": out})
	require.NoError(t, err)
	assert.JSONEq(t, `{"events":[]}`, string(body))
}

func TestParseRangeEnd_BareDateIsInclusive(t *testing.T) {
	ts, err := parseRangeEnd("2026-03-08")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC), ts.AsTime())

	ts, err = parseRangeEnd("2026-03-08T12:00:00Z")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC), ts.AsTime(), "full timestamps stay as given")

	_, err = parseRangeEnd("garbage")
	require.Error(t, err)
}

func TestHolidayRange(t *testing.T) {
	t.Run("year selects the calendar year", func(t *testing.T) {
		s, e, err := holidayRange(url.Values{"year": {"2026"}, "country_code": {"DE"}})
		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), s.AsTime())
		assert.Equal(t, time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), e.AsTime())
	})
	t.Run("explicit start and end win over year", func(t *testing.T) {
		s, e, err := holidayRange(url.Values{"start": {"2026-05-01"}, "end": {"2026-05-31"}, "year": {"2030"}})
		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), s.AsTime())
		assert.Equal(t, time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC), e.AsTime())
	})
	t.Run("nothing given is an error, not a 1970 window", func(t *testing.T) {
		_, _, err := holidayRange(url.Values{})
		require.Error(t, err)
	})
	t.Run("invalid year", func(t *testing.T) {
		for _, y := range []string{"abc", "1999", "2101"} {
			_, _, err := holidayRange(url.Values{"year": {y}})
			require.Error(t, err, y)
		}
	})
}

func TestHandleListHolidays_RequiresYearOrRange(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/holidays?country_code=DE", nil)
	req = withAuth(req, "11111111-1111-1111-1111-111111111111", testTenantID)
	routes.HandleListHolidays(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func decodeRecurring(t *testing.T, body string) updateRecurringEventRequest {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/x", strings.NewReader(body))
	rec := httptest.NewRecorder()
	got, ok := decodeAndValidate[updateRecurringEventRequest](rec, req)
	require.True(t, ok, rec.Body.String())
	return got
}

func TestUpdateRecurringRequest_FlatBodyCarriesChanges(t *testing.T) {
	req := decodeRecurring(t, `{"scope":"all","original_date":"2026-03-02","title":"Neu","is_all_day":true,"start_time":"2026-03-02T08:00:00Z"}`)
	c := req.changeSet()
	require.NotNil(t, c.Title)
	assert.Equal(t, "Neu", *c.Title)
	require.NotNil(t, c.IsAllDay)
	assert.True(t, *c.IsAllDay)
	require.NotNil(t, c.StartTime)
}

func TestUpdateRecurringRequest_NestedChangesStillWork(t *testing.T) {
	req := decodeRecurring(t, `{"scope":"this","original_date":"2026-03-02","changes":{"title":"Nested"}}`)
	c := req.changeSet()
	require.NotNil(t, c.Title)
	assert.Equal(t, "Nested", *c.Title)
}

func TestCreateEventRequest_ReminderMinutesValidated(t *testing.T) {
	ok := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(
		`{"calendar_id":"11111111-1111-1111-1111-111111111111","title":"t","start_time":"a","end_time":"b","reminder_minutes":[15]}`))
	got, valid := decodeAndValidate[createEventRequest](httptest.NewRecorder(), ok)
	require.True(t, valid)
	assert.Equal(t, []int32{15}, got.ReminderMinutes)

	bad := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(
		`{"calendar_id":"11111111-1111-1111-1111-111111111111","title":"t","start_time":"a","end_time":"b","reminder_minutes":[0]}`))
	_, valid = decodeAndValidate[createEventRequest](httptest.NewRecorder(), bad)
	assert.False(t, valid)
}
