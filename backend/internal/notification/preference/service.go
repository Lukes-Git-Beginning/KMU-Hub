package preference

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// DeliveryDecision represents the result of evaluating a user's preferences
// for a specific event.
type DeliveryDecision struct {
	Deliver     bool   `json:"deliver"`
	InApp       bool   `json:"in_app"`
	DesktopPush bool   `json:"desktop_push"`
	Sound       string `json:"sound"`
	Reason      string `json:"reason"`
}

// Service handles notification preference evaluation and CRUD.
type Service struct {
	repo Repository
}

// NewService creates a new preference service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Evaluate determines whether and how to deliver a notification to a user.
// Pipeline: urgent bypass -> resource mute -> event-type pref -> module default
// -> system default -> low priority override -> quiet hours check
func (s *Service) Evaluate(ctx context.Context, userID uuid.UUID, event models.EventPayload) (*DeliveryDecision, error) {
	// Step 1: Urgent priority bypasses all filters
	if event.Priority == models.PriorityUrgent {
		return &DeliveryDecision{
			Deliver:     true,
			InApp:       true,
			DesktopPush: true,
			Sound:       "urgent",
			Reason:      "urgent priority bypasses all filters",
		}, nil
	}

	// Step 2: Check resource muting
	if event.ResourceID != "" {
		muted, err := s.repo.IsResourceMuted(ctx, userID, event.ModuleID, event.ResourceID)
		if err != nil {
			return nil, err
		}
		if muted {
			return &DeliveryDecision{
				Deliver: false,
				Reason:  "resource muted",
			}, nil
		}
	}

	// Step 3: Check event-type preference
	pref, err := s.repo.GetEventTypePreference(ctx, userID, event.Type)
	if err != nil && !errors.Is(err, ErrPreferenceNotFound) {
		return nil, err
	}

	// Step 4: Fall back to module default
	if pref == nil {
		pref, err = s.repo.GetModuleDefault(ctx, userID, event.ModuleID)
		if err != nil && !errors.Is(err, ErrPreferenceNotFound) {
			return nil, err
		}
	}

	// Step 5: System default if no preference found
	decision := &DeliveryDecision{
		Deliver:     true,
		InApp:       true,
		DesktopPush: true,
		Sound:       "default",
		Reason:      "system default",
	}

	if pref != nil {
		decision.InApp = pref.InApp
		decision.DesktopPush = pref.DesktopPush
		decision.Sound = pref.Sound
		decision.Reason = "user preference"

		// If user explicitly disabled all delivery channels
		if !pref.InApp && !pref.DesktopPush {
			decision.Deliver = false
			decision.Reason = "all delivery channels disabled by preference"
			return decision, nil
		}
	}

	// Step 6: Low priority override - never desktop push regardless of preferences
	if event.Priority == models.PriorityLow {
		decision.DesktopPush = false
		decision.Sound = ""
		decision.Reason = "low priority: in-app only"
		return decision, nil
	}

	// Step 7: Check quiet hours (only for normal priority)
	inQuietHours, err := s.isInQuietHours(ctx, userID)
	if err != nil {
		slog.Warn("quiet hours check failed, proceeding with normal delivery",
			"user_id", userID,
			"error", err,
		)
	} else if inQuietHours {
		decision.DesktopPush = false
		decision.Sound = ""
		decision.Reason = "quiet hours: in-app only"
		return decision, nil
	}

	return decision, nil
}

// isInQuietHours checks whether the user is currently in their quiet hours.
func (s *Service) isInQuietHours(ctx context.Context, userID uuid.UUID) (bool, error) {
	qh, err := s.repo.GetQuietHours(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrQuietHoursNotFound) {
			return false, nil
		}
		return false, err
	}

	// Check manual DND first
	if qh.ManualDND {
		if qh.ManualDNDUntil != nil {
			// Timed DND - check if still active
			if time.Now().Before(*qh.ManualDNDUntil) {
				return true, nil
			}
			// DND expired, but don't auto-update here (let a background task do it)
			return false, nil
		}
		// Indefinite manual DND
		return true, nil
	}

	// Scheduled quiet hours
	if !qh.Enabled {
		return false, nil
	}

	// Load timezone
	loc, err := time.LoadLocation(qh.Timezone)
	if err != nil {
		return false, ErrInvalidTimezone
	}

	now := time.Now().In(loc)

	// Check day of week (1=Monday, 7=Sunday)
	// Go's time.Weekday: 0=Sunday, 1=Monday, ..., 6=Saturday
	goWeekday := int(now.Weekday())
	isoWeekday := goWeekday
	if goWeekday == 0 {
		isoWeekday = 7 // Sunday = 7 in ISO
	}

	dayMatch := false
	for _, day := range qh.DaysOfWeek {
		if day == isoWeekday {
			dayMatch = true
			break
		}
	}
	if !dayMatch {
		return false, nil
	}

	// Parse start and end times
	startTime, err := time.Parse("15:04", qh.StartTime)
	if err != nil {
		return false, ErrInvalidTimeFormat
	}
	endTime, err := time.Parse("15:04", qh.EndTime)
	if err != nil {
		return false, ErrInvalidTimeFormat
	}

	currentMinutes := now.Hour()*60 + now.Minute()
	startMinutes := startTime.Hour()*60 + startTime.Minute()
	endMinutes := endTime.Hour()*60 + endTime.Minute()

	if startMinutes <= endMinutes {
		// Same-day window (e.g., 09:00 - 17:00)
		return currentMinutes >= startMinutes && currentMinutes < endMinutes, nil
	}

	// Overnight window (e.g., 18:00 - 08:00)
	return currentMinutes >= startMinutes || currentMinutes < endMinutes, nil
}

