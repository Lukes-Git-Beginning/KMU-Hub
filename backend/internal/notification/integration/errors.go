package integration

import "errors"

var (
	// ErrConfigNotFound is returned when an integration config does not exist.
	ErrConfigNotFound = errors.New("integration config not found")

	// ErrConfigAlreadyExists is returned when a config for the given platform already exists.
	ErrConfigAlreadyExists = errors.New("integration config already exists for this platform")

	// ErrMappingNotFound is returned when a channel mapping does not exist.
	ErrMappingNotFound = errors.New("channel mapping not found")

	// ErrAccountLinkNotFound is returned when an account link does not exist.
	ErrAccountLinkNotFound = errors.New("account link not found")

	// ErrAccountNotLinked is returned when the external user has no linked KMU Hub account.
	ErrAccountNotLinked = errors.New("external account not linked to a KMU Hub user")

	// ErrLinkTokenExpired is returned when a link token has passed its expiration time.
	ErrLinkTokenExpired = errors.New("link token has expired")

	// ErrLinkTokenUsed is returned when a link token has already been consumed.
	ErrLinkTokenUsed = errors.New("link token has already been used")

	// ErrLinkTokenNotFound is returned when a link token does not exist.
	ErrLinkTokenNotFound = errors.New("link token not found")

	// ErrPlatformDisabled is returned when the platform integration is not active.
	ErrPlatformDisabled = errors.New("platform integration is not active")

	// ErrInvalidPlatform is returned when an unsupported platform value is provided.
	ErrInvalidPlatform = errors.New("invalid platform: must be teams or slack")
)
