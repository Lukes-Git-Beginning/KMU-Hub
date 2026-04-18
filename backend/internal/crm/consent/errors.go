package consent

import "errors"

var (
	ErrConsentNotFound         = errors.New("consent record not found")
	ErrContactNotFound         = errors.New("contact not found")
	ErrInvalidConsentType      = errors.New("invalid consent type")
	ErrInvalidLegalBasis       = errors.New("invalid legal basis")
	ErrDeletionRequestNotFound = errors.New("deletion request not found")
	ErrDeletionAlreadyComplete = errors.New("deletion request already completed")

	// ErrNoConsent is returned by Asserter when no active consent exists for
	// the given contact + channel. Callers must not proceed with the send.
	ErrNoConsent = errors.New("consent: no active consent for channel")

	// ErrContactMissing is returned by Asserter when the contact record cannot
	// be found during the consent pre-check.
	ErrContactMissing = errors.New("consent: contact not found")
)
