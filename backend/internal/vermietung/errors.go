package vermietung

import "errors"

var (
	ErrObjectNotFound          = errors.New("rental object not found")
	ErrRentalNotFound          = errors.New("rental not found")
	ErrInspectionNotFound      = errors.New("rental inspection not found")
	ErrInvalidInput            = errors.New("vermietung input is invalid")
	ErrRentalConflict          = errors.New("rental period overlaps with an existing booking")
	ErrInvalidStateTransition  = errors.New("rental state transition is not allowed")
)
