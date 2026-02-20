package caldav

import (
	"fmt"
	"strings"
	"time"

	ical "github.com/emersion/go-ical"

	"github.com/kmuhub/kmuhub/internal/models"
)

// CalEventInput represents parsed iCalendar event data for creating/updating events.
// This is an intermediate type -- the caller maps it to gRPC create/update requests.
type CalEventInput struct {
	Title       string
	Description *string
	Location    *string
	StartTime   time.Time
	EndTime     time.Time
	IsAllDay    bool
	Timezone    string
	RRule       *string
}

// CalExceptionInput represents a parsed recurring event exception from iCalendar.
type CalExceptionInput struct {
	OriginalDate time.Time
	IsCancelled  bool
	Title        *string
	StartTime    *time.Time
	EndTime      *time.Time
}

// EventToICal converts a CalendarEvent with its exceptions and attendees to an iCalendar object.
func EventToICal(event *models.CalendarEvent, exceptions []models.EventException, attendees []models.EventAttendee) (*ical.Calendar, error) {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//KMU Hub//CalDAV//DE")

	// Collect unique timezones and add VTIMEZONE components
	timezones := make(map[string]bool)
	if event.Timezone != "" {
		timezones[event.Timezone] = true
	}
	for tzName := range timezones {
		vtComp, err := GenerateVTimezone(tzName)
		if err != nil {
			return nil, fmt.Errorf("generate vtimezone for %q: %w", tzName, err)
		}
		cal.Children = append(cal.Children, vtComp)
	}

	// Build master VEVENT
	vevent := ical.NewEvent()
	vevent.Props.SetText(ical.PropUID, event.ID.String())
	vevent.Props.SetDateTime(ical.PropDateTimeStamp, event.UpdatedAt.UTC())
	vevent.Props.SetText(ical.PropSummary, event.Title)

	// DTSTART / DTEND
	setEventTimes(vevent, event.StartTime, event.EndTime, event.IsAllDay, event.Timezone)

	// Optional properties
	if event.Description != nil && *event.Description != "" {
		vevent.Props.SetText(ical.PropDescription, *event.Description)
	}
	if event.Location != nil && *event.Location != "" {
		vevent.Props.SetText(ical.PropLocation, *event.Location)
	}

	// RRULE
	if event.RRule != nil && *event.RRule != "" {
		rruleProp := ical.NewProp(ical.PropRecurrenceRule)
		rruleProp.SetValueType(ical.ValueRecurrence)
		rruleProp.Value = *event.RRule
		vevent.Props.Set(rruleProp)
	}

	// EXDATE for cancelled exceptions
	for _, exc := range exceptions {
		if exc.IsCancelled {
			exdateProp := ical.NewProp(ical.PropExceptionDates)
			if event.IsAllDay {
				exdateProp.SetValueType(ical.ValueDate)
				exdateProp.Value = exc.OriginalDate.Format("20060102")
			} else {
				exdateProp.SetValueType(ical.ValueDateTime)
				if event.Timezone != "" {
					exdateProp.Params.Set(ical.ParamTimezoneID, event.Timezone)
					exdateProp.Value = exc.OriginalDate.Format("20060102T150405")
				} else {
					exdateProp.Value = exc.OriginalDate.UTC().Format("20060102T150405Z")
				}
			}
			vevent.Props.Add(exdateProp)
		}
	}

	// ATTENDEE properties
	for _, att := range attendees {
		attendeeProp := ical.NewProp(ical.PropAttendee)
		attendeeProp.Value = fmt.Sprintf("urn:uuid:%s", att.UserID.String())
		cn := strings.TrimSpace(fmt.Sprintf("%s %s", att.FirstName, att.LastName))
		if cn != "" {
			attendeeProp.Params.Set(ical.ParamCommonName, cn)
		}
		attendeeProp.Params.Set(ical.ParamParticipationStatus, rsvpToPartStat(att.RSVPStatus))
		vevent.Props.Add(attendeeProp)
	}

	cal.Children = append(cal.Children, vevent.Component)

	// Additional VEVENTs for non-cancelled exceptions with overrides (RECURRENCE-ID)
	for _, exc := range exceptions {
		if exc.IsCancelled {
			continue
		}
		excEvent := ical.NewEvent()
		excEvent.Props.SetText(ical.PropUID, event.ID.String())
		excEvent.Props.SetDateTime(ical.PropDateTimeStamp, exc.UpdatedAt.UTC())

		// RECURRENCE-ID
		recIDProp := ical.NewProp(ical.PropRecurrenceID)
		if event.IsAllDay {
			recIDProp.SetValueType(ical.ValueDate)
			recIDProp.Value = exc.OriginalDate.Format("20060102")
		} else {
			recIDProp.SetValueType(ical.ValueDateTime)
			if event.Timezone != "" {
				recIDProp.Params.Set(ical.ParamTimezoneID, event.Timezone)
				recIDProp.Value = exc.OriginalDate.Format("20060102T150405")
			} else {
				recIDProp.Value = exc.OriginalDate.UTC().Format("20060102T150405Z")
			}
		}
		excEvent.Props.Set(recIDProp)

		// Use overridden title or fall back to master
		title := event.Title
		if exc.Title != nil && *exc.Title != "" {
			title = *exc.Title
		}
		excEvent.Props.SetText(ical.PropSummary, title)

		// Use overridden times or fall back to master times
		startTime := event.StartTime
		endTime := event.EndTime
		if exc.StartTime != nil {
			startTime = *exc.StartTime
		}
		if exc.EndTime != nil {
			endTime = *exc.EndTime
		}
		setEventTimes(excEvent, startTime, endTime, event.IsAllDay, event.Timezone)

		// Carry over description/location from master if not overridden in exception
		if event.Description != nil && *event.Description != "" {
			excEvent.Props.SetText(ical.PropDescription, *event.Description)
		}
		if event.Location != nil && *event.Location != "" {
			excEvent.Props.SetText(ical.PropLocation, *event.Location)
		}

		cal.Children = append(cal.Children, excEvent.Component)
	}

	return cal, nil
}

