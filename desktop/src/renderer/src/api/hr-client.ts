/**
 * HR API client -- typed fetch wrapper for all HR HTTP endpoints.
 *
 * Follows the same auth/refresh/offline pattern as finance-client.ts.
 * Gateway routes: /api/v1/hr/*
 */
import type {
  LeaveRequest,
  LeaveType,
  LeaveBalance,
  WorkTimeEntry,
  BreakEntry,
  WorkTimeStatus,
  DailySummary,
  WeeklySummary,
  ArbZGComplianceResult,
  AbsenceEntry,
  EmployeeProfile,
  EmployeeDocument,
  HRDocumentCategory,
  HRSettings,
  CreateLeaveRequestInput,
  CreateEmployeeInput,
  ApproveRejectInput,
  RecordSickLeaveInput,
  SubmitCorrectionInput,
  UpdateEmployeeInput,
  UpdateSelfProfileInput,
  UploadDocumentInput,
} from './hr-types'
import { authenticatedRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const method = (options.method ?? 'GET').toUpperCase()
  let body: unknown = undefined
  if (options.body !== undefined) {
    if (typeof options.body === 'string') {
      try { body = JSON.parse(options.body) } catch { body = options.body }
    } else {
      body = options.body
    }
  }
  return authenticatedRequest<T>({ method, path, body })
}

function qs(params: Record<string, unknown>): string {
  const entries = Object.entries(params).filter(
    ([, v]) => v !== undefined && v !== '' && v !== null,
  )
  if (entries.length === 0) return ''
  return (
    '?' +
    new URLSearchParams(entries.map(([k, v]) => [k, String(v)])).toString()
  )
}

// ---------------------------------------------------------------------------
// Leave
// ---------------------------------------------------------------------------

