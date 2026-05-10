package contact

import "errors"

var (
	ErrContactNotFound    = errors.New("contact not found")
	ErrEmailExists        = errors.New("contact with this email already exists")
	ErrFirstNameRequired  = errors.New("first name is required")
	ErrLastNameRequired   = errors.New("last name is required")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrCompanyNotFound    = errors.New("company not found")
	ErrContactInUse       = errors.New("contact is in use and cannot be deleted")
	ErrTagNotFound        = errors.New("tag not found")
	ErrInvalidCustomField = errors.New("invalid custom field")
	ErrCannotMergeSelf    = errors.New("cannot merge a contact with itself")
	ErrAlreadyMerged      = errors.New("contact has already been merged")
	ErrInvalidTenant      = errors.New("tenant_id is required")
)
