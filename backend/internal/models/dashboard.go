package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DashboardDefault holds the default dashboard layout for a given role.
type DashboardDefault struct {
	ID            string          `json:"id"`
	Role          string          `json:"role"`
	Layout        json.RawMessage `json:"layout"`
	ActiveWidgets json.RawMessage `json:"active_widgets"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// UserDashboardLayout holds a user's personal dashboard layout override.
type UserDashboardLayout struct {
	ID            string          `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	UserID        string          `json:"user_id"`
	Layout        json.RawMessage `json:"layout"`
	ActiveWidgets json.RawMessage `json:"active_widgets"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// DashboardLayoutResponse is the unified response returned to clients.
// It may originate from either a user override or a role default.
type DashboardLayoutResponse struct {
	Layout        json.RawMessage `json:"layout"`
	ActiveWidgets json.RawMessage `json:"active_widgets"`
	IsCustom      bool            `json:"is_custom"`
	UpdatedAt     time.Time       `json:"updated_at"`
}
