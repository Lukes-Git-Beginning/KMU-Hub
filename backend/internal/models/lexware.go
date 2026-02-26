package models

import (
	"time"

	"github.com/google/uuid"
)

// LexwareSyncConfig holds per-tenant Lexware sync configuration.
type LexwareSyncConfig struct {
	ID                       uuid.UUID  `json:"id"`
	ConfigID                 uuid.UUID  `json:"config_id"`
	ContactSyncEnabled       bool       `json:"contact_sync_enabled"`
	ContactSyncIntervalMin   int        `json:"contact_sync_interval_minutes"`
	InvoicePushEnabled       bool       `json:"invoice_push_enabled"`
	QuotePushEnabled         bool       `json:"quote_push_enabled"`
	WebhookEnabled           bool       `json:"webhook_enabled"`
	LastContactSyncAt        *time.Time `json:"last_contact_sync_at,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

// LexwareEntityMapping tracks KMU Hub ID <-> Lexware ID pairs.
// Lexware uses UUID strings (not integers like Bexio).
type LexwareEntityMapping struct {
	ID               uuid.UUID  `json:"id"`
	ConfigID         uuid.UUID  `json:"config_id"`
	EntityType       string     `json:"entity_type"`
	KmuhubID         uuid.UUID  `json:"kmuhub_id"`
	LexwareID        string     `json:"lexware_id"`
	LexwareVersion   int        `json:"lexware_version"`
	LastSyncedAt     time.Time  `json:"last_synced_at"`
	LexwareUpdatedAt *time.Time `json:"lexware_updated_at,omitempty"`
	KmuhubUpdatedAt  *time.Time `json:"kmuhub_updated_at,omitempty"`
	SyncDirection    string     `json:"sync_direction"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// LexwareFieldMapping holds configurable field mapping for one entity type.
type LexwareFieldMapping struct {
	ID         uuid.UUID                 `json:"id"`
	ConfigID   uuid.UUID                 `json:"config_id"`
	EntityType string                    `json:"entity_type"`
	Mappings   []LexwareFieldMappingEntry `json:"mappings"`
	CreatedAt  time.Time                 `json:"created_at"`
	UpdatedAt  time.Time                 `json:"updated_at"`
}

// LexwareFieldMappingEntry represents a single field mapping rule.
type LexwareFieldMappingEntry struct {
	KmuhubField  string `json:"kmuhub_field"`
	LexwareField string `json:"lexware_field"`
	Direction    string `json:"direction"`
	Required     bool   `json:"required"`
}

// LexwareSyncLog records a single sync operation for audit purposes.
type LexwareSyncLog struct {
	ID             uuid.UUID      `json:"id"`
	ConfigID       uuid.UUID      `json:"config_id"`
	SyncType       string         `json:"sync_type"`
	Status         string         `json:"status"`
	ItemsProcessed int            `json:"items_processed"`
	ItemsCreated   int            `json:"items_created"`
	ItemsUpdated   int            `json:"items_updated"`
	ItemsFailed    int            `json:"items_failed"`
	ErrorMessage   *string        `json:"error_message,omitempty"`
	StartedAt      time.Time      `json:"started_at"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
	Metadata       map[string]any `json:"metadata"`
}

// LexwareSyncStatus provides a computed overview of sync state.
type LexwareSyncStatus struct {
	ContactSyncEnabled     bool       `json:"contact_sync_enabled"`
	InvoicePushEnabled     bool       `json:"invoice_push_enabled"`
	QuotePushEnabled       bool       `json:"quote_push_enabled"`
	WebhookEnabled         bool       `json:"webhook_enabled"`
	LastContactSyncAt      *time.Time `json:"last_contact_sync_at,omitempty"`
	TotalContactsMapped    int        `json:"total_contacts_mapped"`
	TotalInvoicesMapped    int        `json:"total_invoices_mapped"`
	TotalQuotesMapped      int        `json:"total_quotes_mapped"`
	LastSyncError          *string    `json:"last_sync_error,omitempty"`
	LastSyncErrorAt        *time.Time `json:"last_sync_error_at,omitempty"`
}
