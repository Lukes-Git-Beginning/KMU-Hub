package customfield

import "errors"

// ErrNotFound is returned when the definition does not exist.
var ErrNotFound = errors.New("custom field definition not found")

// ErrDuplicateName is returned when a definition with the same name already exists.
var ErrDuplicateName = errors.New("custom field name already exists")

// ErrInvalidFieldType is returned when an unsupported field_type is provided.
var ErrInvalidFieldType = errors.New("invalid custom field type")

// ErrNameRequired is returned when name is empty.
var ErrNameRequired = errors.New("custom field name is required")
