/**
 * TanStack Query hooks for HR module operations.
 *
 * Query keys follow the pattern ['hr', domain, ...params] for consistent
 * cache invalidation. Mutations invalidate related queries automatically.
 *
 * ArbZG compliance toasts are triggered on clock-in/out success callbacks.
 * Work time status polls every 30 seconds for real-time header display.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import i18next from 'i18next'
import {
  hrLeaveApi,
  hrTimeApi,
  hrAbsenceApi,
  hrEmployeeApi,
  hrSettingsApi,
} from '../hr-client'
import type {
  CreateLeaveRequestInput,
  CreateEmployeeInput,
  ApproveRejectInput,
  RecordSickLeaveInput,
  SubmitCorrectionInput,
  CreateManualEntryInput,
  TimeAnalyticsRange,
  UpdateEmployeeInput,
  UpdateSelfProfileInput,
  UploadDocumentInput,
  ListLeaveRequestsParams,
  ListWorkTimeEntriesParams,
  ListEmployeesParams,
  AbsenceCalendarParams,
  ArbZGComplianceResult,
  HRSettings,
} from '../hr-types'

// ---------------------------------------------------------------------------
// Query key factory
// ---------------------------------------------------------------------------

export const hrKeys = {
  all: ['hr'] as const,

  // Leave
  leaveRequests: (params?: ListLeaveRequestsParams) =>
    ['hr', 'leave', 'requests', params] as const,
  leaveRequest: (id: string) => ['hr', 'leave', 'requests', id] as const,
  leaveBalance: () => ['hr', 'leave', 'balance'] as const,
  employeeLeaveBalance: (userId: string) =>
    ['hr', 'leave', 'balance', userId] as const,
  leaveTypes: () => ['hr', 'leave', 'types'] as const,

  // Time
  workTimeStatus: () => ['hr', 'time', 'status'] as const,
  activeShift: () => ['hr', 'time', 'active'] as const,
  workTimeEntries: (params?: ListWorkTimeEntriesParams) =>
    ['hr', 'time', 'entries', params] as const,
  dailySummary: (date: string) => ['hr', 'time', 'summary', 'daily', date] as const,
  weeklySummary: (weekStart: string) =>
    ['hr', 'time', 'summary', 'weekly', weekStart] as const,
  timeBalance: () => ['hr', 'time', 'balance'] as const,
  timeProjects: () => ['hr', 'time', 'projects'] as const,
  timeAnalytics: (range: string) => ['hr', 'time', 'analytics', range] as const,
  teamTime: (weekStart: string) => ['hr', 'time', 'team', weekStart] as const,
  myWeekStatus: (weekStart: string) => ['hr', 'time', 'weeks', 'status', weekStart] as const,

  // Absences
  absenceCalendar: (params: AbsenceCalendarParams) =>
    ['hr', 'absences', 'calendar', params] as const,

  // Employees
  employees: (params?: ListEmployeesParams) =>
    ['hr', 'employees', params] as const,
  selfProfile: () => ['hr', 'employees', 'me'] as const,
  employee: (id: string) => ['hr', 'employees', id] as const,
  employeeDocuments: (employeeId: string) =>
    ['hr', 'employees', employeeId, 'documents'] as const,
  documentCategories: (employeeId: string) =>
    ['hr', 'employees', employeeId, 'documents', 'categories'] as const,

  // Settings
  hrSettings: () => ['hr', 'settings'] as const,
}

// ---------------------------------------------------------------------------
// ArbZG toast helper
// ---------------------------------------------------------------------------

function showArbZGToast(compliance: ArbZGComplianceResult | undefined) {
  if (!compliance) return
  switch (compliance.severity) {
    case 'info':
      toast.info(compliance.message || i18next.t('api.hr.arbzg.info'))
      break
    case 'warning':
      toast.warning(compliance.message || i18next.t('api.hr.arbzg.warning'))
      break
    case 'error':
      toast.error(
        compliance.message || i18next.t('api.hr.arbzg.error'),
      )
      break
  }
  if (compliance.restViolation) {
    toast.warning(
      i18next.t('api.hr.arbzg.restViolation', { hours: compliance.restHoursActual ?? 0 }),
    )
  }
}

// ---------------------------------------------------------------------------
// Leave hooks
// ---------------------------------------------------------------------------

export function useLeaveRequests(params?: ListLeaveRequestsParams) {
  return useQuery({
    queryKey: hrKeys.leaveRequests(params),
    queryFn: () =>
      hrLeaveApi.listRequests(params as Record<string, unknown> ?? {}),
  })
}

export function useLeaveRequest(id: string) {
  return useQuery({
    queryKey: hrKeys.leaveRequest(id),
    queryFn: () => hrLeaveApi.getRequest(id),
    enabled: !!id,
    select: (data) => data.request,
  })
}

export function useLeaveBalance() {
  return useQuery({
    queryKey: hrKeys.leaveBalance(),
    queryFn: () => hrLeaveApi.getBalance(),
    select: (data) => data.balance,
  })
}

export function useEmployeeLeaveBalance(userId: string) {
  return useQuery({
    queryKey: hrKeys.employeeLeaveBalance(userId),
    queryFn: () => hrLeaveApi.getEmployeeBalance(userId),
    enabled: !!userId,
    select: (data) => data.balance,
  })
}

export function useLeaveTypes() {
  return useQuery({
    queryKey: hrKeys.leaveTypes(),
    queryFn: () => hrLeaveApi.listTypes(),
    staleTime: 5 * 60 * 1000, // 5 minutes
    select: (data) => data.types,
  })
}

export function useCreateLeaveRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateLeaveRequestInput) =>
      hrLeaveApi.createRequest(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['hr', 'leave', 'requests'] })
      qc.invalidateQueries({ queryKey: hrKeys.leaveBalance() })
      toast.success(i18next.t('api.hr.leave.submitted'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.leave.error.submit'))
    },
  })
}

export function useApproveLeaveRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data?: ApproveRejectInput }) =>
      hrLeaveApi.approveRequest(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['hr', 'leave', 'requests'] })
      qc.invalidateQueries({ queryKey: ['hr', 'leave', 'balance'] })
      toast.success(i18next.t('api.hr.leave.approved'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.leave.error.approve'))
    },
  })
}

export function useRejectLeaveRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data?: ApproveRejectInput }) =>
      hrLeaveApi.rejectRequest(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['hr', 'leave', 'requests'] })
      toast.success(i18next.t('api.hr.leave.rejected'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.leave.error.reject'))
    },
  })
}

export function useCancelLeaveRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => hrLeaveApi.cancelRequest(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['hr', 'leave', 'requests'] })
      qc.invalidateQueries({ queryKey: hrKeys.leaveBalance() })
      toast.success(i18next.t('api.hr.leave.cancelled'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.leave.error.cancel'))
    },
  })
}

export function useRecordSickLeave() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: RecordSickLeaveInput) =>
      hrLeaveApi.recordSickLeave(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['hr', 'leave', 'requests'] })
      toast.success(i18next.t('api.hr.leave.sickLeaveRecorded'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.leave.error.sickLeave'))
    },
  })
}

// ---------------------------------------------------------------------------
// Time tracking hooks
// ---------------------------------------------------------------------------

export function useWorkTimeStatus() {
  return useQuery({
    queryKey: hrKeys.workTimeStatus(),
    queryFn: () => hrTimeApi.getStatus(),
    refetchInterval: 300_000, // 5min — header clock is cosmetic
  })
}

export function useActiveShift() {
  return useQuery({
    queryKey: hrKeys.activeShift(),
    queryFn: () => hrTimeApi.getActiveShift(),
  })
}

export function useClockIn() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => hrTimeApi.clockIn(),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: hrKeys.workTimeStatus() })
      qc.invalidateQueries({ queryKey: hrKeys.activeShift() })
      showArbZGToast(data.compliance)
      toast.success(i18next.t('api.hr.time.clockedIn'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.time.error.clockIn'))
    },
  })
}

export function useClockOut() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => hrTimeApi.clockOut(),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: hrKeys.workTimeStatus() })
      qc.invalidateQueries({ queryKey: hrKeys.activeShift() })
      qc.invalidateQueries({ queryKey: ['hr', 'time', 'entries'] })
      qc.invalidateQueries({ queryKey: ['hr', 'time', 'summary'] })
      qc.invalidateQueries({ queryKey: hrKeys.timeBalance() })
      showArbZGToast(data.compliance)
      toast.success(i18next.t('api.hr.time.clockedOut'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.time.error.clockOut'))
    },
  })
}

export function useStartBreak() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => hrTimeApi.startBreak(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: hrKeys.activeShift() })
      qc.invalidateQueries({ queryKey: hrKeys.workTimeStatus() })
      toast.info(i18next.t('api.hr.time.breakStarted'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.time.error.breakStart'))
    },
  })
}

export function useEndBreak() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => hrTimeApi.endBreak(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: hrKeys.activeShift() })
      qc.invalidateQueries({ queryKey: hrKeys.workTimeStatus() })
      toast.info(i18next.t('api.hr.time.breakEnded'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.time.error.breakEnd'))
    },
  })
}

export function useWorkTimeEntries(params?: ListWorkTimeEntriesParams) {
  return useQuery({
    queryKey: hrKeys.workTimeEntries(params),
    queryFn: () =>
      hrTimeApi.listEntries(params as Record<string, unknown> ?? {}),
  })
}

export function useDailySummary(date: string) {
  return useQuery({
    queryKey: hrKeys.dailySummary(date),
    queryFn: () => hrTimeApi.getDailySummary(date),
    enabled: !!date,
    select: (data) => data.summary,
  })
}

export function useWeeklySummary(weekStart: string) {
  return useQuery({
    queryKey: hrKeys.weeklySummary(weekStart),
    queryFn: () => hrTimeApi.getWeeklySummary(weekStart),
    enabled: !!weekStart,
    select: (data) => data.summary,
  })
}

/** Cumulative flextime/overtime balance (Stundenkonto). */
export function useTimeBalance() {
  return useQuery({
    queryKey: hrKeys.timeBalance(),
    queryFn: () => hrTimeApi.getBalance(),
    select: (data) => data.balance,
  })
}

