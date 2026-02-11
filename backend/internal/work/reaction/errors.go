package reaction

import "errors"

var (
	// ErrEmojiRequired is returned when an empty emoji string is provided.
	ErrEmojiRequired = errors.New("emoji is required")

	// ErrEmojiTooLong is returned when the emoji exceeds 32 bytes.
	// 32 bytes accommodates multi-codepoint emoji such as flag sequences.
	ErrEmojiTooLong = errors.New("emoji must be at most 32 bytes")
)
