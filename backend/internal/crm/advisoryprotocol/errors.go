package advisoryprotocol

import "errors"

// Sentinel errors for the advisory protocol domain.
var (
	ErrProtocolNotFound   = errors.New("advisory protocol not found")
	ErrProtocolFinalized  = errors.New("advisory protocol is finalized and immutable")
	ErrContactNotFound    = errors.New("contact not found in tenant")
	ErrInvalidTenant      = errors.New("invalid tenant id")
	ErrInvalidRiskClass   = errors.New("risk class must be between 1 and 7")
)
