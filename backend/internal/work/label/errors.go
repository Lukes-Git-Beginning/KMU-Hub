package label

import "errors"

// ErrNotFound is returned when a label does not exist.
var ErrNotFound = errors.New("label not found")

// ErrDuplicateName is returned when a label with the same name already exists
// within the tenant.
var ErrDuplicateName = errors.New("label name already exists")
