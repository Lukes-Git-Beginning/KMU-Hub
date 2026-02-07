/**
 * Type-safe API client built on openapi-fetch.
 *
 * Features:
 * - Automatic Authorization header injection from auth store
 * - 401 response interception with transparent token refresh
 * - Concurrent refresh request de-duplication
 * - Offline guard: blocks mutations (POST/PUT/DELETE) when offline
 */
import createClient from 'openapi-fetch'
import type { paths } from './types'
import { API_BASE_URL } from '@/lib/constants'

/**
 * Custom error thrown when a mutation is attempted while offline.
 * The UI can check `error instanceof OfflineError` to show a specific message.
 */
export class OfflineError extends Error {
  constructor() {
    super('Aenderungen sind offline nicht moeglich. Bitte stellen Sie die Internetverbindung wieder her.')
    this.name = 'OfflineError'
  }
}

const apiClient = createClient<paths>({ baseUrl: API_BASE_URL })

/**
 * Shared refresh promise to prevent concurrent token refresh attempts.
 * When multiple requests hit 401 simultaneously, only one refresh is issued.
 */
let refreshPromise: Promise<string | null> | null = null

/** HTTP methods that mutate data and must be blocked when offline. */
const MUTATION_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH'])

apiClient.use({
  async onRequest({ request }) {
    // Block mutations when offline -- allow GETs to proceed (served from cache)
    if (!navigator.onLine && MUTATION_METHODS.has(request.method)) {
      throw new OfflineError()
    }

    // Lazy import to break circular dependency (auth store imports apiClient)
    const { useAuthStore } = await import('@/stores/auth')
    const { accessToken } = useAuthStore.getState()
    if (accessToken) {
      request.headers.set('Authorization', `Bearer ${accessToken}`)
    }
    return request
  },

  async onResponse({ request, response }) {
    if (response.status !== 401) {
      return response
    }

    // Skip refresh for auth endpoints themselves to avoid infinite loops
    const url = new URL(request.url)
    if (
      url.pathname.includes('/auth/login') ||
      url.pathname.includes('/auth/refresh') ||
      url.pathname.includes('/auth/register')
    ) {
      return response
    }

    const { useAuthStore } = await import('@/stores/auth')
    const store = useAuthStore.getState()

    // De-duplicate concurrent refresh requests
    if (!refreshPromise) {
      refreshPromise = store.refreshToken().finally(() => {
        refreshPromise = null
      })
    }

    const newAccessToken = await refreshPromise

    if (!newAccessToken) {
      // Refresh failed -- force logout
      store.logout()
      return response
    }

    // Retry original request with new token
    const retryRequest = new Request(request, {
      headers: new Headers(request.headers),
    })
    retryRequest.headers.set('Authorization', `Bearer ${newAccessToken}`)
    return fetch(retryRequest)
  },
})

export { apiClient }