// ICalToEventInput parses an iCalendar object into a CalEventInput and optional exceptions.
func ICalToEventInput(cal *ical.Calendar) (*CalEventInput, []*CalExceptionInput, error) {
	events := cal.Events()
	if len(events) == 0 {
		return nil, nil, fmt.Errorf("no VEVENT found in calendar object")
	}

	// Find the master event (no RECURRENCE-ID) and exception events
	var master *ical.Event
	var exceptionEvents []ical.Event

	for i := range events {
		if events[i].Props.Get(ical.PropRecurrenceID) != nil {
			exceptionEvents = append(exceptionEvents, events[i])
		} else if master == nil {
			master = &events[i]
		}
	}

	if master == nil {
		// If all events have RECURRENCE-ID, use the first one as master
		master = &events[0]
	}

	input, err := parseVEvent(master)
	if err != nil {
		return nil, nil, fmt.Errorf("parse master VEVENT: %w", err)
	}

	// Parse EXDATE properties as cancelled exceptions
	var exceptions []*CalExceptionInput
	for _, exdateProp := range master.Props.Values(ical.PropExceptionDates) {
		loc := time.UTC
		if tzid := exdateProp.Params.Get(ical.ParamTimezoneID); tzid != "" {
			if parsedLoc, err := time.LoadLocation(tzid); err == nil {
				loc = parsedLoc
			}
		}
		exdate, err := exdateProp.DateTime(loc)
		if err != nil {
			return nil, nil, fmt.Errorf("parse EXDATE: %w", err)
		}
		exceptions = append(exceptions, &CalExceptionInput{
			OriginalDate: exdate,
			IsCancelled:  true,
		})
	}

	// Parse exception VEVENTs (modified occurrences)
	for _, excEvt := range exceptionEvents {
		recIDProp := excEvt.Props.Get(ical.PropRecurrenceID)
		if recIDProp == nil {
			continue
		}

		loc := time.UTC
		if tzid := recIDProp.Params.Get(ical.ParamTimezoneID); tzid != "" {
			if parsedLoc, err := time.LoadLocation(tzid); err == nil {
				loc = parsedLoc
			}
		}
		origDate, err := recIDProp.DateTime(loc)
		if err != nil {
			return nil, nil, fmt.Errorf("parse RECURRENCE-ID: %w", err)
		}

		excInput := &CalExceptionInput{
			OriginalDate: origDate,
			IsCancelled:  false,
		}

		// Extract overridden title
		if summaryProp := excEvt.Props.Get(ical.PropSummary); summaryProp != nil {
			text, _ := summaryProp.Text()
			if text != "" {
				excInput.Title = &text
			}
		}

		// Extract overridden times
		startProp := excEvt.Props.Get(ical.PropDateTimeStart)
		if startProp != nil {
			startLoc := time.UTC
			if tzid := startProp.Params.Get(ical.ParamTimezoneID); tzid != "" {
				if parsedLoc, err := time.LoadLocation(tzid); err == nil {
					startLoc = parsedLoc
				}
			}
			startTime, err := startProp.DateTime(startLoc)
			if err == nil {
				excInput.StartTime = &startTime
			}
		}

		endProp := excEvt.Props.Get(ical.PropDateTimeEnd)
		if endProp != nil {
			endLoc := time.UTC
			if tzid := endProp.Params.Get(ical.ParamTimezoneID); tzid != "" {
				if parsedLoc, err := time.LoadLocation(tzid); err == nil {
					endLoc = parsedLoc
				}
			}
			endTime, err := endProp.DateTime(endLoc)
			if err == nil {
				excInput.EndTime = &endTime
			}
		}

		exceptions = append(exceptions, excInput)
	}

	return input, exceptions, nil
}

