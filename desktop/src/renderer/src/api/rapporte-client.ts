/**
 * Lightweight fetch wrapper for Rapporte API endpoints.
 *
 * Follows the same pattern as inventar-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry.
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  WorkReport,
  ReportLine,
  ReportAttachment,
  ReportStats,
  CreateReportInput,
  UpdateReportInput,
  ListReportsParams,
  ApproveReportInput,
  RejectReportInput,
  AddLineInput,
  UpdateLineInput,
  ListAttachmentsParams,
  UploadAttachmentInput,
  ListPendingApprovalsParams,
  ListReportsResponse,
  ListLinesResponse,
  ListAttachmentsResponse,
  ListPendingApprovalsResponse,
} from './rapporte-types'

// ---------------------------------------------------------------------------
// Request helper
// ---------------------------------------------------------------------------

const MUTATION_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH'])

async function getAuthToken(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken
}

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | undefined>
}

async function request<T>(opts: RequestOptions): Promise<T> {
  if (!navigator.onLine && MUTATION_METHODS.has(opts.method)) {
    throw new Error('Änderungen sind offline nicht möglich.')
  }

  const url = new URL(`${API_BASE_URL}${opts.path}`)

  if (opts.params) {
    for (const [key, value] of Object.entries(opts.params)) {
      if (value === undefined) continue
      url.searchParams.set(key, String(value))
    }
  }

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

  const response = await fetch(url.toString(), init)

  if (!response.ok) {
    if (response.status === 401) {
      const { useAuthStore } = await import('@/stores/auth')
      const store = useAuthStore.getState()
      const newToken = await store.refreshToken()

      if (newToken) {
        headers['Authorization'] = `Bearer ${newToken}`
        const retryResponse = await fetch(url.toString(), { ...init, headers })

        if (!retryResponse.ok) {
          const errBody = await retryResponse.json().catch(() => ({}))
          throw new Error(
            (errBody as Record<string, string>).error ||
              `Request failed: ${retryResponse.status}`,
          )
        }

        if (retryResponse.status === 204) return {} as T
        return retryResponse.json() as Promise<T>
      }

      store.logout()
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

// ---------------------------------------------------------------------------
// Base path
// ---------------------------------------------------------------------------

const BASE = '/api/v1/rapporte'

// ---------------------------------------------------------------------------
// Reports
// ---------------------------------------------------------------------------

export function listReports(params?: ListReportsParams) {
  return request<ListReportsResponse>({
    method: 'GET',
    path: `${BASE}/reports`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getReport(id: string) {
  return request<{ report: WorkReport }>({ method: 'GET', path: `${BASE}/reports/${id}` })
}

export function createReport(body: CreateReportInput) {
  return request<{ report: WorkReport }>({ method: 'POST', path: `${BASE}/reports`, body })
}

export function updateReport(id: string, body: UpdateReportInput) {
  return request<{ report: WorkReport }>({ method: 'PATCH', path: `${BASE}/reports/${id}`, body })
}

export function deleteReport(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/reports/${id}` })
}

// ---------------------------------------------------------------------------
// State transitions
// ---------------------------------------------------------------------------

export function submitReport(id: string) {
  return request<{ report: WorkReport }>({ method: 'POST', path: `${BASE}/reports/${id}/submit` })
}

export function approveReport(id: string, body: ApproveReportInput) {
  return request<{ report: WorkReport }>({
    method: 'POST',
    path: `${BASE}/reports/${id}/approve`,
    body,
  })
}

export function rejectReport(id: string, body: RejectReportInput) {
  return request<{ report: WorkReport }>({
    method: 'POST',
    path: `${BASE}/reports/${id}/reject`,
    body,
  })
}

// ---------------------------------------------------------------------------
// Lines
// ---------------------------------------------------------------------------

export function listLines(reportId: string) {
  return request<ListLinesResponse>({ method: 'GET', path: `${BASE}/reports/${reportId}/lines` })
}

export function addLine(reportId: string, body: AddLineInput) {
  return request<{ line: ReportLine }>({
    method: 'POST',
    path: `${BASE}/reports/${reportId}/lines`,
    body,
  })
}

export function updateLine(lineId: string, body: UpdateLineInput) {
  return request<{ line: ReportLine }>({ method: 'PATCH', path: `${BASE}/lines/${lineId}`, body })
}

export function deleteLine(lineId: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/lines/${lineId}` })
}

// ---------------------------------------------------------------------------
// Attachments
// ---------------------------------------------------------------------------

export function listAttachments(reportId: string, params?: ListAttachmentsParams) {
  return request<ListAttachmentsResponse>({
    method: 'GET',
    path: `${BASE}/reports/${reportId}/attachments`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function uploadAttachment(reportId: string, body: UploadAttachmentInput) {
  return request<{ attachment: ReportAttachment }>({
    method: 'POST',
    path: `${BASE}/reports/${reportId}/attachments`,
    body,
  })
}

export function deleteAttachment(attachmentId: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/attachments/${attachmentId}` })
}

// ---------------------------------------------------------------------------
// Stats & approvals
// ---------------------------------------------------------------------------

export function getReportStats() {
  return request<ReportStats>({ method: 'GET', path: `${BASE}/stats` })
}

export function listPendingApprovals(params?: ListPendingApprovalsParams) {
  return request<ListPendingApprovalsResponse>({
    method: 'GET',
    path: `${BASE}/pending`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

export function getExportPDFUrl(reportId: string) {
  return `${API_BASE_URL}${BASE}/reports/${reportId}/export/pdf`
}
