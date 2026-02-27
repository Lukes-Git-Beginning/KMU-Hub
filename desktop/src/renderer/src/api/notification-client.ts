/**
 * Notification API client for quiet hours, DND, and muting.
 *
 * Complements the openapi-fetch based hooks in useNotifications.ts
 * for endpoints not yet in the openapi spec.
 */
import { API_BASE_URL } from '@/lib/constants'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface QuietHours {
  start_time: string   // "22:00"
  end_time: string     // "07:00"
  days: number[]       // 0=Sun..6=Sat
  timezone: string
  is_active: boolean
}

export interface DNDStatus {
  is_active: boolean
  expires_at?: string  // ISO
}

export interface MutedResource {
  id: string
  resource_type: string  // 'conversation' | 'channel' | 'thread'
  resource_id: string
  resource_label?: string
  muted_at: string
}

// ---------------------------------------------------------------------------
// Internal helpers (same pattern as finance-client)
// ---------------------------------------------------------------------------

const MUTATION_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH'])

let refreshPromise: Promise<string | null> | null = null

async function getToken(): Promise<string | undefined> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken
}

async function refreshTokenFn(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  const store = useAuthStore.getState()
  if (!refreshPromise) {
    refreshPromise = store.refreshToken().finally(() => {
      refreshPromise = null
    })
  }
  return refreshPromise
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const method = options.method ?? 'GET'
  if (!navigator.onLine && MUTATION_METHODS.has(method)) {
    throw new Error('Aenderungen sind offline nicht moeglich.')
  }

  const token = await getToken()
  const headers = new Headers(options.headers)
  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (options.body && typeof options.body === 'string') {
    headers.set('Content-Type', 'application/json')
  }

  let res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers })

  if (res.status === 401 && !path.includes('/auth/')) {
    const newToken = await refreshTokenFn()
    if (!newToken) {
      const { useAuthStore } = await import('@/stores/auth')
      useAuthStore.getState().logout()
      throw new Error('Session abgelaufen')
    }
    headers.set('Authorization', `Bearer ${newToken}`)
    res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers })
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error ?? body.message ?? `HTTP ${res.status}`)
  }

  if (res.status === 204) return {} as T
  return res.json() as Promise<T>
}

// ---------------------------------------------------------------------------
// Quiet Hours API
// ---------------------------------------------------------------------------

export const quietHoursApi = {
  get() {
    return request<{ quiet_hours: QuietHours }>('/api/v1/notifications/quiet-hours')
  },
  update(data: Partial<QuietHours>) {
    return request<{ quiet_hours: QuietHours }>('/api/v1/notifications/quiet-hours', {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  },
}

// ---------------------------------------------------------------------------
// DND API
// ---------------------------------------------------------------------------

export const dndApi = {
  getStatus() {
    return request<DNDStatus>('/api/v1/notifications/dnd')
  },
  enable(expiresAt?: string) {
    return request<DNDStatus>('/api/v1/notifications/dnd', {
      method: 'POST',
      body: JSON.stringify({ expires_at: expiresAt }),
    })
  },
  disable() {
    return request<Record<string, never>>('/api/v1/notifications/dnd', {
      method: 'DELETE',
    })
  },
}

// ---------------------------------------------------------------------------
// Muting API
// ---------------------------------------------------------------------------

export const mutingApi = {
  list() {
    return request<{ mutes: MutedResource[] }>('/api/v1/notifications/mutes')
  },
  mute(resourceType: string, resourceId: string) {
    return request<{ mute: MutedResource }>('/api/v1/notifications/mutes', {
      method: 'POST',
      body: JSON.stringify({ resource_type: resourceType, resource_id: resourceId }),
    })
  },
  unmute(muteId: string) {
    return request<Record<string, never>>(`/api/v1/notifications/mutes/${muteId}`, {
      method: 'DELETE',
    })
  },
}
