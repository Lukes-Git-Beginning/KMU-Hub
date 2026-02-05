package company

import "errors"

var (
	ErrCompanyNotFound    = errors.New("company not found")
	ErrNameRequired       = errors.New("company name is required")
	ErrCompanyInUse       = errors.New("company has contacts and cannot be deleted")
	ErrTagNotFound        = errors.New("tag not found")
	ErrTagWrongType       = errors.New("tag is not for companies")
	ErrInvalidCustomField = errors.New("invalid custom field")
)
