package recurring

import "errors"

var (
	// ErrNotFound is returned when no schedule with the given id exists for the tenant.
	ErrNotFound = errors.New("recurring invoice not found")
	// ErrNoLineItems is returned when a schedule would carry no billable position.
	ErrNoLineItems = errors.New("recurring invoice requires at least one line item")
	// ErrInvalidInterval is returned for an interval outside the four supported ones.
	ErrInvalidInterval = errors.New("invalid recurrence interval")
	// ErrInvalidStatus is returned for a status outside active/paused/ended.
	ErrInvalidStatus = errors.New("invalid recurring invoice status")
	// ErrInvalidDateRange is returned when end_date lies before start_date.
	ErrInvalidDateRange = errors.New("end date must not precede start date")
	// ErrNotActive is returned when generation is requested for a paused or ended
	// schedule. Resuming is an explicit user action, never an implicit side effect
	// of pressing "generate".
	ErrNotActive = errors.New("recurring invoice is not active")
	// ErrScheduleExhausted is returned when the next run would fall past the end date.
	ErrScheduleExhausted = errors.New("recurring invoice has reached its end date")
)