/** Billable projects (Kunde → Projekt) a time entry can be attributed to. */
export function useTimeProjects() {
  return useQuery({
    queryKey: hrKeys.timeProjects(),
    queryFn: () => hrTimeApi.listProjects(),
    staleTime: 5 * 60 * 1000,
    select: (data) => data.projects,
  })
}

/** Aggregated time-tracking analytics (KPIs, day trend, project + billable split). */
export function useTimeAnalytics(range: TimeAnalyticsRange) {
  return useQuery({
    queryKey: hrKeys.timeAnalytics(range),
    queryFn: () => hrTimeApi.getAnalytics(range),
    select: (data) => data.analytics,
  })
}

/** Team weekly time overview (manager view). */
export function useTeamTime(weekStart: string) {
  return useQuery({
    queryKey: hrKeys.teamTime(weekStart),
    queryFn: () => hrTimeApi.getTeamTime(weekStart),
    enabled: !!weekStart,
    select: (data) => data.rows,
  })
}

/** Current user's week submission status. */
export function useMyWeekStatus(weekStart: string) {
  return useQuery({
    queryKey: hrKeys.myWeekStatus(weekStart),
    queryFn: () => hrTimeApi.getMyWeekStatus(weekStart),
    enabled: !!weekStart,
    select: (data) => data.status,
  })
}

