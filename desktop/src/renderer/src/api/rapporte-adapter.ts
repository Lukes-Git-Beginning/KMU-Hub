/**
 * Adapter — maps API WorkReport (rapporte-types.ts) to the UI FieldReport shape
 * used by RapportePage and its subcomponents.
 *
 * Fields not present in the API response are defaulted to safe zero-values.
 * When the backend gains richer fields (workers, materials, …) we can enrich here.
 */
import type { WorkReport, ReportStats } from './rapporte-types'
import type { FieldReport } from '@/stores/rapporte'

/** Map a single WorkReport wire record to the UI FieldReport shape. */
export function adaptWorkReport(api: WorkReport): FieldReport {
  return {
    id: api.id,
    // report_date is ISO date string (YYYY-MM-DD or ISO 8601)
    date: api.report_date ? api.report_date.split('T')[0] : api.created_at.split('T')[0],
    // The API does not carry project info on the report — use title as project name
    projectId: api.id, // no separate project FK yet; use id as placeholder key
    projectName: api.title,
    author: api.author_id,
    // Weather / timing fields not in API — safe defaults
    weather: 'sunny' as const,
    temperature: 10,
    workStart: '08:00',
    workEnd: '17:00',
    breakMinutes: 60,
    workers: [],
    // Derive a single activity from description so the list renders something
    activities: api.description
      ? [{ description: api.description, category: '' }]
      : [],
    materials: [],
    photos: [],
    notes: api.review_note ?? '',
    signatureStatus: 'pending' as const,
    signatureDataUrl: undefined,
    approvalStatus: api.status as FieldReport['approvalStatus'],
    approvalComment: api.review_note ?? undefined,
    approvedBy: api.reviewer_id ?? undefined,
  }
}

/** Map the stats response to summary numbers used by the stats row. */
export interface RapporteUISummary {
  totalReports: number
  approvedCount: number
  submittedCount: number
  draftCount: number
}

export function adaptReportStats(stats: ReportStats): RapporteUISummary {
  return {
    totalReports: stats.total_reports,
    approvedCount: stats.approved_count,
    submittedCount: stats.submitted_count,
    draftCount: stats.draft_count,
  }
}