// ============================================================================
// CRUD Operations
// ============================================================================

// GetPreferences returns all preferences for a user, optionally filtered by module.
func (s *Service) GetPreferences(ctx context.Context, userID uuid.UUID, moduleID *string) ([]*models.NotificationPreference, error) {
	return s.repo.ListPreferences(ctx, userID, moduleID)
}

// UpdatePreference creates or updates a notification preference.
func (s *Service) UpdatePreference(ctx context.Context, pref *models.NotificationPreference) error {
	if pref.ID == uuid.Nil {
		pref.ID = uuid.New()
	}
	now := time.Now()
	if pref.CreatedAt.IsZero() {
		pref.CreatedAt = now
	}
	pref.UpdatedAt = now

	return s.repo.UpsertPreference(ctx, pref)
}

// MuteResource mutes notifications for a specific resource.
func (s *Service) MuteResource(ctx context.Context, userID uuid.UUID, moduleID, resourceID string) (*models.NotificationMute, error) {
	// Check if already muted
	muted, err := s.repo.IsResourceMuted(ctx, userID, moduleID, resourceID)
	if err != nil {
		return nil, err
	}
	if muted {
		return nil, ErrMuteAlreadyExists
	}

	mute := &models.NotificationMute{
		ID:         uuid.New(),
		UserID:     userID,
		ModuleID:   moduleID,
		ResourceID: resourceID,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.CreateMute(ctx, mute); err != nil {
		return nil, err
	}
	return mute, nil
}

// UnmuteResource removes a mute for a specific resource.
func (s *Service) UnmuteResource(ctx context.Context, userID uuid.UUID, moduleID, resourceID string) error {
	return s.repo.DeleteMute(ctx, userID, moduleID, resourceID)
}

// ListMutedResources returns muted resources for a user.
func (s *Service) ListMutedResources(ctx context.Context, userID uuid.UUID, moduleID *string, page, pageSize int) ([]*models.NotificationMute, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize
	return s.repo.ListMutes(ctx, userID, moduleID, offset, pageSize)
}

// GetQuietHours returns the quiet hours configuration for a user.
func (s *Service) GetQuietHours(ctx context.Context, userID uuid.UUID) (*models.QuietHours, error) {
	return s.repo.GetQuietHours(ctx, userID)
}

// UpdateQuietHours creates or updates quiet hours for a user.
func (s *Service) UpdateQuietHours(ctx context.Context, userID uuid.UUID, startTime, endTime, timezone string, daysOfWeek []int, enabled bool) (*models.QuietHours, error) {
	// Validate timezone
	if _, err := time.LoadLocation(timezone); err != nil {
		return nil, ErrInvalidTimezone
	}

	// Validate time format
	if _, err := time.Parse("15:04", startTime); err != nil {
		return nil, ErrInvalidTimeFormat
	}
	if _, err := time.Parse("15:04", endTime); err != nil {
		return nil, ErrInvalidTimeFormat
	}

	// Validate days of week
	for _, day := range daysOfWeek {
		if day < 1 || day > 7 {
			return nil, ErrInvalidDayOfWeek
		}
	}

	// Retrieve existing or create new
	qh, err := s.repo.GetQuietHours(ctx, userID)
	if err != nil && !errors.Is(err, ErrQuietHoursNotFound) {
		return nil, err
	}

	now := time.Now()
	if qh == nil {
		qh = &models.QuietHours{
			ID:        uuid.New(),
			UserID:    userID,
			CreatedAt: now,
		}
	}

	qh.StartTime = startTime
	qh.EndTime = endTime
	qh.Timezone = timezone
	qh.DaysOfWeek = daysOfWeek
	qh.Enabled = enabled
	qh.UpdatedAt = now

	if err := s.repo.UpsertQuietHours(ctx, qh); err != nil {
		return nil, err
	}
	return qh, nil
}

// ToggleManualDND toggles manual Do Not Disturb mode.
func (s *Service) ToggleManualDND(ctx context.Context, userID uuid.UUID, enabled bool, until *time.Time) (*models.QuietHours, error) {
	qh, err := s.repo.GetQuietHours(ctx, userID)
	if err != nil && !errors.Is(err, ErrQuietHoursNotFound) {
		return nil, err
	}

	now := time.Now()
	if qh == nil {
		qh = &models.QuietHours{
			ID:         uuid.New(),
			UserID:     userID,
			StartTime:  "18:00",
			EndTime:    "08:00",
			Timezone:   "Europe/Berlin",
			DaysOfWeek: []int{1, 2, 3, 4, 5, 6, 7},
			Enabled:    false,
			CreatedAt:  now,
		}
	}

	qh.ManualDND = enabled
	qh.ManualDNDUntil = until
	qh.UpdatedAt = now

	if err := s.repo.UpsertQuietHours(ctx, qh); err != nil {
		return nil, err
	}
	return qh, nil
}
