package calendar

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for calendar persistence
type Repository interface {
	// Calendars
	Create(ctx context.Context, calendar *models.Calendar) error
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Calendar, error)
	ListByUser(ctx context.Context, userID, tenantID uuid.UUID) ([]models.CalendarWithMemberInfo, error)
	Update(ctx context.Context, calendar *models.Calendar) error
	Delete(ctx context.Context, id, tenantID uuid.UUID) error

	// Members
	AddMember(ctx context.Context, member *models.CalendarMember) error
	RemoveMember(ctx context.Context, calendarID, userID uuid.UUID) error
	ListMembers(ctx context.Context, calendarID uuid.UUID) ([]models.CalendarMember, error)
	GetMember(ctx context.Context, calendarID, userID uuid.UUID) (*models.CalendarMember, error)
	UpdateMemberPermission(ctx context.Context, calendarID, userID uuid.UUID, permission string) error
	UpdateMemberVisibility(ctx context.Context, calendarID, userID uuid.UUID, visible bool) error
	UpdateMemberColorOverride(ctx context.Context, calendarID, userID uuid.UUID, color *string) error

	// Discovery & Subscription
	ListBrowsable(ctx context.Context, userID, tenantID uuid.UUID) ([]models.Calendar, error)
	Subscribe(ctx context.Context, calendarID, userID uuid.UUID) error
	Unsubscribe(ctx context.Context, calendarID, userID uuid.UUID) error

	// Event Categories
	CreateCategory(ctx context.Context, category *models.EventCategory) error
	ListCategories(ctx context.Context, userID, tenantID uuid.UUID) ([]models.EventCategory, error)
	DeleteCategory(ctx context.Context, id, userID, tenantID uuid.UUID) error

	// Preferences
	GetPreferences(ctx context.Context, userID uuid.UUID) (*models.UserCalendarPreferences, error)
	UpsertPreferences(ctx context.Context, prefs *models.UserCalendarPreferences) error

	// Auto-create personal calendar on first access
	EnsurePersonalCalendar(ctx context.Context, userID, tenantID uuid.UUID) (*models.Calendar, error)
}
