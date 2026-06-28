package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// 2FA fields
	TwoFactorEnabled         bool       `json:"two_factor_enabled"`
	TwoFactorSecretEncrypted string     `json:"-"`
	TwoFactorPendingSecret   string     `json:"-"`
	TwoFactorEnabledAt       *time.Time `json:"two_factor_enabled_at,omitempty"`

	// Locale preference
	Locale string `json:"locale"`

	// Avatar object key ({tenant_id}/avatar/{uuid}{ext}) in MinIO; resolved to a
	// viewable URL on the client via the presigned-download endpoint.
	AvatarURL string `json:"avatar_url"`
}

type Role struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Permission struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Resource  string    `json:"resource"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}
