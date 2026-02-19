package routing

import "errors"

var (
	// ErrRuleNotFound is returned when a routing rule does not exist.
	ErrRuleNotFound = errors.New("routing rule not found")

	// ErrInvalidConditions is returned when conditions JSON is malformed.
	ErrInvalidConditions = errors.New("invalid routing rule conditions")

	// ErrInvalidAction is returned when an unknown action type is encountered.
	ErrInvalidAction = errors.New("invalid routing rule action type")

	// ErrAutoReplyNonEmail is returned when auto-reply is attempted on a non-email channel.
	ErrAutoReplyNonEmail = errors.New("auto-reply is only supported for email channel messages")
)
