package dialer

import (
	"errors"

	"github.com/kmuhub/kmuhub/internal/dachfmt"
)

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
	// ErrInvalidPhoneNumber aliases the dachfmt sentinel so existing
	// errors.Is(err, dialer.ErrInvalidPhoneNumber) checks keep working after
	// the phone normalisation moved to internal/dachfmt.
	ErrInvalidPhoneNumber      = dachfmt.ErrInvalidPhoneNumber
	ErrCampaignHasNoContacts   = errors.New("campaign has no contacts")
	ErrContactNotInCampaign    = errors.New("contact not in campaign")
	ErrCampaignContactNotFound = errors.New("campaign contact not found")
	// ErrConsentNotConfigured is returned when an outbound call is attempted on a
	// service built without a consent asserter (the bare NewService path).
	// Fail-closed guard (DSGVO / §7 UWG): such a service must never place calls.
	ErrConsentNotConfigured = errors.New("dialer consent asserter not configured")
)
