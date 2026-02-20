package integration

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Platform constants for supported integration platforms.
const (
	PlatformTeams = "teams"
	PlatformSlack = "slack"
)

// ValidPlatforms for validation.
var ValidPlatforms = map[string]bool{
	PlatformTeams: true,
	PlatformSlack: true,
}

// DeliveryStatus constants for delivery log entries.
const (
	DeliveryStatusSent        = "sent"
	DeliveryStatusFailed      = "failed"
	DeliveryStatusRateLimited = "rate_limited"
)

// IntegrationConfig represents a platform integration configuration (one per platform per org).
type IntegrationConfig struct {
	ID                  uuid.UUID       `json:"id"`
	Platform            string          `json:"platform"`
	IsActive            bool            `json:"is_active"`
	CredentialsVaultKey string          `json:"credentials_vault_key"`
	Metadata            json.RawMessage `json:"metadata"`
	CreatedBy           uuid.UUID       `json:"created_by"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// ChannelMapping maps a set of KMU Hub modules to an external platform channel.
type ChannelMapping struct {
	ID           uuid.UUID       `json:"id"`
	ConfigID     uuid.UUID       `json:"config_id"`
	ChannelID    string          `json:"channel_id"`
	ChannelName  string          `json:"channel_name"`
	Modules      []string        `json:"modules"`
	IsActive     bool            `json:"is_active"`
	PlatformData json.RawMessage `json:"platform_data"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// AccountLink maps an external platform user to a KMU Hub user.
type AccountLink struct {
	ID                  uuid.UUID `json:"id"`
	Platform            string    `json:"platform"`
	ExternalUserID      string    `json:"external_user_id"`
	KMUHubUserID        uuid.UUID `json:"kmuhub_user_id"`
	ExternalDisplayName string    `json:"external_display_name,omitempty"`
	LinkedAt            time.Time `json:"linked_at"`
}

// DeliveryLogEntry records the result of forwarding a notification to an external platform.
type DeliveryLogEntry struct {
	ID                uuid.UUID `json:"id"`
	NotificationID    uuid.UUID `json:"notification_id"`
	MappingID         uuid.UUID `json:"mapping_id"`
	Platform          string    `json:"platform"`
	Status            string    `json:"status"`
	PlatformMessageID string    `json:"platform_message_id,omitempty"`
	ErrorMessage      string    `json:"error_message,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// LinkToken represents a short-lived token for the account linking flow.
type LinkToken struct {
	ID             uuid.UUID `json:"id"`
	Platform       string    `json:"platform"`
	ExternalUserID string    `json:"external_user_id"`
	TokenHash      string    `json:"token_hash"`
	ExpiresAt      time.Time `json:"expires_at"`
	Used           bool      `json:"used"`
	CreatedAt      time.Time `json:"created_at"`
}

// ModuleColors maps module IDs to their brand colors for notification cards.
var ModuleColors = map[string]string{
	"crm":      "#3b82f6", // Blue
	"work":     "#8b5cf6", // Purple
	"hr":       "#22c55e", // Green
	"biz":      "#f59e0b", // Amber (Finance)
	"email":    "#0ea5e9", // Sky
	"chat":     "#10b981", // Emerald
	"calendar": "#f97316", // Orange
	"document": "#64748b", // Slate
	"system":   "#ef4444", // Red
}
