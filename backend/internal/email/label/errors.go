package label

import "errors"

var (
	// ErrNotFound is returned when a label does not exist within the tenant.
	ErrNotFound = errors.New("email label not found")

	// ErrDuplicateName is returned when a label with the same name already
	// exists within the tenant.
	ErrDuplicateName = errors.New("email label name already exists")

	// ErrNameRequired is returned when the label name is blank.
	ErrNameRequired = errors.New("email label name is required")

	// ErrColorInvalid is returned for a colour that is not a 6-digit hex code.
	ErrColorInvalid = errors.New("email label color must be a valid hex color (e.g. #6B7280)")

	// ErrMessageNotFound is returned when an assignment targets a message that
	// does not exist within the tenant.
	ErrMessageNotFound = errors.New("email message not found")

	// ErrTargetInvalid is returned when an assignment carries a label id that
	// does not belong to the tenant.
	ErrTargetInvalid = errors.New("one or more label ids do not belong to the tenant")
)
