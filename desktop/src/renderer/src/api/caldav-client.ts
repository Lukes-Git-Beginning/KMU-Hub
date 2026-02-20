/**
 * CalDAV/CardDAV settings API client.
 *
 * Fetch wrapper for app-specific password management and admin CalDAV settings.
 * Follows the calendar-client.ts pattern: typed fetch with auth header injection
 * and 401 retry.
 *
 * Gateway routes: /api/v1/caldav/* and /api/v1/admin/caldav/*
 */
import { API_BASE_URL } from '@/lib/constants'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface AppPassword {
  id: string
  label: string
  password_prefix: string
  last_used_at: string | null
  created_at: string
  revoked_at: string | null
}

export interface CreatePasswordResponse {
  password: string
  id: string
  label: string
  prefix: string
}

export interface CalDAVStatus {
  org_enabled: boolean
  user_enabled: boolean
  password_count: number
  user_id: string
  caldav_url: string
  carddav_url: string
}

export interface AdminCalDAVSettings {
  enabled: boolean
}

export interface AdminCalDAVUser {
  id: string
  email: string
  first_name: string
  last_name: string
  caldav_enabled: boolean
  password_count: number
  last_used: string | null
}

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
// User API
// ---------------------------------------------------------------------------

export function getAppPasswords() {
  return request<{ passwords: AppPassword[] }>({
    method: 'GET',
    path: '/api/v1/caldav/passwords',
  })
}

export function createAppPassword(label: string) {
  return request<CreatePasswordResponse>({
    method: 'POST',
    path: '/api/v1/caldav/passwords',
    body: { label },
  })
}

export function revokeAppPassword(id: string) {
  return request<{ status: string }>({
    method: 'DELETE',
    path: `/api/v1/caldav/passwords/${id}`,
  })
}

export function getCalDAVStatus() {
  return request<CalDAVStatus>({
    method: 'GET',
    path: '/api/v1/caldav/status',
  })
}

export function enableCalDAV() {
  return request<{ status: string }>({
    method: 'PUT',
    path: '/api/v1/caldav/enable',
  })
}

export function disableCalDAV() {
  return request<{ status: string }>({
    method: 'PUT',
    path: '/api/v1/caldav/disable',
  })
}

// ---------------------------------------------------------------------------
// Admin API
// ---------------------------------------------------------------------------

export function getAdminCalDAVSettings() {
  return request<AdminCalDAVSettings>({
    method: 'GET',
    path: '/api/v1/admin/caldav/settings',
  })
}

export function setAdminCalDAVSettings(enabled: boolean) {
  return request<AdminCalDAVSettings>({
    method: 'PUT',
    path: '/api/v1/admin/caldav/settings',
    body: { enabled },
  })
}

export function getAdminCalDAVUsers() {
  return request<{ users: AdminCalDAVUser[] }>({
    method: 'GET',
    path: '/api/v1/admin/caldav/users',
  })
}

export function adminRevokeUserPasswords(userId: string) {
  return request<{ revoked_count: number }>({
    method: 'DELETE',
    path: `/api/v1/admin/caldav/users/${userId}/passwords`,
  })
}
