/**
 * Type-safe API client built on openapi-fetch.
 *
 * Features:
 * - Automatic Authorization header injection from auth store
 * - 401 response interception with transparent token refresh
 * - Concurrent refresh request de-duplication
 * - Offline mutations: enqueued in IndexedDB instead of throwing OfflineError
 * - Idempotency-Key header auto-injected for all mutations
 */
import createClient from 'openapi-fetch'
import type { paths } from './types'
import { API_BASE_URL } from '@/lib/constants'
import { enqueue } from './offline-queue'
import { withIdempotencyKey } from './idempotency'

/**
 * Custom error thrown when a mutation is attempted while offline and
 * enqueueing is explicitly disabled (non-standard use). Kept for
 * backwards-compatibility with existing error-boundary checks.
 */
export class OfflineError extends Error {
  constructor() {
    super('Änderungen sind offline nicht möglich. Bitte stellen Sie die Internetverbindung wieder her.')
    this.name = 'OfflineError'
  }
}

/**
 * Returned in place of a real API response when a mutation was enqueued
 * for later replay (device offline at call time).
 */
export interface QueuedResponse {
  queued: true
  idempotencyKey: string
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
    // Offline handling for mutations: enqueue instead of throwing.
    if (!navigator.onLine && MUTATION_METHODS.has(request.method)) {
      const url = new URL(request.url)
      const bodyText = await request.text()
      const headers: Record<string, string> = {}
      request.headers.forEach((value, key) => {
        if (key.toLowerCase() !== 'authorization') {
          headers[key] = value
        }
      })

      // Ensure idempotency key is stable for offline items.
      const idemHeaders = withIdempotencyKey(headers)
      const idempotencyKey = idemHeaders.get('Idempotency-Key')!
      headers['Idempotency-Key'] = idempotencyKey

      await enqueue(request.method, url.pathname + url.search, bodyText || null, headers, idempotencyKey)

      // Return a synthetic 202 response; caller can check data.queued === true.
      const queuedBody: QueuedResponse = { queued: true, idempotencyKey }
      return new Response(JSON.stringify(queuedBody), {
        status: 202,
        headers: { 'Content-Type': 'application/json', 'X-Cosmi-Queued': 'true' },
      })
    }

    // Inject Idempotency-Key for online mutations.
    if (MUTATION_METHODS.has(request.method)) {
      const updatedHeaders = withIdempotencyKey(request.headers)
      updatedHeaders.forEach((value, key) => request.headers.set(key, value))
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
