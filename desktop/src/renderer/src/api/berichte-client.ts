/**
 * Lightweight fetch wrapper for the Berichte (Reports/BI) API endpoints.
 *
 * Follows the same pattern as wiki-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry. Adds a blob variant for the
 * PDF/CSV/XLSX export endpoint (Content-Disposition filename extraction).
 */
import type {
  BuilderQueryConfig,
  CreateDefinitionInput,
  CreateReportDocumentInput,
  CreateScheduleInput,
  DashboardKPIsResponse,
  DefinitionResponse,
  ExportReportInput,
  ExportedReport,
  InvalidateCacheResponse,
  ListDefinitionsParams,
  ListDefinitionsResponse,
  ListReportDocumentsParams,
  ListReportDocumentsResponse,
  ListReportTemplatesResponse,
  ListSchedulesParams,
  ListSchedulesResponse,
  PreviewReportResponse,
  ReportDocumentResponse,
  ReportFormat,
  RunReportInput,
  RunReportResponse,
  ScheduleResponse,
  UpdateDefinitionInput,
  UpdateReportDocumentInput,
  UpdateScheduleInput,
} from './berichte-types'
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

async function requestBlobWithHeaders(
  path: string,
  method: string,
  body: unknown,
  fallbackFormat: ReportFormat,
  params?: RequestOptions['params'],
): Promise<ExportedReport> {
  const response = await authenticatedBlobRequest({ method, path, body, params: params as Record<string, string | number | boolean | null | undefined> })

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
    : `report.${fallbackFormat}`
  const contentType =
    response.headers.get('Content-Type') ?? 'application/octet-stream'

  return { blob, filename, content_type: contentType }
}

// ---------------------------------------------------------------------------
// Base path
// ---------------------------------------------------------------------------

const BASE = '/api/v1/berichte'

// ---------------------------------------------------------------------------
// Definitions
// ---------------------------------------------------------------------------

export function listDefinitions(params?: ListDefinitionsParams) {
  return request<ListDefinitionsResponse>({
    method: 'GET',
    path: `${BASE}/definitions`,
    params: params as RequestOptions['params'],
  })
}

export function getDefinition(id: string) {
  return request<DefinitionResponse>({
    method: 'GET',
    path: `${BASE}/definitions/${id}`,
  })
}

export function createDefinition(body: CreateDefinitionInput) {
  return request<DefinitionResponse>({
    method: 'POST',
    path: `${BASE}/definitions`,
    body,
  })
}

export function updateDefinition(id: string, body: UpdateDefinitionInput) {
  return request<DefinitionResponse>({
    method: 'PATCH',
    path: `${BASE}/definitions/${id}`,
    body,
  })
}

export function deleteDefinition(id: string) {
  return request<void>({
    method: 'DELETE',
    path: `${BASE}/definitions/${id}`,
  })
}

// ---------------------------------------------------------------------------
// Run / Cache / Export
// ---------------------------------------------------------------------------

export function runReport(definitionId: string, body?: RunReportInput) {
  return request<RunReportResponse>({
    method: 'POST',
    path: `${BASE}/definitions/${definitionId}/run`,
    body: body ?? {},
  })
}

export function exportReport(
  definitionId: string,
  input: ExportReportInput,
): Promise<ExportedReport> {
  return requestBlobWithHeaders(
    `${BASE}/definitions/${definitionId}/export`,
    'POST',
    { params: input.params ?? {} },
    input.format,
    { format: input.format },
  )
}

export function invalidateCache(definitionId: string) {
  return request<InvalidateCacheResponse>({
    method: 'DELETE',
    path: `${BASE}/definitions/${definitionId}/cache`,
  })
}

/** Builder live preview — runs a no-code query and returns a ReportResult. */
export function previewReport(query: BuilderQueryConfig) {
  return request<PreviewReportResponse>({
    method: 'POST',
    path: `${BASE}/preview`,
    body: { query },
  })
}

// ---------------------------------------------------------------------------
// Schedules
// ---------------------------------------------------------------------------

export function listSchedules(params?: ListSchedulesParams) {
  return request<ListSchedulesResponse>({
    method: 'GET',
    path: `${BASE}/schedules`,
    params: params as RequestOptions['params'],
  })
}

export function createSchedule(body: CreateScheduleInput) {
  return request<ScheduleResponse>({
    method: 'POST',
    path: `${BASE}/schedules`,
    body,
  })
}

