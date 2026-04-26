package vertraege

import "errors"

var (
	ErrContractNotFound      = errors.New("contract not found")
	ErrContractNumberTaken   = errors.New("contract number already in use for this tenant")
	ErrPartyNotFound         = errors.New("contract party not found")
	ErrReminderNotFound      = errors.New("contract reminder not found")
	ErrInvalidInput          = errors.New("invalid input")
	ErrDeleteNonDraft        = errors.New("only draft contracts can be deleted")
)
