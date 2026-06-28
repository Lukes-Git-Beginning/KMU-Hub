package thread

import "errors"

// ErrCannedResponseNotFound is returned when a canned response does not exist
// for the tenant.
var ErrCannedResponseNotFound = errors.New("canned response not found")
