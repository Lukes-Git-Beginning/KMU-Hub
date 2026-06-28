package meeting

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for meeting persistence
type Repository interface {
	// Meeting CRUD
	CreateMeeting(ctx context.Context, m *Meeting) error
	GetMeeting(ctx context.Context, id, tenantID uuid.UUID) (*Meeting, error)
	UpdateMeeting(ctx context.Context, m *Meeting) error
	// UpdateAISummary persists an LLM-generated summary on the meeting row (Wave 7C).
	UpdateAISummary(ctx context.Context, tenantID, meetingID uuid.UUID, summary string, at time.Time) error
	DeleteMeeting(ctx context.Context, id, tenantID uuid.UUID) error
	ListMeetings(ctx context.Context, filter MeetingFilter) ([]Meeting, error)

	// Cross-tenant system queries (used under WithSystemContext, no tenant filter)
	GetMeetingByRoomName(ctx context.Context, roomName string) (*Meeting, error)
	ListStaleMeetings(ctx context.Context, cutoff time.Time) ([]Meeting, error)

	// Attendees
	AddAttendee(ctx context.Context, meetingID, userID uuid.UUID) error
	RemoveAttendee(ctx context.Context, meetingID, userID uuid.UUID) error
	UpdateAttendeeRSVP(ctx context.Context, meetingID, userID uuid.UUID, rsvp string) error
	// GetAttendees returns attendees with denormalized user info (first_name, last_name).
	GetAttendees(ctx context.Context, meetingID uuid.UUID) ([]MeetingAttendeeWithUser, error)

	// Notes
	SaveNotes(ctx context.Context, notes *MeetingNotes) error
	GetNotes(ctx context.Context, meetingID, authorID uuid.UUID) (*MeetingNotes, error)
	GetAllNotes(ctx context.Context, meetingID uuid.UUID) ([]MeetingNotes, error)
	GetPreviousMeetingNotes(ctx context.Context, recurringMeetingID uuid.UUID, beforeDate time.Time) (*MeetingNotes, error)

	// Action Items
	CreateActionItem(ctx context.Context, item *MeetingActionItem) error
	GetActionItemByID(ctx context.Context, id, tenantID uuid.UUID) (*MeetingActionItem, error)
	UpdateActionItem(ctx context.Context, item *MeetingActionItem, tenantID uuid.UUID) error
	DeleteActionItem(ctx context.Context, id, tenantID uuid.UUID) error
	ListActionItems(ctx context.Context, meetingID, tenantID uuid.UUID) ([]MeetingActionItem, error)
	UpdateActionItemTaskID(ctx context.Context, itemID, taskID uuid.UUID) error

	// Chat
	SaveChatMessage(ctx context.Context, msg *MeetingChatMessage) error
	ListChatMessages(ctx context.Context, meetingID, tenantID uuid.UUID, limit int) ([]MeetingChatMessage, error)

	// Co-hosts
	// AddCoHost grants co-host rights to userID for the given meeting. Idempotent
	// (ON CONFLICT DO NOTHING) — safe to call when already a co-host.
	AddCoHost(ctx context.Context, tenantID, meetingID, userID, grantedBy uuid.UUID) error
	// RemoveCoHost revokes co-host rights. Idempotent — returns nil when not found.
	RemoveCoHost(ctx context.Context, tenantID, meetingID, userID uuid.UUID) error
	// IsCoHost reports whether userID currently has co-host rights for meetingID.
	IsCoHost(ctx context.Context, tenantID, meetingID, userID uuid.UUID) (bool, error)
	// ListCoHosts returns all current co-hosts for a meeting.
	ListCoHosts(ctx context.Context, tenantID, meetingID uuid.UUID) ([]MeetingCoHost, error)

	// Lock
	// SetLocked updates the locked column on the meeting row.
	SetLocked(ctx context.Context, tenantID, meetingID uuid.UUID, locked bool) error

	// Breakout Rooms (Wave 6A)
	CreateBreakoutRoom(ctx context.Context, r *BreakoutRoom) error
	ListBreakoutRooms(ctx context.Context, meetingID, tenantID uuid.UUID) ([]*BreakoutRoom, error)
	GetBreakoutRoom(ctx context.Context, id, tenantID uuid.UUID) (*BreakoutRoom, error)
	// CloseAllBreakoutRooms sets status='closed' on all open rooms; returns the room_names closed.
	CloseAllBreakoutRooms(ctx context.Context, meetingID, tenantID uuid.UUID) ([]string, error)
	// UpsertBreakoutAssignment assigns or re-assigns a user to a breakout room. Idempotent.
	UpsertBreakoutAssignment(ctx context.Context, meetingID, breakoutRoomID, userID, assignedBy uuid.UUID) error
	// GetBreakoutAssignmentForUser returns the open breakout room the user is assigned to,
	// or nil, nil when no open assignment exists.
	GetBreakoutAssignmentForUser(ctx context.Context, meetingID, userID, tenantID uuid.UUID) (*BreakoutRoom, error)
	ListBreakoutAssignments(ctx context.Context, meetingID, tenantID uuid.UUID) ([]BreakoutAssignment, error)
	ClearBreakoutAssignment(ctx context.Context, meetingID, userID, tenantID uuid.UUID) error
	ClearAllBreakoutAssignments(ctx context.Context, meetingID, tenantID uuid.UUID) error
}

// MeetingFilter contains filtering parameters for listing meetings
type MeetingFilter struct {
	TenantID    uuid.UUID
	OrganizerID *uuid.UUID
	AttendeeID  *uuid.UUID
	Status      *string
	StartAfter  *time.Time
	StartBefore *time.Time
	Limit       int
	Offset      int
}