package absence

import "errors"

var (
	// ErrInvalidDateRange is returned when end date is before start date.
	ErrInvalidDateRange = errors.New("end date must be on or after start date")
)
