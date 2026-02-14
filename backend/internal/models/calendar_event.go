package models

import (
	"time"

	"github.com/google/uuid"
)

// RSVP status constants
const (
	RSVPPending   = "pending"
	RSVPAccepted  = "accepted"
	RSVPDeclined  = "declined"
	RSVPTentative = "tentative"
)

// ValidRSVPStatuses contains all valid RSVP status values
var ValidRSVPStatuses = map[string]bool{
	RSVPPending:   true,
	RSVPAccepted:  true,
	RSVPDeclined:  true,
	RSVPTentative: true,
}

// Recurring event edit scope constants
const (
	ScopeThisEvent      = "this"
	ScopeThisAndFuture  = "this_and_future"
	ScopeAllEvents      = "all"
)

// ValidRecurringEditScopes contains all valid recurring event edit scopes
var ValidRecurringEditScopes = map[string]bool{
	ScopeThisEvent:     true,
	ScopeThisAndFuture: true,
	ScopeAllEvents:     true,
}

// CalendarEvent represents an event on a calendar
type CalendarEvent struct {
	ID              uuid.UUID  `json:"id"`
	CalendarID      uuid.UUID  `json:"calendar_id"`
	Title           string     `json:"title"`
	Description     *string    `json:"description,omitempty"`
	Location        *string    `json:"location,omitempty"`
	ResourceID      *uuid.UUID `json:"resource_id,omitempty"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         time.Time  `json:"end_time"`
	IsAllDay        bool       `json:"is_all_day"`
	Timezone        string     `json:"timezone"`
	RRule           *string    `json:"rrule,omitempty"`
	RecurrenceEnd   *time.Time `json:"recurrence_end,omitempty"`
	HasVideoCall    bool       `json:"has_video_call"`
	LiveKitRoomName *string    `json:"livekit_room_name,omitempty"`
	CategoryID      *uuid.UUID `json:"category_id,omitempty"`
	CreatedBy       uuid.UUID  `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ExpandedEvent represents a calendar event with denormalized display data,
// including expanded recurring event instances
type ExpandedEvent struct {
	CalendarEvent
	OriginalEventID *uuid.UUID `json:"original_event_id,omitempty"` // set for recurring instances
	OriginalDate    *time.Time `json:"original_date,omitempty"`
	CalendarName    string     `json:"calendar_name"`
	CalendarColor   string     `json:"calendar_color"`
	DisplayColor    string     `json:"display_color"` // COALESCE(color_override, calendar_color)
	CategoryName    *string    `json:"category_name,omitempty"`
	CategoryColor   *string    `json:"category_color,omitempty"`
	ResourceName    *string    `json:"resource_name,omitempty"`
	AttendeeCount   int        `json:"attendee_count"`
	MyRSVP          *string    `json:"my_rsvp,omitempty"`
}

// EventAttendee represents a user invited to a calendar event with RSVP status
type EventAttendee struct {
	EventID     uuid.UUID  `json:"event_id"`
	UserID      uuid.UUID  `json:"user_id"`
	RSVPStatus  string     `json:"rsvp_status"`
	RespondedAt *time.Time `json:"responded_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	FirstName   string     `json:"first_name"` // denormalized from users table
	LastName    string     `json:"last_name"`  // denormalized from users table
}

// EventException represents a modification or cancellation of a single
// occurrence in a recurring event series
type EventException struct {
	ID           uuid.UUID  `json:"id"`
	EventID      uuid.UUID  `json:"event_id"`
	OriginalDate time.Time  `json:"original_date"` // DATE type
	IsCancelled  bool       `json:"is_cancelled"`
	Title        *string    `json:"title,omitempty"`
	Description  *string    `json:"description,omitempty"`
	Location     *string    `json:"location,omitempty"`
	StartTime    *time.Time `json:"start_time,omitempty"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	ResourceID   *uuid.UUID `json:"resource_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// EventReminder represents a reminder for a calendar event
type EventReminder struct {
	ID            uuid.UUID `json:"id"`
	EventID       uuid.UUID `json:"event_id"`
	MinutesBefore int       `json:"minutes_before"`
	CreatedAt     time.Time `json:"created_at"`
}

// UserCalendarPreferences stores per-user global calendar settings
type UserCalendarPreferences struct {
	UserID                     uuid.UUID `json:"user_id"`
	DefaultView                string    `json:"default_view"`
	WeekDays                   int       `json:"week_days"`
	DefaultReminderMinutes     int       `json:"default_reminder_minutes"`
	DefaultAllDayReminderMinutes int    `json:"default_allday_reminder_minutes"`
	SubdivisionCode            *string   `json:"subdivision_code,omitempty"`
	ShowTaskDeadlines          bool      `json:"show_task_deadlines"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

// TaskDeadlineStub is a lightweight task representation for the calendar
// deadline overlay layer
type TaskDeadlineStub struct {
	TaskID     uuid.UUID `json:"task_id"`
	Title      string    `json:"title"`
	DueDate    time.Time `json:"due_date"`
	ProjectKey *string   `json:"project_key,omitempty"`
	Priority   string    `json:"priority"`
}
