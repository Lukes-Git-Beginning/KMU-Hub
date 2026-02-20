package leave

import "errors"

var (
	// ErrLeaveRequestNotFound is returned when a leave request does not exist.
	ErrLeaveRequestNotFound = errors.New("leave request not found")
	// ErrInvalidTransition is returned when an invalid status transition is attempted.
	ErrInvalidTransition = errors.New("invalid leave request status transition")
	// ErrInsufficientBalance is returned when leave balance is insufficient for the request.
	ErrInsufficientBalance = errors.New("insufficient leave balance")
	// ErrOverlappingApproved is returned as a warning when overlapping leave exists.
	ErrOverlappingApproved = errors.New("overlapping leave requests exist")
	// ErrCannotApproveOwnRequest is returned when an employee tries to approve their own request.
	ErrCannotApproveOwnRequest = errors.New("cannot approve own leave request")
	// ErrAUDocumentRequired is returned when AU document is needed but not provided.
	ErrAUDocumentRequired = errors.New("AU document required for sick leave exceeding threshold")
	// ErrNotAuthorizedToApprove is returned when the caller is not the employee's manager or HR.
	ErrNotAuthorizedToApprove = errors.New("not authorized to approve this leave request")
	// ErrInvalidDateRange is returned when end date is before start date.
	ErrInvalidDateRange = errors.New("end date must be on or after start date")
	// ErrLeaveTypeNotFound is returned when a leave type does not exist.
	ErrLeaveTypeNotFound = errors.New("leave type not found")
	// ErrSettingsNotFound is returned when HR company settings do not exist.
	ErrSettingsNotFound = errors.New("HR company settings not found")
	// ErrCannotCancelPastLeave is returned when trying to cancel approved leave that has already started.
	ErrCannotCancelPastLeave = errors.New("cannot cancel approved leave that has already started")
)
