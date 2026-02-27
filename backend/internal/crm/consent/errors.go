package consent

import "errors"

var (
	ErrConsentNotFound         = errors.New("consent record not found")
	ErrContactNotFound         = errors.New("contact not found")
	ErrInvalidConsentType      = errors.New("invalid consent type")
	ErrInvalidLegalBasis       = errors.New("invalid legal basis")
	ErrDeletionRequestNotFound = errors.New("deletion request not found")
	ErrDeletionAlreadyComplete = errors.New("deletion request already completed")
)
