package employee

import "errors"

var (
	// ErrEmployeeNotFound is returned when an employee profile does not exist.
	ErrEmployeeNotFound = errors.New("employee profile not found")
	// ErrProfileAlreadyExists is returned when an employee profile already exists for the user.
	ErrProfileAlreadyExists = errors.New("employee profile already exists for this user")
	// ErrUnauthorizedFieldUpdate is returned when a caller tries to update fields they don't have permission for.
	ErrUnauthorizedFieldUpdate = errors.New("unauthorized to update these fields")
	// ErrDocumentCategoryNotFound is returned when a document category does not exist.
	ErrDocumentCategoryNotFound = errors.New("document category not found")
	// ErrDocumentNotFound is returned when an employee document does not exist.
	ErrDocumentNotFound = errors.New("employee document not found")
	// ErrInsufficientDocumentAccess is returned when a caller does not have access to view a document.
	ErrInsufficientDocumentAccess = errors.New("insufficient access to view this document")
)
