package tag

import "errors"

var (
	// ErrTagNotFound is returned when a tag with the given ID does not exist.
	ErrTagNotFound = errors.New("tag not found")

	// ErrTagNameRequired is returned when a tag name is empty.
	ErrTagNameRequired = errors.New("tag name is required")

	// ErrTagNameTooLong is returned when a tag name exceeds 100 characters.
	ErrTagNameTooLong = errors.New("tag name too long (maximum 100 characters)")

	// ErrTagNameConflict is returned when a tag with the same name already exists.
	ErrTagNameConflict = errors.New("tag name already exists")

	// ErrInvalidColor is returned when the color format is not a valid #RRGGBB hex string.
	ErrInvalidColor = errors.New("invalid color format (must be #RRGGBB)")

	// ErrTagAlreadyAssigned is returned for informational purposes; tagging is
	// idempotent so this is not treated as a hard error.
	ErrTagAlreadyAssigned = errors.New("tag is already assigned to this file")
)
