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

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | string[] | undefined>
}

function request<T>(opts: RequestOptions): Promise<T> {
  return authenticatedRequest<T>(opts)
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