// parseVEvent extracts CalEventInput from a single VEVENT component.
func parseVEvent(evt *ical.Event) (*CalEventInput, error) {
	input := &CalEventInput{}

	// SUMMARY -> Title
	if summaryProp := evt.Props.Get(ical.PropSummary); summaryProp != nil {
		text, err := summaryProp.Text()
		if err != nil {
			return nil, fmt.Errorf("parse SUMMARY: %w", err)
		}
		input.Title = text
	}

	// DESCRIPTION
	if descProp := evt.Props.Get(ical.PropDescription); descProp != nil {
		text, err := descProp.Text()
		if err == nil && text != "" {
			input.Description = &text
		}
	}

	// LOCATION
	if locProp := evt.Props.Get(ical.PropLocation); locProp != nil {
		text, err := locProp.Text()
		if err == nil && text != "" {
			input.Location = &text
		}
	}

	// DTSTART
	startProp := evt.Props.Get(ical.PropDateTimeStart)
	if startProp == nil {
		return nil, fmt.Errorf("DTSTART is required")
	}

	// Detect all-day event from VALUE=DATE
	isAllDay := startProp.ValueType() == ical.ValueDate ||
		(startProp.ValueType() == ical.ValueDefault && len(startProp.Value) == 8)
	input.IsAllDay = isAllDay

	// Determine timezone
	timezone := "Europe/Berlin" // Deutschland-First default
	if tzid := startProp.Params.Get(ical.ParamTimezoneID); tzid != "" {
		timezone = tzid
	}
	input.Timezone = timezone

	loc := time.UTC
	if !isAllDay {
		if parsedLoc, err := time.LoadLocation(timezone); err == nil {
			loc = parsedLoc
		}
	}

	startTime, err := startProp.DateTime(loc)
	if err != nil {
		return nil, fmt.Errorf("parse DTSTART: %w", err)
	}
	input.StartTime = startTime

	// DTEND
	endTime, err := evt.DateTimeEnd(loc)
	if err != nil {
		return nil, fmt.Errorf("parse DTEND: %w", err)
	}
	input.EndTime = endTime

	// RRULE
	if rruleProp := evt.Props.Get(ical.PropRecurrenceRule); rruleProp != nil {
		rrule := rruleProp.Value
		if rrule != "" {
			input.RRule = &rrule
		}
	}

	return input, nil
}

// setEventTimes sets DTSTART and DTEND properties on a VEVENT component.
func setEventTimes(evt *ical.Event, start, end time.Time, isAllDay bool, timezone string) {
	if isAllDay {
		evt.Props.SetDate(ical.PropDateTimeStart, start)
		evt.Props.SetDate(ical.PropDateTimeEnd, end)
	} else {
		if timezone != "" {
			loc, err := time.LoadLocation(timezone)
			if err == nil {
				start = start.In(loc)
				end = end.In(loc)
			}
		}
		evt.Props.SetDateTime(ical.PropDateTimeStart, start)
		evt.Props.SetDateTime(ical.PropDateTimeEnd, end)
	}
}

// rsvpToPartStat maps KMU Hub RSVP status to iCalendar PARTSTAT parameter value.
func rsvpToPartStat(rsvp string) string {
	switch rsvp {
	case models.RSVPAccepted:
		return "ACCEPTED"
	case models.RSVPDeclined:
		return "DECLINED"
	case models.RSVPTentative:
		return "TENTATIVE"
	default:
		return "NEEDS-ACTION"
	}
}
