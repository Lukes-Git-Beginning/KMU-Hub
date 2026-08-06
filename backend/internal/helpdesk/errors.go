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
	ErrInvalidChannel         = errors.New("invalid ticket intake channel")
	ErrInvalidRequesterEmail  = errors.New("invalid requester email")
	ErrRequesterNameTooLong   = errors.New("requester name must be at most 200 characters")
	ErrInvalidCustomFields    = errors.New("custom fields must be a flat map of string, number or boolean values")
	// ErrMissingRequester is the service-side mirror of the DB CHECK
	// chk_tickets_requester_identity: a ticket names either a tenant user or an
	// external requester reachable by mail, never neither.
	ErrMissingRequester = errors.New("ticket needs either a requester_id or an external requester with an email")
	ErrMessageAlreadyLinked   = errors.New("inbox message already linked to a ticket")
	ErrInvalidCsatRating      = errors.New("csat rating must be between 1 and 5")
	ErrInvalidCsatDelay       = errors.New("csat survey delay must be between 0 and 20160 minutes")
	ErrCsatQuestionTooLong    = errors.New("csat survey question must be at most 500 characters")
	ErrCsatCommentTooLong     = errors.New("csat comment must be at most 2000 characters")
	// ErrCsatSurveyNotFound is the single verdict the public redeem path gives
	// for an unknown, malformed, expired, revoked or already-redeemed survey
	// link. Splitting it into finer errors would turn the endpoint into an
	// oracle for which tokens exist.
	ErrCsatSurveyNotFound = errors.New("csat survey link not found")
)
