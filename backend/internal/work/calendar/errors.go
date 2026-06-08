package calendar

import "errors"

var (
	// Booking Page errors
	ErrBookingPageNotFound            = errors.New("booking page not found")
	ErrBookingPageSlugRequired        = errors.New("booking page slug is required")
	ErrBookingPageCompanyNameRequired = errors.New("company name is required")
	ErrBookingPageInvalidRules        = errors.New("invalid availability rules")
	ErrBookingServiceNotFound         = errors.New("service not found on this booking page")
	ErrBookingSlotUnavailable         = errors.New("the requested time slot is not available")
	ErrBookingSlotInPast              = errors.New("the requested time slot is in the past or within the lead time")

	// Calendar errors
	ErrCalendarNotFound          = errors.New("calendar not found")
	ErrCalendarNameRequired      = errors.New("calendar name is required")
	ErrInvalidCalendarType       = errors.New("invalid calendar type")
	ErrMemberAlreadyExists       = errors.New("user is already a member of this calendar")
	ErrMemberNotFound            = errors.New("user is not a member of this calendar")
	ErrNotCalendarOwner          = errors.New("user is not the owner of this calendar")
	ErrNotCalendarAdmin          = errors.New("user is not an admin of this calendar")
	ErrInsufficientPermission    = errors.New("insufficient permission on this calendar")
	ErrCannotDeleteDefaultCalendar = errors.New("cannot delete the default calendar")
	ErrCannotDeletePersonalCalendar = errors.New("only the owner can delete a personal calendar")
	ErrCategoryNotFound          = errors.New("event category not found")
	ErrCategoryNameRequired      = errors.New("category name is required")
	ErrDuplicateCategoryName     = errors.New("category name already exists")
	ErrSubscriptionExists        = errors.New("already subscribed to this calendar")
	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrInvalidPermission         = errors.New("invalid calendar permission")
	ErrInvalidColor              = errors.New("invalid color format")
	ErrCannotSubscribePersonal   = errors.New("cannot subscribe to a personal calendar")
	ErrOwnerCannotBeAdded        = errors.New("owner is implicitly a member and cannot be added")
)
