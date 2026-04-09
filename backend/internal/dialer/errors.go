package dialer

import "errors"

var (
	ErrCampaignNotFound         = errors.New("campaign not found")
	ErrCampaignNotDraft         = errors.New("campaign must be in draft status")
	ErrCampaignNotActive        = errors.New("campaign must be in active status")
	ErrInvalidStatusTransition  = errors.New("invalid campaign status transition")
	ErrNoContactsAvailable      = errors.New("no contacts available in queue")
	ErrContactAlreadyInCampaign = errors.New("contact already in campaign")
	ErrCallSessionNotFound      = errors.New("call session not found")
	ErrOutcomeNotFound          = errors.New("outcome not found")
	ErrAgentNotAvailable        = errors.New("agent is not available")
	ErrInvalidPhoneNumber       = errors.New("invalid phone number")
	ErrCampaignHasNoContacts    = errors.New("campaign has no contacts")
	ErrContactNotInCampaign     = errors.New("contact not in campaign")
)
