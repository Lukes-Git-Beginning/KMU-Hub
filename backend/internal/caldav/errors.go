package caldav

import "errors"

var (
	// ErrInvalidCredentials is returned when app-specific password validation fails.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrPasswordNotFound is returned when the specified app-specific password does not exist.
	ErrPasswordNotFound = errors.New("app-specific password not found")

	// ErrPasswordRevoked is returned when the app-specific password has been revoked.
	ErrPasswordRevoked = errors.New("app-specific password has been revoked")

	// ErrCalDAVDisabled is returned when CalDAV/CardDAV is disabled at the org level.
	ErrCalDAVDisabled = errors.New("caldav/carddav is disabled")

	// ErrInvalidSyncToken is returned when a sync token cannot be parsed.
	ErrInvalidSyncToken = errors.New("invalid sync token")
)
