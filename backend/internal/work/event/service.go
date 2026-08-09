package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/work/calendar"
)

// Service handles event business logic
type Service struct {
	repo         Repository
	calendarRepo calendar.Repository
	emitter      EventEmitter
}

// NewService creates a new event service
func NewService(repo Repository, calendarRepo calendar.Repository) *Service {
	return &Service{
		repo:         repo,
		calendarRepo: calendarRepo,
	}
}

// SetEventEmitter sets the optional event emitter for notification events.
func (s *Service) SetEventEmitter(emitter EventEmitter) {
	s.emitter = emitter
}

// CreateInput contains the data needed to create an event
type CreateInput struct {
	TenantID     uuid.UUID
	CalendarID   uuid.UUID
	Title        string
	Description  *string
	Location     *string
	ResourceID   *uuid.UUID
	StartTime    time.Time
	EndTime      time.Time
	IsAllDay     bool
	Timezone     string
	RRule        *string
	HasVideoCall bool
	CategoryID   *uuid.UUID
	AttendeeIDs  []uuid.UUID
}

// Create creates a new calendar event
func (s *Service) Create(ctx context.Context, actorID uuid.UUID, input CreateInput) (*models.CalendarEvent, error) {
	// Validate title
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrEventTitleRequired
	}

	// Validate time range
	if !input.IsAllDay && !input.EndTime.After(input.StartTime) {
		return nil, ErrInvalidTimeRange
	}

	// Validate RRULE if present
	if input.RRule != nil && *input.RRule != "" {
		if err := ValidateRRule(*input.RRule); err != nil {
			return nil, err
		}
	}

	// Check calendar permissions
	cal, err := s.calendarRepo.GetByID(ctx, input.CalendarID, input.TenantID)
	if err != nil {
		return nil, err
	}
	if cal.OwnerID != actorID {
		member, memberErr := s.calendarRepo.GetMember(ctx, input.CalendarID, actorID)
		if memberErr != nil {
			return nil, ErrInsufficientPermission
		}
		if member.Permission == models.CalendarPermissionView {
			return nil, ErrInsufficientPermission
		}
	}

	// Default timezone
	tz := input.Timezone
	if tz == "" {
		tz = cal.Timezone
	}

	now := time.Now()
	eventID := uuid.New()

	// Generate LiveKit room name if video call
	var livekitRoomName *string
	if input.HasVideoCall {
		roomName := fmt.Sprintf("cal-%s", eventID.String()[:8])
		livekitRoomName = &roomName
	}

	evt := &models.CalendarEvent{
		ID:              eventID,
		TenantID:        input.TenantID,
		CalendarID:      input.CalendarID,
		Title:           title,
		Description:     input.Description,
		Location:        input.Location,
		ResourceID:      input.ResourceID,
		StartTime:       input.StartTime,
		EndTime:         input.EndTime,
		IsAllDay:        input.IsAllDay,
		Timezone:        tz,
		RRule:           input.RRule,
		HasVideoCall:    input.HasVideoCall,
		LiveKitRoomName: livekitRoomName,
		CategoryID:      input.CategoryID,
		CreatedBy:       actorID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if createErr := s.repo.Create(ctx, evt); createErr != nil {
		return nil, createErr
	}

	// Add creator as attendee with 'accepted'
	creatorAttendee := &models.EventAttendee{
		EventID:    eventID,
		UserID:     actorID,
		RSVPStatus: models.RSVPAccepted,
		CreatedAt:  now,
	}
	if addErr := s.repo.AddAttendee(ctx, creatorAttendee); addErr != nil {
		slog.Warn("failed to add creator as attendee", "error", addErr)
	}

	// Add additional attendees as pending
	for _, attendeeID := range input.AttendeeIDs {
		if attendeeID == actorID {
			continue // already added
		}
		attendee := &models.EventAttendee{
			EventID:    eventID,
			UserID:     attendeeID,
			RSVPStatus: models.RSVPPending,
			CreatedAt:  now,
		}
		if addErr := s.repo.AddAttendee(ctx, attendee); addErr != nil {
			slog.Warn("failed to add attendee", "user_id", attendeeID, "error", addErr)
		}

		// Emit invite notification
		s.emitEvent(ctx, "calendar.event.invited", actorID, eventID, input.CalendarID, []string{attendeeID.String()})
	}

	// Emit creation event
	s.emitEvent(ctx, "calendar.event.created", actorID, eventID, input.CalendarID, nil)

	slog.Info("event created",
		"event_id", eventID,
		"calendar_id", input.CalendarID,
		"title", title,
	)

	return evt, nil
}

