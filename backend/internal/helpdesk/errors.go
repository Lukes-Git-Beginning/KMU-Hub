package helpdesk

import "errors"

var (
	ErrTicketNotFound      = errors.New("ticket not found")
	ErrQueueNotFound       = errors.New("ticket queue not found")
	ErrSLAPolicyNotFound   = errors.New("sla policy not found")
	ErrInvalidStatus       = errors.New("invalid ticket status")
	ErrInvalidPriority     = errors.New("invalid ticket priority")
	ErrCannotMergeSelf     = errors.New("cannot merge ticket into itself")
	ErrAlreadyMerged       = errors.New("ticket is already merged")
	ErrCannedResponseNotFound = errors.New("canned response not found")
)
