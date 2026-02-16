package account

import "errors"

var (
	// ErrAccountNotFound is returned when an email account does not exist.
	ErrAccountNotFound = errors.New("email account not found")

	// ErrAccountExists is returned when a user already has an email account configured.
	ErrAccountExists = errors.New("email account already exists for this user")

	// ErrInvalidCredentials is returned when IMAP/SMTP credentials are rejected by the server.
	ErrInvalidCredentials = errors.New("invalid email credentials")

	// ErrConnectionFailed is returned when connecting to the IMAP or SMTP server fails.
	ErrConnectionFailed = errors.New("email server connection failed")
)