// Get retrieves an event by ID
func (s *Service) Get(ctx context.Context, eventID, tenantID uuid.UUID) (*models.CalendarEvent, error) {
	return s.repo.GetByID(ctx, eventID, tenantID)
}

// UpdateInput contains fields that can be updated on an event
type UpdateInput struct {
	Title        *string
	Description  *string
	Location     *string
	ResourceID   *uuid.UUID
	StartTime    *time.Time
	EndTime      *time.Time
	IsAllDay     *bool
	CategoryID   *uuid.UUID
	HasVideoCall *bool
}

// UpdateEvent updates a non-recurring event
func (s *Service) UpdateEvent(ctx context.Context, eventID, actorID, tenantID uuid.UUID, changes UpdateInput) (*models.CalendarEvent, error) {
	evt, err := s.repo.GetByID(ctx, eventID, tenantID)
	if err != nil {
		return nil, err
	}

	// Check permission: must be creator or have edit access to calendar
	if evt.CreatedBy != actorID {
		if permErr := s.requireCalendarEditPermission(ctx, evt.CalendarID, actorID, evt.TenantID); permErr != nil {
			return nil, permErr
		}
	}

	applyEventChanges(evt, changes)
	evt.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, evt); updateErr != nil {
		return nil, updateErr
	}

	s.emitEvent(ctx, "calendar.event.updated", actorID, eventID, evt.CalendarID, nil)

	slog.Info("event updated", "event_id", eventID)

	return evt, nil
}

// UpdateRecurringEvent handles the three-way edit for recurring events
func (s *Service) UpdateRecurringEvent(ctx context.Context, eventID, actorID, tenantID uuid.UUID, scope string, originalDate time.Time, changes UpdateInput) (*models.CalendarEvent, error) {
	if !models.ValidRecurringEditScopes[scope] {
		return nil, ErrInvalidRecurringEditScope
	}

	evt, err := s.repo.GetByID(ctx, eventID, tenantID)
	if err != nil {
		return nil, err
	}

	if evt.RRule == nil || *evt.RRule == "" {
		return nil, ErrEventNotRecurring
	}

	// Check permission
	if evt.CreatedBy != actorID {
		if permErr := s.requireCalendarEditPermission(ctx, evt.CalendarID, actorID, evt.TenantID); permErr != nil {
			return nil, permErr
		}
	}

	switch scope {
	case models.ScopeThisEvent:
		return s.editThisEvent(ctx, evt, originalDate, changes)
	case models.ScopeThisAndFuture:
		return s.editThisAndFuture(ctx, evt, actorID, originalDate, changes)
	case models.ScopeAllEvents:
		return s.editAllEvents(ctx, evt, actorID, changes)
	default:
		return nil, ErrInvalidRecurringEditScope
	}
}

