/**
 * TypeScript types for the Lexware integration module.
 *
 * Covers: API key connection, sync status, sync logs, field mappings.
 * Matches the gateway endpoints defined in route_lexware.go.
 */

// ---------------------------------------------------------------------------
// Connection
// ---------------------------------------------------------------------------

export interface LexwareConnectionStatus {
  connected: boolean
  connected_at?: string
}

// ---------------------------------------------------------------------------
// Sync Status
// ---------------------------------------------------------------------------

export interface LexwareSyncStatus {
  contact_sync: LexwareEntitySyncStatus
  invoice_sync: LexwareEntitySyncStatus
  quote_sync: LexwareEntitySyncStatus
}

export interface LexwareEntitySyncStatus {
  last_sync_at?: string
  items_synced: number
  items_failed: number
  status: 'idle' | 'running' | 'completed' | 'failed'
}

// ---------------------------------------------------------------------------
// Sync Logs
// ---------------------------------------------------------------------------

export interface LexwareSyncLogEntry {
  id: string
  sync_type:
    | 'contact_full'
    | 'contact_delta'
    | 'invoice_push'
    | 'quote_push'
    | 'credit_note_push'
    | 'webhook_event'
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

export interface LexwareFieldMappingEntry {
  kmuhub_field: string
  lexware_field: string
  direction: 'inbound' | 'outbound' | 'both'
  required: boolean
}

// ---------------------------------------------------------------------------
// Known Lexware fields (nested) for field mapping editor dropdowns
// ---------------------------------------------------------------------------

export const LEXWARE_CONTACT_FIELDS = [
  { value: 'person.firstName', label: 'Vorname (Person)' },
  { value: 'person.lastName', label: 'Nachname (Person)' },
  { value: 'company.name', label: 'Firmenname' },
  { value: 'emailAddresses.business', label: 'E-Mail (Business)' },
  { value: 'phoneNumbers.business', label: 'Telefon (Business)' },
  { value: 'phoneNumbers.mobile', label: 'Mobil' },
  { value: 'addresses.billing.street', label: 'Strasse (Rechnung)' },
  { value: 'addresses.billing.city', label: 'Stadt (Rechnung)' },
  { value: 'addresses.billing.zip', label: 'PLZ (Rechnung)' },
  { value: 'addresses.billing.countryCode', label: 'Land (Rechnung)' },
  { value: 'note', label: 'Notiz' },
] as const

export const KMUHUB_CONTACT_FIELDS = [
  { value: 'first_name', label: 'Vorname' },
  { value: 'last_name', label: 'Nachname' },
  { value: 'email', label: 'E-Mail' },
  { value: 'phone', label: 'Telefon' },
  { value: 'company_name', label: 'Firma' },
  { value: 'notes', label: 'Notizen' },
] as const

// ---------------------------------------------------------------------------
// Default field mappings (used in setup wizard)
// ---------------------------------------------------------------------------

export const DEFAULT_CONTACT_MAPPINGS: LexwareFieldMappingEntry[] = [
  { kmuhub_field: 'first_name', lexware_field: 'person.firstName', direction: 'both', required: true },
  { kmuhub_field: 'last_name', lexware_field: 'person.lastName', direction: 'both', required: false },
  { kmuhub_field: 'email', lexware_field: 'emailAddresses.business', direction: 'both', required: false },
  { kmuhub_field: 'phone', lexware_field: 'phoneNumbers.business', direction: 'both', required: false },
  { kmuhub_field: 'notes', lexware_field: 'note', direction: 'both', required: false },
]
