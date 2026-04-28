/**
 * TanStack Query hooks for the Rapporte module.
 *
 * Queries for reports, lines, attachments, stats, and pending approvals.
 * Mutations for report lifecycle, state transitions, line management,
 * and attachment uploads.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listReports,
  getReport,
  createReport,
  updateReport,
  deleteReport,
  submitReport,
  approveReport,
  rejectReport,
  listLines,
  addLine,
  updateLine,
  deleteLine,
  listAttachments,
  uploadAttachment,
  deleteAttachment,
  getReportStats,
  listPendingApprovals,
} from '../rapporte-client'
import type {
  ListReportsParams,
  CreateReportInput,
  UpdateReportInput,
  ApproveReportInput,
  RejectReportInput,
  AddLineInput,
  UpdateLineInput,
  ListAttachmentsParams,
  UploadAttachmentInput,
  ListPendingApprovalsParams,
} from '../rapporte-types'

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const rapporteKeys = {
  all: ['rapporte'] as const,
  reports: (params?: ListReportsParams) => ['rapporte', 'reports', params] as const,
  report: (id: string) => ['rapporte', 'report', id] as const,
  lines: (reportId: string) => ['rapporte', 'report', reportId, 'lines'] as const,
  attachments: (reportId: string, params?: ListAttachmentsParams) =>
    ['rapporte', 'report', reportId, 'attachments', params] as const,
  stats: () => ['rapporte', 'stats'] as const,
  pending: (params?: ListPendingApprovalsParams) => ['rapporte', 'pending', params] as const,
}

// ---------------------------------------------------------------------------
// Queries — Reports
// ---------------------------------------------------------------------------

export function useRapporteList(params?: ListReportsParams) {
  return useQuery({
    queryKey: rapporteKeys.reports(params),
    queryFn: () => listReports(params),
    staleTime: 30_000,
  })
}

export function useRapporte(id: string) {
  return useQuery({
    queryKey: rapporteKeys.report(id),
    queryFn: () => getReport(id),
    enabled: !!id,
    staleTime: 30_000,
  })
}

// ---------------------------------------------------------------------------
// Queries — Lines
// ---------------------------------------------------------------------------

export function useReportLines(reportId: string) {
  return useQuery({
    queryKey: rapporteKeys.lines(reportId),
    queryFn: () => listLines(reportId),
    enabled: !!reportId,
    staleTime: 30_000,
  })
}

// ---------------------------------------------------------------------------
// Queries — Attachments
// ---------------------------------------------------------------------------

export function useReportAttachments(reportId: string, params?: ListAttachmentsParams) {
  return useQuery({
    queryKey: rapporteKeys.attachments(reportId, params),
    queryFn: () => listAttachments(reportId, params),
    enabled: !!reportId,
    staleTime: 30_000,
  })
}

// ---------------------------------------------------------------------------
// Queries — Stats
// ---------------------------------------------------------------------------

export function useReportStats() {
  return useQuery({
    queryKey: rapporteKeys.stats(),
    queryFn: getReportStats,
    staleTime: 30_000,
  })
}

// ---------------------------------------------------------------------------
// Queries — Pending approvals
// ---------------------------------------------------------------------------

export function usePendingApprovals(params?: ListPendingApprovalsParams) {
  return useQuery({
    queryKey: rapporteKeys.pending(params),
    queryFn: () => listPendingApprovals(params),
    staleTime: 30_000,
  })
}

// ---------------------------------------------------------------------------
// Mutations — Report lifecycle
// ---------------------------------------------------------------------------

export function useCreateReport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateReportInput) => createReport(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['rapporte', 'reports'] })
      qc.invalidateQueries({ queryKey: rapporteKeys.stats() })
    },
  })
}

export function useUpdateReport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateReportInput & { id: string }) => updateReport(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: rapporteKeys.report(variables.id) })
      qc.invalidateQueries({ queryKey: ['rapporte', 'reports'] })
    },
  })
}

export function useDeleteReport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteReport(id),
    // Bug #19: also invalidate the detail cache so stale report data is evicted
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: rapporteKeys.report(id) })
      qc.invalidateQueries({ queryKey: ['rapporte', 'reports'] })
      qc.invalidateQueries({ queryKey: rapporteKeys.stats() })
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — State transitions
// ---------------------------------------------------------------------------

export function useSubmitReport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => submitReport(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: rapporteKeys.report(id) })
      qc.invalidateQueries({ queryKey: ['rapporte', 'reports'] })
      qc.invalidateQueries({ queryKey: rapporteKeys.stats() })
    },
  })
}

export function useApproveReport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: ApproveReportInput & { id: string }) =>
      approveReport(id, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: rapporteKeys.report(variables.id) })
      qc.invalidateQueries({ queryKey: ['rapporte', 'reports'] })
      qc.invalidateQueries({ queryKey: ['rapporte', 'pending'] })
      qc.invalidateQueries({ queryKey: rapporteKeys.stats() })
    },
  })
}

export function useRejectReport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: RejectReportInput & { id: string }) =>
      rejectReport(id, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: rapporteKeys.report(variables.id) })
      qc.invalidateQueries({ queryKey: ['rapporte', 'reports'] })
      qc.invalidateQueries({ queryKey: ['rapporte', 'pending'] })
      qc.invalidateQueries({ queryKey: rapporteKeys.stats() })
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — Lines
// ---------------------------------------------------------------------------

export function useAddLine() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ reportId, ...body }: AddLineInput & { reportId: string }) =>
      addLine(reportId, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: rapporteKeys.lines(variables.reportId) })
    },
  })
}

export function useUpdateLine() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, reportId, ...body }: UpdateLineInput & { id: string; reportId: string }) =>
      updateLine(id, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: rapporteKeys.lines(variables.reportId) })
    },
  })
}

export function useDeleteLine() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id }: { id: string; reportId: string }) => deleteLine(id),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: rapporteKeys.lines(variables.reportId) })
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — Attachments
// ---------------------------------------------------------------------------

export function useUploadAttachment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ reportId, ...body }: UploadAttachmentInput & { reportId: string }) =>
      uploadAttachment(reportId, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: rapporteKeys.attachments(variables.reportId) })
    },
  })
}

export function useDeleteAttachment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id }: { id: string; reportId: string }) => deleteAttachment(id),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: rapporteKeys.attachments(variables.reportId) })
    },
  })
}
