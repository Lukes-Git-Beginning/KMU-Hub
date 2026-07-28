package berichte

import "errors"

var (
	ErrDefinitionNotFound  = errors.New("berichte definition not found")
	ErrScheduleNotFound    = errors.New("berichte schedule not found")
	ErrInvalidQueryConfig  = errors.New("berichte query config is invalid")
	ErrInvalidCron         = errors.New("berichte cron expression is invalid")
	ErrInvalidFormat       = errors.New("berichte report format is invalid")
	ErrInvalidModule       = errors.New("berichte module is invalid")
	ErrInvalidKind         = errors.New("berichte kind is invalid")
	ErrCacheMiss           = errors.New("berichte cache miss")
	ErrExecutorUnavailable = errors.New("berichte executor unavailable")
	ErrDocumentNotFound    = errors.New("berichte document not found")
	ErrInvalidTitle        = errors.New("berichte document title is required")
	ErrInvalidStatus       = errors.New("berichte document status is invalid")
	ErrInvalidRows         = errors.New("berichte document rows must be a JSON array")
	ErrInvalidSettings     = errors.New("berichte document settings must be a JSON object")
	ErrDocumentTooLarge    = errors.New("berichte document exceeds the size limit")

	// ErrShareNotFound covers all three ways a share link stops working:
	// unknown, expired and revoked. They are deliberately indistinguishable to
	// the unauthenticated caller — a distinct "expired" answer would confirm
	// that the token was once valid, and 403 on a revoked link would say the
	// document exists.
	ErrShareNotFound = errors.New("berichte share link not found")
	// ErrSharePasswordRequired means the link is password-protected and the
	// request carried none. Distinct from a wrong password only in the message:
	// the caller already holds the token, so the existence of the protection is
	// not the secret — the password is.
	ErrSharePasswordRequired = errors.New("berichte share link requires a password")
	ErrSharePasswordInvalid  = errors.New("berichte share link password is invalid")
	ErrInvalidShareExpiry    = errors.New("berichte share link expiry is invalid")
	ErrSharePasswordTooLong  = errors.New("berichte share link password is too long")
)
