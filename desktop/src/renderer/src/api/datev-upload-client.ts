/**
 * DATEV upload integration API client.
 *
 * Follows the bexio-client.ts pattern: typed fetch helper with auth
 * header injection and 401 retry. All endpoints target the DATEV gateway
 * routes under /api/v1/finance/datev/*.
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  DatevConnectionStatus,
  DatevUploadConfig,
  DatevUploadLogEntry,
} from './datev-upload-types'

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

class OfflineError extends Error {
  constructor() {
    super('Änderungen sind offline nicht möglich.')
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