/** Submit the current week for manager approval. */
export function useSubmitWeek() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (weekStart: string) => hrTimeApi.submitWeek(weekStart),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['hr', 'time', 'weeks'] })
      qc.invalidateQueries({ queryKey: ['hr', 'time', 'team'] })
      toast.success(i18next.t('api.hr.time.weekSubmitted'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.time.error.weekSubmit'))
    },
  })
}

/** Approve a team member's submitted week. */
export function useApproveWeek() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ employeeId, weekStart }: { employeeId: string; weekStart: string }) =>
      hrTimeApi.approveWeek(employeeId, weekStart),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['hr', 'time', 'team'] })
      toast.success(i18next.t('api.hr.time.weekApproved'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.time.error.weekApprove'))
    },
  })
}

/** Reject a team member's submitted week. */
export function useRejectWeek() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ employeeId, weekStart, reason }: { employeeId: string; weekStart: string; reason: string }) =>
      hrTimeApi.rejectWeek(employeeId, weekStart, reason),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['hr', 'time', 'team'] })
      toast.success(i18next.t('api.hr.time.weekRejected'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.time.error.weekReject'))
    },
  })
}

/** Create a manual time entry (back-dated / corrected day). */
export function useCreateTimeEntry() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateManualEntryInput) => hrTimeApi.createEntry(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['hr', 'time', 'entries'] })
      qc.invalidateQueries({ queryKey: ['hr', 'time', 'summary'] })
      qc.invalidateQueries({ queryKey: hrKeys.timeBalance() })
      toast.success(i18next.t('api.hr.time.entryCreated'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.time.error.entryCreate'))
    },
  })
}

