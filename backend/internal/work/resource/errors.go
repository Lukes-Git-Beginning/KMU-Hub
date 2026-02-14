package resource

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors for resource operations
var (
	ErrResourceNotFound      = fmt.Errorf("resource not found")
	ErrResourceNameRequired  = fmt.Errorf("resource name is required")
	ErrInvalidResourceType   = fmt.Errorf("invalid resource type")
	ErrResourceInactive      = fmt.Errorf("resource is inactive")
	ErrBookingNotFound       = fmt.Errorf("booking not found")
	ErrBookingConflict       = fmt.Errorf("resource booking conflict")
	ErrBookingAlreadyCancelled = fmt.Errorf("booking is already cancelled")
	ErrNotBookingOwner       = fmt.Errorf("not the booking owner")
	ErrInvalidBookingTimeRange = fmt.Errorf("booking end time must be after start time")
	ErrNotAuthorized         = fmt.Errorf("not authorized for this resource operation")
)

// AlternativeResource describes an available substitute resource
type AlternativeResource struct {
	ResourceID   uuid.UUID
	ResourceName string
	Capacity     *int
	Floor        *string
}

// BookingConflictError carries details about a double-booking conflict
// including alternative resources that are available for the same time range
type BookingConflictError struct {
	ResourceName  string
	ConflictStart time.Time
	ConflictEnd   time.Time
	Alternatives  []AlternativeResource
}

func (e *BookingConflictError) Error() string {
	return fmt.Sprintf("resource %s is booked from %s to %s", e.ResourceName, e.ConflictStart.Format(time.RFC3339), e.ConflictEnd.Format(time.RFC3339))
}
