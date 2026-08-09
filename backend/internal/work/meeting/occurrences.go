package meeting

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/work/event"
)

// maxMeetingOccurrences caps a single expansion. The calendar's own expansion
// (event.Service.ListEventsInRange) is bounded only by the requested window,
// which is acceptable for rendering a month grid but not for an endpoint a
// client can point at a decade with FREQ=HOURLY. Anything past the cap is
// dropped and reported through the truncated flag.
const maxMeetingOccurrences = 200

// SeriesSource supplies the recurring calendar event a meeting is linked to.
// *event.Service satisfies it.
//
// The series definition deliberately lives on the calendar event and not on
// the meeting: meetings already carry calendar_event_id (migration 000037),
// and a second RRULE column on meetings would be a second source of truth for
// the same series.
type SeriesSource interface {
	Get(ctx context.Context, eventID, tenantID uuid.UUID) (*models.CalendarEvent, error)
	ListExceptions(ctx context.Context, eventID uuid.UUID, start, end time.Time) ([]models.EventException, error)
}

// WithSeriesSource attaches the calendar event service so recurring meetings
// can be expanded, and returns the service for chaining. Without it
// ListOccurrences reports ErrSeriesUnavailable rather than pretending the
// meeting is a one-off.
func (s *Service) WithSeriesSource(src SeriesSource) *Service {
	s.series = src
	return s
}

// MeetingOccurrence is one dated instance of a recurring meeting. Occurrences
// are computed on read and never stored, so they have no id of their own.
type MeetingOccurrence struct {
	MeetingID       uuid.UUID `json:"meeting_id"`
	CalendarEventID uuid.UUID `json:"calendar_event_id"`
	Start           time.Time `json:"start"`
	End             time.Time `json:"end"`
}

// ListOccurrences expands the series of a recurring meeting inside [start, end).
//
// The recurrence rule comes from the calendar event the meeting is linked to
// and is expanded with the calendar's own event.ExpandRecurrence, so a meeting
// series and its calendar series can never drift apart. Occurrence timing
// follows the calendar; occurrence length follows the meeting, because the
// meeting is what repeats and it carries its own scheduled duration.
//
// The second return value is true when the cap cut the list short.
func (s *Service) ListOccurrences(ctx context.Context, meetingID, tenantID uuid.UUID, start, end time.Time) ([]MeetingOccurrence, bool, error) {
	if !end.After(start) {
		return nil, false, ErrInvalidTimeRange
	}
	if s.series == nil {
		return nil, false, ErrSeriesUnavailable
	}

	m, err := s.repo.GetMeeting(ctx, meetingID, tenantID)
	if err != nil {
		return nil, false, err
	}
	if m.CalendarEventID == nil {
		return nil, false, ErrNotRecurring
	}

	evt, err := s.series.Get(ctx, *m.CalendarEventID, tenantID)
	if err != nil {
		return nil, false, fmt.Errorf("load linked calendar event: %w", err)
	}
	if evt.RRule == nil || strings.TrimSpace(*evt.RRule) == "" {
		return nil, false, ErrNotRecurring
	}

	// The series ends where the calendar says it ends, even if the caller asks
	// for a wider window.
	windowEnd := end
	if evt.RecurrenceEnd != nil && evt.RecurrenceEnd.Before(windowEnd) {
		windowEnd = *evt.RecurrenceEnd
	}
	if !windowEnd.After(start) {
		return []MeetingOccurrence{}, false, nil
	}

	dates, err := event.ExpandRecurrence(*evt.RRule, evt.StartTime, start, windowEnd)
	if err != nil {
		// Stored data, not caller input: the rule was accepted when the event
		// was written, so a parse failure here is a data defect worth naming
		// rather than an empty list the caller would read as "no occurrences".
		slog.Error("failed to expand meeting series",
			"meeting_id", meetingID,
			"calendar_event_id", evt.ID,
			"error", err,
		)
		return nil, false, ErrInvalidRecurrence
	}

	// Cancellations and time overrides must apply here too — without them a
	// cancelled occurrence would still be offered as a meeting.
	exceptions, err := s.series.ListExceptions(ctx, evt.ID, start, windowEnd)
	if err != nil {
		return nil, false, fmt.Errorf("load series exceptions: %w", err)
	}
	exceptionsByDate := make(map[string]*models.EventException, len(exceptions))
	for i := range exceptions {
		exceptionsByDate[exceptions[i].OriginalDate.Format("2006-01-02")] = &exceptions[i]
	}

	duration := m.ScheduledEnd.Sub(m.ScheduledStart)
	occurrences := make([]MeetingOccurrence, 0, len(dates))
	truncated := false

	for _, date := range dates {
		occStart := date
		if ex, ok := exceptionsByDate[date.Format("2006-01-02")]; ok {
			if ex.IsCancelled {
				continue
			}
			// lean: only the moved start is carried over; a calendar
			// exception's title/location overrides are ignored because a
			// meeting keeps its own title and room. Carry them over once an
			// occurrence is allowed to differ from its meeting.
			if ex.StartTime != nil {
				occStart = *ex.StartTime
			}
		}

		if len(occurrences) == maxMeetingOccurrences {
			truncated = true
			break
		}

		occurrences = append(occurrences, MeetingOccurrence{
			MeetingID:       m.ID,
			CalendarEventID: evt.ID,
			Start:           occStart,
			End:             occStart.Add(duration),
		})
	}

	return occurrences, truncated, nil
}
