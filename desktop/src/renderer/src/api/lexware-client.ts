/**
 * Lexware integration API client.
 *
 * Follows the bexio-client.ts pattern: typed fetch helper with auth
 * header injection and 401 retry. All endpoints target the Lexware gateway
 * routes under /api/v1/integrations/lexware/*.
 */
import type {
  LexwareConnectionStatus,
  LexwareSyncStatus,
  LexwareSyncLogEntry,
  LexwareFieldMappingEntry,
} from './lexware-types'
import { authenticatedRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

function request<T>(opts: { method: string; path: string; body?: unknown }): Promise<T> {
  return authenticatedRequest<T>(opts)
}

// ---------------------------------------------------------------------------
// Connect / Disconnect
// ---------------------------------------------------------------------------

export function connect(apiKey: string) {
  return request<{ success: boolean; error_message?: string }>({
    method: 'POST',
    path: '/api/v1/integrations/lexware/connect',
    body: { api_key: apiKey },
  })
}

export function disconnect() {
  return request<{ success: boolean }>({
    method: 'POST',
    path: '/api/v1/integrations/lexware/disconnect',
  })
}

export function getConnectionStatus() {
  return request<LexwareConnectionStatus>({
    method: 'GET',
    path: '/api/v1/integrations/lexware/status',
  })
}

export function testConnection() {
  return request<{ success: boolean; error_message?: string }>({
    method: 'POST',
    path: '/api/v1/integrations/lexware/test',
  })
}

// ---------------------------------------------------------------------------
// Sync
// ---------------------------------------------------------------------------

export function triggerSync(syncType?: string) {
  return request<{ sync_id: string; status: string }>({
    method: 'POST',
    path: '/api/v1/integrations/lexware/sync/trigger',
    body: syncType ? { sync_type: syncType } : undefined,
  })
}

export function getSyncStatus() {
  return request<LexwareSyncStatus>({
    method: 'GET',
    path: '/api/v1/integrations/lexware/sync/status',
  })
}

export function listSyncLogs(limit?: number) {
  const params = limit ? `?limit=${limit}` : ''
  return request<LexwareSyncLogEntry[]>({
    method: 'GET',
    path: `/api/v1/integrations/lexware/sync/logs${params}`,
  })
}

// ---------------------------------------------------------------------------
// Field Mappings
// ---------------------------------------------------------------------------

export function getFieldMappings(entityType: string) {
  return request<LexwareFieldMappingEntry[]>({
    method: 'GET',
    path: `/api/v1/integrations/lexware/mappings/${entityType}`,
  })
}

export function updateFieldMappings(
  entityType: string,
  mappings: LexwareFieldMappingEntry[],
) {
  return request<{ success: boolean }>({
    method: 'PUT',
    path: `/api/v1/integrations/lexware/mappings/${entityType}`,
    body: { mappings },
  })
}

// ---------------------------------------------------------------------------
// Manual Push
// ---------------------------------------------------------------------------

export function pushInvoice(invoiceId: string) {
  return request<{ success: boolean; lexware_id?: string; error_message?: string }>({
    method: 'POST',
    path: `/api/v1/integrations/lexware/push/invoice/${invoiceId}`,
  })
}

export function pushQuote(quoteId: string) {
  return request<{ success: boolean; lexware_id?: string; error_message?: string }>({
    method: 'POST',
    path: `/api/v1/integrations/lexware/push/quote/${quoteId}`,
  })
}
