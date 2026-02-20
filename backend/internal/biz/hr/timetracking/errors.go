package timetracking

import "errors"

var (
	// ErrAlreadyClockedIn is returned when attempting to clock in while already clocked in.
	ErrAlreadyClockedIn = errors.New("already clocked in")
	// ErrNotClockedIn is returned when attempting to clock out or start a break without an active shift.
	ErrNotClockedIn = errors.New("not clocked in")
	// ErrAlreadyOnBreak is returned when attempting to start a break while already on break.
	ErrAlreadyOnBreak = errors.New("already on break")
	// ErrNotOnBreak is returned when attempting to end a break without an active break.
	ErrNotOnBreak = errors.New("not on break")
	// ErrShiftAlreadyCompleted is returned when attempting to modify a completed shift.
	ErrShiftAlreadyCompleted = errors.New("shift already completed")
	// ErrCorrectionNotFound is returned when a time correction entry does not exist.
	ErrCorrectionNotFound = errors.New("time correction not found")
	// ErrNotAuthorizedToApproveCorrection is returned when the caller cannot approve a correction.
	ErrNotAuthorizedToApproveCorrection = errors.New("not authorized to approve this time correction")
	// ErrMaxDailyHoursExceeded is returned when today's work time has reached the ArbZG 10h maximum.
	ErrMaxDailyHoursExceeded = errors.New("ArbZG: maximum daily work time of 10 hours reached")
	// ErrWorkTimeEntryNotFound is returned when a work time entry does not exist.
	ErrWorkTimeEntryNotFound = errors.New("work time entry not found")
)
