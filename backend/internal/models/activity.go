package models

import (
	"time"

	"github.com/google/uuid"
)

// ActivityType represents the type of activity
type ActivityType string

const (
	ActivityTypeCall    ActivityType = "call"
	ActivityTypeMeeting ActivityType = "meeting"
	ActivityTypeNote    ActivityType = "note"
	ActivityTypeEmail   ActivityType = "email"
	ActivityTypeTask    ActivityType = "task"
)

// ValidActivityTypes for validation
var ValidActivityTypes = map[ActivityType]bool{
	ActivityTypeCall:    true,
	ActivityTypeMeeting: true,
	ActivityTypeNote:    true,
	ActivityTypeEmail:   true,
	ActivityTypeTask:    true,
}

// IsValid checks if the activity type is valid
func (at ActivityType) IsValid() bool {
	return ValidActivityTypes[at]
}

// String returns the string representation of the activity type
func (at ActivityType) String() string {
	return string(at)
}

// Activity represents a CRM activity (call, meeting, note, email, task)
type Activity struct {
	ID           uuid.UUID    `json:"id"`
	TenantID     uuid.UUID    `json:"tenant_id"`
	ActivityType ActivityType `json:"activity_type"`
	Subject      string       `json:"subject"`
	Description  *string      `json:"description,omitempty"`
	ContactID    *uuid.UUID   `json:"contact_id,omitempty"`
	CompanyID    *uuid.UUID   `json:"company_id,omitempty"`
	DealID       *uuid.UUID   `json:"deal_id,omitempty"`
	AssignedTo   *uuid.UUID   `json:"assigned_to,omitempty"`
	DueDate      *time.Time   `json:"due_date,omitempty"`
	IsCompleted  bool         `json:"is_completed"`
	CompletedAt  *time.Time   `json:"completed_at,omitempty"`
	CreatedBy    uuid.UUID    `json:"created_by"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// ActivityWithRelations includes associated data for API responses
type ActivityWithRelations struct {
	Activity
	ContactName    *string        `json:"contact_name,omitempty"`
	CompanyName    *string        `json:"company_name,omitempty"`
	DealName       *string        `json:"deal_name,omitempty"`
	AssignedToName *string        `json:"assigned_to_name,omitempty"`
	Tags           []*Tag         `json:"tags,omitempty"`
	CustomFields   map[string]any `json:"custom_fields,omitempty"`
}
