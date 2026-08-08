package schichten

import "errors"

var (
	ErrShiftNotFound         = errors.New("shift not found")
	ErrTemplateNotFound      = errors.New("shift template not found")
	ErrAssignmentNotFound    = errors.New("shift assignment not found")
	ErrInvalidInput          = errors.New("schichten input is invalid")
	ErrAlreadyAssigned       = errors.New("employee is already assigned to this shift")
	ErrArbzgViolation        = errors.New("ArbZG §5 violation: minimum 11 hours rest period not met between consecutive shifts")
	ErrShiftFull             = errors.New("shift has reached maximum capacity")
	ErrSwapRequestNotFound   = errors.New("swap request not found")
	ErrSwapAlreadyProcessed  = errors.New("swap request has already been processed")

	// JArbSchG (Jugendarbeitsschutzgesetz) violations. They only ever apply to
	// employees whose HR profile carries is_minor.
	ErrJArbSchGNightWork  = errors.New("JArbSchG §14 violation: minors must not be scheduled between 20:00 and 06:00")
	ErrJArbSchGDailyHours = errors.New("JArbSchG §8 violation: minors must not work more than 8 hours per shift")
	ErrJArbSchGWeekend    = errors.New("JArbSchG §16/§17 violation: minors must not be scheduled on Saturdays or Sundays")
)
