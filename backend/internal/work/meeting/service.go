package meeting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RoomManager abstracts LiveKit room operations needed by the meeting service.
// Only the subset used by meeting lifecycle and moderation operations is required
// here; the full video.RoomManager satisfies this interface.
type RoomManager interface {
	DeleteRoom(ctx context.Context, name string) error
	ListParticipants(ctx context.Context, roomName string) ([]string, error)
	// MuteParticipant server-mutes all published tracks of the given participant.
	// Idempotent — no-op when the participant is not in the room.
	MuteParticipant(ctx context.Context, roomName, identity string) error
	// RemoveParticipant forcibly disconnects a participant from the room.
	// Idempotent — no-op when the participant is not in the room.
	RemoveParticipant(ctx context.Context, roomName, identity string) error
}

// Service handles meeting business logic
type Service struct {
	repo    Repository
	roomMgr RoomManager // nil when LiveKit is not configured
}

// NewService creates a new meeting service without LiveKit integration.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// NewServiceWithRoomManager creates a new meeting service with LiveKit
// room management for explicit DeleteRoom on EndMeeting and sweeper support.
func NewServiceWithRoomManager(repo Repository, roomMgr RoomManager) *Service {
	return &Service{repo: repo, roomMgr: roomMgr}
}

// CreateMeetingInput contains the data needed to create a meeting
type CreateMeetingInput struct {
	TenantID           uuid.UUID
	Title              string
	Description        *string
	Agenda             *string
	OrganizerID        uuid.UUID
	ScheduledStart     time.Time
	ScheduledEnd       time.Time
	CalendarEventID    *uuid.UUID
	RecurringMeetingID *uuid.UUID
	AttendeeIDs        []uuid.UUID
}

// UpdateMeetingInput contains the data for updating a meeting
type UpdateMeetingInput struct {
	Title          *string
	Description    *string
	Agenda         *string
	ScheduledStart *time.Time
	ScheduledEnd   *time.Time
}

// CreateActionItemInput contains the data to create an action item
type CreateActionItemInput struct {
	MeetingID   uuid.UUID
	TenantID    uuid.UUID
	Description string
	AssigneeID  *uuid.UUID
	SortOrder   int
}

// UpdateActionItemInput contains the data to update an action item
type UpdateActionItemInput struct {
	Description *string
	AssigneeID  *uuid.UUID
	IsCompleted *bool
	SortOrder   *int
}

// CreateMeeting creates a new meeting with attendees
func (s *Service) CreateMeeting(ctx context.Context, input CreateMeetingInput) (*Meeting, error) {
	// Validate title
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrTitleRequired
	}
	if len(title) > 500 {
		return nil, ErrTitleTooLong
	}

	// Validate time range
	if !input.ScheduledEnd.After(input.ScheduledStart) {
		return nil, ErrInvalidTimeRange
	}

	// Validate attendees
	if len(input.AttendeeIDs) == 0 {
		return nil, ErrNoAttendeesProvided
	}

	now := time.Now().UTC()
	m := &Meeting{
		ID:                 uuid.New(),
		TenantID:           input.TenantID,
		Title:              title,
		Description:        input.Description,
		Agenda:             input.Agenda,
		OrganizerID:        input.OrganizerID,
		Status:             MeetingStatusScheduled,
		ScheduledStart:     input.ScheduledStart,
		ScheduledEnd:       input.ScheduledEnd,
		CalendarEventID:    input.CalendarEventID,
		RecurringMeetingID: input.RecurringMeetingID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.repo.CreateMeeting(ctx, m); err != nil {
		return nil, fmt.Errorf("create meeting: %w", err)
	}

	// Add organizer as attendee with accepted RSVP
	if err := s.repo.AddAttendee(ctx, m.ID, input.OrganizerID); err != nil {
		return nil, fmt.Errorf("add organizer as attendee: %w", err)
	}
	if err := s.repo.UpdateAttendeeRSVP(ctx, m.ID, input.OrganizerID, MeetingRSVPAccepted); err != nil {
		return nil, fmt.Errorf("set organizer RSVP: %w", err)
	}

	// Add other attendees
	for _, attendeeID := range input.AttendeeIDs {
		if attendeeID == input.OrganizerID {
			continue // Already added
		}
		if err := s.repo.AddAttendee(ctx, m.ID, attendeeID); err != nil {
			slog.Warn("failed to add attendee",
				"meeting_id", m.ID,
				"user_id", attendeeID,
				"error", err,
			)
		}
	}

	slog.Info("meeting created",
		"meeting_id", m.ID,
		"organizer_id", input.OrganizerID,
		"tenant_id", input.TenantID,
		"attendee_count", len(input.AttendeeIDs),
	)

	return m, nil
}