export function updateSchedule(id: string, body: UpdateScheduleInput) {
  return request<ScheduleResponse>({
    method: 'PATCH',
    path: `${BASE}/schedules/${id}`,
    body,
  })
}

export function deleteSchedule(id: string) {
  return request<void>({
    method: 'DELETE',
    path: `${BASE}/schedules/${id}`,
  })
}

export function toggleSchedule(id: string, active: boolean) {
  return request<ScheduleResponse>({
    method: 'POST',
    path: `${BASE}/schedules/${id}/toggle`,
    body: { active },
  })
}

/** Trigger an immediate manual run of a schedule ("send now"). */
export function runScheduleNow(id: string) {
  return request<ScheduleResponse>({
    method: 'POST',
    path: `${BASE}/schedules/${id}/run`,
  })
}

// ---------------------------------------------------------------------------
// Integration (R-5) — attach a report as a reference to other modules.
// These talk to non-berichte endpoints on purpose (cross-module linking).
// ---------------------------------------------------------------------------

/** A lightweight report reference used when attaching to tasks/contacts. */
export interface ReportRef {
  id: string
  title: string
}

/** Attach a report document as a reference (not a file copy) to a work task. */
export function attachReportToTask(taskId: string, doc: ReportRef) {
  return request<{ file: unknown }>({
    method: 'POST',
    path: `/api/v1/tasks/${taskId}/files`,
    body: {
      filename: doc.title,
      mime_type: 'application/cosmi-report',
      file_size: 0,
      storage_key: `report://${doc.id}`,
      thumbnail_key: '',
    },
  })
}

/** Attach a report document as a reference to a CRM contact. */
export function attachReportToContact(contactId: string, doc: ReportRef) {
  return request<{ file: unknown }>({
    method: 'POST',
    path: `/api/v1/contacts/${contactId}/files`,
    body: {
      filename: doc.title,
      mime_type: 'application/cosmi-report',
      storage_key: `report://${doc.id}`,
    },
  })
}

export interface SavedDocumentFile {
  id: string
  name: string
}

/**
 * Save the report as a PDF file into the documents module (R-5b). The real
 * PDF bytes come from the server export (R-3b, Luke); the demo files a small
 * placeholder PDF so the entry appears in documents with the "Bericht" tag.
 */
export function saveReportToDocuments(folderId: string, doc: ReportRef) {
  const placeholder = new Blob([`%PDF-1.4\n% Cosmi-Bericht: ${doc.title}\n`], {
    type: 'application/pdf',
  })
  const file = new File([placeholder], `${doc.title}.pdf`, { type: 'application/pdf' })
  const form = new FormData()
  form.append('file', file)
  form.append('folder_id', folderId)
  form.append('tag_id', 't-bericht')
  return request<{ file: SavedDocumentFile }>({
    method: 'POST',
    path: '/api/v1/documents/files/upload',
    body: form,
  })
}

// ---------------------------------------------------------------------------
// Dashboard KPIs
// ---------------------------------------------------------------------------

export function getDashboardKPIs(modules?: string[]) {
  return request<DashboardKPIsResponse>({
    method: 'GET',
    path: `${BASE}/kpis`,
    params: modules && modules.length > 0 ? { modules } : undefined,
  })
}

// ---------------------------------------------------------------------------
// Report Documents (multi-page authoring)
// ---------------------------------------------------------------------------

export function listReportDocuments(params?: ListReportDocumentsParams) {
  return request<ListReportDocumentsResponse>({
    method: 'GET',
    path: `${BASE}/documents`,
    params: params as RequestOptions['params'],
  })
}

export function getReportDocument(id: string) {
  return request<ReportDocumentResponse>({
    method: 'GET',
    path: `${BASE}/documents/${id}`,
  })
}

export function createReportDocument(body: CreateReportDocumentInput) {
  return request<ReportDocumentResponse>({
    method: 'POST',
    path: `${BASE}/documents`,
    body,
  })
}

export function updateReportDocument(id: string, body: UpdateReportDocumentInput) {
  return request<ReportDocumentResponse>({
    method: 'PATCH',
    path: `${BASE}/documents/${id}`,
    body,
  })
}

export function deleteReportDocument(id: string) {
  return request<void>({
    method: 'DELETE',
    path: `${BASE}/documents/${id}`,
  })
}

export function listReportTemplates() {
  return request<ListReportTemplatesResponse>({
    method: 'GET',
    path: `${BASE}/templates`,
  })
}
