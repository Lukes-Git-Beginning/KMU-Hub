package customfield

import "errors"

var (
	ErrFieldNotFound      = errors.New("custom field not found")
	ErrFieldExists        = errors.New("custom field with this name already exists for entity type")
	ErrInvalidFieldType   = errors.New("invalid field type")
	ErrInvalidEntityType  = errors.New("invalid entity type")
	ErrOptionsRequired    = errors.New("options are required for select/multiselect fields")
	ErrFieldNameRequired  = errors.New("field name is required")
	ErrFieldLabelRequired = errors.New("field label is required")
	ErrFieldInUse         = errors.New("custom field is in use and cannot be deleted")
)
