/**
 * Notification API client for quiet hours, DND, and muting.
 *
 * Complements the openapi-fetch based hooks in useNotifications.ts
 * for endpoints not yet in the openapi spec.
 */
import { authenticatedRequest } from './utils/authenticatedFetch'

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
// Internal helpers
// ---------------------------------------------------------------------------

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const method = (options.method ?? 'GET').toUpperCase()
  let body: unknown = undefined
  if (options.body !== undefined) {
    if (typeof options.body === 'string') {
      try { body = JSON.parse(options.body) } catch { body = options.body }
    } else {
      body = options.body
    }
  }
  return authenticatedRequest<T>({ method, path, body })
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
