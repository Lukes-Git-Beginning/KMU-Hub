/**
 * Bexio integration API client.
 *
 * Follows the integration-client.ts pattern: typed fetch helper with auth
 * header injection and 401 retry. All endpoints target the Bexio gateway
 * routes under /api/v1/integrations/bexio/*.
 */
import type {
  BexioConnectionStatus,
  BexioSyncStatus,
  BexioSyncConfig,
  BexioSyncLogEntry,
  BexioFieldMappingEntry,
  BexioEntityType,
} from './bexio-types'
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

export function getAuthorizationURL() {
  return request<{ authorization_url: string }>({
    method: 'GET',
    path: '/api/v1/integrations/bexio/oauth/authorize',
  })
}

export function disconnect() {
  return request<void>({
    method: 'POST',
    path: '/api/v1/integrations/bexio/disconnect',
  })
}

export function getConnectionStatus() {
  return request<BexioConnectionStatus>({
    method: 'GET',
    path: '/api/v1/integrations/bexio/status',
  })
}

// ---------------------------------------------------------------------------
// Sync
// ---------------------------------------------------------------------------

export function triggerSync(syncType?: string) {
  return request<{ sync_id: string }>({
    method: 'POST',
    path: '/api/v1/integrations/bexio/sync/trigger',
    body: syncType ? { sync_type: syncType } : undefined,
  })
}

export function getSyncStatus() {
  return request<BexioSyncStatus>({
    method: 'GET',
    path: '/api/v1/integrations/bexio/sync/status',
  })
}

export function listSyncLogs(limit?: number) {
  const params = limit ? `?limit=${limit}` : ''
  return request<BexioSyncLogEntry[]>({
    method: 'GET',
    path: `/api/v1/integrations/bexio/sync/logs${params}`,
  })
}

export function updateSyncConfig(config: BexioSyncConfig) {
  return request<void>({
    method: 'PUT',
    path: '/api/v1/integrations/bexio/sync/config',
    body: config,
  })
}

// ---------------------------------------------------------------------------
// Field Mappings
// ---------------------------------------------------------------------------

export function getFieldMappings(entityType: BexioEntityType) {
  return request<{ mappings: BexioFieldMappingEntry[] }>({
    method: 'GET',
    path: `/api/v1/integrations/bexio/mappings/${entityType}`,
  })
}

export function updateFieldMappings(
  entityType: BexioEntityType,
  mappings: BexioFieldMappingEntry[],
) {
  return request<void>({
    method: 'PUT',
    path: `/api/v1/integrations/bexio/mappings/${entityType}`,
    body: { mappings },
  })
}

// ---------------------------------------------------------------------------
// Manual Push
// ---------------------------------------------------------------------------

export function pushInvoice(invoiceId: string) {
  return request<{ bexio_id: number }>({
    method: 'POST',
    path: `/api/v1/integrations/bexio/push/invoice/${invoiceId}`,
  })
}

export function pushQuote(quoteId: string) {
  return request<{ bexio_id: number }>({
    method: 'POST',
    path: `/api/v1/integrations/bexio/push/quote/${quoteId}`,
  })
}
