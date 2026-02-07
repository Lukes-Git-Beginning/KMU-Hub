package models

import (
	"time"

	"github.com/google/uuid"
)

// MentionType represents the type of mention
type MentionType string

const (
	MentionTypeUser     MentionType = "user"
	MentionTypeChannel  MentionType = "channel"
	MentionTypeEveryone MentionType = "everyone"
)

// Mention represents a mention in a message
type Mention struct {
	MessageID   uuid.UUID   `json:"message_id"`
	UserID      uuid.UUID   `json:"user_id"`
	MentionType MentionType `json:"mention_type"`
	CreatedAt   time.Time   `json:"created_at"`
}

// MentionWithUser includes user details for API responses
type MentionWithUser struct {
	Mention
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
