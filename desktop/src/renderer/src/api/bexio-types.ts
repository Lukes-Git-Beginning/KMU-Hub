/**
 * TypeScript types for the Bexio integration module.
 *
 * Covers: OAuth connection, sync status, sync logs, field mappings,
 * and sync configuration. Matches the gateway endpoints defined in
 * route_bexio.go.
 */

// ---------------------------------------------------------------------------
// Connection
// ---------------------------------------------------------------------------

export interface BexioConnectionStatus {
  connected: boolean
  org_name?: string
  connected_at?: string
}

// ---------------------------------------------------------------------------
// Sync Status
// ---------------------------------------------------------------------------

export interface BexioSyncStatus {
  contact_sync: BexioEntitySyncStatus
  invoice_sync: BexioEntitySyncStatus
  quote_sync: BexioEntitySyncStatus
  payment_poll: BexioEntitySyncStatus
}

export interface BexioEntitySyncStatus {
  last_sync_at?: string
  items_synced: number
  items_failed: number
  status: 'idle' | 'running' | 'completed' | 'failed'
}

// ---------------------------------------------------------------------------
// Sync Logs
// ---------------------------------------------------------------------------

export interface BexioSyncLogEntry {
  id: string
  sync_type:
    | 'contact_full'
    | 'contact_delta'
    | 'invoice_push'
    | 'quote_push'
    | 'payment_poll'
  status: 'running' | 'completed' | 'failed' | 'partial'
  items_processed: number
  items_created: number
  items_updated: number
  items_failed: number
  error_message?: string
  started_at: string
  completed_at?: string
}

// ---------------------------------------------------------------------------
// Field Mappings
// ---------------------------------------------------------------------------

export interface BexioFieldMappingEntry {
  kmuhub_field: string
  bexio_field: string
  direction: 'inbound' | 'outbound' | 'both'
  required: boolean
}

export type BexioEntityType = 'contact' | 'invoice' | 'quote'

// ---------------------------------------------------------------------------
// Sync Configuration (for setup wizard)
// ---------------------------------------------------------------------------

export interface BexioSyncConfig {
  contact_sync_enabled: boolean
  contact_sync_interval_minutes: number
  invoice_push_enabled: boolean
  quote_push_enabled: boolean
  payment_poll_enabled: boolean
  payment_poll_interval_minutes: number
}

// ---------------------------------------------------------------------------
// Known Bexio fields (for field mapping editor dropdowns)
// ---------------------------------------------------------------------------

export const BEXIO_CONTACT_FIELDS = [
  { id: 'name_1', label: 'Name 1 (Firma/Vorname)', type: 'string' },
  { id: 'name_2', label: 'Name 2 (Nachname)', type: 'string' },
  { id: 'mail', label: 'E-Mail', type: 'string' },
  { id: 'phone_fixed', label: 'Telefon (Festnetz)', type: 'string' },
  { id: 'phone_mobile', label: 'Telefon (Mobil)', type: 'string' },
  { id: 'address', label: 'Adresse', type: 'string' },
  { id: 'city', label: 'Stadt', type: 'string' },
  { id: 'postcode', label: 'PLZ', type: 'string' },
  { id: 'country_id', label: 'Land', type: 'number' },
  { id: 'remarks', label: 'Bemerkungen', type: 'string' },
  { id: 'url', label: 'Website', type: 'string' },
] as const

export const KMUHUB_CONTACT_FIELDS = [
  { id: 'first_name', label: 'Vorname', type: 'string' },
  { id: 'last_name', label: 'Nachname', type: 'string' },
  { id: 'email', label: 'E-Mail', type: 'string' },
  { id: 'phone', label: 'Telefon', type: 'string' },
  { id: 'mobile', label: 'Mobil', type: 'string' },
  { id: 'company_name', label: 'Firma', type: 'string' },
  { id: 'address', label: 'Adresse', type: 'string' },
  { id: 'city', label: 'Stadt', type: 'string' },
  { id: 'zip', label: 'PLZ', type: 'string' },
  { id: 'country', label: 'Land', type: 'string' },
  { id: 'website', label: 'Website', type: 'string' },
  { id: 'notes', label: 'Notizen', type: 'string' },
] as const

// ---------------------------------------------------------------------------
// Default field mappings (used in setup wizard step 3)
// ---------------------------------------------------------------------------

export const DEFAULT_CONTACT_MAPPINGS: BexioFieldMappingEntry[] = [
  { kmuhub_field: 'first_name', bexio_field: 'name_1', direction: 'both', required: true },
  { kmuhub_field: 'last_name', bexio_field: 'name_2', direction: 'both', required: true },
  { kmuhub_field: 'email', bexio_field: 'mail', direction: 'both', required: true },
  { kmuhub_field: 'phone', bexio_field: 'phone_fixed', direction: 'both', required: false },
  { kmuhub_field: 'mobile', bexio_field: 'phone_mobile', direction: 'both', required: false },
  { kmuhub_field: 'address', bexio_field: 'address', direction: 'both', required: false },
  { kmuhub_field: 'city', bexio_field: 'city', direction: 'both', required: false },
  { kmuhub_field: 'zip', bexio_field: 'postcode', direction: 'both', required: false },
  { kmuhub_field: 'country', bexio_field: 'country_id', direction: 'both', required: false },
  { kmuhub_field: 'website', bexio_field: 'url', direction: 'both', required: false },
]
