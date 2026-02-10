/**
 * Lightweight fetch wrapper for Calendar API endpoints.
 *
 * The main apiClient (openapi-fetch) is typed against the auto-generated
 * OpenAPI spec. Calendar endpoints are not yet in the spec (backend built
 * in parallel), so this module provides a typed fetch helper that mirrors
 * the same auth header injection and error handling patterns.
 *
 * Once calendar routes are added to openapi.yaml and types regenerated,
 * hooks can migrate to the typed apiClient.
 */
import { API_BASE_URL } from '@/lib/constants'

/** Error thrown when a mutation is attempted while offline. */
class OfflineError extends Error {
  constructor() {
    super(
      'Aenderungen sind offline nicht moeglich. Bitte stellen Sie die Internetverbindung wieder her.',
    )
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
  params?: Record<string, string | number | boolean | string[] | undefined>
}

/**
 * Make an authenticated request to the calendar API.
 * Returns parsed JSON data or throws on error.
 */
async function request<T>(opts: RequestOptions): Promise<T> {
  if (!navigator.onLine && MUTATION_METHODS.has(opts.method)) {
    throw new OfflineError()
  }

  const url = new URL(`${API_BASE_URL}${opts.path}`)

  if (opts.params) {
    for (const [key, value] of Object.entries(opts.params)) {
      if (value === undefined) continue
      if (Array.isArray(value)) {
        for (const v of value) {
          url.searchParams.append(key, v)
        }
      } else {
        url.searchParams.set(key, String(value))
      }
    }
  }

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

  const response = await fetch(url.toString(), init)

  if (!response.ok) {
    // Attempt 401 token refresh once
    if (response.status === 401) {
      const { useAuthStore } = await import('@/stores/auth')
      const store = useAuthStore.getState()
      const newToken = await store.refreshToken()

      if (newToken) {
        headers['Authorization'] = `Bearer ${newToken}`
        const retryResponse = await fetch(url.toString(), {
          ...init,
          headers,
        })

        if (!retryResponse.ok) {
          const errBody = await retryResponse.json().catch(() => ({}))
          throw new Error(
            (errBody as Record<string, string>).error ||
              `Request failed: ${retryResponse.status}`,
          )
        }

        if (retryResponse.status === 204) return {} as T
        return retryResponse.json() as Promise<T>
      }

      // Refresh failed -- force logout
      store.logout()
      throw new Error('Authentication expired')
    }

    const errBody = await response.json().catch(() => ({}))
    throw new Error(
      (errBody as Record<string, string>).error ||
        `Request failed: ${response.status}`,
    )
  }

  if (response.status === 204) return {} as T
  return response.json() as Promise<T>
}

// Convenience methods matching openapi-fetch API style
export const calendarApi = {
  GET: <T>(path: string, params?: RequestOptions['params']) =>
    request<T>({ method: 'GET', path, params }),

  POST: <T>(path: string, body?: unknown) =>
    request<T>({ method: 'POST', path, body }),

  PUT: <T>(path: string, body?: unknown) =>
    request<T>({ method: 'PUT', path, body }),

  DELETE: <T = void>(path: string) =>
    request<T>({ method: 'DELETE', path }),
}
