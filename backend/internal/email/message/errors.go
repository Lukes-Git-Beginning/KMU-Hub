package message

import "errors"

var (
	// ErrMessageNotFound is returned when an email message does not exist.
	ErrMessageNotFound = errors.New("email message not found")

	// ErrFolderNotFound is returned when an email folder does not exist.
	ErrFolderNotFound = errors.New("email folder not found")

	// ErrThreadNotFound is returned when a thread ID has no associated messages.
	ErrThreadNotFound = errors.New("email thread not found")
)
