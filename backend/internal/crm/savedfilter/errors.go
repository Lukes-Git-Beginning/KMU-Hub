package savedfilter

import "errors"

var (
	ErrFilterNotFound    = errors.New("saved filter not found")
	ErrNameRequired      = errors.New("filter name is required")
	ErrInvalidEntityType = errors.New("invalid entity type")
	ErrInvalidFilterJSON = errors.New("invalid filter JSON")
	ErrFilterNotOwned    = errors.New("filter does not belong to this user")
)
