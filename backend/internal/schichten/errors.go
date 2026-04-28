package schichten

import "errors"

var (
	ErrShiftNotFound      = errors.New("shift not found")
	ErrTemplateNotFound   = errors.New("shift template not found")
	ErrAssignmentNotFound = errors.New("shift assignment not found")
	ErrInvalidInput       = errors.New("schichten input is invalid")
	ErrAlreadyAssigned    = errors.New("employee is already assigned to this shift")
	ErrArbzgViolation     = errors.New("ArbZG §5 violation: minimum 11 hours rest period not met between consecutive shifts")
	ErrShiftFull          = errors.New("shift has reached maximum capacity")
)