// editThisEvent creates an exception for a single occurrence
func (s *Service) editThisEvent(ctx context.Context, evt *models.CalendarEvent, originalDate time.Time, changes UpdateInput) (*models.CalendarEvent, error) {
	now := time.Now()
	exception := &models.EventException{
		ID:           uuid.New(),
		EventID:      evt.ID,
		OriginalDate: originalDate,
		IsCancelled:  false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Apply changes to exception fields
	if changes.Title != nil {
		exception.Title = changes.Title
	}
	if changes.Description != nil {
		exception.Description = changes.Description
	}
	if changes.Location != nil {
		exception.Location = changes.Location
	}
	if changes.StartTime != nil {
		exception.StartTime = changes.StartTime
	}
	if changes.EndTime != nil {
		exception.EndTime = changes.EndTime
	}
	if changes.ResourceID != nil {
		exception.ResourceID = changes.ResourceID
	}

	if createErr := s.repo.CreateException(ctx, exception); createErr != nil {
		return nil, createErr
	}

	slog.Info("recurring event exception created",
		"event_id", evt.ID,
		"original_date", originalDate,
	)

	return evt, nil
}

// editThisAndFuture splits a recurring series at the given date
func (s *Service) editThisAndFuture(ctx context.Context, evt *models.CalendarEvent, actorID uuid.UUID, originalDate time.Time, changes UpdateInput) (*models.CalendarEvent, error) { //nolint:unparam
	// Set UNTIL on original RRULE to the day before originalDate
	dayBefore := originalDate.Add(-24 * time.Hour)
	newRRule, err := SetUntil(*evt.RRule, dayBefore)
	if err != nil {
		return nil, err
	}
	evt.RRule = &newRRule
	evt.RecurrenceEnd = &dayBefore
	evt.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, evt); updateErr != nil {
		return nil, updateErr
	}

	// Delete exceptions on/after originalDate from original event
	if deleteErr := s.repo.DeleteExceptionsAfterDate(ctx, evt.ID, originalDate); deleteErr != nil {
		slog.Warn("failed to delete exceptions after split date", "error", deleteErr)
	}

	// Create new event with changed properties starting from originalDate
	now := time.Now()
	newEventID := uuid.New()

	// Build new RRULE without UNTIL (clean split)
	cleanRRule, _ := RemoveUntil(*evt.RRule)
	// Actually we want the original RRULE without the UNTIL we just set
	originalRRuleParts := strings.Split(newRRule, ";")
	var cleanParts []string
	for _, part := range originalRRuleParts {
		if strings.HasPrefix(strings.ToUpper(part), "UNTIL=") {
			continue
		}
		cleanParts = append(cleanParts, part)
	}
	cleanRRule = strings.Join(cleanParts, ";")

	newEvt := &models.CalendarEvent{
		ID:              newEventID,
		TenantID:        evt.TenantID,
		CalendarID:      evt.CalendarID,
		Title:           evt.Title,
		Description:     evt.Description,
		Location:        evt.Location,
		ResourceID:      evt.ResourceID,
		StartTime:       originalDate,
		EndTime:         originalDate.Add(evt.EndTime.Sub(evt.StartTime)),
		IsAllDay:        evt.IsAllDay,
		Timezone:        evt.Timezone,
		RRule:           &cleanRRule,
		HasVideoCall:    evt.HasVideoCall,
		LiveKitRoomName: evt.LiveKitRoomName,
		CategoryID:      evt.CategoryID,
		CreatedBy:       actorID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Apply changes to the new event
	applyEventChanges(newEvt, changes)

	if createErr := s.repo.Create(ctx, newEvt); createErr != nil {
		return nil, createErr
	}

	s.emitEvent(ctx, "calendar.event.updated", actorID, newEventID, evt.CalendarID, nil)

	slog.Info("recurring event split",
		"original_event_id", evt.ID,
		"new_event_id", newEventID,
		"split_date", originalDate,
	)

	return newEvt, nil
}

// editAllEvents updates the master event directly
func (s *Service) editAllEvents(ctx context.Context, evt *models.CalendarEvent, actorID uuid.UUID, changes UpdateInput) (*models.CalendarEvent, error) {
	applyEventChanges(evt, changes)
	evt.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, evt); updateErr != nil {
		return nil, updateErr
	}

	s.emitEvent(ctx, "calendar.event.updated", actorID, evt.ID, evt.CalendarID, nil)

	slog.Info("recurring event updated (all)", "event_id", evt.ID)

	return evt, nil
}

// DeleteEvent deletes an event. Only creator can delete.
func (s *Service) DeleteEvent(ctx context.Context, eventID, actorID, tenantID uuid.UUID) error {
	evt, err := s.repo.GetByID(ctx, eventID, tenantID)
	if err != nil {
		return err
	}

	if evt.CreatedBy != actorID {
		return ErrNotEventCreator
	}

	if deleteErr := s.repo.Delete(ctx, eventID, tenantID); deleteErr != nil {
		return deleteErr
	}

	s.emitEvent(ctx, "calendar.event.cancelled", actorID, eventID, evt.CalendarID, nil)

	slog.Info("event deleted", "event_id", eventID)

	return nil
}

