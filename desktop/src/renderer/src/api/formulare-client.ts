/**
 * Lightweight fetch wrapper for the Formulare (Forms) API endpoints.
 *
 * Follows the same pattern as berichte-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry. Adds a blob variant for the
 * CSV/XLSX export endpoint (Content-Disposition filename extraction).
 */
import type {
  CreateFormSchemaInput,
  CreateShareLinkInput,
  CreateSubmissionInput,
  CreateWebhookInput,
  DuplicateFormSchemaInput,
  ExportFormat,
  ExportedSubmissions,
  FieldStat,
  FormSchema,
  FormShareLink,
  FormSubmission,
  FormWebhook,
  ListDeliveriesQuery,
  ListFormSchemasResponse,
  ListSchemasQuery,
  ListSubmissionsQuery,
  ListSubmissionsResponse,
  UpdateFormSchemaInput,
  UpdateShareLinkInput,
  UpdateSubmissionStatusInput,
  UpdateWebhookInput,
  WebhookDelivery,
} from './formulare-types'
import { authenticatedRequest, authenticatedBlobRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Request helpers
// ---------------------------------------------------------------------------

type QueryPrimitive = string | number | boolean | undefined | null

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, QueryPrimitive | QueryPrimitive[]>
}

function request<T>(opts: RequestOptions): Promise<T> {
  return authenticatedRequest<T>(opts)
}

async function requestBlob(
  path: string,
  params: RequestOptions['params'],
  fallbackFormat: ExportFormat,
): Promise<ExportedSubmissions> {
  const response = await authenticatedBlobRequest({ method: 'GET', path, params: params as Record<string, string | number | boolean | null | undefined> })

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
// Share links (FD-1 / FD-2)
// ---------------------------------------------------------------------------

export function listShareLinks(formSchemaId: string) {
  return request<FormShareLink[]>({
    method: 'GET',
    path: `${BASE}/schemas/${formSchemaId}/share-links`,
  })
}

export function createShareLink(formSchemaId: string, body: CreateShareLinkInput) {
  return request<FormShareLink>({
    method: 'POST',
    path: `${BASE}/schemas/${formSchemaId}/share-links`,
    body,
  })
}

export function updateShareLink(id: string, body: UpdateShareLinkInput) {
  return request<FormShareLink>({
    method: 'PATCH',
    path: `${BASE}/share-links/${id}`,
    body,
  })
}

export function deleteShareLink(id: string) {
  return request<void>({
    method: 'DELETE',
    path: `${BASE}/share-links/${id}`,
  })
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
  /** FT-3a — per-field analysis, keyed by field id. */
  fieldStats?: Record<string, FieldStat>
  /** FT-3b — aggregated share-link views → submissions conversion. */
  totalViews?: number
  conversionRate?: number
  /** FT-3b — simulated per-page drop-off (only for multi-page forms). */
  pageDropoff?: { page: number; percent: number }[]
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
