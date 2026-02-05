package activity

import "errors"

var (
	ErrActivityNotFound    = errors.New("activity not found")
	ErrSubjectRequired     = errors.New("activity subject is required")
	ErrInvalidActivityType = errors.New("invalid activity type")
	ErrContactNotFound     = errors.New("contact not found")
	ErrCompanyNotFound     = errors.New("company not found")
	ErrDealNotFound        = errors.New("deal not found")
	ErrUserNotFound        = errors.New("user not found")
	ErrTagNotFound         = errors.New("tag not found")
	ErrInvalidCustomField  = errors.New("invalid custom field")
)
