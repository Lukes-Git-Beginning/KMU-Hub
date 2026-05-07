package calendar

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// validCalendarViews contains the allowed default view types for preferences
var validCalendarViews = map[string]bool{
	"day":   true,
	"week":  true,
	"month": true,
}

// Service handles calendar business logic
type Service struct {
	repo Repository
}

// NewService creates a new calendar service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains the data needed to create a calendar
type CreateInput struct {
	Name         string
	Description  *string
	CalendarType string
	Color        string
	Timezone     string
}

// Create creates a new calendar
func (s *Service) Create(ctx context.Context, userID, tenantID uuid.UUID, input CreateInput) (*models.Calendar, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrCalendarNameRequired
	}

	// Validate type
	if !models.ValidCalendarTypes[input.CalendarType] {
		return nil, ErrInvalidCalendarType
	}

	// Validate color
	color := input.Color
	if color == "" {
		color = "#4285F4"
	}
	if !hexColorRegex.MatchString(color) {
		return nil, ErrInvalidColor
	}

	// Default timezone
	tz := input.Timezone
	if tz == "" {
		tz = "Europe/Berlin"
	}

	now := time.Now()
	cal := &models.Calendar{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Name:         name,
		Description:  input.Description,
		CalendarType: input.CalendarType,
		Color:        color,
		OwnerID:      userID,
		IsDefault:    false,
		Timezone:     tz,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if createErr := s.repo.Create(ctx, cal); createErr != nil {
		return nil, createErr
	}

	slog.Info("calendar created",
		"calendar_id", cal.ID,
		"name", cal.Name,
		"type", cal.CalendarType,
		"owner_id", userID,
		"tenant_id", tenantID,
	)

	return cal, nil
}

// Get retrieves a calendar by ID
func (s *Service) Get(ctx context.Context, calendarID, tenantID uuid.UUID) (*models.Calendar, error) {
	return s.repo.GetByID(ctx, calendarID, tenantID)
}

// ListByUser lists all calendars a user owns or is a member of.
// Ensures the personal calendar exists first.
func (s *Service) ListByUser(ctx context.Context, userID, tenantID uuid.UUID) ([]models.CalendarWithMemberInfo, error) {
	// Ensure personal calendar exists
	if _, err := s.repo.EnsurePersonalCalendar(ctx, userID, tenantID); err != nil {
		return nil, err
	}

	return s.repo.ListByUser(ctx, userID, tenantID)
}

// UpdateInput contains fields that can be updated on a calendar
type UpdateInput struct {
	Name        *string
	Description *string
	Color       *string
	Timezone    *string
}

// Update updates an existing calendar. Only owner or admin can update.
func (s *Service) Update(ctx context.Context, calendarID, actorID, tenantID uuid.UUID, input UpdateInput) (*models.Calendar, error) {
	cal, err := s.repo.GetByID(ctx, calendarID, tenantID)
	if err != nil {
		return nil, err
	}

	// Check permission: must be owner or admin member
	if cal.OwnerID != actorID {
		if permErr := s.requirePermission(ctx, calendarID, actorID, models.CalendarPermissionAdmin); permErr != nil {
			return nil, permErr
		}
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrCalendarNameRequired
		}
		cal.Name = name
	}

	if input.Description != nil {
		cal.Description = input.Description
	}

	if input.Color != nil {
		if !hexColorRegex.MatchString(*input.Color) {
			return nil, ErrInvalidColor
		}
		cal.Color = *input.Color
	}

	if input.Timezone != nil {
		cal.Timezone = *input.Timezone
	}

	cal.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, cal); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("calendar updated", "calendar_id", cal.ID)

	return cal, nil
}

// Delete deletes a calendar. Only owner can delete. Cannot delete default calendar.
func (s *Service) Delete(ctx context.Context, calendarID, actorID, tenantID uuid.UUID) error {
	cal, err := s.repo.GetByID(ctx, calendarID, tenantID)
	if err != nil {
		return err
	}

	if cal.OwnerID != actorID {
		return ErrNotCalendarOwner
	}

	if cal.IsDefault {
		return ErrCannotDeleteDefaultCalendar
	}

	if deleteErr := s.repo.Delete(ctx, calendarID, tenantID); deleteErr != nil {
		return deleteErr
	}

	slog.Info("calendar deleted", "calendar_id", calendarID)

	return nil
}

