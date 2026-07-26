package message

import "errors"

var (
	// ErrMessageNotFound is returned when an inbox message does not exist.
	ErrMessageNotFound = errors.New("inbox message not found")

	// ErrAlreadyAssigned is returned when trying to claim a message that is already assigned.
	ErrAlreadyAssigned = errors.New("message is already assigned to another user")

	// ErrInvalidSnoozeTime is returned when snooze_until is in the past.
	ErrInvalidSnoozeTime = errors.New("snooze time must be in the future")

	// ErrAdapterNotFound is returned when no adapter is registered for a channel.
	ErrAdapterNotFound = errors.New("no adapter registered for channel")

	// ErrDuplicateMessage is returned when a source_id already exists for the user+channel.
	ErrDuplicateMessage = errors.New("duplicate inbox message for this user and channel")

	// ErrInvalidStatus is returned when a status value is not one of the allowed states.
	ErrInvalidStatus = errors.New("status must be one of: open, pending, resolved, closed")
)

// ValidStatuses lists the allowed conversation-level status values.
var ValidStatuses = map[string]bool{
	"open":     true,
	"pending":  true,
	"resolved": true,
	"closed":   true,
}
