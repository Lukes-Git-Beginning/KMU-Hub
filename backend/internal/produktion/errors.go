package produktion

import "errors"

var (
	ErrOrderNotFound          = errors.New("production order not found")
	ErrOrderNumberTaken       = errors.New("production order number already exists for this tenant")
	ErrOrderNotDeletable      = errors.New("production order can only be deleted in planned status")
	ErrOrderInvalidTransition = errors.New("invalid status transition for production order")
	ErrBookingNotFound        = errors.New("machine booking not found")
	ErrBookingConflict        = errors.New("machine booking overlaps with an existing booking")
	ErrPlanNotFound           = errors.New("production plan not found")
	ErrInvalidInput           = errors.New("invalid input")
)
