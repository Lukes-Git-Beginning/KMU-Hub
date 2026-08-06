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
	// ErrEmployeeRequired is returned when a document upload names no employee.
	ErrEmployeeRequired = errors.New("employee id is required")
	// ErrDocumentNotFound is returned when an employee document does not exist.
	ErrDocumentNotFound = errors.New("employee document not found")
	// ErrInsufficientDocumentAccess is returned when a caller does not have access to view a document.
	ErrInsufficientDocumentAccess = errors.New("insufficient access to view this document")

	// ErrAlreadyOffboarded is returned when the personnel record has already left.
	ErrAlreadyOffboarded = errors.New("employee has already been offboarded")
	// ErrSelfOffboard is returned when a caller tries to offboard their own account.
	ErrSelfOffboard = errors.New("an account cannot offboard itself")
	// ErrLastRoleAdmin is returned when offboarding would leave the tenant without
	// an active account that can hand out roles.
	ErrLastRoleAdmin = errors.New("the last account carrying role administration cannot be offboarded")
	// ErrSuccessorRequired is returned when the leaver still has direct reports
	// and no successor was named.
	ErrSuccessorRequired = errors.New("a successor is required: the employee still has direct reports")
	// ErrInvalidSuccessor is returned when the named successor is the leaver, is
	// not an employee of this tenant, or is already inactive.
	ErrInvalidSuccessor = errors.New("the named successor is not an active employee of this tenant")
	// ErrInvalidExitType is returned when the exit type is outside the accepted set.
	ErrInvalidExitType = errors.New("unknown exit type")
	// ErrExitBeforeLastWorkDay is returned when the exit date precedes the last working day.
	ErrExitBeforeLastWorkDay = errors.New("the exit date cannot precede the last working day")
)
