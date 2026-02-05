package report

import "errors"

var (
	ErrInvalidDateRange = errors.New("invalid date range: start date must be before end date")
	ErrStartDateRequired = errors.New("start date is required")
	ErrEndDateRequired   = errors.New("end date is required")
)