// AddMember adds a user as a member of a calendar. Only owner or admin can add members.
func (s *Service) AddMember(ctx context.Context, calendarID, actorID, targetUserID uuid.UUID, tenantID uuid.UUID, permission string) error {
	cal, err := s.repo.GetByID(ctx, calendarID, tenantID)
	if err != nil {
		return err
	}

	// Cannot add owner as member (owner is implicit admin)
	if cal.OwnerID == targetUserID {
		return ErrOwnerCannotBeAdded
	}

	// Validate permission
	if !models.ValidCalendarPermissions[permission] {
		return ErrInvalidPermission
	}

	// Check actor is owner or admin
	if cal.OwnerID != actorID {
		if permErr := s.requirePermission(ctx, calendarID, actorID, models.CalendarPermissionAdmin); permErr != nil {
			return permErr
		}
	}

	now := time.Now()
	member := &models.CalendarMember{
		CalendarID: calendarID,
		TenantID:   tenantID,
		UserID:     targetUserID,
		Permission: permission,
		IsVisible:  true,
		CreatedAt:  now,
	}

	if addErr := s.repo.AddMember(ctx, member); addErr != nil {
		return addErr
	}

	slog.Info("calendar member added",
		"calendar_id", calendarID,
		"user_id", targetUserID,
		"permission", permission,
	)

	return nil
}

// RemoveMember removes a user from a calendar. Only owner or admin can remove members.
func (s *Service) RemoveMember(ctx context.Context, calendarID, actorID, targetUserID, tenantID uuid.UUID) error {
	cal, err := s.repo.GetByID(ctx, calendarID, tenantID)
	if err != nil {
		return err
	}

	// Cannot remove owner
	if cal.OwnerID == targetUserID {
		return ErrNotCalendarOwner
	}

	// Check actor is owner or admin
	if cal.OwnerID != actorID {
		if permErr := s.requirePermission(ctx, calendarID, actorID, models.CalendarPermissionAdmin); permErr != nil {
			return permErr
		}
	}

	if removeErr := s.repo.RemoveMember(ctx, calendarID, targetUserID); removeErr != nil {
		return removeErr
	}

	slog.Info("calendar member removed",
		"calendar_id", calendarID,
		"user_id", targetUserID,
	)

	return nil
}

// UpdateMemberPermission changes a member's permission level. Only owner or admin can change.
func (s *Service) UpdateMemberPermission(ctx context.Context, calendarID, actorID, targetUserID, tenantID uuid.UUID, permission string) error {
	cal, err := s.repo.GetByID(ctx, calendarID, tenantID)
	if err != nil {
		return err
	}

	// Cannot change owner's permission (implicit admin)
	if cal.OwnerID == targetUserID {
		return ErrOwnerCannotBeAdded
	}

	// Validate permission
	if !models.ValidCalendarPermissions[permission] {
		return ErrInvalidPermission
	}

	// Check actor is owner or admin
	if cal.OwnerID != actorID {
		if permErr := s.requirePermission(ctx, calendarID, actorID, models.CalendarPermissionAdmin); permErr != nil {
			return permErr
		}
	}

	// Verify target is actually a member
	if _, memberErr := s.repo.GetMember(ctx, calendarID, targetUserID); memberErr != nil {
		return memberErr
	}

	if updateErr := s.repo.UpdateMemberPermission(ctx, calendarID, targetUserID, permission); updateErr != nil {
		return updateErr
	}

	slog.Info("calendar member permission updated",
		"calendar_id", calendarID,
		"user_id", targetUserID,
		"permission", permission,
	)

	return nil
}

// ListMembers returns all members of a calendar
func (s *Service) ListMembers(ctx context.Context, calendarID, tenantID uuid.UUID) ([]models.CalendarMember, error) {
	if _, err := s.repo.GetByID(ctx, calendarID, tenantID); err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx, calendarID)
}

// ListBrowsable lists shared calendars the user can subscribe to
func (s *Service) ListBrowsable(ctx context.Context, userID, tenantID uuid.UUID) ([]models.Calendar, error) {
	return s.repo.ListBrowsable(ctx, userID, tenantID)
}

// Subscribe subscribes a user to a shared calendar with view permission
func (s *Service) Subscribe(ctx context.Context, calendarID, userID, tenantID uuid.UUID) error {
	cal, err := s.repo.GetByID(ctx, calendarID, tenantID)
	if err != nil {
		return err
	}

	// Can only subscribe to shared calendars
	if cal.CalendarType == models.CalendarTypePersonal {
		return ErrCannotSubscribePersonal
	}

	// Cannot subscribe to own calendar
	if cal.OwnerID == userID {
		return ErrSubscriptionExists
	}

	if subscribeErr := s.repo.Subscribe(ctx, calendarID, userID); subscribeErr != nil {
		return subscribeErr
	}

	slog.Info("calendar subscribed",
		"calendar_id", calendarID,
		"user_id", userID,
	)

	return nil
}

