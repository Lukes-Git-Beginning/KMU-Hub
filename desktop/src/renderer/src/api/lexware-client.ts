/**
 * Lexware integration API client.
 *
 * Follows the bexio-client.ts pattern: typed fetch helper with auth
 * header injection and 401 retry. All endpoints target the Lexware gateway
 * routes under /api/v1/integrations/lexware/*.
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  LexwareConnectionStatus,
  LexwareSyncStatus,
  LexwareSyncLogEntry,
  LexwareFieldMappingEntry,
} from './lexware-types'

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

class OfflineError extends Error {
  constructor() {
    super('Aenderungen sind offline nicht moeglich.')
    this.name = 'OfflineError'
  }
}

const MUTATION_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH'])

async function getAuthToken(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken
}

interface RequestOptions {
  method: string
  path: string
  body?: unknown
}

async function request<T>(opts: RequestOptions): Promise<T> {
  if (!navigator.onLine && MUTATION_METHODS.has(opts.method)) {
    throw new OfflineError()
  }

  const url = `${API_BASE_URL}${opts.path}`
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  const token = await getAuthToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const init: RequestInit = {
    method: opts.method,
    headers,
  }

  if (opts.body !== undefined) {
    init.body = JSON.stringify(opts.body)
  }

  const res = await fetch(url, init)

  if (!res.ok) {
    if (res.status === 401) {
      const { useAuthStore } = await import('@/stores/auth')
      const store = useAuthStore.getState()
      const newToken = await store.refreshToken()

      if (newToken) {
        headers['Authorization'] = `Bearer ${newToken}`
        const retryRes = await fetch(url, { ...init, headers })
        if (!retryRes.ok) {
          const err = await retryRes.json().catch(() => ({}))
          throw new Error(
            (err as Record<string, string>).error ||
              `Request failed: ${retryRes.status}`,
          )
        }
        if (retryRes.status === 204) return {} as T
        return retryRes.json() as Promise<T>
      }

      store.logout()
      throw new Error('Authentication expired')
    }

    const errBody = await res.json().catch(() => ({}))
    throw new Error(
      (errBody as Record<string, string>).error ||
        `Request failed: ${res.status}`,
    )
  }

  if (res.status === 204) return {} as T
  return res.json() as Promise<T>
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
