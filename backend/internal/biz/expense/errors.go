package expense

import "errors"

var (
	// ErrNotFound is returned when an expense does not exist in the caller's tenant.
	ErrNotFound = errors.New("expense not found")
	// ErrNotPending is returned when approve/reject is asked for an expense that
	// was already decided. The decision is not reversible through this path.
	ErrNotPending = errors.New("expense is not pending")
	// ErrSelfApproval is returned when the submitter tries to decide their own
	// expense. Enforced server-side: the frontend hides the buttons, which is a
	// courtesy, not a control.
	ErrSelfApproval = errors.New("an expense cannot be decided by the person who submitted it")
	// ErrDecided is returned when an already decided expense is edited or deleted.
	ErrDecided = errors.New("a decided expense can no longer be modified")
	// ErrInvalidAmount is returned for a non-positive or unparsable amount.
	ErrInvalidAmount = errors.New("amount must be a positive number")
	// ErrInvalidDate is returned when the expense date is missing or not YYYY-MM-DD.
	ErrInvalidDate = errors.New("date must be a valid YYYY-MM-DD date")
	// ErrDescriptionRequired is returned for an empty description.
	ErrDescriptionRequired = errors.New("description is required")
	// ErrMissingActor is returned when no authenticated user reached the service.
	// Without one, neither ownership nor the four-eyes rule can hold.
	ErrMissingActor = errors.New("expense operations require an authenticated user")
)
