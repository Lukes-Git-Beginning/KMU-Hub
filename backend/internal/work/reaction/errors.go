package reaction

import "errors"

var (
	// ErrMessageNotFound is returned when the message does not exist for the
	// tenant or lives in a private channel the user is not a member of. The two
	// cases are indistinguishable on purpose: a private channel must not reveal
	// that a message ID exists.
	ErrMessageNotFound = errors.New("message not found")

	// ErrEmojiRequired is returned when an empty emoji string is provided.
	ErrEmojiRequired = errors.New("emoji is required")

	// ErrEmojiTooLong is returned when the emoji exceeds 32 bytes.
	// 32 bytes accommodates multi-codepoint emoji such as flag sequences.
	ErrEmojiTooLong = errors.New("emoji must be at most 32 bytes")
)
