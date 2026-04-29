package event

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ReminderWithEvent combines a reminder with its event and target users
type ReminderWithEvent struct {
	Reminder models.EventReminder
	Event    models.CalendarEvent
	UserIDs  []uuid.UUID // creator + attendees who accepted
}

// Repository defines the interface for event persistence
type Repository interface {
	// Events
	Create(ctx context.Context, event *models.CalendarEvent) error
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.CalendarEvent, error)
	Update(ctx context.Context, event *models.CalendarEvent) error
	Delete(ctx context.Context, id, tenantID uuid.UUID) error
	ListInRange(ctx context.Context, calendarIDs []uuid.UUID, start, end time.Time, userID, tenantID uuid.UUID) ([]models.ExpandedEvent, error)
	ListRecurringOverlapping(ctx context.Context, calendarIDs []uuid.UUID, start, end time.Time, tenantID uuid.UUID) ([]models.CalendarEvent, error)

	// Attendees
	AddAttendee(ctx context.Context, attendee *models.EventAttendee) error
	RemoveAttendee(ctx context.Context, eventID, userID uuid.UUID) error
	UpdateRSVP(ctx context.Context, eventID, userID uuid.UUID, status string) error
	ListAttendees(ctx context.Context, eventID uuid.UUID) ([]models.EventAttendee, error)
	ListAttendeeEventIDs(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]uuid.UUID, error)

	// Exceptions
	CreateException(ctx context.Context, exception *models.EventException) error
	ListExceptions(ctx context.Context, eventID uuid.UUID, start, end time.Time) ([]models.EventException, error)
	DeleteExceptionsAfterDate(ctx context.Context, eventID uuid.UUID, date time.Time) error

	// Reminders
	SetReminders(ctx context.Context, eventID uuid.UUID, minutesBefore []int) error
	ListReminders(ctx context.Context, eventID uuid.UUID) ([]models.EventReminder, error)
	ListUpcomingReminders(ctx context.Context, windowStart, windowEnd time.Time) ([]ReminderWithEvent, error)

	// Task deadlines
	ListTaskDeadlinesInRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]models.TaskDeadlineStub, error)
}