export function useSubmitCorrection() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: SubmitCorrectionInput) =>
      hrTimeApi.submitCorrection(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['hr', 'time', 'entries'] })
      toast.success(i18next.t('api.hr.time.correctionSubmitted'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.time.error.correction'))
    },
  })
}

export function useApproveCorrection() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => hrTimeApi.approveCorrection(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['hr', 'time', 'entries'] })
      toast.success(i18next.t('api.hr.time.correctionApproved'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.time.error.correctionApprove'))
    },
  })
}

// ---------------------------------------------------------------------------
// Absence calendar hooks
// ---------------------------------------------------------------------------

export function useAbsenceCalendar(params: AbsenceCalendarParams) {
  return useQuery({
    queryKey: hrKeys.absenceCalendar(params),
    queryFn: () =>
      hrAbsenceApi.getCalendar(params as unknown as Record<string, unknown>),
    enabled: !!params.start_date && !!params.end_date,
    select: (data) => data.entries,
  })
}

// ---------------------------------------------------------------------------
// Employee hooks
// ---------------------------------------------------------------------------

export function useEmployees(params?: ListEmployeesParams) {
  return useQuery({
    queryKey: hrKeys.employees(params),
    queryFn: () =>
      hrEmployeeApi.list(params as Record<string, unknown> ?? {}),
  })
}

export function useSelfProfile() {
  return useQuery({
    queryKey: hrKeys.selfProfile(),
    queryFn: () => hrEmployeeApi.getSelf(),
    select: (data) => data.employee,
  })
}

export function useEmployee(id: string) {
  return useQuery({
    queryKey: hrKeys.employee(id),
    queryFn: () => hrEmployeeApi.get(id),
    enabled: !!id,
    select: (data) => data.employee,
  })
}

export function useCreateEmployee() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateEmployeeInput) => hrEmployeeApi.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['hr', 'employees'] })
      toast.success(i18next.t('api.hr.employee.created'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.employee.error.create'))
    },
  })
}

export function useUpdateEmployee() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateEmployeeInput }) =>
      hrEmployeeApi.update(id, data),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['hr', 'employees'] })
      qc.invalidateQueries({ queryKey: hrKeys.employee(vars.id) })
      toast.success(i18next.t('api.hr.employee.updated'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.employee.error.update'))
    },
  })
}

export function useUpdateSelfProfile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: UpdateSelfProfileInput) =>
      hrEmployeeApi.updateSelf(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: hrKeys.selfProfile() })
      toast.success(i18next.t('api.hr.employee.profileUpdated'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.employee.error.profileUpdate'))
    },
  })
}

export function useEmployeeDocuments(employeeId: string) {
  return useQuery({
    queryKey: hrKeys.employeeDocuments(employeeId),
    queryFn: () => hrEmployeeApi.listDocuments(employeeId),
    enabled: !!employeeId,
    select: (data) => data.documents,
  })
}

export function useUploadEmployeeDocument() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      employeeId,
      data,
    }: {
      employeeId: string
      data: UploadDocumentInput
    }) => hrEmployeeApi.uploadDocument(employeeId, data),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({
        queryKey: hrKeys.employeeDocuments(vars.employeeId),
      })
      toast.success(i18next.t('api.hr.employee.documentUploaded'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.employee.error.documentUpload'))
    },
  })
}

export function useDocumentCategories(employeeId: string) {
  return useQuery({
    queryKey: hrKeys.documentCategories(employeeId),
    queryFn: () => hrEmployeeApi.listDocumentCategories(employeeId),
    enabled: !!employeeId,
    staleTime: 5 * 60 * 1000, // 5 minutes
    select: (data) => data.categories,
  })
}

// ---------------------------------------------------------------------------
// Settings hooks
// ---------------------------------------------------------------------------

export function useHRSettings() {
  return useQuery({
    queryKey: hrKeys.hrSettings(),
    queryFn: () => hrSettingsApi.get(),
    staleTime: 5 * 60 * 1000, // 5 minutes
    select: (data) => data.settings,
  })
}

export function useUpdateHRSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<HRSettings>) => hrSettingsApi.update(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: hrKeys.hrSettings() })
      toast.success(i18next.t('api.hr.settings.updated'))
    },
    onError: (err: Error) => {
      toast.error(err.message || i18next.t('api.hr.settings.error.update'))
    },
  })
}