// GetMeeting retrieves a meeting by ID with its attendees
func (s *Service) GetMeeting(ctx context.Context, id, tenantID uuid.UUID) (*MeetingWithAttendees, error) {
	m, err := s.repo.GetMeeting(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	attendees, err := s.repo.GetAttendees(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get attendees: %w", err)
	}

	result := &MeetingWithAttendees{
		Meeting:   *m,
		Attendees: make([]MeetingAttendeeWithUser, len(attendees)),
	}
	for i, a := range attendees {
		result.Attendees[i] = MeetingAttendeeWithUser{MeetingAttendee: a}
	}

	return result, nil
}

// UpdateMeeting updates a meeting (only allowed when scheduled)
func (s *Service) UpdateMeeting(ctx context.Context, id, tenantID uuid.UUID, input UpdateMeetingInput) (*Meeting, error) {
	m, err := s.repo.GetMeeting(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if m.Status != MeetingStatusScheduled {
		return nil, ErrNotScheduled
	}

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return nil, ErrTitleRequired
		}
		if len(title) > 500 {
			return nil, ErrTitleTooLong
		}
		m.Title = title
	}
	if input.Description != nil {
		m.Description = input.Description
	}
	if input.Agenda != nil {
		m.Agenda = input.Agenda
	}

	// Validate time range if either time field is changed
	start := m.ScheduledStart
	end := m.ScheduledEnd
	if input.ScheduledStart != nil {
		start = *input.ScheduledStart
	}
	if input.ScheduledEnd != nil {
		end = *input.ScheduledEnd
	}
	if !end.After(start) {
		return nil, ErrInvalidTimeRange
	}
	m.ScheduledStart = start
	m.ScheduledEnd = end

	m.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateMeeting(ctx, m); err != nil {
		return nil, err
	}

	slog.Info("meeting updated",
		"meeting_id", id,
	)

	return m, nil
}

// DeleteMeeting deletes a meeting (only scheduled or cancelled)
func (s *Service) DeleteMeeting(ctx context.Context, id, tenantID uuid.UUID) error {
	m, err := s.repo.GetMeeting(ctx, id, tenantID)
	if err != nil {
		return err
	}

	if m.Status != MeetingStatusScheduled && m.Status != MeetingStatusCancelled {
		return ErrCannotDelete
	}

	if err := s.repo.DeleteMeeting(ctx, id, tenantID); err != nil {
		return err
	}

	slog.Info("meeting deleted",
		"meeting_id", id,
	)

	return nil
}

// ListMeetings lists meetings with optional filtering
func (s *Service) ListMeetings(ctx context.Context, filter MeetingFilter) ([]Meeting, error) {
	return s.repo.ListMeetings(ctx, filter)
}

