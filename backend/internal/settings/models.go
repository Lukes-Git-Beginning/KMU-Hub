package settings

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ModuleLead represents a user who has been granted Modul-Leiter rights
// for a specific module within a tenant.
type ModuleLead struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	ModuleID  string
	GrantedBy *uuid.UUID
	GrantedAt time.Time
}

// UserModuleGrant records that a user has access to a module within a tenant.
// UserName is joined from the users table for display and is not stored on the
// grant row itself. LastActiveAt is reserved for a future activity pipeline and
// stays nil until that lands.
type UserModuleGrant struct {
	TenantID     uuid.UUID
	UserID       uuid.UUID
	UserName     string
	ModuleID     string
	GrantedBy    *uuid.UUID
	GrantedAt    time.Time
	LastActiveAt *time.Time
}

// ModuleGrantRef is a (user, module) pair targeted by a bulk revoke.
type ModuleGrantRef struct {
	UserID   uuid.UUID
	ModuleID string
}

// SettingEntry is a key→JSONB value pair for a module setting.
// Value is stored as raw JSON to accommodate any shape the FE defines
// (string, number, boolean, array, object).
type SettingEntry struct {
	Key   string
	Value json.RawMessage
}

// TenantSetting is the full DB row for a tenant-scoped setting.
type TenantSetting struct {
	TenantID  uuid.UUID
	ModuleID  string
	Key       string
	Value     json.RawMessage
	UpdatedBy *uuid.UUID
	UpdatedAt time.Time
}

// UserSetting is the full DB row for a user-scoped setting.
type UserSetting struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	ModuleID  string
	Key       string
	Value     json.RawMessage
	UpdatedAt time.Time
}

// TenantModule is one catalogue module together with this tenant's activation
// state. Group and the module id come from internal/modules; Active and
// AssignedSeats come from the database.
type TenantModule struct {
	ModuleID      string
	Group         string
	Active        bool
	AssignedSeats int32
	// FlagKey names the deployment-wide feature flag gating the module, or "".
	// The gateway resolves it; it never reaches the HTTP response.
	FlagKey string
}

// TenantSubscription is the booked contract of a tenant. BillingPeriodEnd and
// TotalSeats are nil when no period has been agreed resp. seats are unlimited.
type TenantSubscription struct {
	PlanType         string
	SupportTier      string
	Status           string
	BillingPeriodEnd *time.Time
	TotalSeats       *int32
}
