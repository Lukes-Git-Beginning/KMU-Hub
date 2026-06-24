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
import { authenticatedRequest } from './utils/authenticatedFetch'
import { normalizeWireTimestamps } from './wire-time'

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | string[] | undefined>
}

/**
 * The calendar gateway serializes responses via `response.JSON`, so
 * `google.protobuf.Timestamp` fields (event start_time/end_time, reminders,
 * created_at, …) arrive as `{seconds, nanos}` rather than RFC3339 strings.
 * Normalize every response to ISO before it reaches the hooks — the walk is a
 * no-op on already-string timestamps (mock mode), so it is safe to apply
 * unconditionally.
 */
async function request<T>(opts: RequestOptions): Promise<T> {
  const result = await authenticatedRequest<T>(opts)
  return normalizeWireTimestamps(result)
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
