package models

import (
	"time"

	"github.com/google/uuid"
)

// App-specific password scope constants
const (
	AppPasswordScopeCalDAV = "caldav_carddav"
)

// CalDAV change type constants
const (
	ChangeTypeCreated  = "created"
	ChangeTypeModified = "modified"
	ChangeTypeDeleted  = "deleted"
)

// CalDAV collection type constants
const (
	CollectionCalendar    = "calendar"
	CollectionAddressBook = "addressbook"
)

// AppSpecificPassword represents a per-user app-specific password
// for CalDAV/CardDAV HTTP Basic Auth.
type AppSpecificPassword struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	UserID         uuid.UUID  `json:"user_id"`
	Label          string     `json:"label"`
	PasswordHash   string     `json:"-"`
	PasswordPrefix string     `json:"password_prefix"`
	Scope          string     `json:"scope"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}

// CalDAVSyncVersion tracks the current sync version for a collection
// (calendar or address book).
type CalDAVSyncVersion struct {
	CollectionType string    `json:"collection_type"`
	CollectionID   uuid.UUID `json:"collection_id"`
	SyncVersion    int64     `json:"sync_version"`
}

// CalDAVChangeLogEntry records a single mutation in a CalDAV collection
// for incremental sync (RFC 6578).
type CalDAVChangeLogEntry struct {
	ID             int64     `json:"id"`
	CollectionType string    `json:"collection_type"`
	CollectionID   uuid.UUID `json:"collection_id"`
	ObjectPath     string    `json:"object_path"`
	ChangeType     string    `json:"change_type"`
	SyncVersion    int64     `json:"sync_version"`
	ChangedAt      time.Time `json:"changed_at"`
}

// CalDAVSetting represents a key-value pair for global CalDAV/CardDAV
// organization settings.
type CalDAVSetting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}
