package attachment

import "errors"

var (
	// ErrAttachmentNotFound is returned when an email attachment does not exist.
	ErrAttachmentNotFound = errors.New("email attachment not found")

	// ErrAttachmentTooLarge is returned when an attachment exceeds the configured size limit.
	ErrAttachmentTooLarge = errors.New("email attachment exceeds size limit")
)
