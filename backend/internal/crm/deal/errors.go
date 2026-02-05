package deal

import "errors"

var (
	ErrDealNotFound       = errors.New("deal not found")
	ErrNameRequired       = errors.New("deal name is required")
	ErrInvalidCurrency    = errors.New("invalid currency code")
	ErrStageNotFound      = errors.New("pipeline stage not found")
	ErrContactNotFound    = errors.New("contact not found")
	ErrCompanyNotFound    = errors.New("company not found")
	ErrOwnerNotFound      = errors.New("owner not found")
	ErrTagNotFound        = errors.New("tag not found")
	ErrInvalidCustomField = errors.New("invalid custom field")
)
