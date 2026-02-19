/**
 * CRM module TypeScript types for import/export and visibility.
 *
 * Manually defined types for the Phase 10 contact import/export API entities.
 * These types mirror the backend email/contact service models
 * (import, export, visibility).
 */

// ---------------------------------------------------------------------------
// Visibility types
// ---------------------------------------------------------------------------

export type ContactVisibility = 'shared' | 'personal'

// ---------------------------------------------------------------------------
// Import types
// ---------------------------------------------------------------------------

export interface ImportPreview {
  columns: string[]
  sample_rows: string[][]
  detected_mapping: Record<string, string>
}

export interface ImportError {
  row: number
  reason: string
}

export interface ImportResult {
  imported_count: number
  merged_count: number
  skipped_count: number
  errors?: ImportError[]
}

// ---------------------------------------------------------------------------
// Export types
// ---------------------------------------------------------------------------

export type ExportFormat = 'csv' | 'vcard'

/** Available CRM fields for CSV export. */
export const EXPORT_FIELDS = [
  { key: 'first_name', label: 'Vorname' },
  { key: 'last_name', label: 'Nachname' },
  { key: 'email', label: 'E-Mail' },
  { key: 'phone', label: 'Telefon' },
  { key: 'company', label: 'Firma' },
  { key: 'position', label: 'Position' },
  { key: 'notes', label: 'Notizen' },
] as const

/** CRM fields available for CSV field mapping during import. */
export const IMPORT_CRM_FIELDS = [
  { key: 'first_name', label: 'Vorname' },
  { key: 'last_name', label: 'Nachname' },
  { key: 'email', label: 'E-Mail' },
  { key: 'phone', label: 'Telefon' },
  { key: 'company', label: 'Firma' },
  { key: 'position', label: 'Position' },
  { key: 'notes', label: 'Notizen' },
  { key: '__ignore', label: '-- Ignorieren' },
] as const