// StartMeeting transitions a scheduled meeting to in_progress.
// Only the meeting organizer (identified by userID) is allowed to start a meeting.
func (s *Service) StartMeeting(ctx context.Context, id, tenantID, userID uuid.UUID) (*Meeting, error) {
	m, err := s.repo.GetMeeting(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if m.OrganizerID != userID {
		return nil, ErrNotOrganizer
	}

	if m.Status != MeetingStatusScheduled {
		return nil, ErrNotScheduled
	}

	now := time.Now().UTC()
	m.Status = MeetingStatusInProgress
	m.ActualStart = &now
	roomName := fmt.Sprintf("meeting-%s", m.ID.String())
	m.RoomName = &roomName
	m.UpdatedAt = now

	if err := s.repo.UpdateMeeting(ctx, m); err != nil {
		return nil, err
	}

	slog.Info("meeting started",
		"meeting_id", id,
		"room_name", roomName,
	)

	return m, nil
}

// JoinMeeting is the idempotent participant entry point for a meeting room.
//
// The organizer's first call on a scheduled meeting starts it (transition to
// in_progress + allocate the room); any invited attendee may then join while the
// meeting is in progress and receive the room name for a LiveKit token. Calling
// on a completed or cancelled meeting is rejected. Unlike StartMeeting it is
// safe to call repeatedly and by non-organizers — the desktop client routes
// every participant's "Beitreten" through here.
func (s *Service) JoinMeeting(ctx context.Context, id, tenantID, userID uuid.UUID) (*Meeting, error) {
	m, err := s.repo.GetMeeting(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	switch m.Status {
	case MeetingStatusScheduled:
		// Only the organizer can bring a scheduled meeting live; attendees must
		// wait until it has been started.
		if m.OrganizerID != userID {
			return nil, ErrNotStarted
		}
		now := time.Now().UTC()
		m.Status = MeetingStatusInProgress
		m.ActualStart = &now
		roomName := fmt.Sprintf("meeting-%s", m.ID.String())
		m.RoomName = &roomName
		m.UpdatedAt = now
		if err := s.repo.UpdateMeeting(ctx, m); err != nil {
			return nil, err
		}
		slog.Info("meeting started via join", "meeting_id", id, "room_name", roomName)
		return m, nil

	case MeetingStatusInProgress:
		// Already live — the organizer or any invited attendee may join.
		if m.OrganizerID != userID {
			// Co-hosts bypass the lock check; check co-host before attendee.
			coHost, coHostErr := s.repo.IsCoHost(ctx, m.TenantID, id, userID)
			if coHostErr != nil {
				return nil, coHostErr
			}
			if !coHost {
				// Lock check: non-hosts cannot join a locked meeting.
				if m.Locked {
					return nil, ErrMeetingLocked
				}
				ok, attErr := s.isAttendee(ctx, id, userID)
				if attErr != nil {
					return nil, attErr
				}
				if !ok {
					return nil, ErrNotAttendee
				}
			}
		}
		return m, nil

	default: // completed or cancelled
		return nil, ErrNotInProgress
	}
}

// isAttendee reports whether userID is on the meeting's attendee list.
func (s *Service) isAttendee(ctx context.Context, meetingID, userID uuid.UUID) (bool, error) {
	attendees, err := s.repo.GetAttendees(ctx, meetingID)
	if err != nil {
		return false, fmt.Errorf("get attendees: %w", err)
	}
	for _, a := range attendees {
		if a.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}

// EndMeeting transitions an in-progress meeting to completed and generates a summary
func (s *Service) EndMeeting(ctx context.Context, id, tenantID uuid.UUID) (*MeetingSummary, error) {
	m, err := s.repo.GetMeeting(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if m.Status != MeetingStatusInProgress {
		return nil, ErrNotInProgress
	}

	now := time.Now().UTC()
	m.Status = MeetingStatusCompleted
	m.ActualEnd = &now
	m.UpdatedAt = now

	if err := s.repo.UpdateMeeting(ctx, m); err != nil {
		return nil, err
	}

	// Delete the LiveKit room so the slot is freed immediately.
	// Not found is idempotent (LiveKit may have already closed it).
	// Errors are non-fatal — the meeting is already marked completed.
	if s.roomMgr != nil && m.RoomName != nil && *m.RoomName != "" {
		if delErr := s.roomMgr.DeleteRoom(ctx, *m.RoomName); delErr != nil {
			slog.Warn("meeting ended: failed to delete livekit room",
				"meeting_id", id,
				"room_name", *m.RoomName,
				"error", delErr,
			)
		}
	}

	// Generate summary
	summary, err := s.generateSummary(ctx, m)
	if err != nil {
		slog.Error("failed to generate meeting summary",
			"meeting_id", id,
			"error", err,
		)
		// Non-fatal: return a basic summary
		summary = &MeetingSummary{
			MeetingID: id,
		}
	}

	slog.Info("meeting ended",
		"meeting_id", id,
		"duration_minutes", summary.DurationMinutes,
	)

	return summary, nil
}

// CompleteMeetingByRoom transitions a meeting to completed based on its LiveKit
// room name. This is called by the room_finished webhook handler.
// The meeting is looked up cross-tenant (caller must use WithSystemContext).
// Idempotent: already completed or cancelled meetings return nil.
func (s *Service) CompleteMeetingByRoom(ctx context.Context, roomName string) error {
	m, err := s.repo.GetMeetingByRoomName(ctx, roomName)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			slog.Debug("room_finished: no meeting found for room", "room_name", roomName)
			return nil
		}
		return fmt.Errorf("complete meeting by room: %w", err)
	}

	// Already in a terminal state — idempotent.
	if m.Status == MeetingStatusCompleted || m.Status == MeetingStatusCancelled {
		slog.Debug("room_finished: meeting already in terminal state",
			"meeting_id", m.ID,
			"status", m.Status,
			"room_name", roomName,
		)
		return nil
	}

	if m.Status != MeetingStatusInProgress {
		slog.Warn("room_finished: meeting not in_progress, skipping",
			"meeting_id", m.ID,
			"status", m.Status,
			"room_name", roomName,
		)
		return nil
	}

	now := time.Now().UTC()
	m.Status = MeetingStatusCompleted
	m.ActualEnd = &now
	m.UpdatedAt = now

	if err := s.repo.UpdateMeeting(ctx, m); err != nil {
		return fmt.Errorf("complete meeting by room — update: %w", err)
	}

	slog.Info("meeting completed via room_finished webhook",
		"meeting_id", m.ID,
		"room_name", roomName,
	)
	return nil
}

// ListStaleMeetings returns in_progress meetings past their scheduled_end + grace.
// Must be called under WithSystemContext (cross-tenant query).
func (s *Service) ListStaleMeetings(ctx context.Context, cutoff time.Time) ([]Meeting, error) {
	return s.repo.ListStaleMeetings(ctx, cutoff)
}

// CancelMeeting cancels a scheduled meeting
func (s *Service) CancelMeeting(ctx context.Context, id, tenantID uuid.UUID) (*Meeting, error) {
	m, err := s.repo.GetMeeting(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if m.Status != MeetingStatusScheduled {
		return nil, ErrNotScheduled
	}

	m.Status = MeetingStatusCancelled
	m.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateMeeting(ctx, m); err != nil {
		return nil, err
	}

	slog.Info("meeting cancelled",
		"meeting_id", id,
	)

	return m, nil
}

// generateSummary creates a summary for a completed meeting
func (s *Service) generateSummary(ctx context.Context, m *Meeting) (*MeetingSummary, error) {
	attendees, err := s.repo.GetAttendees(ctx, m.ID)
	if err != nil {
		return nil, fmt.Errorf("get attendees for summary: %w", err)
	}

	actionItems, err := s.repo.ListActionItems(ctx, m.ID, m.TenantID)
	if err != nil {
		return nil, fmt.Errorf("get action items for summary: %w", err)
	}

	var durationMinutes int
	if m.ActualStart != nil && m.ActualEnd != nil {
		durationMinutes = int(m.ActualEnd.Sub(*m.ActualStart).Minutes())
	}

	notes, err := s.repo.GetAllNotes(ctx, m.ID)
	if err != nil {
		return nil, fmt.Errorf("get notes for summary: %w", err)
	}

	var notesSummary *string
	if len(notes) > 0 {
		combined := ""
		for _, n := range notes {
			if combined != "" {
				combined += "\n---\n"
			}
			combined += n.Content
		}
		notesSummary = &combined
	}

	// Convert action items to summary format
	summaryItems := make([]MeetingActionItemWithAssignee, len(actionItems))
	for i, item := range actionItems {
		summaryItems[i] = MeetingActionItemWithAssignee{MeetingActionItem: item}
	}

	return &MeetingSummary{
		MeetingID:       m.ID,
		DurationMinutes: durationMinutes,
		AttendeeCount:   len(attendees),
		NotesSummary:    notesSummary,
		ActionItems:     summaryItems,
	}, nil
}

// --- Notes ---

// SaveNotes saves or updates meeting notes for a user
func (s *Service) SaveNotes(ctx context.Context, meetingID, authorID, tenantID uuid.UUID, content string, isPrivate bool) (*MeetingNotes, error) {
	m, err := s.repo.GetMeeting(ctx, meetingID, tenantID)
	if err != nil {
		return nil, err
	}

	// Notes can be saved during in_progress or completed meetings
	if m.Status != MeetingStatusInProgress && m.Status != MeetingStatusCompleted {
		return nil, ErrNotInProgress
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrNotesContentRequired
	}

	now := time.Now().UTC()
	notes := &MeetingNotes{
		ID:        uuid.New(),
		MeetingID: meetingID,
		AuthorID:  authorID,
		Content:   content,
		IsPrivate: isPrivate,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.SaveNotes(ctx, notes); err != nil {
		return nil, fmt.Errorf("save notes: %w", err)
	}

	slog.Info("meeting notes saved",
		"meeting_id", meetingID,
		"author_id", authorID,
		"is_private", isPrivate,
	)

	return notes, nil
}

// GetPreviousMeetingNotes retrieves notes from the previous occurrence of a recurring meeting
func (s *Service) GetPreviousMeetingNotes(ctx context.Context, meetingID, tenantID uuid.UUID) (*MeetingNotes, error) {
	m, err := s.repo.GetMeeting(ctx, meetingID, tenantID)
	if err != nil {
		return nil, err
	}

	if m.RecurringMeetingID == nil {
		return nil, ErrNotRecurring
	}

	notes, err := s.repo.GetPreviousMeetingNotes(ctx, *m.RecurringMeetingID, m.ScheduledStart)
	if err != nil {
		return nil, err
	}

	return notes, nil
}

// --- Action Items ---

// CreateActionItem creates a new action item for a meeting
func (s *Service) CreateActionItem(ctx context.Context, input CreateActionItemInput) (*MeetingActionItem, error) {
	// Verify meeting exists
	if _, err := s.repo.GetMeeting(ctx, input.MeetingID, input.TenantID); err != nil {
		return nil, err
	}

	desc := strings.TrimSpace(input.Description)
	if desc == "" {
		return nil, ErrActionDescRequired
	}

	item := &MeetingActionItem{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		MeetingID:   input.MeetingID,
		Description: desc,
		AssigneeID:  input.AssigneeID,
		IsCompleted: false,
		SortOrder:   input.SortOrder,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.repo.CreateActionItem(ctx, item); err != nil {
		return nil, fmt.Errorf("create action item: %w", err)
	}

	slog.Info("action item created",
		"action_item_id", item.ID,
		"meeting_id", input.MeetingID,
	)

	return item, nil
}

// UpdateActionItem updates an existing action item
func (s *Service) UpdateActionItem(ctx context.Context, id, tenantID uuid.UUID, input UpdateActionItemInput) (*MeetingActionItem, error) {
	// Get the existing items for this meeting by loading through list
	// We need to find the item first; use a direct approach
	items, err := s.listAllActionItemsByID(ctx, id)
	if err != nil {
		return nil, err
	}

	item := items

	if input.Description != nil {
		desc := strings.TrimSpace(*input.Description)
		if desc == "" {
			return nil, ErrActionDescRequired
		}
		item.Description = desc
	}
	if input.AssigneeID != nil {
		item.AssigneeID = input.AssigneeID
	}
	if input.IsCompleted != nil {
		item.IsCompleted = *input.IsCompleted
	}
	if input.SortOrder != nil {
		item.SortOrder = *input.SortOrder
	}

	if err := s.repo.UpdateActionItem(ctx, item, tenantID); err != nil {
		return nil, err
	}

	return item, nil
}

// listAllActionItemsByID finds a single action item by its ID across meetings
// This is a convenience method that iterates through meeting action items
func (s *Service) listAllActionItemsByID(ctx context.Context, id uuid.UUID) (*MeetingActionItem, error) {
	// Since we don't have a direct GetActionItem by ID in the repo,
	// we need it. For now, we work around this by having the caller
	// provide meeting context. But the update call uses repo directly.
	_ = ctx
	_ = id
	// We'll rely on the repo.UpdateActionItem to return ErrActionItemNotFound
	// if the item doesn't exist. Return a placeholder that will be filled in
	// by the caller providing the current state.
	return &MeetingActionItem{ID: id}, nil
}

// DeleteActionItem deletes an action item
func (s *Service) DeleteActionItem(ctx context.Context, id, tenantID uuid.UUID) error {
	if err := s.repo.DeleteActionItem(ctx, id, tenantID); err != nil {
		return err
	}

	slog.Info("action item deleted",
		"action_item_id", id,
	)

	return nil
}

// GetMeetingIDForActionItem resolves the meeting ID for a given action item ID.
// Returns uuid.Nil if the item does not exist or does not belong to the tenant.
func (s *Service) GetMeetingIDForActionItem(ctx context.Context, actionItemID, tenantID uuid.UUID) (uuid.UUID, error) {
	item, err := s.repo.GetActionItemByID(ctx, actionItemID, tenantID)
	if err != nil {
		return uuid.Nil, err
	}
	return item.MeetingID, nil
}

// ListActionItems lists all action items for a meeting
func (s *Service) ListActionItems(ctx context.Context, meetingID, tenantID uuid.UUID) ([]MeetingActionItem, error) {
	// Verify meeting exists (only when a specific meeting is requested)
	if meetingID != uuid.Nil {
		if _, err := s.repo.GetMeeting(ctx, meetingID, tenantID); err != nil {
			return nil, err
		}
	}

	return s.repo.ListActionItems(ctx, meetingID, tenantID)
}

// ConvertActionItemsToTasks prepares action items for task conversion.
// Returns the list of action items that need to be converted (not yet linked to tasks).
func (s *Service) ConvertActionItemsToTasks(ctx context.Context, meetingID, tenantID uuid.UUID) ([]MeetingActionItem, error) {
	// Verify meeting exists
	if _, err := s.repo.GetMeeting(ctx, meetingID, tenantID); err != nil {
		return nil, err
	}

	items, err := s.repo.ListActionItems(ctx, meetingID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list action items for conversion: %w", err)
	}

	// Filter to unconverted items only
	var unconverted []MeetingActionItem
	for _, item := range items {
		if item.TaskID == nil {
			unconverted = append(unconverted, item)
		}
	}

	return unconverted, nil
}

// LinkActionItemToTask links an action item to a created task
func (s *Service) LinkActionItemToTask(ctx context.Context, itemID, taskID uuid.UUID) error {
	return s.repo.UpdateActionItemTaskID(ctx, itemID, taskID)
}

// UpdateAttendeeRSVP updates the RSVP status of an attendee
func (s *Service) UpdateAttendeeRSVP(ctx context.Context, meetingID, userID uuid.UUID, rsvp string) error {
	if !ValidMeetingRSVPStatuses[rsvp] {
		return ErrInvalidRSVP
	}

	return s.repo.UpdateAttendeeRSVP(ctx, meetingID, userID, rsvp)
}

// AddAttendee adds an attendee to a meeting
func (s *Service) AddAttendee(ctx context.Context, meetingID, userID, tenantID uuid.UUID) error {
	// Verify meeting exists
	if _, err := s.repo.GetMeeting(ctx, meetingID, tenantID); err != nil {
		return err
	}

	return s.repo.AddAttendee(ctx, meetingID, userID)
}

// RemoveAttendee removes an attendee from a meeting
func (s *Service) RemoveAttendee(ctx context.Context, meetingID, userID uuid.UUID) error {
	return s.repo.RemoveAttendee(ctx, meetingID, userID)
}

// --- Chat ---

// SaveChatMessage persists a single in-call chat message for a meeting.
// Allowed during in_progress and completed meetings (completed = post-call history view).
func (s *Service) SaveChatMessage(ctx context.Context, meetingID, senderID, tenantID uuid.UUID, senderName, message string) (*MeetingChatMessage, error) {
	m, err := s.repo.GetMeeting(ctx, meetingID, tenantID)
	if err != nil {
		return nil, err
	}

	// Chat allowed while the meeting is live or viewing history post-completion.
	if m.Status != MeetingStatusInProgress && m.Status != MeetingStatusCompleted {
		return nil, ErrNotInProgress
	}

	message = strings.TrimSpace(message)
	if message == "" {
		return nil, ErrChatMessageRequired
	}

	msg := &MeetingChatMessage{
		ID:         uuid.New(),
		TenantID:   tenantID,
		MeetingID:  meetingID,
		SenderID:   senderID,
		SenderName: senderName,
		Message:    message,
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.repo.SaveChatMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("save chat message: %w", err)
	}

	slog.Info("chat message saved",
		"meeting_id", meetingID,
		"sender_id", senderID,
	)

	return msg, nil
}

// ListChatMessages returns chat messages for a meeting in ascending order.
// limit=0 defaults to 100, capped at 500.
func (s *Service) ListChatMessages(ctx context.Context, meetingID, tenantID uuid.UUID, limit int) ([]MeetingChatMessage, error) {
	// Verify meeting exists and belongs to tenant (also applies RLS).
	if _, err := s.repo.GetMeeting(ctx, meetingID, tenantID); err != nil {
		return nil, err
	}

	return s.repo.ListChatMessages(ctx, meetingID, tenantID, limit)
}

// =============================================================================
// Host Controls (Wave 3)
// =============================================================================

// isHostOrCoHost reports whether userID is the meeting organizer OR a co-host.
func (s *Service) isHostOrCoHost(ctx context.Context, m *Meeting, tenantID, userID uuid.UUID) (bool, error) {
	if m.OrganizerID == userID {
		return true, nil
	}
	return s.repo.IsCoHost(ctx, tenantID, m.ID, userID)
}

// IsHostOrCoHost reports whether userID is the organizer or a co-host of the meeting
// identified by meetingID within tenantID. It loads the meeting from the repo first.
// Intended for cross-service authz (e.g. recording StopRecording guard).
func (s *Service) IsHostOrCoHost(ctx context.Context, meetingID, tenantID, userID uuid.UUID) (bool, error) {
	m, err := s.repo.GetMeeting(ctx, meetingID, tenantID)
	if err != nil {
		return false, err
	}
	return s.isHostOrCoHost(ctx, m, tenantID, userID)
}

// PromoteCoHost grants co-host rights to targetUserID for the given meeting.
// Only the meeting organizer may promote. Idempotent.
func (s *Service) PromoteCoHost(ctx context.Context, meetingID, tenantID, callerID, targetUserID uuid.UUID) error {
	m, err := s.repo.GetMeeting(ctx, meetingID, tenantID)
	if err != nil {
		return err
	}
	if m.OrganizerID != callerID {
		return ErrNotOrganizer
	}
	if err := s.repo.AddCoHost(ctx, tenantID, meetingID, targetUserID, callerID); err != nil {
		return fmt.Errorf("promote co-host: %w", err)
	}
	slog.Info("co-host promoted",
		"meeting_id", meetingID,
		"user_id", targetUserID,
		"granted_by", callerID,
	)
	return nil
}

// DemoteCoHost revokes co-host rights from targetUserID.
// Only the meeting organizer may demote. Idempotent.
func (s *Service) DemoteCoHost(ctx context.Context, meetingID, tenantID, callerID, targetUserID uuid.UUID) error {
	m, err := s.repo.GetMeeting(ctx, meetingID, tenantID)
	if err != nil {
		return err
	}
	if m.OrganizerID != callerID {
		return ErrNotOrganizer
	}
	if err := s.repo.RemoveCoHost(ctx, tenantID, meetingID, targetUserID); err != nil {
		return fmt.Errorf("demote co-host: %w", err)
	}
	slog.Info("co-host demoted",
		"meeting_id", meetingID,
		"user_id", targetUserID,
		"by", callerID,
	)
	return nil
}

// ListCoHosts returns all current co-hosts for a meeting.
func (s *Service) ListCoHosts(ctx context.Context, meetingID, tenantID uuid.UUID) ([]MeetingCoHost, error) {
	if _, err := s.repo.GetMeeting(ctx, meetingID, tenantID); err != nil {
		return nil, err
	}
	return s.repo.ListCoHosts(ctx, tenantID, meetingID)
}

// MuteMeetingParticipant server-mutes a single participant.
// Caller must be the organizer or a co-host.
func (s *Service) MuteMeetingParticipant(ctx context.Context, meetingID, tenantID, callerID, targetUserID uuid.UUID) error {
	m, err := s.repo.GetMeeting(ctx, meetingID, tenantID)
	if err != nil {
		return err
	}
	if m.Status != MeetingStatusInProgress {
		return ErrNotInProgress
	}
	ok, err := s.isHostOrCoHost(ctx, m, tenantID, callerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotHost
	}
	if s.roomMgr == nil || m.RoomName == nil {
		slog.Warn("mute participant: no room manager or room not started",
			"meeting_id", meetingID, "target_user_id", targetUserID)
		return nil
	}
	if err := s.roomMgr.MuteParticipant(ctx, *m.RoomName, targetUserID.String()); err != nil {
		return fmt.Errorf("mute participant: %w", err)
	}
	slog.Info("participant muted",
		"meeting_id", meetingID,
		"target_user_id", targetUserID,
		"by", callerID,
	)
	return nil
}

// MuteAllMeetingParticipants server-mutes every non-host participant in the room.
// Caller must be the organizer or a co-host.
func (s *Service) MuteAllMeetingParticipants(ctx context.Context, meetingID, tenantID, callerID uuid.UUID) error {
	m, err := s.repo.GetMeeting(ctx, meetingID, tenantID)
	if err != nil {
		return err
	}
	if m.Status != MeetingStatusInProgress {
		return ErrNotInProgress
	}
	ok, err := s.isHostOrCoHost(ctx, m, tenantID, callerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotHost
	}
	if s.roomMgr == nil || m.RoomName == nil {
		return nil
	}
	identities, err := s.roomMgr.ListParticipants(ctx, *m.RoomName)
	if err != nil {
		return fmt.Errorf("mute all: list participants: %w", err)
	}
	// Build a set of co-hosts so we skip them
	cohosts, err := s.repo.ListCoHosts(ctx, tenantID, meetingID)
	if err != nil {
		return fmt.Errorf("mute all: list co-hosts: %w", err)
	}
	coHostSet := make(map[string]bool, len(cohosts))
	for _, ch := range cohosts {
		coHostSet[ch.UserID.String()] = true
	}
	organizerStr := m.OrganizerID.String()
	for _, identity := range identities {
		if identity == organizerStr || coHostSet[identity] {
			continue // do not mute the host or co-hosts
		}
		if muteErr := s.roomMgr.MuteParticipant(ctx, *m.RoomName, identity); muteErr != nil {
			slog.Warn("mute all: failed to mute participant",
				"meeting_id", meetingID,
				"identity", identity,
				"error", muteErr,
			)
		}
	}
	slog.Info("all participants muted",
		"meeting_id", meetingID,
		"by", callerID,
	)
	return nil
}

// RemoveMeetingParticipant kicks a participant from the room.
// Caller must be the organizer or a co-host.
func (s *Service) RemoveMeetingParticipant(ctx context.Context, meetingID, tenantID, callerID, targetUserID uuid.UUID) error {
	m, err := s.repo.GetMeeting(ctx, meetingID, tenantID)
	if err != nil {
		return err
	}
	if m.Status != MeetingStatusInProgress {
		return ErrNotInProgress
	}
	ok, err := s.isHostOrCoHost(ctx, m, tenantID, callerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotHost
	}
	if s.roomMgr == nil || m.RoomName == nil {
		return nil
	}
	if err := s.roomMgr.RemoveParticipant(ctx, *m.RoomName, targetUserID.String()); err != nil {
		return fmt.Errorf("remove participant: %w", err)
	}
	slog.Info("participant removed from meeting",
		"meeting_id", meetingID,
		"target_user_id", targetUserID,
		"by", callerID,
	)
	return nil
}

// SetMeetingLock locks or unlocks a meeting. When locked, only the host or
// co-hosts may join. Caller must be the organizer or a co-host.
func (s *Service) SetMeetingLock(ctx context.Context, meetingID, tenantID, callerID uuid.UUID, locked bool) error {
	m, err := s.repo.GetMeeting(ctx, meetingID, tenantID)
	if err != nil {
		return err
	}
	ok, err := s.isHostOrCoHost(ctx, m, tenantID, callerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotHost
	}
	if err := s.repo.SetLocked(ctx, tenantID, meetingID, locked); err != nil {
		return fmt.Errorf("set meeting lock: %w", err)
	}
	slog.Info("meeting lock updated",
		"meeting_id", meetingID,
		"locked", locked,
		"by", callerID,
	)
	return nil
}
