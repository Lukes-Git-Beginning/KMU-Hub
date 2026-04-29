/**
 * Lightweight fetch wrapper for the Berichte (Reports/BI) API endpoints.
 *
 * Follows the same pattern as wiki-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry. Adds a blob variant for the
 * PDF/CSV/XLSX export endpoint (Content-Disposition filename extraction).
 */
import type {
  CreateDefinitionInput,
  CreateScheduleInput,
  DashboardKPIsResponse,
  DefinitionResponse,
  ExportReportInput,
  ExportedReport,
  InvalidateCacheResponse,
  ListDefinitionsParams,
  ListDefinitionsResponse,
  ListSchedulesParams,
  ListSchedulesResponse,
  ReportFormat,
  RunReportInput,
  RunReportResponse,
  ScheduleResponse,
  UpdateDefinitionInput,
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
