package guest

import "errors"

var (
	ErrSessionNotFound     = errors.New("guest session not found")
	ErrSessionExpired      = errors.New("guest session expired")
	ErrSessionInactive     = errors.New("guest session deactivated")
	ErrChannelNotGuest     = errors.New("channel is not guest-enabled")
	ErrConfigNotFound      = errors.New("guest channel config not found")
	ErrConfigAlreadyExists = errors.New("guest channel config already exists")
	ErrDisplayNameRequired = errors.New("display name is required")
	ErrDisplayNameTooLong  = errors.New("display name exceeds 200 characters")
	ErrRateLimited         = errors.New("message rate limit exceeded")
	ErrInvalidToken        = errors.New("invalid guest token")
	ErrTooManyFailures     = errors.New("too many failed validation attempts")
)
