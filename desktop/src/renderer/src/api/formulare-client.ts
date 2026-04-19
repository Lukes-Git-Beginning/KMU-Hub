/**
 * Lightweight fetch wrapper for the Formulare (Forms) API endpoints.
 *
 * Follows the same pattern as berichte-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry. Adds a blob variant for the
 * CSV/XLSX export endpoint (Content-Disposition filename extraction).
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  CreateFormSchemaInput,
  CreateSubmissionInput,
  CreateWebhookInput,
  DuplicateFormSchemaInput,
  ExportFormat,
  ExportedSubmissions,
  FormSchema,
  FormSubmission,
  FormWebhook,
  ListDeliveriesQuery,
  ListFormSchemasResponse,
  ListSchemasQuery,
  ListSubmissionsQuery,
  ListSubmissionsResponse,
  UpdateFormSchemaInput,
  UpdateSubmissionStatusInput,
  UpdateWebhookInput,
  WebhookDelivery,
} from './formulare-types'

// ---------------------------------------------------------------------------
// Request helpers (identical pattern to berichte-client.ts)
// ---------------------------------------------------------------------------

const MUTATION_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH'])

async function getAuthToken(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken
}

async function refreshAuthToken(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().refreshToken()
}

async function logoutAuth(): Promise<void> {
  const { useAuthStore } = await import('@/stores/auth')
  useAuthStore.getState().logout()
}

type QueryPrimitive = string | number | boolean | undefined | null

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, QueryPrimitive | QueryPrimitive[]>
}

function buildUrl(path: string, params?: RequestOptions['params']): string {
  const url = new URL(`${API_BASE_URL}${path}`)
  if (!params) return url.toString()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null) continue
    if (Array.isArray(value)) {
      const filtered = value.filter((v) => v !== undefined && v !== null)
      if (filtered.length === 0) continue
      url.searchParams.set(key, filtered.join(','))
      continue
    }
    url.searchParams.set(key, String(value))
  }
  return url.toString()
}

async function request<T>(opts: RequestOptions): Promise<T> {
  if (!navigator.onLine && MUTATION_METHODS.has(opts.method)) {
    throw new Error('Änderungen sind offline nicht möglich.')
  }

  const url = buildUrl(opts.path, opts.params)
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  const token = await getAuthToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const init: RequestInit = { method: opts.method, headers }
  if (opts.body !== undefined) {
    init.body = JSON.stringify(opts.body)
  }

  const response = await fetch(url, init)

  if (!response.ok) {
    if (response.status === 401) {
      const newToken = await refreshAuthToken()
      if (newToken) {
        headers['Authorization'] = `Bearer ${newToken}`
        const retry = await fetch(url, { ...init, headers })
        if (!retry.ok) {
          const errBody = await retry.json().catch(() => ({}))
          throw new Error(
            (errBody as Record<string, string>).error ||
              `Request failed: ${retry.status}`,
          )
        }
        if (retry.status === 204) return {} as T
        return retry.json() as Promise<T>
      }
      await logoutAuth()
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

async function requestBlob(
  path: string,
  params: RequestOptions['params'],
  fallbackFormat: ExportFormat,
): Promise<ExportedSubmissions> {
  if (!navigator.onLine) {
    throw new Error('Änderungen sind offline nicht möglich.')
  }

  const url = buildUrl(path, params)
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  const token = await getAuthToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const init: RequestInit = { method: 'GET', headers }

  let response = await fetch(url, init)

  if (response.status === 401) {
    const newToken = await refreshAuthToken()
    if (!newToken) {
      await logoutAuth()
      throw new Error('Authentication expired')
    }
    headers['Authorization'] = `Bearer ${newToken}`
    response = await fetch(url, { ...init, headers })
  }

  if (!response.ok) {
    const errBody = await response.json().catch(() => ({}))
    throw new Error(
      (errBody as Record<string, string>).error ||
        `Export failed: ${response.status}`,
    )
  }

  const blob = await response.blob()
  const disposition = response.headers.get('Content-Disposition') ?? ''
  const match = /filename\*?=(?:UTF-8'')?"?([^";]+)"?/i.exec(disposition)
  const filename = match?.[1]
    ? decodeURIComponent(match[1].trim())
    : `submissions.${fallbackFormat}`
  const contentType =
    response.headers.get('Content-Type') ?? 'application/octet-stream'

  return { blob, filename, contentType }
}

// ---------------------------------------------------------------------------
// Base path
// ---------------------------------------------------------------------------

const BASE = '/api/v1/formulare'

// ---------------------------------------------------------------------------
// Schemas
// ---------------------------------------------------------------------------

export function listFormSchemas(query?: ListSchemasQuery) {
  return request<ListFormSchemasResponse>({
    method: 'GET',
    path: `${BASE}/schemas`,
    params: query as RequestOptions['params'],
  })
}

export function getFormSchema(id: string) {
  return request<FormSchema>({
    method: 'GET',
    path: `${BASE}/schemas/${id}`,
  })
}

export function createFormSchema(body: CreateFormSchemaInput) {
  return request<FormSchema>({
    method: 'POST',
    path: `${BASE}/schemas`,
    body,
  })
}

export function updateFormSchema(id: string, body: UpdateFormSchemaInput) {
  return request<FormSchema>({
    method: 'PATCH',
    path: `${BASE}/schemas/${id}`,
    body,
  })
}

export function deleteFormSchema(id: string) {
  return request<void>({
    method: 'DELETE',
    path: `${BASE}/schemas/${id}`,
  })
}

export function duplicateFormSchema(id: string, body: DuplicateFormSchemaInput) {
  return request<FormSchema>({
    method: 'POST',
    path: `${BASE}/schemas/${id}/duplicate`,
    body,
  })
}

// ---------------------------------------------------------------------------
// Submissions
// ---------------------------------------------------------------------------

export function listSubmissions(formSchemaId: string, query?: ListSubmissionsQuery) {
  return request<ListSubmissionsResponse>({
    method: 'GET',
    path: `${BASE}/schemas/${formSchemaId}/submissions`,
    params: query as RequestOptions['params'],
  })
}

export function getSubmission(id: string) {
  return request<FormSubmission>({
    method: 'GET',
    path: `${BASE}/submissions/${id}`,
  })
}

export function createSubmission(formSchemaId: string, body: CreateSubmissionInput) {
  return request<FormSubmission>({
    method: 'POST',
    path: `${BASE}/schemas/${formSchemaId}/submissions`,
    body,
  })
}

export function updateSubmissionStatus(id: string, body: UpdateSubmissionStatusInput) {
  return request<FormSubmission>({
    method: 'PATCH',
    path: `${BASE}/submissions/${id}`,
    body,
  })
}

export function exportSubmissions(
  formSchemaId: string,
  format: ExportFormat,
): Promise<ExportedSubmissions> {
  return requestBlob(
    `${BASE}/schemas/${formSchemaId}/submissions/export`,
    { format },
    format,
  )
}

// ---------------------------------------------------------------------------
// Webhooks
// ---------------------------------------------------------------------------

export function listWebhooks(formSchemaId: string) {
  return request<FormWebhook[]>({
    method: 'GET',
    path: `${BASE}/schemas/${formSchemaId}/webhooks`,
  })
}

export function getWebhook(id: string) {
  return request<FormWebhook>({
    method: 'GET',
    path: `${BASE}/webhooks/${id}`,
  })
}

export function createWebhook(formSchemaId: string, body: CreateWebhookInput) {
  return request<FormWebhook>({
    method: 'POST',
    path: `${BASE}/schemas/${formSchemaId}/webhooks`,
    body,
  })
}

export function updateWebhook(id: string, body: UpdateWebhookInput) {
  return request<FormWebhook>({
    method: 'PATCH',
    path: `${BASE}/webhooks/${id}`,
    body,
  })
}

export function deleteWebhook(id: string) {
  return request<void>({
    method: 'DELETE',
    path: `${BASE}/webhooks/${id}`,
  })
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

export interface FormStatsResponse {
  totalSubmissions: number
  newSubmissions: number
  readSubmissions: number
  archivedSubmissions: number
  submissionsThisWeek: number
  submissionsThisMonth: number
  averageCompletionRate: number
}

export function getFormStats(schemaId: string) {
  return request<FormStatsResponse>({
    method: 'GET',
    path: `${BASE}/schemas/${schemaId}/stats`,
  })
}

// ---------------------------------------------------------------------------
// Deliveries
// ---------------------------------------------------------------------------

export function listWebhookDeliveries(webhookId: string) {
  return request<WebhookDelivery[]>({
    method: 'GET',
    path: `${BASE}/webhooks/${webhookId}/deliveries`,
  })
}

export function listAllDeliveries(query?: ListDeliveriesQuery) {
  return request<WebhookDelivery[]>({
    method: 'GET',
    path: `${BASE}/deliveries`,
    params: query as RequestOptions['params'],
  })
}
