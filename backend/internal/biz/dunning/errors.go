package dunning

import "errors"

var (
	// ErrDunningNotFound is returned when a dunning record does not exist.
	ErrDunningNotFound = errors.New("dunning record not found")
	// ErrDunningNotDraft is returned when an operation requires draft status.
	ErrDunningNotDraft = errors.New("dunning record is not in draft status")
	// ErrDunningMaxLevel is returned when the maximum dunning level (3) has already been reached.
	ErrDunningMaxLevel = errors.New("maximum dunning level (3) already reached for this invoice")
	// ErrConfigNotFound is returned when no dunning configuration exists.
	ErrConfigNotFound = errors.New("dunning configuration not found")
	// ErrCompanySettingsMissing is returned by emailNotice when the tenant has no
	// company settings row, so the dunning PDF cannot be rendered. This is a
	// configuration gap, not a delivery failure: sendAndNotify treats it as
	// non-fatal (status still flips to sent, the miss is logged) while a genuine
	// SMTP/PDF error stays fail-closed.
	ErrCompanySettingsMissing = errors.New("no company settings configured for tenant")
)
