package presence

import "errors"

var (
	ErrInvalidManualStatus = errors.New("only online, away, and dnd can be set manually")
	ErrInvalidAwayTimeout  = errors.New("away timeout must be between 60 and 3600 seconds")
	// ErrConfigNotFound signals that the tenant has never written a presence
	// config. Since 000274 that is the normal state of a fresh tenant, not an
	// error condition — the service answers it with the code default.
	ErrConfigNotFound = errors.New("presence config not found for tenant")
	// ErrMissingTenant signals a presence config call without a tenant on the
	// context. Reading or writing another tenant's row is worse than failing.
	ErrMissingTenant = errors.New("missing tenant context")
)
