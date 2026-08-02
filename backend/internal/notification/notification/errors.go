package notification

import "errors"

var (
	// ErrNotificationNotFound is returned when a notification does not exist.
	ErrNotificationNotFound = errors.New("notification not found")

	// ErrInvalidPriority is returned when an invalid priority value is provided.
	ErrInvalidPriority = errors.New("invalid priority: must be urgent, normal, or low")

	// ErrUnauthorized is returned when a user tries to access another user's notification.
	ErrUnauthorized = errors.New("unauthorized: notification belongs to another user")

	// ErrInvalidSnoozeTime is returned when snoozed_until is in the past.
	ErrInvalidSnoozeTime = errors.New("snooze time must be in the future")
)
