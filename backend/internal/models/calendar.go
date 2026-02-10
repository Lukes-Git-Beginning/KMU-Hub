package models

import (
	"time"

	"github.com/google/uuid"
)

// Calendar type constants
const (
	CalendarTypePersonal = "personal"
	CalendarTypeShared   = "shared"
	CalendarTypeResource = "resource"
)

// ValidCalendarTypes contains all valid calendar types
var ValidCalendarTypes = map[string]bool{
	CalendarTypePersonal: true,
	CalendarTypeShared:   true,
	CalendarTypeResource: true,
}

// Calendar permission constants
const (
	CalendarPermissionView  = "view"
	CalendarPermissionEdit  = "edit"
	CalendarPermissionAdmin = "admin"
)

// ValidCalendarPermissions contains all valid calendar member permissions
var ValidCalendarPermissions = map[string]bool{
	CalendarPermissionView:  true,
	CalendarPermissionEdit:  true,
	CalendarPermissionAdmin: true,
}

// Calendar represents a personal or shared calendar
type Calendar struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	CalendarType string    `json:"calendar_type"`
	Color        string    `json:"color"`
	OwnerID      uuid.UUID `json:"owner_id"`
	IsDefault    bool      `json:"is_default"`
	Timezone     string    `json:"timezone"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CalendarWithMemberInfo includes the current user's membership details for list queries
type CalendarWithMemberInfo struct {
	Calendar
	Permission    string  `json:"permission"`
	ColorOverride *string `json:"color_override,omitempty"`
	IsVisible     bool    `json:"is_visible"`
}

// CalendarMember represents a user's membership in a calendar
type CalendarMember struct {
	CalendarID    uuid.UUID `json:"calendar_id"`
	UserID        uuid.UUID `json:"user_id"`
	Permission    string    `json:"permission"`
	ColorOverride *string   `json:"color_override,omitempty"`
	IsVisible     bool      `json:"is_visible"`
	CreatedAt     time.Time `json:"created_at"`
	FirstName     string    `json:"first_name"` // denormalized from users table
	LastName      string    `json:"last_name"`  // denormalized from users table
}