// DeleteRecurringInstance cancels a single occurrence of a recurring event
func (s *Service) DeleteRecurringInstance(ctx context.Context, eventID, actorID, tenantID uuid.UUID, originalDate time.Time) error {
	evt, err := s.repo.GetByID(ctx, eventID, tenantID)
	if err != nil {
		return err
	}

	if evt.RRule == nil || *evt.RRule == "" {
		return ErrEventNotRecurring
	}

	if evt.CreatedBy != actorID {
		return ErrNotEventCreator
	}

	now := time.Now()
	exception := &models.EventException{
		ID:           uuid.New(),
		EventID:      eventID,
		OriginalDate: originalDate,
		IsCancelled:  true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if createErr := s.repo.CreateException(ctx, exception); createErr != nil {
		return createErr
	}

	slog.Info("recurring event instance cancelled",
		"event_id", eventID,
		"original_date", originalDate,
	)

	return nil
}

// RSVPToEvent updates a user's RSVP status for an event
func (s *Service) RSVPToEvent(ctx context.Context, eventID, userID, tenantID uuid.UUID, status string) error {
	if !models.ValidRSVPStatuses[status] {
		return ErrInvalidRSVPStatus
	}

	evt, err := s.repo.GetByID(ctx, eventID, tenantID)
	if err != nil {
		return err
	}

	if rsvpErr := s.repo.UpdateRSVP(ctx, eventID, userID, status); rsvpErr != nil {
		return rsvpErr
	}

	s.emitEvent(ctx, "calendar.event.rsvp_changed", userID, eventID, evt.CalendarID,
		[]string{evt.CreatedBy.String()})

	slog.Info("RSVP updated",
		"event_id", eventID,
		"user_id", userID,
		"status", status,
	)

	return nil
}

// ListAttendees lists all attendees of an event
func (s *Service) ListAttendees(ctx context.Context, eventID, tenantID uuid.UUID) ([]models.EventAttendee, error) {
	if _, err := s.repo.GetByID(ctx, eventID, tenantID); err != nil {
		return nil, err
	}
	return s.repo.ListAttendees(ctx, eventID)
}

// SetReminders sets reminders for an event (replaces all existing)
func (s *Service) SetReminders(ctx context.Context, eventID, actorID, tenantID uuid.UUID, minutesBefore []int) error {
	if len(minutesBefore) > 3 {
		return ErrReminderLimitExceeded
	}

	for _, m := range minutesBefore {
		if m <= 0 {
			return ErrInvalidReminderMinutes
		}
	}

	evt, err := s.repo.GetByID(ctx, eventID, tenantID)
	if err != nil {
		return err
	}

	// Check permission
	if evt.CreatedBy != actorID {
		if permErr := s.requireCalendarEditPermission(ctx, evt.CalendarID, actorID, evt.TenantID); permErr != nil {
			return permErr
		}
	}

	if setErr := s.repo.SetReminders(ctx, eventID, minutesBefore); setErr != nil {
		return setErr
	}

	slog.Info("reminders set",
		"event_id", eventID,
		"minutes", minutesBefore,
	)

	return nil
}

// ListReminders lists reminders for an event
func (s *Service) ListReminders(ctx context.Context, eventID uuid.UUID) ([]models.EventReminder, error) {
	return s.repo.ListReminders(ctx, eventID)
}

// ListExceptions lists the per-occurrence exceptions (cancellations and
// overrides) of a recurring event inside a window. Exposed on the service so
// other domains that expand this event's series -- the meeting service, which
// reads the series a meeting is linked to -- can honour the same exceptions
// ListEventsInRange applies, instead of reaching for the repository.
func (s *Service) ListExceptions(ctx context.Context, eventID uuid.UUID, start, end time.Time) ([]models.EventException, error) {
	return s.repo.ListExceptions(ctx, eventID, start, end)
}

// ListEventsInRange returns all events (expanded recurring + non-recurring) in a time window
func (s *Service) ListEventsInRange(ctx context.Context, calendarIDs []uuid.UUID, start, end time.Time, userID, tenantID uuid.UUID) ([]models.ExpandedEvent, error) {
	// Get non-recurring events from repo
	nonRecurring, err := s.repo.ListInRange(ctx, calendarIDs, start, end, userID, tenantID)
	if err != nil {
		return nil, err
	}

	// Get recurring events and expand
	recurring, err := s.repo.ListRecurringOverlapping(ctx, calendarIDs, start, end, tenantID)
	if err != nil {
		return nil, err
	}

	for _, evt := range recurring {
		if evt.RRule == nil || *evt.RRule == "" {
			continue
		}

		// Expand RRULE into concrete dates
		dates, expandErr := ExpandRecurrence(*evt.RRule, evt.StartTime, start, end)
		if expandErr != nil {
			slog.Warn("failed to expand RRULE",
				"event_id", evt.ID,
				"rrule", *evt.RRule,
				"error", expandErr,
			)
			continue
		}

		// Fetch exceptions for this event in the range
		exceptions, exErr := s.repo.ListExceptions(ctx, evt.ID, start, end)
		if exErr != nil {
			slog.Warn("failed to fetch exceptions", "event_id", evt.ID, "error", exErr)
		}

		exceptionMap := make(map[string]*models.EventException)
		for i := range exceptions {
			key := exceptions[i].OriginalDate.Format("2006-01-02")
			exceptionMap[key] = &exceptions[i]
		}

		duration := evt.EndTime.Sub(evt.StartTime)

		for _, date := range dates {
			dateKey := date.Format("2006-01-02")

			// Check for exception
			if ex, hasException := exceptionMap[dateKey]; hasException {
				if ex.IsCancelled {
					continue // Skip cancelled occurrence
				}
				// Use modified fields from exception
				expanded := models.ExpandedEvent{
					CalendarEvent: evt,
				}
				expanded.OriginalEventID = &evt.ID
				expanded.OriginalDate = &date
				expanded.StartTime = date
				expanded.EndTime = date.Add(duration)

				if ex.Title != nil {
					expanded.Title = *ex.Title
				}
				if ex.Description != nil {
					expanded.Description = ex.Description
				}
				if ex.Location != nil {
					expanded.Location = ex.Location
				}
				if ex.StartTime != nil {
					expanded.StartTime = *ex.StartTime
				}
				if ex.EndTime != nil {
					expanded.EndTime = *ex.EndTime
				}
				if ex.ResourceID != nil {
					expanded.ResourceID = ex.ResourceID
				}

				nonRecurring = append(nonRecurring, expanded)
				continue
			}

			// Normal occurrence
			expanded := models.ExpandedEvent{
				CalendarEvent: evt,
			}
			expanded.OriginalEventID = &evt.ID
			expanded.OriginalDate = &date
			expanded.StartTime = date
			expanded.EndTime = date.Add(duration)

			nonRecurring = append(nonRecurring, expanded)
		}
	}

	// Sort by start time
	sort.Slice(nonRecurring, func(i, j int) bool {
		return nonRecurring[i].StartTime.Before(nonRecurring[j].StartTime)
	})

	return nonRecurring, nil
}

// ListTaskDeadlinesInRange returns task deadlines for a user in a time window
func (s *Service) ListTaskDeadlinesInRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]models.TaskDeadlineStub, error) {
	return s.repo.ListTaskDeadlinesInRange(ctx, userID, start, end)
}

