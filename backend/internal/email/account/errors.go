package account

import "errors"

var (
	// ErrAccountNotFound is returned when an email account does not exist.
	ErrAccountNotFound = errors.New("email account not found")

	// ErrInvalidCredentials is returned when IMAP/SMTP credentials are rejected by the server.
	ErrInvalidCredentials = errors.New("invalid email credentials")

	// ErrConnectionFailed is returned when connecting to the IMAP or SMTP server fails.
	ErrConnectionFailed = errors.New("email server connection failed")

	// ErrHostNotAllowed is returned when an IMAP or SMTP host resolves to an
	// internal address (loopback, private, link-local) that is not on the
	// EMAIL_DIAL_ALLOWLIST. The resolved address stays out of the message.
	ErrHostNotAllowed = errors.New("email server host is not allowed")
)
