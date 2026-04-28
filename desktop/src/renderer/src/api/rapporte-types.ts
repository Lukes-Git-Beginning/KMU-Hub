/**
 * TypeScript types for the Rapporte module.
 *
 * Mirrors backend/internal/rapporte/models.go and route_rapporte.go request types.
 * UUIDs are represented as strings; dates as ISO 8601 strings.
 */

// ---------------------------------------------------------------------------
// Enums
// ---------------------------------------------------------------------------

export type ReportStatus = 'draft' | 'submitted' | 'approved' | 'rejected'

// ---------------------------------------------------------------------------
// Domain models
// ---------------------------------------------------------------------------

export interface WorkReport {
  id: string
  tenant_id: string
  title: string
  description: string
  status: ReportStatus
  author_id: string
  reviewer_id: string | null
  reviewed_at: string | null
  review_note: string
  lat: number | null
  lon: number | null
  report_date: string
  created_at: string
  updated_at: string
  deleted_at: string | null
}

export interface ReportLine {
  id: string
  tenant_id: string
  report_id: string
  position: number
  description: string
  quantity: number
  unit: string
  notes: string
  created_at: string
  updated_at: string
}

export interface ReportAttachment {
  id: string
  tenant_id: string
  report_id: string
  line_id: string | null
  filename: string
  content_type: string
  size_bytes: number
  object_key: string
  uploaded_by: string
  created_at: string
}

export interface ReportStats {
  total_reports: number
  draft_count: number
  submitted_count: number
  approved_count: number
  rejected_count: number
}

// ---------------------------------------------------------------------------
// Request input types
// ---------------------------------------------------------------------------

export interface CreateReportInput {
  title: string
  description?: string
  author_id: string
  lat?: number
  lon?: number
  report_date?: string
}

export interface UpdateReportInput {
  title?: string
  description?: string
  lat?: number
  lon?: number
  report_date?: string
}

export interface ListReportsParams {
  search?: string
  status?: ReportStatus
  author_id?: string
  page?: number
  page_size?: number
}

export interface ApproveReportInput {
  reviewer_id: string
  review_note?: string
}

export interface RejectReportInput {
  reviewer_id: string
  review_note?: string
}

export interface AddLineInput {
  description: string
  quantity?: number
  unit?: string
  notes?: string
  position?: number
}

export interface UpdateLineInput {
  description?: string
  quantity?: number
  unit?: string
  notes?: string
  position?: number
}

export interface ListAttachmentsParams {
  line_id?: string
}

export interface UploadAttachmentInput {
  line_id?: string
  filename: string
  content_type?: string
  size_bytes?: number
  object_key: string
  uploaded_by: string
}

export interface ListPendingApprovalsParams {
  page?: number
  page_size?: number
}

// ---------------------------------------------------------------------------
// Response wrapper types
// ---------------------------------------------------------------------------

export interface ListReportsResponse {
  reports: WorkReport[]
  total: number
}

export interface ListLinesResponse {
  lines: ReportLine[]
}

export interface ListAttachmentsResponse {
  attachments: ReportAttachment[]
}

export interface ListPendingApprovalsResponse {
  reports: WorkReport[]
  total: number
}

export interface ExportPDFResponse {
  payload: Uint8Array
  filename: string
  content_type: string
}
