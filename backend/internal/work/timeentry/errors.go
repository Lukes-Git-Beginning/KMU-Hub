package timeentry

import "errors"

var (
	ErrNotFound          = errors.New("time entry not found")
	ErrInvalidTaskID     = errors.New("invalid task ID")
	ErrInvalidUserID     = errors.New("invalid user ID")
	ErrAlreadyRunning    = errors.New("a timer is already running for this user")
	ErrNoActiveTimer     = errors.New("no active timer for this user")
	ErrInvalidDuration   = errors.New("duration must be positive")
	ErrInvalidStartTime  = errors.New("start time cannot be in the future")
	ErrDescriptionTooLong = errors.New("description must be 2000 characters or less")
	ErrCannotEditOthers  = errors.New("cannot edit another user's time entry")
	ErrCannotDeleteOthers = errors.New("cannot delete another user's time entry")
)
