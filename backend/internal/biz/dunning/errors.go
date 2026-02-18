package dunning

import "errors"

var (
	// ErrDunningNotFound is returned when a dunning record does not exist.
	ErrDunningNotFound = errors.New("dunning record not found")
	// ErrDunningNotDraft is returned when an operation requires draft status.
	ErrDunningNotDraft = errors.New("dunning record is not in draft status")
	// ErrDunningMaxLevel is returned when the maximum dunning level (3) has already been reached.
	ErrDunningMaxLevel = errors.New("maximum dunning level (3) already reached for this invoice")
	// ErrConfigNotFound is returned when no dunning configuration exists.
	ErrConfigNotFound = errors.New("dunning configuration not found")
)
