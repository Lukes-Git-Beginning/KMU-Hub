package folder

import "errors"

// Sentinel errors for folder operations.
var (
	ErrFolderNotFound          = errors.New("folder not found")
	ErrFolderNameRequired      = errors.New("folder name is required")
	ErrFolderNameTooLong       = errors.New("folder name must be 255 characters or less")
	ErrInvalidSpaceType        = errors.New("invalid space type: must be personal, team, or project")
	ErrCannotDeleteSystemFolder = errors.New("cannot delete a system folder")
	ErrCannotRenameSystemFolder = errors.New("cannot rename a system folder")
	ErrCannotMoveToSelf        = errors.New("cannot move folder into itself")
	ErrCircularParent          = errors.New("cannot move folder: would create circular reference")
	ErrFolderNameConflict      = errors.New("a folder with this name already exists in the same parent")
)
