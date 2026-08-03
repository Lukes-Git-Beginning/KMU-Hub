package changerequest

import "errors"

var (
	// ErrNotFound is returned when a change request does not exist in the tenant.
	ErrNotFound = errors.New("change request not found")
	// ErrPendingRequestExists is returned when the proposer already has an open
	// request for the same field.
	ErrPendingRequestExists = errors.New("a pending request for this field already exists")
	// ErrFieldNotProposable is returned when the field is not one an employee may
	// propose a change for.
	ErrFieldNotProposable = errors.New("field cannot be proposed")
	// ErrNotPending is returned when a decision or cancellation targets a request
	// that was already decided.
	ErrNotPending = errors.New("change request is not pending")
	// ErrNotProposer is returned when somebody other than the proposer tries to
	// cancel a request.
	ErrNotProposer = errors.New("only the proposer may cancel this request")
	// ErrOutOfScope is returned when the approver's permission scope does not
	// reach the proposer — a manager deciding about somebody who does not report
	// to them.
	ErrOutOfScope = errors.New("proposer is outside the approver's scope")
	// ErrReasonRequired is returned when a rejection carries no reason.
	ErrReasonRequired = errors.New("rejection reason is required")
	// ErrProfileNotFound is returned when the proposer has no employee profile,
	// so an approval would have nothing to write to.
	ErrProfileNotFound = errors.New("employee profile not found for proposer")
)
