/**
 * Lightweight fetch wrapper for Rapporte API endpoints.
 *
 * Follows the same pattern as inventar-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry.
 */
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
import { authenticatedRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Request helper
// ---------------------------------------------------------------------------

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | undefined>
}

function request<T>(opts: RequestOptions): Promise<T> {
  return authenticatedRequest<T>(opts)
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

// NOTE: the former getExportPDFUrl() helper was dead code referencing a missing
// API_BASE_URL — the report PDF is generated client-side (modules/rapporte/rapporte-export.ts).
