package template

import "errors"

var (
	// ErrTemplateNotFound is returned when a template does not exist, is not
	// visible to the caller (foreign personal template), or belongs to
	// another tenant.
	ErrTemplateNotFound = errors.New("email template not found")
	// ErrNameRequired is returned when the template name is empty.
	ErrNameRequired = errors.New("template name is required")
	// ErrInvalidVisibility is returned for a visibility value other than
	// "personal" or "shared".
	ErrInvalidVisibility = errors.New("visibility must be personal or shared")
)
