/**
 * Lightweight fetch wrapper for the Berichte (Reports/BI) API endpoints.
 *
 * Follows the same pattern as wiki-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry. Adds a blob variant for the
 * PDF/CSV/XLSX export endpoint (Content-Disposition filename extraction).
 */
import { API_BASE_URL } from '@/lib/constants'
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

// ---------------------------------------------------------------------------
// Request helpers
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

async function requestBlobWithHeaders(
  path: string,
  method: string,
  body: unknown,
  fallbackFormat: ReportFormat,
  params?: RequestOptions['params'],
): Promise<ExportedReport> {
  if (!navigator.onLine) {
    throw new Error('Änderungen sind offline nicht möglich.')
  }

  const url = buildUrl(path, params)
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  const token = await getAuthToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const init: RequestInit = { method, headers }
  if (body !== undefined) init.body = JSON.stringify(body)

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
