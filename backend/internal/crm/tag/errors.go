package tag

import "errors"

var (
	ErrTagNotFound       = errors.New("tag not found")
	ErrTagExists         = errors.New("tag with this name already exists for entity type")
	ErrTagNameRequired   = errors.New("tag name is required")
	ErrInvalidColor      = errors.New("invalid color format, must be hex color (e.g., #ff5733)")
	ErrInvalidEntityType = errors.New("invalid entity type")
	ErrTagInUse          = errors.New("tag is in use and cannot be deleted")
)
