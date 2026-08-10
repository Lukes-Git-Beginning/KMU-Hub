package caldav

import (
	"bytes"
	"strings"
	"testing"
	"time"

	ical "github.com/emersion/go-ical"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// encodeDecode runs a Calendar through the real wire format (ICS text) and
// back, exactly like a CalDAV client/server would, instead of comparing
// in-memory ical.Calendar structs directly.
func encodeDecode(t *testing.T, cal *ical.Calendar) *ical.Calendar {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, ical.NewEncoder(&buf).Encode(cal))
	decoded, err := ical.NewDecoder(&buf).Decode()
	require.NoError(t, err)
	return decoded
}

func baseEvent() *models.CalendarEvent {
	loc, _ := time.LoadLocation("Europe/Berlin")
	return &models.CalendarEvent{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		CalendarID: uuid.New(),
		Title:      "Kickoff Meeting",
		StartTime:  time.Date(2026, 3, 10, 9, 0, 0, 0, loc),
		EndTime:    time.Date(2026, 3, 10, 10, 0, 0, 0, loc),
		Timezone:   "Europe/Berlin",
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func TestEventToICal_RoundTrip_Normal(t *testing.T) {
	event := baseEvent()
	desc := "Quarterly planning"
	loc := "Room 4"
	event.Description = &desc
	event.Location = &loc

	cal, err := EventToICal(event, nil, nil)
	require.NoError(t, err)

	decoded := encodeDecode(t, cal)
	input, exceptions, err := ICalToEventInput(decoded)
	require.NoError(t, err)
	assert.Empty(t, exceptions)

	assert.Equal(t, event.Title, input.Title)
	require.NotNil(t, input.Description)
	assert.Equal(t, desc, *input.Description)
	require.NotNil(t, input.Location)
	assert.Equal(t, loc, *input.Location)
	assert.False(t, input.IsAllDay)
	assert.Equal(t, "Europe/Berlin", input.Timezone)
	assert.True(t, event.StartTime.Equal(input.StartTime), "start time must survive the roundtrip: got %v want %v", input.StartTime, event.StartTime)
	assert.True(t, event.EndTime.Equal(input.EndTime), "end time must survive the roundtrip: got %v want %v", input.EndTime, event.EndTime)
	assert.Nil(t, input.RRule)
}

func TestEventToICal_RoundTrip_AllDay(t *testing.T) {
	event := baseEvent()
	event.IsAllDay = true
	event.StartTime = time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	event.EndTime = time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC)

	cal, err := EventToICal(event, nil, nil)
	require.NoError(t, err)

	decoded := encodeDecode(t, cal)
	input, _, err := ICalToEventInput(decoded)
	require.NoError(t, err)

	assert.True(t, input.IsAllDay)
	assert.Equal(t, event.StartTime.Format("20060102"), input.StartTime.Format("20060102"))
	assert.Equal(t, event.EndTime.Format("20060102"), input.EndTime.Format("20060102"))
}

func TestEventToICal_RoundTrip_Recurring(t *testing.T) {
	event := baseEvent()
	rrule := "FREQ=WEEKLY;BYDAY=TU;COUNT=5"
	event.RRule = &rrule

	cal, err := EventToICal(event, nil, nil)
	require.NoError(t, err)

	decoded := encodeDecode(t, cal)
	input, _, err := ICalToEventInput(decoded)
	require.NoError(t, err)

	require.NotNil(t, input.RRule)
	assert.Equal(t, rrule, *input.RRule)
}

func TestEventToICal_RoundTrip_TimezoneSurvives(t *testing.T) {
	event := baseEvent()
	event.Timezone = "America/New_York"
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	event.StartTime = time.Date(2026, 6, 1, 14, 0, 0, 0, loc)
	event.EndTime = time.Date(2026, 6, 1, 15, 0, 0, 0, loc)

	cal, err := EventToICal(event, nil, nil)
	require.NoError(t, err)

	decoded := encodeDecode(t, cal)
	input, _, err := ICalToEventInput(decoded)
	require.NoError(t, err)

	assert.Equal(t, "America/New_York", input.Timezone)
	assert.True(t, event.StartTime.Equal(input.StartTime))
}

func TestEventToICal_RoundTrip_Attendees(t *testing.T) {
	event := baseEvent()
	attendees := []models.EventAttendee{
		{
			EventID:    event.ID,
			UserID:     uuid.New(),
			RSVPStatus: models.RSVPAccepted,
			FirstName:  "Anna",
			LastName:   "Muster",
		},
		{
			EventID:    event.ID,
			UserID:     uuid.New(),
			RSVPStatus: models.RSVPDeclined,
			FirstName:  "Ben",
			LastName:   "Beispiel",
		},
	}

	cal, err := EventToICal(event, nil, attendees)
	require.NoError(t, err)

	decoded := encodeDecode(t, cal)
	events := decoded.Events()
	require.Len(t, events, 1)

	attendeeProps := events[0].Props.Values(ical.PropAttendee)
	require.Len(t, attendeeProps, 2)

	// ATTENDEE is not surfaced back through CalEventInput (there's no field
	// for it), but it must survive at the iCalendar wire-format level -- CN
	// and PARTSTAT are what a real CalDAV client renders in its UI.
	got := map[string]string{}
	for _, p := range attendeeProps {
		got[p.Params.Get(ical.ParamCommonName)] = p.Params.Get(ical.ParamParticipationStatus)
	}
	assert.Equal(t, "ACCEPTED", got["Anna Muster"])
	assert.Equal(t, "DECLINED", got["Ben Beispiel"])
}

func TestRsvpToPartStat(t *testing.T) {
	cases := []struct {
		rsvp string
		want string
	}{
		{models.RSVPAccepted, "ACCEPTED"},
		{models.RSVPDeclined, "DECLINED"},
		{models.RSVPTentative, "TENTATIVE"},
		{models.RSVPPending, "NEEDS-ACTION"},
		{"", "NEEDS-ACTION"},
		{"garbage", "NEEDS-ACTION"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, rsvpToPartStat(tc.rsvp), "rsvp=%q", tc.rsvp)
	}
}

func TestEventToICal_RoundTrip_CancelledException(t *testing.T) {
	event := baseEvent()
	rrule := "FREQ=WEEKLY;COUNT=10"
	event.RRule = &rrule

	excDate := event.StartTime.AddDate(0, 0, 7)
	exceptions := []models.EventException{
		{
			ID:           uuid.New(),
			EventID:      event.ID,
			OriginalDate: excDate,
			IsCancelled:  true,
			UpdatedAt:    time.Now(),
		},
	}

	cal, err := EventToICal(event, exceptions, nil)
	require.NoError(t, err)

	decoded := encodeDecode(t, cal)
	_, parsedExceptions, err := ICalToEventInput(decoded)
	require.NoError(t, err)
	require.Len(t, parsedExceptions, 1)

	assert.True(t, parsedExceptions[0].IsCancelled)
	assert.Equal(t, excDate.Format("20060102T150405"), parsedExceptions[0].OriginalDate.Format("20060102T150405"))
}

func TestEventToICal_RoundTrip_OverriddenException(t *testing.T) {
	event := baseEvent()
	rrule := "FREQ=WEEKLY;COUNT=10"
	event.RRule = &rrule

	origDate := event.StartTime.AddDate(0, 0, 7)
	overriddenTitle := "Verschobenes Meeting"
	overriddenStart := origDate.Add(2 * time.Hour)
	overriddenEnd := overriddenStart.Add(time.Hour)
	exceptions := []models.EventException{
		{
			ID:           uuid.New(),
			EventID:      event.ID,
			OriginalDate: origDate,
			IsCancelled:  false,
			Title:        &overriddenTitle,
			StartTime:    &overriddenStart,
			EndTime:      &overriddenEnd,
			UpdatedAt:    time.Now(),
		},
	}

	cal, err := EventToICal(event, exceptions, nil)
	require.NoError(t, err)

	decoded := encodeDecode(t, cal)
	_, parsedExceptions, err := ICalToEventInput(decoded)
	require.NoError(t, err)
	require.Len(t, parsedExceptions, 1)

	got := parsedExceptions[0]
	assert.False(t, got.IsCancelled)
	require.NotNil(t, got.Title)
	assert.Equal(t, overriddenTitle, *got.Title)
	require.NotNil(t, got.StartTime)
	assert.True(t, overriddenStart.Equal(*got.StartTime))
	require.NotNil(t, got.EndTime)
	assert.True(t, overriddenEnd.Equal(*got.EndTime))
}

func TestEventToICal_EmptyDescriptionAndLocation_NoCrash(t *testing.T) {
	event := baseEvent()
	empty := ""
	event.Description = &empty
	event.Location = &empty

	cal, err := EventToICal(event, nil, nil)
	require.NoError(t, err)

	decoded := encodeDecode(t, cal)
	input, _, err := ICalToEventInput(decoded)
	require.NoError(t, err)

	// EventToICal skips empty description/location on write, so they must
	// come back nil rather than a pointer to an empty string.
	assert.Nil(t, input.Description)
	assert.Nil(t, input.Location)
}

func TestICalToEventInput_NoVEvent_ReturnsError(t *testing.T) {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")

	input, exceptions, err := ICalToEventInput(cal)
	assert.Error(t, err)
	assert.Nil(t, input)
	assert.Nil(t, exceptions)
}

func TestICalToEventInput_MissingDTStart_ReturnsError(t *testing.T) {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	vevent := ical.NewEvent()
	vevent.Props.SetText(ical.PropUID, "test-uid")
	vevent.Props.SetText(ical.PropSummary, "No DTSTART")
	cal.Children = append(cal.Children, vevent.Component)

	input, exceptions, err := ICalToEventInput(cal)
	assert.Error(t, err)
	assert.Nil(t, input)
	assert.Nil(t, exceptions)
}

func TestDecoder_MalformedICS_ReturnsErrorNotPanic(t *testing.T) {
	malformed := "this is not a valid ICS document at all\r\n\x00\x01garbage"

	assert.NotPanics(t, func() {
		_, err := ical.NewDecoder(strings.NewReader(malformed)).Decode()
		assert.Error(t, err)
	})
}

func TestDecoder_TruncatedVEvent_ReturnsErrorNotPanic(t *testing.T) {
	// A VEVENT that opens but never closes, and is missing required
	// properties -- a real-world case of a client sending a truncated body.
	truncated := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:abc\r\n"

	var decoded *ical.Calendar
	var decodeErr error
	assert.NotPanics(t, func() {
		decoded, decodeErr = ical.NewDecoder(strings.NewReader(truncated)).Decode()
	})
	if decodeErr == nil {
		// Some decoders tolerate a missing END: and still round-trip -- if
		// so, ICalToEventInput must still fail cleanly (no DTSTART) rather
		// than panic.
		assert.NotPanics(t, func() {
			_, _, err := ICalToEventInput(decoded)
			assert.Error(t, err)
		})
	}
}
