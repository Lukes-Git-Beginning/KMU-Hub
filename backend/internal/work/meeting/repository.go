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
	GetMeeting(ctx context.Context, id uuid.UUID) (*Meeting, error)
	UpdateMeeting(ctx context.Context, m *Meeting) error
	DeleteMeeting(ctx context.Context, id uuid.UUID) error
	ListMeetings(ctx context.Context, filter MeetingFilter) ([]Meeting, error)

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
	UpdateActionItem(ctx context.Context, item *MeetingActionItem) error
	DeleteActionItem(ctx context.Context, id uuid.UUID) error
	ListActionItems(ctx context.Context, meetingID uuid.UUID) ([]MeetingActionItem, error)
	UpdateActionItemTaskID(ctx context.Context, itemID, taskID uuid.UUID) error
}

// MeetingFilter contains filtering parameters for listing meetings
type MeetingFilter struct {
	OrganizerID *uuid.UUID
	AttendeeID  *uuid.UUID
	Status      *string
	StartAfter  *time.Time
	StartBefore *time.Time
	Limit       int
	Offset      int
}
