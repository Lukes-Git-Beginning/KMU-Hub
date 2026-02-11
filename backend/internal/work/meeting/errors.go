package meeting

import "errors"

var (
	ErrNotFound              = errors.New("meeting not found")
	ErrTitleRequired         = errors.New("meeting title is required")
	ErrTitleTooLong          = errors.New("meeting title must be at most 500 characters")
	ErrInvalidTimeRange      = errors.New("scheduled end must be after scheduled start")
	ErrStartInPast           = errors.New("scheduled start must be in the future")
	ErrInvalidStatus         = errors.New("invalid meeting status")
	ErrInvalidRSVP           = errors.New("invalid RSVP status")
	ErrNotScheduled          = errors.New("meeting must be in scheduled status")
	ErrNotInProgress         = errors.New("meeting must be in progress")
	ErrCannotDelete          = errors.New("can only delete scheduled or cancelled meetings")
	ErrNotesContentRequired  = errors.New("notes content is required")
	ErrActionDescRequired    = errors.New("action item description is required")
	ErrActionItemNotFound    = errors.New("action item not found")
	ErrNotRecurring          = errors.New("meeting is not part of a recurring series")
	ErrNoPreviousNotes       = errors.New("no previous meeting notes found")
	ErrNoAttendeesProvided   = errors.New("at least one attendee is required")
	ErrInvalidAwayTimeout    = errors.New("away timeout must be between 60 and 3600 seconds")
)