// applyEventChanges applies UpdateInput fields to an event
func applyEventChanges(evt *models.CalendarEvent, changes UpdateInput) {
	if changes.Title != nil {
		evt.Title = strings.TrimSpace(*changes.Title)
	}
	if changes.Description != nil {
		evt.Description = changes.Description
	}
	if changes.Location != nil {
		evt.Location = changes.Location
	}
	if changes.ResourceID != nil {
		evt.ResourceID = changes.ResourceID
	}
	if changes.StartTime != nil {
		evt.StartTime = *changes.StartTime
	}
	if changes.EndTime != nil {
		evt.EndTime = *changes.EndTime
	}
	if changes.IsAllDay != nil {
		evt.IsAllDay = *changes.IsAllDay
	}
	if changes.CategoryID != nil {
		evt.CategoryID = changes.CategoryID
	}
	if changes.HasVideoCall != nil {
		evt.HasVideoCall = *changes.HasVideoCall
		if evt.HasVideoCall && evt.LiveKitRoomName == nil {
			roomName := fmt.Sprintf("cal-%s", evt.ID.String()[:8])
			evt.LiveKitRoomName = &roomName
		}
	}
}

// emitEvent sends a notification event if emitter is set
func (s *Service) emitEvent(ctx context.Context, eventType string, actorID, eventID, calendarID uuid.UUID, targetUserIDs []string) {
	if s.emitter == nil {
		return
	}

	payload := models.EventPayload{
		Type:          eventType,
		Priority:      "normal",
		ActorID:       actorID.String(),
		ResourceID:    eventID.String(),
		ModuleID:      "calendar",
		TargetUserIDs: targetUserIDs,
		Payload:       json.RawMessage(fmt.Sprintf(`{"calendar_id":"%s","event_id":"%s"}`, calendarID, eventID)),
		Timestamp:     time.Now(),
	}

	if emitErr := s.emitter.EmitCalendarEvent(ctx, payload); emitErr != nil {
		slog.Warn("failed to emit calendar event",
			"type", eventType,
			"event_id", eventID,
			"error", emitErr,
		)
	}
}

// requireCalendarEditPermission checks if the actor has edit or admin on the calendar
func (s *Service) requireCalendarEditPermission(ctx context.Context, calendarID, actorID, tenantID uuid.UUID) error {
	cal, err := s.calendarRepo.GetByID(ctx, calendarID, tenantID)
	if err != nil {
		return ErrInsufficientPermission
	}
	if cal.OwnerID == actorID {
		return nil
	}

	member, memberErr := s.calendarRepo.GetMember(ctx, calendarID, actorID)
	if memberErr != nil {
		return ErrInsufficientPermission
	}

	if member.Permission == models.CalendarPermissionView {
		return ErrInsufficientPermission
	}

	return nil
}
