package send

import "errors"

var (
	// ErrSendFailed is returned when SMTP sending fails after establishing a connection.
	ErrSendFailed = errors.New("failed to send email")

	// ErrSMTPAuthFailed is returned when SMTP authentication is rejected by the server.
	ErrSMTPAuthFailed = errors.New("SMTP authentication failed")

	// ErrInvalidRecipient is returned when a recipient email address is malformed.
	ErrInvalidRecipient = errors.New("invalid recipient email address")
)
