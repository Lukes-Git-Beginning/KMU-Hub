package presence

import (
	"time"

	"github.com/google/uuid"
)

// Presence level constants
const (
	PresenceOnline  = "online"
	PresenceAway    = "away"
	PresenceDND     = "dnd"
	PresenceInCall  = "in_call"
	PresenceOffline = "offline"
)

// ValidPresenceLevels contains all valid presence level values
var ValidPresenceLevels = map[string]bool{
	PresenceOnline:  true,
	PresenceAway:    true,
	PresenceDND:     true,
	PresenceInCall:  true,
	PresenceOffline: true,
}

// UserPresence represents the current presence status of a user.
// This is primarily stored in Redis for real-time access; the struct
// is used for serialization/deserialization.
type UserPresence struct {
	UserID         uuid.UUID  `json:"user_id"`
	Status         string     `json:"status"`
	ManualOverride bool       `json:"manual_override"`
	LastActivity   *time.Time `json:"last_activity,omitempty"`
	// Denormalized user info for API responses
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// PresenceConfig holds system-wide presence configuration
type PresenceConfig struct {
	ID                  uuid.UUID  `json:"id"`
	AwayTimeoutSeconds  int        `json:"away_timeout_seconds"`
	UpdatedAt           time.Time  `json:"updated_at"`
	UpdatedBy           *uuid.UUID `json:"updated_by,omitempty"`
}

// DefaultAwayTimeoutSeconds is the default idle timeout before marking a user as away
const DefaultAwayTimeoutSeconds = 300
