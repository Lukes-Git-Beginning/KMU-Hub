package video

import (
	"time"

	"github.com/google/uuid"
)

// Call type constants
const (
	CallTypeOneToOne = "one_to_one"
	CallTypeGroup    = "group"
)

// ValidCallTypes contains all valid call type values
var ValidCallTypes = map[string]bool{
	CallTypeOneToOne: true,
	CallTypeGroup:    true,
}

// Call status constants
const (
	CallStatusRinging = "ringing"
	CallStatusActive  = "active"
	CallStatusEnded   = "ended"
)

// ValidCallStatuses contains all valid call status values
var ValidCallStatuses = map[string]bool{
	CallStatusRinging: true,
	CallStatusActive:  true,
	CallStatusEnded:   true,
}

// CallSession represents an active or completed video/voice call
type CallSession struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	CallType        string     `json:"call_type"`
	Status          string     `json:"status"`
	RoomName        string     `json:"room_name"`
	InitiatorID     uuid.UUID  `json:"initiator_id"`
	ChannelID       *uuid.UUID `json:"channel_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	DurationSeconds *int       `json:"duration_seconds,omitempty"`
}

// CallParticipant represents a user participating in a call
type CallParticipant struct {
	CallID   uuid.UUID  `json:"call_id"`
	TenantID uuid.UUID  `json:"tenant_id"`
	UserID   uuid.UUID  `json:"user_id"`
	JoinedAt time.Time  `json:"joined_at"`
	LeftAt   *time.Time `json:"left_at,omitempty"`
	HasVideo bool       `json:"has_video"`
	HasAudio bool       `json:"has_audio"`
}

// CallParticipantWithUser includes denormalized user info for API responses
type CallParticipantWithUser struct {
	CallParticipant
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// CallSessionWithParticipants combines a call session with its participants
type CallSessionWithParticipants struct {
	CallSession
	Participants []CallParticipantWithUser `json:"participants"`
}
