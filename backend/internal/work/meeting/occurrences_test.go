package meeting

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// stubSeries stands in for *event.Service. It hands back one calendar event
// and its exceptions, and can be told to fail either lookup so the error paths
// are reachable without a database.
type stubSeries struct {
	event      *models.CalendarEvent
	exceptions []models.EventException
	getErr     error
	excErr     error

	gotEventID  uuid.UUID
	gotTenantID uuid.UUID
	excStart    time.Time
	excEnd      time.Time
}

func (s *stubSeries) Get(_ context.Context, eventID, tenantID uuid.UUID) (*models.CalendarEvent, error) {
	s.gotEventID = eventID
	s.gotTenantID = tenantID
	if s.getErr != nil {
		return nil, s.getErr
	}
	cp := *s.event
	return &cp, nil
}

func (s *stubSeries) ListExceptions(_ context.Context, _ uuid.UUID, start, end time.Time) ([]models.EventException, error) {
	s.excStart, s.excEnd = start, end
	if s.excErr != nil {
		return nil, s.excErr
	}
	return s.exceptions, nil
}

// occurrenceFixture wires a scheduled meeting linked to a recurring calendar
// event. dtstart is both the event's start and the meeting's scheduled start;
// the meeting runs 30 minutes.
func occurrenceFixture(t *testing.T, rrule string, dtstart time.Time) (*Service, *stubSeries, uuid.UUID) {
	t.Helper()

	repo := newMockRepo()
	eventID := uuid.New()
	meetingID := uuid.New()

	repo.meetings[meetingID] = &Meeting{
		ID:              meetingID,
		TenantID:        testTenantID,
		Title:           "Weekly sync",
		OrganizerID:     uuid.New(),
		Status:          MeetingStatusScheduled,
		ScheduledStart:  dtstart,
		ScheduledEnd:    dtstart.Add(30 * time.Minute),
		CalendarEventID: &eventID,
	}

	rule := rrule
	series := &stubSeries{
		event: &models.CalendarEvent{
			ID:        eventID,
			TenantID:  testTenantID,
			Title:     "Weekly sync",
			StartTime: dtstart,
			EndTime:   dtstart.Add(time.Hour),
			RRule:     &rule,
		},
	}

	return NewService(repo).WithSeriesSource(series), series, meetingID
}

func TestListOccurrences_ExpandsLinkedCalendarSeries(t *testing.T) {
	dtstart := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	svc, series, meetingID := occurrenceFixture(t, "FREQ=WEEKLY", dtstart)

	occ, truncated, err := svc.ListOccurrences(context.Background(), meetingID, testTenantID,
		dtstart, dtstart.Add(21*24*time.Hour))
	require.NoError(t, err)
	assert.False(t, truncated)
	require.Len(t, occ, 4, "four weekly occurrences in a three-week window, both ends inclusive")

	for i, o := range occ {
		assert.Equal(t, meetingID, o.MeetingID)
		assert.Equal(t, series.event.ID, o.CalendarEventID)
		assert.Equal(t, dtstart.Add(time.Duration(i)*7*24*time.Hour), o.Start)
		// Length comes from the meeting (30 min), not from the calendar event
		// (60 min) -- the meeting is what repeats.
		assert.Equal(t, 30*time.Minute, o.End.Sub(o.Start))
	}

	// The series is read tenant-scoped, from the linked event.
	assert.Equal(t, testTenantID, series.gotTenantID)
	assert.Equal(t, series.event.ID, series.gotEventID)
}

func TestListOccurrences_SkipsCancelledAndHonoursMovedStart(t *testing.T) {
	dtstart := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	svc, series, meetingID := occurrenceFixture(t, "FREQ=WEEKLY", dtstart)

	cancelled := dtstart.Add(7 * 24 * time.Hour)
	moved := dtstart.Add(14 * 24 * time.Hour)
	movedTo := moved.Add(2 * time.Hour)
	series.exceptions = []models.EventException{
		{EventID: series.event.ID, OriginalDate: cancelled, IsCancelled: true},
		{EventID: series.event.ID, OriginalDate: moved, StartTime: &movedTo},
	}

	occ, _, err := svc.ListOccurrences(context.Background(), meetingID, testTenantID,
		dtstart, dtstart.Add(21*24*time.Hour))
	require.NoError(t, err)
	require.Len(t, occ, 3, "the cancelled occurrence is dropped")

	assert.Equal(t, dtstart, occ[0].Start)
	assert.Equal(t, movedTo, occ[1].Start, "moved occurrence reports its new start")
	assert.Equal(t, movedTo.Add(30*time.Minute), occ[1].End, "and keeps the meeting's length")
	assert.Equal(t, dtstart.Add(21*24*time.Hour), occ[2].Start)
}

func TestListOccurrences_CapsInstances(t *testing.T) {
	dtstart := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	svc, _, meetingID := occurrenceFixture(t, "FREQ=HOURLY", dtstart)

	// 30 days of hourly occurrences is 721 dates -- well past the cap.
	occ, truncated, err := svc.ListOccurrences(context.Background(), meetingID, testTenantID,
		dtstart, dtstart.Add(30*24*time.Hour))
	require.NoError(t, err)
	assert.True(t, truncated, "the caller must learn the window was cut short")
	assert.Len(t, occ, maxMeetingOccurrences)
}

