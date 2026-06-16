/**
 * Shared authenticated fetch helper for custom API clients.
 *
 * Centralises the repetitive auth + idempotency + offline-guard boilerplate
 * that was duplicated across all 29 *-client.ts modules. Each module keeps
 * its own domain types and convenience wrappers, but delegates the actual
 * HTTP call to `authenticatedRequest`.
 *
 * Features
 * ─────────
 * - Injects `Authorization: Bearer <token>` via the auth store.
 * - Adds `Idempotency-Key` header for every POST / PUT / PATCH / DELETE
 *   request (idempotent GETs must not consume keys).
 * - Attempts a single transparent token refresh on HTTP 401.
 * - Throws `OfflineError` when the device is offline and a mutation is
 *   attempted (callers that want to enqueue instead must handle this).
 *
 * Body handling
 * ─────────────
 * Pass a plain JS object for JSON bodies — this helper sets Content-Type
 * automatically. For multipart (FormData / Blob), pass the value directly;
 * Content-Type is intentionally omitted so the browser sets the correct
 * boundary.
 */

import { generateIdempotencyKey } from '../idempotency'
import { API_BASE_URL } from '@/lib/constants'

// ---------------------------------------------------------------------------
// Shared error type
// ---------------------------------------------------------------------------

/** Thrown when a mutation is attempted while the device is offline. */
export class OfflineError extends Error {
  constructor() {
    super(
      'Änderungen sind offline nicht möglich. Bitte stellen Sie die Internetverbindung wieder her.',
    )
    this.name = 'OfflineError'
  }
}

// ---------------------------------------------------------------------------
// HTTP method classification
// ---------------------------------------------------------------------------

const MUTATION_METHODS = new Set(['POST', 'PUT', 'PATCH', 'DELETE'])

// ---------------------------------------------------------------------------
// Auth helpers (lazy-imported to break circular dependency)
// ---------------------------------------------------------------------------

async function getToken(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken
}

async function doTokenRefresh(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().refreshToken()
}

async function doLogout(): Promise<void> {
  const { useAuthStore } = await import('@/stores/auth')
  useAuthStore.getState().logout()
}

// ---------------------------------------------------------------------------
// Request options
// ---------------------------------------------------------------------------

export type QueryValue = string | number | boolean | null | undefined

export interface AuthenticatedRequestOptions {
  /** HTTP method (GET, POST, PUT, PATCH, DELETE, …). */
  method: string
  /**
   * Path portion of the URL, e.g. `/api/v1/contacts`.
   * `API_BASE_URL` is prepended automatically unless the path starts with
   * `http://` or `https://`.
   */
  path: string
  /** Query-string parameters. Arrays are appended as repeated params. */
  params?: Record<string, QueryValue | QueryValue[]>
  /**
   * Request body.
   * - Plain object / array → serialised to JSON, Content-Type is set.
   * - `FormData` / `Blob` / `string` → passed through as-is, Content-Type
   *   is NOT set (browser or caller is responsible).
   * - `undefined` → no body.
   */
  body?: unknown
}

// ---------------------------------------------------------------------------
// Core implementation
// ---------------------------------------------------------------------------

/**
 * Makes an authenticated HTTP request, injecting auth + idempotency headers
 * and handling 401 refresh transparently.
 *
 * @throws {OfflineError} when offline and `method` is a mutation.
 * @throws {Error} on non-2xx responses.
 */
