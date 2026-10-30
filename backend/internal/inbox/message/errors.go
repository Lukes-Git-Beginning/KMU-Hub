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

	// ErrInvalidTag is returned when a tag is empty after trimming whitespace.
	ErrInvalidTag = errors.New("tag must not be empty")

	// ErrForwardNotSupported is returned when the message's channel adapter
	// has no concept of forwarding to an arbitrary recipient.
	ErrForwardNotSupported = errors.New("forward is not supported for this channel")

	// ErrReplyNotSupported is returned when the message's channel has no way to
	// deliver a reply (notifications have no recipient on the other side).
	ErrReplyNotSupported = errors.New("reply is not supported for this channel")

	// ErrChannelUnavailable is returned when the reply cannot be delivered because
	// the originating system (e.g. the chat service) is not configured or not reachable.
	ErrChannelUnavailable = errors.New("the originating channel is not reachable, reply not sent")

	// ErrReplyNotPermitted is returned when the originating system refuses the
	// reply for this user (e.g. not a member of the chat channel).
	ErrReplyNotPermitted = errors.New("the originating channel refused the reply for this user")

	// ErrRecipientNotInTenant is returned by Create when the message's user is not a user of the
	// message's tenant. The user_id foreign key does not check that (FK lookups bypass RLS), and
	// the consumer would otherwise file -- and push over WebSocket -- a message for a foreign user.
	ErrRecipientNotInTenant = errors.New("inbox message recipient is not a user of the tenant")
)

// ValidStatuses lists the allowed conversation-level status values.
var ValidStatuses = map[string]bool{
	"open":     true,
	"pending":  true,
	"resolved": true,
	"closed":   true,
}