func TestListOccurrences_ClampsToRecurrenceEnd(t *testing.T) {
	dtstart := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	svc, series, meetingID := occurrenceFixture(t, "FREQ=WEEKLY", dtstart)

	recurrenceEnd := dtstart.Add(8 * 24 * time.Hour) // covers two occurrences
	series.event.RecurrenceEnd = &recurrenceEnd

	occ, _, err := svc.ListOccurrences(context.Background(), meetingID, testTenantID,
		dtstart, dtstart.Add(365*24*time.Hour))
	require.NoError(t, err)
	assert.Len(t, occ, 2, "the series ends where the calendar says, not where the caller asks")
	assert.Equal(t, recurrenceEnd, series.excEnd, "exceptions are fetched for the clamped window")
}

func TestListOccurrences_EmptyWhenSeriesEndedBeforeWindow(t *testing.T) {
	dtstart := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	svc, series, meetingID := occurrenceFixture(t, "FREQ=WEEKLY", dtstart)

	ended := dtstart.Add(24 * time.Hour)
	series.event.RecurrenceEnd = &ended

	occ, truncated, err := svc.ListOccurrences(context.Background(), meetingID, testTenantID,
		dtstart.Add(30*24*time.Hour), dtstart.Add(60*24*time.Hour))
	require.NoError(t, err)
	assert.False(t, truncated)
	assert.NotNil(t, occ, "empty list, never nil")
	assert.Empty(t, occ)
}

func TestListOccurrences_ErrorPaths(t *testing.T) {
	dtstart := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	window := dtstart.Add(21 * 24 * time.Hour)
	downstream := errors.New("boom")

	t.Run("end not after start", func(t *testing.T) {
		svc, _, meetingID := occurrenceFixture(t, "FREQ=WEEKLY", dtstart)
		_, _, err := svc.ListOccurrences(context.Background(), meetingID, testTenantID, dtstart, dtstart)
		assert.ErrorIs(t, err, ErrInvalidTimeRange)
	})

	t.Run("series source not wired", func(t *testing.T) {
		svc, _, meetingID := occurrenceFixture(t, "FREQ=WEEKLY", dtstart)
		svc.series = nil
		_, _, err := svc.ListOccurrences(context.Background(), meetingID, testTenantID, dtstart, window)
		assert.ErrorIs(t, err, ErrSeriesUnavailable)
	})

	t.Run("unknown meeting", func(t *testing.T) {
		svc, _, _ := occurrenceFixture(t, "FREQ=WEEKLY", dtstart)
		_, _, err := svc.ListOccurrences(context.Background(), uuid.New(), testTenantID, dtstart, window)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("foreign tenant", func(t *testing.T) {
		svc, _, meetingID := occurrenceFixture(t, "FREQ=WEEKLY", dtstart)
		_, _, err := svc.ListOccurrences(context.Background(), meetingID, uuid.New(), dtstart, window)
		assert.ErrorIs(t, err, ErrNotFound, "a foreign tenant must not learn the meeting exists")
	})

	t.Run("meeting not linked to a calendar event", func(t *testing.T) {
		svc, _, meetingID := occurrenceFixture(t, "FREQ=WEEKLY", dtstart)
		repo := svc.repo.(*mockRepo)
		repo.meetings[meetingID].CalendarEventID = nil
		_, _, err := svc.ListOccurrences(context.Background(), meetingID, testTenantID, dtstart, window)
		assert.ErrorIs(t, err, ErrNotRecurring)
	})

	t.Run("linked event carries no rule", func(t *testing.T) {
		svc, series, meetingID := occurrenceFixture(t, "FREQ=WEEKLY", dtstart)
		blank := "   "
		series.event.RRule = &blank
		_, _, err := svc.ListOccurrences(context.Background(), meetingID, testTenantID, dtstart, window)
		assert.ErrorIs(t, err, ErrNotRecurring)
	})

	t.Run("unusable rule", func(t *testing.T) {
		svc, _, meetingID := occurrenceFixture(t, "FREQ=NEVER", dtstart)
		_, _, err := svc.ListOccurrences(context.Background(), meetingID, testTenantID, dtstart, window)
		assert.ErrorIs(t, err, ErrInvalidRecurrence)
	})

	t.Run("event lookup fails", func(t *testing.T) {
		svc, series, meetingID := occurrenceFixture(t, "FREQ=WEEKLY", dtstart)
		series.getErr = downstream
		_, _, err := svc.ListOccurrences(context.Background(), meetingID, testTenantID, dtstart, window)
		assert.ErrorIs(t, err, downstream)
	})

	t.Run("exception lookup fails", func(t *testing.T) {
		svc, series, meetingID := occurrenceFixture(t, "FREQ=WEEKLY", dtstart)
		series.excErr = downstream
		_, _, err := svc.ListOccurrences(context.Background(), meetingID, testTenantID, dtstart, window)
		assert.ErrorIs(t, err, downstream,
			"a half-known exception set would show cancelled occurrences as live meetings")
	})
}
