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
	DeleteMeeting(ctx context.Context, id, tenantID uuid.UUID) error
	ListMeetings(ctx context.Context, filter MeetingFilter) ([]Meeting, error)

	// Cross-tenant system queries (used under WithSystemContext, no tenant filter)
	GetMeetingByRoomName(ctx context.Context, roomName string) (*Meeting, error)
	ListStaleMeetings(ctx context.Context, cutoff time.Time) ([]Meeting, error)

	// Attendees
	AddAttendee(ctx context.Context, meetingID, userID uuid.UUID) error
	RemoveAttendee(ctx context.Context, meetingID, userID uuid.UUID) error
	UpdateAttendeeRSVP(ctx context.Context, meetingID, userID uuid.UUID, rsvp string) error
	GetAttendees(ctx context.Context, meetingID uuid.UUID) ([]MeetingAttendee, error)

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
