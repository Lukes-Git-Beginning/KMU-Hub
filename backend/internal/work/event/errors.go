package event

import "errors"

var (
	ErrEventNotFound           = errors.New("event not found")
	ErrEventTitleRequired      = errors.New("event title is required")
	ErrInvalidTimeRange        = errors.New("end time must be after start time")
	ErrInvalidRRule            = errors.New("invalid RRULE string")
	ErrInvalidRSVPStatus       = errors.New("invalid RSVP status")
	ErrAlreadyAttendee         = errors.New("user is already an attendee")
	ErrNotAttendee             = errors.New("user is not an attendee of this event")
	ErrNotEventCreator         = errors.New("user is not the creator of this event")
	ErrExceptionNotFound       = errors.New("exception not found")
	ErrExceptionAlreadyExists  = errors.New("exception already exists for this date")
	ErrInvalidRecurringEditScope = errors.New("invalid recurring event edit scope")
	ErrEventNotRecurring       = errors.New("event is not a recurring event")
	ErrReminderLimitExceeded   = errors.New("maximum 3 reminders per event")
	ErrInvalidReminderMinutes  = errors.New("reminder minutes must be greater than zero")
	ErrInsufficientPermission  = errors.New("insufficient permission on this calendar")
)
