/**
 * Bexio integration API client.
 *
 * Follows the integration-client.ts pattern: typed fetch helper with auth
 * header injection and 401 retry. All endpoints target the Bexio gateway
 * routes under /api/v1/integrations/bexio/*.
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  BexioConnectionStatus,
  BexioSyncStatus,
  BexioSyncLogEntry,
  BexioFieldMappingEntry,
  BexioEntityType,
} from './bexio-types'

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
// OAuth
// ---------------------------------------------------------------------------

export function getAuthorizationURL() {
  return request<{ authorization_url: string }>({
    method: 'GET',
    path: '/api/v1/integrations/bexio/auth-url',
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
    path: '/api/v1/integrations/bexio/sync',
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
  return request<{ logs: BexioSyncLogEntry[] }>({
    method: 'GET',
    path: `/api/v1/integrations/bexio/sync/logs${params}`,
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
