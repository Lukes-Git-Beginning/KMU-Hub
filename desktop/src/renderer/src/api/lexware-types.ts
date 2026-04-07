/**
 * TypeScript types for the Lexware integration module.
 *
 * Covers: API key connection, sync status, sync logs, field mappings.
 * Matches the gateway endpoints defined in route_lexware.go.
 */
import i18next from 'i18next'

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

const LEXWARE_CONTACT_FIELD_KEYS = [
  { value: 'person.firstName', labelKey: 'config.lexwareFields.firstNamePerson' },
  { value: 'person.lastName', labelKey: 'config.lexwareFields.lastNamePerson' },
  { value: 'company.name', labelKey: 'config.lexwareFields.companyName' },
  { value: 'emailAddresses.business', labelKey: 'config.lexwareFields.emailBusiness' },
  { value: 'phoneNumbers.business', labelKey: 'config.lexwareFields.phoneBusiness' },
  { value: 'phoneNumbers.mobile', labelKey: 'config.lexwareFields.mobile' },
  { value: 'addresses.billing.street', labelKey: 'config.lexwareFields.billingStreet' },
  { value: 'addresses.billing.city', labelKey: 'config.lexwareFields.billingCity' },
  { value: 'addresses.billing.zip', labelKey: 'config.lexwareFields.billingZip' },
  { value: 'addresses.billing.countryCode', labelKey: 'config.lexwareFields.billingCountry' },
  { value: 'note', labelKey: 'config.lexwareFields.note' },
] as const

/** Returns Lexware contact fields with translated labels. Call at render time, not at module load. */
export function getLexwareContactFields() {
  return LEXWARE_CONTACT_FIELD_KEYS.map((f) => ({ value: f.value, label: i18next.t(f.labelKey) }))
}

/** @deprecated Use getLexwareContactFields() for translated labels. */
export const LEXWARE_CONTACT_FIELDS = LEXWARE_CONTACT_FIELD_KEYS

const KMUHUB_CONTACT_FIELD_KEYS = [
  { value: 'first_name', labelKey: 'config.lexwareFields.cosmiFirstName' },
  { value: 'last_name', labelKey: 'config.lexwareFields.cosmiLastName' },
  { value: 'email', labelKey: 'config.lexwareFields.cosmiEmail' },
  { value: 'phone', labelKey: 'config.lexwareFields.cosmiPhone' },
  { value: 'company_name', labelKey: 'config.lexwareFields.cosmiCompany' },
  { value: 'notes', labelKey: 'config.lexwareFields.cosmiNotes' },
] as const

/** Returns Cosmi contact fields with translated labels. Call at render time, not at module load. */
export function getKmuhubContactFields() {
  return KMUHUB_CONTACT_FIELD_KEYS.map((f) => ({ value: f.value, label: i18next.t(f.labelKey) }))
}

/** @deprecated Use getKmuhubContactFields() for translated labels. */
export const KMUHUB_CONTACT_FIELDS = KMUHUB_CONTACT_FIELD_KEYS

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
