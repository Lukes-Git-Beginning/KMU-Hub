package meeting

import (
	"time"

	"github.com/google/uuid"
)

// Meeting status constants
const (
	MeetingStatusScheduled  = "scheduled"
	MeetingStatusInProgress = "in_progress"
	MeetingStatusCompleted  = "completed"
	MeetingStatusCancelled  = "cancelled"
)

// ValidMeetingStatuses contains all valid meeting status values
var ValidMeetingStatuses = map[string]bool{
	MeetingStatusScheduled:  true,
	MeetingStatusInProgress: true,
	MeetingStatusCompleted:  true,
	MeetingStatusCancelled:  true,
}

// RSVP status constants for meeting attendees
const (
	MeetingRSVPPending   = "pending"
	MeetingRSVPAccepted  = "accepted"
	MeetingRSVPDeclined  = "declined"
	MeetingRSVPTentative = "tentative"
)

// ValidMeetingRSVPStatuses contains all valid meeting RSVP status values
var ValidMeetingRSVPStatuses = map[string]bool{
	MeetingRSVPPending:   true,
	MeetingRSVPAccepted:  true,
	MeetingRSVPDeclined:  true,
	MeetingRSVPTentative: true,
}

// Meeting represents a scheduled or in-progress meeting
type Meeting struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	Title              string     `json:"title"`
	Description        *string    `json:"description,omitempty"`
	Agenda             *string    `json:"agenda,omitempty"`
	OrganizerID        uuid.UUID  `json:"organizer_id"`
	Status             string     `json:"status"`
	Locked             bool       `json:"locked"`
	ScheduledStart     time.Time  `json:"scheduled_start"`
	ScheduledEnd       time.Time  `json:"scheduled_end"`
	ActualStart        *time.Time `json:"actual_start,omitempty"`
	ActualEnd          *time.Time `json:"actual_end,omitempty"`
	RoomName           *string    `json:"room_name,omitempty"`
	CalendarEventID    *uuid.UUID `json:"calendar_event_id,omitempty"`
	RecurringMeetingID *uuid.UUID `json:"recurring_meeting_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// MeetingCoHost represents a user who has been granted co-host privileges
// for a specific meeting by the meeting organizer.
type MeetingCoHost struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	MeetingID uuid.UUID `json:"meeting_id"`
	UserID    uuid.UUID `json:"user_id"`
	GrantedBy uuid.UUID `json:"granted_by"`
	CreatedAt time.Time `json:"created_at"`
}

// MeetingWithAttendees combines a meeting with its attendee list
type MeetingWithAttendees struct {
	Meeting
	Attendees []MeetingAttendeeWithUser `json:"attendees"`
}

// MeetingAttendee represents a user invited to a meeting
type MeetingAttendee struct {
	MeetingID  uuid.UUID `json:"meeting_id"`
	UserID     uuid.UUID `json:"user_id"`
	RSVPStatus string    `json:"rsvp_status"`
}

// MeetingAttendeeWithUser includes denormalized user info
type MeetingAttendeeWithUser struct {
	MeetingAttendee
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// MeetingNotes represents notes taken during a meeting
type MeetingNotes struct {
	ID        uuid.UUID `json:"id"`
	MeetingID uuid.UUID `json:"meeting_id"`
	AuthorID  uuid.UUID `json:"author_id"`
	Content   string    `json:"content"`
	IsPrivate bool      `json:"is_private"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MeetingActionItem represents a follow-up action from a meeting
type MeetingActionItem struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	MeetingID   uuid.UUID  `json:"meeting_id"`
	Description string     `json:"description"`
	AssigneeID  *uuid.UUID `json:"assignee_id,omitempty"`
	IsCompleted bool       `json:"is_completed"`
	TaskID      *uuid.UUID `json:"task_id,omitempty"`
	SortOrder   int        `json:"sort_order"`
	CreatedAt   time.Time  `json:"created_at"`
}

// MeetingActionItemWithAssignee includes denormalized assignee info
type MeetingActionItemWithAssignee struct {
	MeetingActionItem
	AssigneeFirstName *string `json:"assignee_first_name,omitempty"`
	AssigneeLastName  *string `json:"assignee_last_name,omitempty"`
}

// MeetingSummary holds a post-meeting summary including duration, attendees, and actions
type MeetingSummary struct {
	MeetingID       uuid.UUID                      `json:"meeting_id"`
	DurationMinutes int                            `json:"duration_minutes"`
	AttendeeCount   int                            `json:"attendee_count"`
	NotesSummary    *string                        `json:"notes_summary,omitempty"`
	ActionItems     []MeetingActionItemWithAssignee `json:"action_items"`
}

// MeetingChatMessage represents a single persisted in-call chat message.
type MeetingChatMessage struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	MeetingID  uuid.UUID `json:"meeting_id"`
	SenderID   uuid.UUID `json:"sender_id"`
	SenderName string    `json:"sender_name"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}
