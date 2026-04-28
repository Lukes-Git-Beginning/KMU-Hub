package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// InboxMessage represents a normalized message in the unified inbox.
// Each message is sourced from a channel (email, chat, notification) and
// identified by a unique (user_id, channel, source_id) combination.
type InboxMessage struct {
	ID            uuid.UUID        `json:"id"`
	TenantID      uuid.UUID        `json:"tenant_id"`
	UserID        uuid.UUID        `json:"user_id"`
	Channel       string           `json:"channel"`
	SourceID      string           `json:"source_id"`
	SenderName    string           `json:"sender_name"`
	SenderID      *uuid.UUID       `json:"sender_id,omitempty"`
	SenderEmail   *string          `json:"sender_email,omitempty"`
	Subject       string           `json:"subject"`
	Preview       string           `json:"preview"`
	IsRead        bool             `json:"is_read"`
	IsStarred     bool             `json:"is_starred"`
	IsArchived    bool             `json:"is_archived"`
	SnoozedUntil  *time.Time       `json:"snoozed_until,omitempty"`
	AssignedTo    *uuid.UUID       `json:"assigned_to,omitempty"`
	TeamInboxID   *uuid.UUID       `json:"team_inbox_id,omitempty"`
	Tags          []string         `json:"tags"`
	DeepLink      string           `json:"deep_link"`
	CRMContactID  *uuid.UUID       `json:"crm_contact_id,omitempty"`
	Metadata      json.RawMessage  `json:"metadata,omitempty"`
	ReceivedAt    time.Time        `json:"received_at"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

// TeamInbox represents a shared inbox for a team.
// Messages can be routed to team inboxes and claimed by members.
type TeamInbox struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Description       *string   `json:"description,omitempty"`
	AssignmentMode    string    `json:"assignment_mode"`
	Visibility        string    `json:"visibility"`
	NextAssigneeIndex int       `json:"next_assignee_index"`
	CreatedBy         uuid.UUID `json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// TeamInboxMember represents a user's membership in a team inbox.
type TeamInboxMember struct {
	TeamInboxID uuid.UUID `json:"team_inbox_id"`
	UserID      uuid.UUID `json:"user_id"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

// RoutingRule defines an automatic routing rule for inbox messages.
// Rules are evaluated in priority order (lower number = higher priority).
type RoutingRule struct {
	ID         uuid.UUID       `json:"id"`
	Name       string          `json:"name"`
	Channel    *string         `json:"channel,omitempty"`
	Conditions json.RawMessage `json:"conditions"`
	Actions    json.RawMessage `json:"actions"`
	Priority   int             `json:"priority"`
	IsActive   bool            `json:"is_active"`
	CreatedBy  uuid.UUID       `json:"created_by"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// Condition represents a rule condition tree with AND/OR logic and leaf comparisons.
// Used by the routing rule evaluator and designed for reuse by the Phase 16 Automation Engine.
type Condition struct {
	And      []Condition `json:"and,omitempty"`
	Or       []Condition `json:"or,omitempty"`
	Field    string      `json:"field,omitempty"`
	Operator string      `json:"operator,omitempty"`
	Value    interface{} `json:"value,omitempty"`
}

// Action represents what happens when a routing rule matches a message.
type Action struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

// UnreadCount is a helper struct for unread count aggregation by channel.
type UnreadCount struct {
	Channel string `json:"channel"`
	Count   int    `json:"count"`
}