// Unsubscribe removes a user's subscription from a calendar
func (s *Service) Unsubscribe(ctx context.Context, calendarID, userID, tenantID uuid.UUID) error {
	cal, err := s.repo.GetByID(ctx, calendarID, tenantID)
	if err != nil {
		return err
	}

	// Cannot unsubscribe from own calendar
	if cal.OwnerID == userID {
		return ErrNotCalendarOwner
	}

	if unsubErr := s.repo.Unsubscribe(ctx, calendarID, userID); unsubErr != nil {
		return unsubErr
	}

	slog.Info("calendar unsubscribed",
		"calendar_id", calendarID,
		"user_id", userID,
	)

	return nil
}

// CreateCategory creates a new event category for a user
func (s *Service) CreateCategory(ctx context.Context, userID, tenantID uuid.UUID, name, color string) (*models.EventCategory, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrCategoryNameRequired
	}

	if color == "" {
		color = "#7986CB"
	}
	if !hexColorRegex.MatchString(color) {
		return nil, ErrInvalidColor
	}

	now := time.Now()
	cat := &models.EventCategory{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		Name:      name,
		Color:     color,
		SortOrder: 0,
		CreatedAt: now,
	}

	if createErr := s.repo.CreateCategory(ctx, cat); createErr != nil {
		return nil, createErr
	}

	slog.Info("event category created",
		"category_id", cat.ID,
		"name", cat.Name,
		"user_id", userID,
	)

	return cat, nil
}

// ListCategories lists all event categories for a user
func (s *Service) ListCategories(ctx context.Context, userID, tenantID uuid.UUID) ([]models.EventCategory, error) {
	return s.repo.ListCategories(ctx, userID, tenantID)
}

// DeleteCategory deletes an event category owned by the user
func (s *Service) DeleteCategory(ctx context.Context, categoryID, userID, tenantID uuid.UUID) error {
	if deleteErr := s.repo.DeleteCategory(ctx, categoryID, userID, tenantID); deleteErr != nil {
		return deleteErr
	}

	slog.Info("event category deleted",
		"category_id", categoryID,
		"user_id", userID,
	)

	return nil
}

// GetPreferences returns the user's calendar preferences, or defaults if not set
func (s *Service) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.UserCalendarPreferences, error) {
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}

	if prefs == nil {
		// Return defaults
		return &models.UserCalendarPreferences{
			UserID:                       userID,
			DefaultView:                  "week",
			WeekDays:                     5,
			DefaultReminderMinutes:       15,
			DefaultAllDayReminderMinutes: 1440,
			ShowTaskDeadlines:            true,
			UpdatedAt:                    time.Now(),
		}, nil
	}

	return prefs, nil
}

// UpsertPreferences creates or updates the user's calendar preferences
func (s *Service) UpsertPreferences(ctx context.Context, prefs *models.UserCalendarPreferences) error {
	// Validate view type
	if !validCalendarViews[prefs.DefaultView] {
		return ErrInvalidCalendarType
	}

	// Validate week days (5 or 7)
	if prefs.WeekDays != 5 && prefs.WeekDays != 7 {
		prefs.WeekDays = 5
	}

	prefs.UpdatedAt = time.Now()

	if upsertErr := s.repo.UpsertPreferences(ctx, prefs); upsertErr != nil {
		return upsertErr
	}

	slog.Info("calendar preferences updated",
		"user_id", prefs.UserID,
		"default_view", prefs.DefaultView,
	)

	return nil
}

// requirePermission checks if the actor has at least the required permission level on a calendar
func (s *Service) requirePermission(ctx context.Context, calendarID, actorID uuid.UUID, required string) error {
	member, err := s.repo.GetMember(ctx, calendarID, actorID)
	if err != nil {
		return ErrInsufficientPermission
	}

	// Permission hierarchy: view < edit < admin
	hierarchy := map[string]int{
		models.CalendarPermissionView:  1,
		models.CalendarPermissionEdit:  2,
		models.CalendarPermissionAdmin: 3,
	}

	actorLevel, ok := hierarchy[member.Permission]
	if !ok {
		return ErrInsufficientPermission
	}

	requiredLevel, ok := hierarchy[required]
	if !ok {
		return ErrInsufficientPermission
	}

	if actorLevel < requiredLevel {
		return ErrInsufficientPermission
	}

	return nil
}
