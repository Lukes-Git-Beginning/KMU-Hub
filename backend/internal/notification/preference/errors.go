package preference

import "errors"

var (
	// ErrPreferenceNotFound is returned when a notification preference does not exist.
	ErrPreferenceNotFound = errors.New("notification preference not found")

	// ErrQuietHoursNotFound is returned when quiet hours configuration does not exist for a user.
	ErrQuietHoursNotFound = errors.New("quiet hours configuration not found")

	// ErrMuteNotFound is returned when a mute entry does not exist.
	ErrMuteNotFound = errors.New("mute not found")

	// ErrMuteAlreadyExists is returned when attempting to mute an already muted resource.
	ErrMuteAlreadyExists = errors.New("resource already muted")

	// ErrInvalidTimezone is returned when an invalid IANA timezone is provided.
	ErrInvalidTimezone = errors.New("invalid timezone: must be a valid IANA timezone")

	// ErrInvalidTimeFormat is returned when time format is invalid.
	ErrInvalidTimeFormat = errors.New("invalid time format: use HH:MM (24-hour)")

	// ErrInvalidDayOfWeek is returned when an invalid day of week is provided.
	ErrInvalidDayOfWeek = errors.New("invalid day of week: must be 1-7 (Monday-Sunday)")
)