export async function authenticatedRequest<T>(
  opts: AuthenticatedRequestOptions,
): Promise<T> {
  const method = opts.method.toUpperCase()

  if (!navigator.onLine && MUTATION_METHODS.has(method)) {
    throw new OfflineError()
  }

  // Build URL
  const base = opts.path.startsWith('http') ? '' : API_BASE_URL
  const url = new URL(`${base}${opts.path}`)

  if (opts.params) {
    for (const [key, value] of Object.entries(opts.params)) {
      if (value === undefined || value === null) continue
      if (Array.isArray(value)) {
        for (const v of value) {
          if (v !== undefined && v !== null) {
            url.searchParams.append(key, String(v))
          }
        }
      } else {
        url.searchParams.set(key, String(value))
      }
    }
  }

  // Build headers
  const headers: Record<string, string> = {}

  // Set Content-Type only for plain JSON bodies (not FormData / Blob / string)
  const isJsonBody =
    opts.body !== undefined &&
    !(opts.body instanceof FormData) &&
    !(opts.body instanceof Blob) &&
    typeof opts.body !== 'string'

  if (isJsonBody) {
    headers['Content-Type'] = 'application/json'
  }

  // Inject auth token
  const token = await getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  // Inject idempotency key for mutations. Set it directly on the existing
  // headers object — do NOT round-trip through a `Headers` instance, which
  // lowercases keys and would leave BOTH `Authorization` and `authorization`
  // in the object. fetch then merges those case-only duplicates into a single
  // comma-joined value ("Bearer x, Bearer x"), which the backend rejects (401).
  if (MUTATION_METHODS.has(method) && !headers['Idempotency-Key']) {
    headers['Idempotency-Key'] = generateIdempotencyKey()
  }

  // Build request init
  const init: RequestInit = { method, headers }
  if (opts.body !== undefined) {
    init.body = isJsonBody ? JSON.stringify(opts.body) : (opts.body as BodyInit)
  }

  // First attempt
  const urlStr = url.toString()
  let response = await fetch(urlStr, init)

  // Single transparent 401 refresh
  if (response.status === 401) {
    const newToken = await doTokenRefresh()
    if (!newToken) {
      await doLogout()
      throw new Error('Authentication expired')
    }
    headers['Authorization'] = `Bearer ${newToken}`
    response = await fetch(urlStr, { ...init, headers })
  }

  if (!response.ok) {
    const errBody = await response.json().catch(() => ({}))
    throw new Error(
      (errBody as Record<string, string>).error ||
        `Request failed: ${response.status}`,
    )
  }

  if (response.status === 204) return {} as T
  return response.json() as Promise<T>
}

/**
 * Variant for endpoints that return a `Blob` response (e.g. file downloads,
 * PDF exports). Idempotency key is still injected for POST/PUT mutations.
 *
 * Returns the raw `Response` so callers can inspect headers (Content-Disposition,
 * Content-Type, etc.) before extracting the blob.
 */
export async function authenticatedBlobRequest(
  opts: AuthenticatedRequestOptions,
): Promise<Response> {
  const method = opts.method.toUpperCase()

  if (!navigator.onLine && MUTATION_METHODS.has(method)) {
    throw new OfflineError()
  }

  const base = opts.path.startsWith('http') ? '' : API_BASE_URL
  const url = new URL(`${base}${opts.path}`)

  if (opts.params) {
    for (const [key, value] of Object.entries(opts.params)) {
      if (value === undefined || value === null) continue
      if (Array.isArray(value)) {
        for (const v of value) {
          if (v !== undefined && v !== null) {
            url.searchParams.append(key, String(v))
          }
        }
      } else {
        url.searchParams.set(key, String(value))
      }
    }
  }

  const headers: Record<string, string> = {}

  const isJsonBody =
    opts.body !== undefined &&
    !(opts.body instanceof FormData) &&
    !(opts.body instanceof Blob) &&
    typeof opts.body !== 'string'

  if (isJsonBody) {
    headers['Content-Type'] = 'application/json'
  }

  const token = await getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  if (MUTATION_METHODS.has(method) && !headers['Idempotency-Key']) {
    // See authenticatedRequest: setting the key via a Headers round-trip would
    // duplicate the Authorization header (capital + lowercase) and trigger 401.
    headers['Idempotency-Key'] = generateIdempotencyKey()
  }

  const init: RequestInit = { method, headers }
  if (opts.body !== undefined) {
    init.body = isJsonBody ? JSON.stringify(opts.body) : (opts.body as BodyInit)
  }

  const urlStr = url.toString()
  let response = await fetch(urlStr, init)

  if (response.status === 401) {
    const newToken = await doTokenRefresh()
    if (!newToken) {
      await doLogout()
      throw new Error('Authentication expired')
    }
    headers['Authorization'] = `Bearer ${newToken}`
    response = await fetch(urlStr, { ...init, headers })
  }

  return response
}
