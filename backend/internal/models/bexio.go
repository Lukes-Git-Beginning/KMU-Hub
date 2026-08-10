package models

import (
	"time"

	"github.com/google/uuid"
)

// BexioSyncConfig holds per-tenant Bexio sync configuration.
type BexioSyncConfig struct {
	ID                       uuid.UUID  `json:"id"`
	TenantID                 uuid.UUID  `json:"tenant_id"`
	ConfigID                 uuid.UUID  `json:"config_id"`
	ContactSyncEnabled       bool       `json:"contact_sync_enabled"`
	ContactSyncIntervalMin   int        `json:"contact_sync_interval_minutes"`
	InvoicePushEnabled       bool       `json:"invoice_push_enabled"`
	QuotePushEnabled         bool       `json:"quote_push_enabled"`
	PaymentPollEnabled       bool       `json:"payment_poll_enabled"`
	PaymentPollIntervalMin   int        `json:"payment_poll_interval_minutes"`
	InvoicePullEnabled       bool       `json:"invoice_pull_enabled"`
	InvoicePullIntervalMin   int        `json:"invoice_pull_interval_minutes"`
	LastContactSyncAt        *time.Time `json:"last_contact_sync_at,omitempty"`
	LastPaymentPollAt        *time.Time `json:"last_payment_poll_at,omitempty"`
	LastInvoicePullAt        *time.Time `json:"last_invoice_pull_at,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

// BexioEntityMapping tracks KMU Hub ID <-> Bexio ID pairs.
type BexioEntityMapping struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	ConfigID        uuid.UUID  `json:"config_id"`
	EntityType      string     `json:"entity_type"`
	KmuhubID        uuid.UUID  `json:"kmuhub_id"`
	BexioID         int        `json:"bexio_id"`
	LastSyncedAt    time.Time  `json:"last_synced_at"`
	BexioUpdatedAt  *time.Time `json:"bexio_updated_at,omitempty"`
	KmuhubUpdatedAt *time.Time `json:"kmuhub_updated_at,omitempty"`
	SyncDirection   string     `json:"sync_direction"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// BexioFieldMapping holds configurable field mapping for one entity type.
type BexioFieldMapping struct {
	ID         uuid.UUID              `json:"id"`
	TenantID   uuid.UUID              `json:"tenant_id"`
	ConfigID   uuid.UUID              `json:"config_id"`
	EntityType string                 `json:"entity_type"`
	Mappings   []BexioFieldMappingEntry `json:"mappings"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// BexioFieldMappingEntry represents a single field mapping rule.
type BexioFieldMappingEntry struct {
	KmuhubField string `json:"kmuhub_field"`
	BexioField  string `json:"bexio_field"`
	Direction   string `json:"direction"`
	Required    bool   `json:"required"`
}

// BexioSyncLog records a single sync operation for audit purposes.
type BexioSyncLog struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	ConfigID       uuid.UUID  `json:"config_id"`
	SyncType       string     `json:"sync_type"`
	Status         string     `json:"status"`
	ItemsProcessed int        `json:"items_processed"`
	ItemsCreated   int        `json:"items_created"`
	ItemsUpdated   int        `json:"items_updated"`
	ItemsFailed    int        `json:"items_failed"`
	ErrorMessage   *string    `json:"error_message,omitempty"`
	StartedAt      time.Time  `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	Metadata       map[string]any `json:"metadata"`
}

// BexioSyncStatus provides a computed overview of sync state.
type BexioSyncStatus struct {
	ContactSyncEnabled     bool       `json:"contact_sync_enabled"`
	InvoicePushEnabled     bool       `json:"invoice_push_enabled"`
	QuotePushEnabled       bool       `json:"quote_push_enabled"`
	PaymentPollEnabled     bool       `json:"payment_poll_enabled"`
	InvoicePullEnabled     bool       `json:"invoice_pull_enabled"`
	LastContactSyncAt      *time.Time `json:"last_contact_sync_at,omitempty"`
	LastPaymentPollAt      *time.Time `json:"last_payment_poll_at,omitempty"`
	LastInvoicePullAt      *time.Time `json:"last_invoice_pull_at,omitempty"`
	TotalContactsMapped    int        `json:"total_contacts_mapped"`
	TotalInvoicesMapped    int        `json:"total_invoices_mapped"`
	TotalQuotesMapped      int        `json:"total_quotes_mapped"`
	LastSyncError          *string    `json:"last_sync_error,omitempty"`
	LastSyncErrorAt        *time.Time `json:"last_sync_error_at,omitempty"`
}
