package signature

import "errors"

var (
	// ErrSignatureNotFound is returned when an email signature does not exist.
	ErrSignatureNotFound = errors.New("email signature not found")
)
