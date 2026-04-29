/**
 * DATEV upload integration API client.
 *
 * Follows the bexio-client.ts pattern: typed fetch helper with auth
 * header injection and 401 retry. All endpoints target the DATEV gateway
 * routes under /api/v1/finance/datev/*.
 */
import type {
  DatevConnectionStatus,
  DatevUploadConfig,
  DatevUploadLogEntry,
} from './datev-upload-types'
import { authenticatedRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

function request<T>(opts: { method: string; path: string; body?: unknown }): Promise<T> {
  return authenticatedRequest<T>(opts)
}

// ---------------------------------------------------------------------------
// OAuth
// ---------------------------------------------------------------------------

export function getAuthorizationURL(redirectUrl: string) {
  return request<{ authorization_url: string }>({
    method: 'GET',
    path: `/api/v1/finance/datev/oauth/authorize?redirect_url=${encodeURIComponent(redirectUrl)}`,
  })
}

export function disconnect() {
  return request<{ success: boolean }>({
    method: 'POST',
    path: '/api/v1/finance/datev/disconnect',
  })
}

export function getConnectionStatus() {
  return request<DatevConnectionStatus>({
    method: 'GET',
    path: '/api/v1/finance/datev/status',
  })
}

// ---------------------------------------------------------------------------
// Upload
// ---------------------------------------------------------------------------

export function uploadBuchungsstapel(startDate: string, endDate: string) {
  return request<{ success: boolean; upload_id?: string; error_message?: string }>({
    method: 'POST',
    path: '/api/v1/finance/datev/upload',
    body: { start_date: startDate, end_date: endDate },
  })
}

export function uploadBeleg(invoiceId: string) {
  return request<{ success: boolean; error_message?: string }>({
    method: 'POST',
    path: `/api/v1/finance/datev/upload/beleg/${invoiceId}`,
  })
}

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

export function getUploadConfig() {
  return request<DatevUploadConfig>({
    method: 'GET',
    path: '/api/v1/finance/datev/config',
  })
}

export function updateUploadConfig(config: Partial<DatevUploadConfig>) {
  return request<{ success: boolean }>({
    method: 'PUT',
    path: '/api/v1/finance/datev/config',
    body: config,
  })
}

// ---------------------------------------------------------------------------
// Logs
// ---------------------------------------------------------------------------

export function listUploadLogs(limit?: number) {
  const params = limit ? `?limit=${limit}` : ''
  return request<DatevUploadLogEntry[]>({
    method: 'GET',
    path: `/api/v1/finance/datev/upload/logs${params}`,
  })
}
