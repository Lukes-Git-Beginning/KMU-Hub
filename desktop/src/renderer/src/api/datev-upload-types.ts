/**
 * TypeScript types for the DATEV upload integration module.
 *
 * Covers: OAuth connection, upload configuration, and upload logs.
 * Matches the gateway endpoints defined in route_datev.go.
 */

// ---------------------------------------------------------------------------
// Connection
// ---------------------------------------------------------------------------

export interface DatevConnectionStatus {
  connected: boolean
  connected_at?: string
}

// ---------------------------------------------------------------------------
// Upload Configuration
// ---------------------------------------------------------------------------

export interface DatevUploadConfig {
  client_number: string
  auto_upload_enabled: boolean
  upload_after_export: boolean
}

// ---------------------------------------------------------------------------
// Upload Logs
// ---------------------------------------------------------------------------

export interface DatevUploadLogEntry {
  id: string
  upload_type: 'buchungsstapel' | 'belegbild'
  status: 'pending' | 'uploading' | 'completed' | 'failed'
  file_size: number
  document_count: number
  error_message?: string
  started_at: string
  completed_at?: string
}
