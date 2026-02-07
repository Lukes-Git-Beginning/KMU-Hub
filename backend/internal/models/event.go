package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType represents a registered event type in the system
type EventType struct {
	ID                 uuid.UUID `json:"id"`
	ModuleID           string    `json:"module_id"`
	EventKey           string    `json:"event_key"`
	DisplayName        string    `json:"display_name"`
	Description        *string   `json:"description,omitempty"`
	DefaultPriority    string    `json:"default_priority"`
	DefaultInApp       bool      `json:"default_in_app"`
	DefaultDesktopPush bool      `json:"default_desktop_push"`
	DefaultSound       string    `json:"default_sound"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Event represents a persisted event in the events durability table
type Event struct {
	ID           uuid.UUID       `json:"id"`
	EventTypeKey string          `json:"event_type_key"`
	ModuleID     string          `json:"module_id"`
	Priority     string          `json:"priority"`
	ActorID      *uuid.UUID      `json:"actor_id,omitempty"`
	ResourceID   *string         `json:"resource_id,omitempty"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	Processed    bool            `json:"processed"`
	CreatedAt    time.Time       `json:"created_at"`
}

// EventPayload represents the payload sent via PostgreSQL LISTEN/NOTIFY.
// This is the compact metadata format (under 8000 bytes).
type EventPayload struct {
	Type          string          `json:"type"`
	Priority      string          `json:"priority"`
	ActorID       string          `json:"actor_id,omitempty"`
	ResourceID    string          `json:"resource_id,omitempty"`
	ModuleID      string          `json:"module_id"`
	TargetUserIDs []string        `json:"target_user_ids,omitempty"`
	Title         string          `json:"title,omitempty"`
	Body          string          `json:"body,omitempty"`
	DeepLink      string          `json:"deep_link,omitempty"`
	GroupKey      string          `json:"group_key,omitempty"`
	Payload       json.RawMessage `json:"payload,omitempty"`
	Timestamp     time.Time       `json:"timestamp"`
}