export const hrLeaveApi = {
  createRequest(data: CreateLeaveRequestInput) {
    return request<{ request: LeaveRequest }>('/api/v1/hr/leave/requests', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },

  listRequests(params: Record<string, unknown> = {}) {
    return request<{ requests: LeaveRequest[]; total: number }>(
      `/api/v1/hr/leave/requests${qs(params)}`,
    )
  },

  getRequest(id: string) {
    return request<{ request: LeaveRequest }>(`/api/v1/hr/leave/requests/${id}`)
  },

  approveRequest(id: string, data?: ApproveRejectInput) {
    return request<Record<string, never>>(
      `/api/v1/hr/leave/requests/${id}/approve`,
      { method: 'POST', body: JSON.stringify(data ?? {}) },
    )
  },

  rejectRequest(id: string, data?: ApproveRejectInput) {
    return request<Record<string, never>>(
      `/api/v1/hr/leave/requests/${id}/reject`,
      { method: 'POST', body: JSON.stringify(data ?? {}) },
    )
  },

  cancelRequest(id: string) {
    return request<Record<string, never>>(
      `/api/v1/hr/leave/requests/${id}/cancel`,
      { method: 'POST' },
    )
  },

  getBalance() {
    return request<{ balance: LeaveBalance }>('/api/v1/hr/leave/balance')
  },

  getEmployeeBalance(userId: string) {
    return request<{ balance: LeaveBalance }>(
      `/api/v1/hr/leave/balance/${userId}`,
    )
  },

  listTypes() {
    return request<{ types: LeaveType[] }>('/api/v1/hr/leave/types')
  },

  recordSickLeave(data: RecordSickLeaveInput) {
    return request<{ request: LeaveRequest }>('/api/v1/hr/leave/sick', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },
}

// ---------------------------------------------------------------------------
// Time tracking
// ---------------------------------------------------------------------------

export const hrTimeApi = {
  clockIn() {
    return request<{ entry: WorkTimeEntry; compliance: ArbZGComplianceResult }>(
      '/api/v1/hr/time/clock-in',
      { method: 'POST' },
    )
  },

  clockOut() {
    return request<{ entry: WorkTimeEntry; compliance: ArbZGComplianceResult }>(
      '/api/v1/hr/time/clock-out',
      { method: 'POST' },
    )
  },

  startBreak() {
    return request<{ break_entry: BreakEntry }>('/api/v1/hr/time/break/start', {
      method: 'POST',
    })
  },

  endBreak() {
    return request<{ break_entry: BreakEntry }>('/api/v1/hr/time/break/end', {
      method: 'POST',
    })
  },

  getActiveShift() {
    return request<{ entry?: WorkTimeEntry; breaks: BreakEntry[] }>(
      '/api/v1/hr/time/active',
    )
  },

  getStatus() {
    return request<WorkTimeStatus>('/api/v1/hr/time/status')
  },

  listEntries(params: Record<string, unknown> = {}) {
    return request<{ entries: WorkTimeEntry[]; total: number }>(
      `/api/v1/hr/time/entries${qs(params)}`,
    )
  },

  getDailySummary(date: string) {
    return request<{ summary: DailySummary }>(
      `/api/v1/hr/time/summary/daily?date=${date}`,
    )
  },

  getWeeklySummary(weekStart: string) {
    return request<{ summary: WeeklySummary }>(
      `/api/v1/hr/time/summary/weekly?week_start=${weekStart}`,
    )
  },

  submitCorrection(data: SubmitCorrectionInput) {
    return request<{ entry: WorkTimeEntry }>('/api/v1/hr/time/corrections', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },

  approveCorrection(id: string) {
    return request<{ entry: WorkTimeEntry }>(
      `/api/v1/hr/time/corrections/${id}/approve`,
      { method: 'POST' },
    )
  },
}

// ---------------------------------------------------------------------------
// Absences
// ---------------------------------------------------------------------------

export const hrAbsenceApi = {
  getCalendar(params: Record<string, unknown>) {
    return request<{ entries: AbsenceEntry[] }>(
      `/api/v1/hr/absences/calendar${qs(params)}`,
    )
  },
}

// ---------------------------------------------------------------------------
// Employees
// ---------------------------------------------------------------------------

export const hrEmployeeApi = {
  list(params: Record<string, unknown> = {}) {
    return request<{ employees: EmployeeProfile[]; total: number }>(
      `/api/v1/hr/employees${qs(params)}`,
    )
  },

  create(data: CreateEmployeeInput) {
    return request<{ employee: EmployeeProfile }>('/api/v1/hr/employees', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },

  getSelf() {
    return request<{ employee: EmployeeProfile }>('/api/v1/hr/employees/me')
  },

  updateSelf(data: UpdateSelfProfileInput) {
    return request<{ employee: EmployeeProfile }>('/api/v1/hr/employees/me', {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  },

  get(id: string) {
    return request<{ employee: EmployeeProfile }>(
      `/api/v1/hr/employees/${id}`,
    )
  },

  update(id: string, data: UpdateEmployeeInput) {
    return request<{ employee: EmployeeProfile }>(
      `/api/v1/hr/employees/${id}`,
      { method: 'PUT', body: JSON.stringify(data) },
    )
  },

  listDocuments(id: string) {
    return request<{ documents: EmployeeDocument[] }>(
      `/api/v1/hr/employees/${id}/documents`,
    )
  },

  uploadDocument(id: string, data: UploadDocumentInput) {
    return request<{ document: EmployeeDocument }>(
      `/api/v1/hr/employees/${id}/documents`,
      { method: 'POST', body: JSON.stringify(data) },
    )
  },

  listDocumentCategories(id: string) {
    return request<{ categories: HRDocumentCategory[] }>(
      `/api/v1/hr/employees/${id}/documents/categories`,
    )
  },
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

export const hrSettingsApi = {
  get() {
    return request<{ settings: HRSettings }>('/api/v1/hr/settings')
  },

  update(data: Partial<HRSettings>) {
    return request<{ settings: HRSettings }>('/api/v1/hr/settings', {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  },
}
