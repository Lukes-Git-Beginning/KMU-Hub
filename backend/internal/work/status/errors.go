package status

import "errors"

var (
	ErrNotFound           = errors.New("status not found")
	ErrNameRequired       = errors.New("status name is required")
	ErrNameTooLong        = errors.New("status name must be at most 100 characters")
	ErrDuplicateName      = errors.New("a status with this name already exists in the project")
	ErrCannotDeleteDefault = errors.New("cannot delete the default status")
	ErrCannotDeleteWithTasks = errors.New("cannot delete status that has tasks assigned")
	ErrInvalidSortOrder   = errors.New("invalid sort order: all status IDs must belong to the same project")
)
