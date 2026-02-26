package guest

import (
	"time"

	"github.com/google/uuid"
)

// GuestSession represents a guest visitor's chat session
type GuestSession struct {
	ID             uuid.UUID  `json:"id"`
	TokenHash      string     `json:"-"` // never exposed in API responses
	ChannelID      uuid.UUID  `json:"channel_id"`
	DisplayName    string     `json:"display_name"`
	Email          *string    `json:"email,omitempty"`
	IPAddress      *string    `json:"ip_address,omitempty"`
	UserAgent      *string    `json:"user_agent,omitempty"`
	LastActivityAt time.Time  `json:"last_activity_at"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
}

// GuestChannelConfig holds per-channel guest chat settings
type GuestChannelConfig struct {
	ID               uuid.UUID `json:"id"`
	ChannelID        uuid.UUID `json:"channel_id"`
	WelcomeMessage   string    `json:"welcome_message"`
	LogoURL          *string   `json:"logo_url,omitempty"`
	PrimaryColor     string    `json:"primary_color"`
	TokenExpiryHours int       `json:"token_expiry_hours"`
	MaxFileSizeMB    int       `json:"max_file_size_mb"`
	AllowedFileTypes string    `json:"allowed_file_types"`
	IsActive         bool      `json:"is_active"`
	CreatedBy        uuid.UUID `json:"created_by"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// CreateSessionInput contains the data needed to create a guest session
type CreateSessionInput struct {
	ChannelID   uuid.UUID
	DisplayName string
	Email       *string
	IPAddress   *string
	UserAgent   *string
}

// CreateConfigInput contains the data needed to create guest channel config
type CreateConfigInput struct {
	ChannelID        uuid.UUID
	WelcomeMessage   *string // nil = use default
	LogoURL          *string
	PrimaryColor     *string // nil = use default #3b82f6
	TokenExpiryHours *int    // nil = use default 168
	MaxFileSizeMB    *int    // nil = use default 10
	AllowedFileTypes *string
	CreatedBy        uuid.UUID
}

// TokenResult is returned when creating a session (contains plain token shown once)
type TokenResult struct {
	Token   string        `json:"token"`
	Session *GuestSession `json:"session"`
}
