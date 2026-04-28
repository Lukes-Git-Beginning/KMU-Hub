package dialer

import (
	"time"

	"github.com/google/uuid"
)

// Campaign status constants
const (
	CampaignStatusDraft     = "draft"
	CampaignStatusActive    = "active"
	CampaignStatusPaused    = "paused"
	CampaignStatusCompleted = "completed"
	CampaignStatusArchived  = "archived"
)

// Campaign mode constants
const (
	CampaignModePreview    = "preview"
	CampaignModePower      = "power"      // Phase 2
	CampaignModePredictive = "predictive" // Phase 3
)

// Campaign contact status constants
const (
	ContactStatusPending    = "pending"
	ContactStatusInProgress = "in_progress"
	ContactStatusCompleted  = "completed"
	ContactStatusSkipped    = "skipped"
	ContactStatusCallback   = "callback"
)

// Agent status constants
const (
	AgentStatusAvailable = "available"
	AgentStatusOnCall    = "on_call"
	AgentStatusWrapUp    = "wrap_up"
	AgentStatusBreak     = "break"
	AgentStatusOffline   = "offline"
)

// Call event type constants
const (
	CallEventInitiated     = "initiated"
	CallEventRinging       = "ringing"
	CallEventAnswered      = "answered"
	CallEventInProgress    = "in_progress"
	CallEventOutcomeLogged = "outcome_logged"
	CallEventWrapUpStarted = "wrap_up_started"
	CallEventCompleted     = "completed"
	CallEventFailed        = "failed"
)

// Campaign represents a dialer campaign.
type Campaign struct {
	ID               uuid.UUID
	TenantID         uuid.UUID
	Name             string
	Description      *string
	Status           string
	Mode             string
	Settings         CampaignSettings
	CreatedBy        uuid.UUID
	AssignedAgentIDs []uuid.UUID
	ContactCount     int
	CompletedCount   int
	CreatedAt        time.Time
	UpdatedAt        time.Time
	StartedAt        *time.Time
	CompletedAt      *time.Time
}

// CampaignSettings holds configurable parameters for a campaign.
type CampaignSettings struct {
	MaxAttempts        int `json:"max_attempts,omitempty"`
	CallbackDelayHours int `json:"callback_delay_hours,omitempty"`
	WrapUpMinSeconds   int `json:"wrap_up_min_seconds,omitempty"`
	// Phase 2 fields
	CallHoursFrom string `json:"call_hours_from,omitempty"`
	CallHoursTo   string `json:"call_hours_to,omitempty"`
	TimeZone      string `json:"time_zone,omitempty"`
}

// CampaignContact represents a contact entry within a campaign queue.
type CampaignContact struct {
	ID           uuid.UUID
	CampaignID   uuid.UUID
	ContactID    uuid.UUID
	Position     int
	Status       string
	OutcomeID    *uuid.UUID
	Notes        *string
	CallbackAt   *time.Time
	LastCalledAt *time.Time
	CallCount    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	// Denormalized for display — filled by service layer, not stored in DB.
	ContactName    string
	ContactPhone   string
	ContactCompany string
}

// CallSession represents a single call attempt against a campaign contact.
type CallSession struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	CampaignContactID uuid.UUID
	CallSessionID     *uuid.UUID // FK to video call_sessions (nullable)
	AgentID           uuid.UUID
	OutcomeID         *uuid.UUID
	DurationSeconds   *int
	Notes             *string
	NextAction        *string
	AppointmentID     *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
	WrapUpCompletedAt *time.Time
}

// CallEvent represents a timestamped event within a call session.
type CallEvent struct {
	ID                  uuid.UUID
	DialerCallSessionID uuid.UUID
	EventType           string
	Payload             map[string]any
	OccurredAt          time.Time
}

// CallOutcome represents a selectable result for a completed call.
type CallOutcome struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	Label         string
	Color         string
	IsPositive    bool
	IsCallback    bool
	IsAppointment bool
	SortOrder     int
	IsActive      bool
	CreatedAt     time.Time
}

// AgentStatusEntry is the current live status of an agent (not stored; derived from log).
type AgentStatusEntry struct {
	UserID     uuid.UUID
	CampaignID *uuid.UUID
	Status     string
	Since      time.Time
	// Denormalized
	FirstName string
	LastName  string
}

// AgentStatusLogEntry is a persisted status-change record.
type AgentStatusLogEntry struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	CampaignID     *uuid.UUID
	Status         string
	PreviousStatus *string
	ChangedAt      time.Time
	ChangedBy      *uuid.UUID
}

// CampaignStats aggregates campaign progress and outcome data.
type CampaignStats struct {
	TotalContacts     int
	CompletedContacts int
	PendingContacts   int
	SkippedContacts   int
	CallbackContacts  int
	TotalCalls        int
	PositiveCalls     int
	AvgDurationSecs   float64
	OutcomeBreakdown  []OutcomeCount
}

// OutcomeCount pairs an outcome definition with its usage count.
type OutcomeCount struct {
	OutcomeID    uuid.UUID
	OutcomeLabel string
	OutcomeColor string
	Count        int
}

// AgentStats aggregates per-agent performance data for a given day.
type AgentStats struct {
	TotalCallsToday    int
	AvgDurationSecs    float64
	CallsPerHour       float64
	OutcomeBreakdown   []OutcomeCount
	ActiveCampaignID   *uuid.UUID
	ActiveCampaignName *string
}
