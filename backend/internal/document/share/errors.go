package share

import "errors"

// Sentinel errors for share operations.
var (
	ErrShareNotFound      = errors.New("share not found")
	ErrInvalidEntityType  = errors.New("invalid entity type: must be file or folder")
	ErrInvalidPermission  = errors.New("invalid permission: must be read or write")
	ErrCannotShareWithSelf = errors.New("cannot share with yourself")
	ErrAlreadyShared      = errors.New("entity is already shared with this user")
)
