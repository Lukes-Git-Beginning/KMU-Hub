package helpdesk

import "errors"

var (
	ErrTicketNotFound         = errors.New("ticket not found")
	ErrQueueNotFound          = errors.New("ticket queue not found")
	ErrSLAPolicyNotFound      = errors.New("sla policy not found")
	ErrInvalidStatus          = errors.New("invalid ticket status")
	ErrInvalidPriority        = errors.New("invalid ticket priority")
	ErrCannotMergeSelf        = errors.New("cannot merge ticket into itself")
	ErrAlreadyMerged          = errors.New("ticket is already merged")
	ErrCannedResponseNotFound = errors.New("canned response not found")
	ErrKBArticleNotFound      = errors.New("kb article not found")
	ErrRoutingRuleNotFound    = errors.New("routing rule not found")
	ErrContactNotFound        = errors.New("contact not found in tenant")
	ErrOrgNotFound            = errors.New("organization not found in tenant")
	ErrInvalidSourceChannel   = errors.New("invalid ticket source channel")
	ErrMessageAlreadyLinked   = errors.New("inbox message already linked to a ticket")
	ErrInvalidCsatRating      = errors.New("csat rating must be between 1 and 5")
)
