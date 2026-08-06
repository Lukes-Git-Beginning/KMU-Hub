package calendar

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// BookingRepository defines persistence for booking pages and public bookings.
type BookingRepository interface {
	// Booking Pages
	CreateBookingPage(ctx context.Context, page *models.BookingPage) error
	GetBookingPageByID(ctx context.Context, id, tenantID uuid.UUID) (*models.BookingPage, error)
	GetBookingPageBySlug(ctx context.Context, slug string) (*models.BookingPage, error)
	UpdateBookingPage(ctx context.Context, page *models.BookingPage) error
	DeleteBookingPage(ctx context.Context, id, tenantID uuid.UUID) error
	ListBookingPages(ctx context.Context, tenantID uuid.UUID, includeInactive bool) ([]models.BookingPage, error)

	// Public Bookings
	CreatePublicBooking(ctx context.Context, booking *models.PublicBooking) error
	GetBookedSlotsForPage(ctx context.Context, bookingPageID uuid.UUID, from, to time.Time) ([]models.PublicBooking, error)
	UpdatePublicBookingCalendarEventID(ctx context.Context, bookingID, tenantID, eventID uuid.UUID) error

	// Calendar Events overlap check (re-uses calendar event rows for staff availability)
	GetCalendarEventsInRange(ctx context.Context, calendarID uuid.UUID, from, to time.Time) ([]eventSlot, error)
}

// eventSlot is a minimal projection of a calendar event used for availability calculation.
type eventSlot struct {
	StartTime time.Time
	EndTime   time.Time
}
