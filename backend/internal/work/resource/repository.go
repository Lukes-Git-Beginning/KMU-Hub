package resource

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ResourceFilters contains filter criteria for listing resources
type ResourceFilters struct {
	TenantID    uuid.UUID
	Type        *string
	MinCapacity *int
	Tags        []string
	Floor       *string
	IsActive    *bool
}

// Repository defines the interface for resource persistence
type Repository interface {
	// Resources
	Create(ctx context.Context, resource *models.Resource) error
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Resource, error)
	List(ctx context.Context, filters ResourceFilters) ([]models.Resource, error)
	Update(ctx context.Context, resource *models.Resource) error
	Delete(ctx context.Context, id, tenantID uuid.UUID) error
	SetTags(ctx context.Context, resourceID uuid.UUID, tags []string) error

	// Bookings
	CreateBooking(ctx context.Context, booking *models.ResourceBooking) error
	CancelBooking(ctx context.Context, bookingID, tenantID uuid.UUID) error
	ListBookings(ctx context.Context, resourceID uuid.UUID, start, end time.Time) ([]models.ResourceBooking, error)
	ListBookingsByEvent(ctx context.Context, eventID uuid.UUID) ([]models.ResourceBooking, error)
	GetBooking(ctx context.Context, bookingID, tenantID uuid.UUID) (*models.ResourceBooking, error)

	// Availability
	FindAvailableResources(ctx context.Context, start, end time.Time, filters ResourceFilters) ([]models.Resource, error)
	FindAlternatives(ctx context.Context, excludeID uuid.UUID, start, end time.Time, resourceType string, tenantID uuid.UUID) ([]AlternativeResource, error)
}
