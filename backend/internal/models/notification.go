package models

import (
	"time"

	"github.com/google/uuid"
)

// Priority constants for notification priority levels
const (
	PriorityUrgent = "urgent"
	PriorityHigh   = "urgent" // alias: frontend sends "high", stored as "urgent"
	PriorityNormal = "normal"
	PriorityLow    = "low"
)

// ValidPriorities for validation
var ValidPriorities = map[string]bool{
	PriorityUrgent: true,
	PriorityNormal: true,
	PriorityLow:    true,
	"high":         true, // frontend alias → mapped to "urgent" before DB write
}

// Notification represents a user notification
type Notification struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	UserID           uuid.UUID  `json:"user_id"`
	EventTypeKey     string     `json:"event_type_key"`
	ModuleID         string     `json:"module_id"`
	Priority         string     `json:"priority"`
	ActorID          *uuid.UUID `json:"actor_id,omitempty"`
	ActorName        *string    `json:"actor_name,omitempty"`
	ResourceID       *string    `json:"resource_id,omitempty"`
	Title            string     `json:"title"`
	Body             *string    `json:"body,omitempty"`
	DeepLink         *string    `json:"deep_link,omitempty"`
	GroupKey         *string    `json:"group_key,omitempty"`
	GroupCount       int        `json:"group_count"`
	IsRead           bool       `json:"is_read"`
	ReadAt           *time.Time `json:"read_at,omitempty"`
	IsPinned         bool       `json:"is_pinned"`
	IsDismissed      bool       `json:"is_dismissed"`
	DeliveredDesktop bool       `json:"delivered_desktop"`
	SnoozedUntil     *time.Time `json:"snoozed_until,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// NotificationPreference represents a user's notification preference
type NotificationPreference struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	UserID       uuid.UUID  `json:"user_id"`
	EventTypeKey *string    `json:"event_type_key,omitempty"`
	ModuleID     *string    `json:"module_id,omitempty"`
	InApp        bool       `json:"in_app"`
	DesktopPush  bool       `json:"desktop_push"`
	Sound        string     `json:"sound"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// NotificationMute represents a muted resource for a user
type NotificationMute struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	UserID     uuid.UUID `json:"user_id"`
	ModuleID   string    `json:"module_id"`
	ResourceID string    `json:"resource_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// QuietHours represents a user's quiet hours configuration
type QuietHours struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	UserID        uuid.UUID  `json:"user_id"`
	StartTime     string     `json:"start_time"`
	EndTime       string     `json:"end_time"`
	Timezone      string     `json:"timezone"`
	DaysOfWeek    []int      `json:"days_of_week"`
	Enabled       bool       `json:"enabled"`
	ManualDND     bool       `json:"manual_dnd"`
	ManualDNDUntil *time.Time `json:"manual_dnd_until,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
