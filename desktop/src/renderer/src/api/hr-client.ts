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
  TimeBalance,
  TimeProject,
  TimeAnalytics,
  TimeAnalyticsRange,
  CreateManualEntryInput,
  ArbZGComplianceResult,
  AbsenceEntry,
  EmployeeProfile,
  ContractType,
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

// ---------------------------------------------------------------------------
// ContractType adapter
//
// The backend serialises Proto enums as integers (encoding/json on pb.go
// structs): 0=unspecified, 1=full_time, 2=part_time, 3=mini_job,
// 4=intern, 5=temporary. The gateway also emits the Go-generated constant
// names (CONTRACT_FULL_TIME etc.) in some edge cases. Demo-mode MSW
// handlers already return camelCase strings. The adapter accepts all three
// formats and normalises to the canonical ContractType union.
// ---------------------------------------------------------------------------

const PROTO_INT_TO_CONTRACT: Record<number, ContractType> = {
  1: 'full_time',
  2: 'part_time',
  3: 'mini_job',
  4: 'intern',
  5: 'temporary',
}

const PROTO_STRING_TO_CONTRACT: Record<string, ContractType> = {
  CONTRACT_FULL_TIME: 'full_time',
  CONTRACT_PART_TIME: 'part_time',
  CONTRACT_MINI_JOB: 'mini_job',
  CONTRACT_INTERN: 'intern',
  CONTRACT_TEMPORARY: 'temporary',
  // legacy FE values — mapped to nearest Proto equivalent
  praktikum: 'intern',
  freelance: 'temporary',
  // already-canonical pass-through
  full_time: 'full_time',
  part_time: 'part_time',
  mini_job: 'mini_job',
  intern: 'intern',
  temporary: 'temporary',
}

function normaliseContractType(raw: unknown): ContractType {
  if (typeof raw === 'number') {
    return PROTO_INT_TO_CONTRACT[raw] ?? 'full_time'
  }
  if (typeof raw === 'string') {
    return PROTO_STRING_TO_CONTRACT[raw] ?? 'full_time'
  }
  return 'full_time'
}

/**
 * Adapts a raw backend or demo-mode employee payload to a typed EmployeeProfile.
 *
 * Tolerates both snake_case (real backend via encoding/json on pb.go) and
 * camelCase (MSW demo handlers). Uses `?? ` chaining: snake_case first,
 * camelCase fallback.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function adaptEmployee(raw: Record<string, any>): EmployeeProfile {
  return {
    id:                    raw.id                                               ?? '',
    userId:                raw.user_id              ?? raw.userId               ?? raw.id ?? '',
    userName:              raw.user_name            ?? raw.userName,
    userEmail:             raw.user_email           ?? raw.userEmail,
    department:            raw.department,
    positionTitle:         raw.position_title       ?? raw.positionTitle,
    contractType:          normaliseContractType(raw.contract_type ?? raw.contractType),
    workDaysPerWeek:       raw.work_days_per_week   ?? raw.workDaysPerWeek      ?? 5,
    annualLeaveDays:       raw.annual_leave_days    ?? raw.annualLeaveDays      ?? 0,
    managerUserId:         raw.manager_user_id      ?? raw.managerUserId,
    managerName:           raw.manager_name         ?? raw.managerName,
    startDate:             raw.start_date           ?? raw.startDate            ?? '',
    emergencyContactName:  raw.emergency_contact_name  ?? raw.emergencyContactName,
    emergencyContactPhone: raw.emergency_contact_phone ?? raw.emergencyContactPhone,
    addressStreet:         raw.address_street       ?? raw.addressStreet,
    addressCity:           raw.address_city         ?? raw.addressCity,
    addressPostalCode:     raw.address_postal_code  ?? raw.addressPostalCode,
    addressCountry:        raw.address_country      ?? raw.addressCountry       ?? '',
    createdAt:             raw.created_at           ?? raw.createdAt            ?? '',
    updatedAt:             raw.updated_at           ?? raw.updatedAt            ?? '',
  }
}

/**
 * Converts a CreateEmployeeInput / UpdateEmployeeInput to the snake_case
 * body the gateway expects (contract_type as string slug, not integer).
 */
function toSnakeCaseEmployeeBody(data: CreateEmployeeInput): Record<string, unknown> {
  return {
    first_name:              data.firstName,
    last_name:               data.lastName,
    email:                   data.email,
    phone:                   data.phone,
    temporary_password:      data.temporaryPassword,
    roles:                   data.roles,
    department:              data.department,
    position_title:          data.positionTitle,
    contract_type:           data.contractType,
    work_days_per_week:      data.workDaysPerWeek,
    annual_leave_days:       data.annualLeaveDays,
    workload_percent:        data.workloadPercent,
    manager_user_id:         data.managerUserId,
    start_date:              data.startDate,
    location:                data.location,
    address_street:          data.addressStreet,
    address_city:            data.addressCity,
    address_postal_code:     data.addressPostalCode,
    address_country:         data.addressCountry,
    emergency_contact_name:  data.emergencyContactName,
    emergency_contact_phone: data.emergencyContactPhone,
    send_invite_email:       data.sendInviteEmail,
  }
}

function toSnakeCaseUpdateBody(data: UpdateEmployeeInput): Record<string, unknown> {
  return {
    department:              data.department,
    position_title:          data.positionTitle,
    contract_type:           data.contractType,
    work_days_per_week:      data.workDaysPerWeek,
    annual_leave_days:       data.annualLeaveDays,
    manager_user_id:         data.managerUserId,
    start_date:              data.startDate,
    emergency_contact_name:  data.emergencyContactName,
    emergency_contact_phone: data.emergencyContactPhone,
    address_street:          data.addressStreet,
    address_city:            data.addressCity,
    address_postal_code:     data.addressPostalCode,
    address_country:         data.addressCountry,
  }
}

function toSnakeCaseSelfBody(data: UpdateSelfProfileInput): Record<string, unknown> {
  return {
    emergency_contact_name:  data.emergencyContactName,
    emergency_contact_phone: data.emergencyContactPhone,
    address_street:          data.addressStreet,
    address_city:            data.addressCity,
    address_postal_code:     data.addressPostalCode,
    address_country:         data.addressCountry,
  }
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

  getBalance() {
    return request<{ balance: TimeBalance }>('/api/v1/hr/time/balance')
  },

  listProjects() {
    return request<{ projects: TimeProject[] }>('/api/v1/hr/time/projects')
  },

  getAnalytics(range: TimeAnalyticsRange) {
    return request<{ analytics: TimeAnalytics }>(`/api/v1/hr/time/analytics?range=${range}`)
  },

  createEntry(data: CreateManualEntryInput) {
    return request<{ entry: WorkTimeEntry }>('/api/v1/hr/time/entries', {
      method: 'POST',
      body: JSON.stringify(data),
    })
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
  async list(params: Record<string, unknown> = {}) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<{ employees: any[]; total: number }>(
      `/api/v1/hr/employees${qs(params)}`,
    )
    return {
      employees: (raw.employees ?? []).map(adaptEmployee),
      total: raw.total ?? 0,
    }
  },

  async create(data: CreateEmployeeInput) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<{ employee: any }>('/api/v1/hr/employees', {
      method: 'POST',
      body: JSON.stringify(toSnakeCaseEmployeeBody(data)),
    })
    return { employee: adaptEmployee(raw.employee) }
  },

  async getSelf() {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<{ employee: any }>('/api/v1/hr/employees/me')
    return { employee: adaptEmployee(raw.employee) }
  },

  async updateSelf(data: UpdateSelfProfileInput) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<{ employee: any }>('/api/v1/hr/employees/me', {
      method: 'PUT',
      body: JSON.stringify(toSnakeCaseSelfBody(data)),
    })
    return { employee: adaptEmployee(raw.employee) }
  },

  async get(id: string) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<{ employee: any }>(`/api/v1/hr/employees/${id}`)
    return { employee: adaptEmployee(raw.employee) }
  },

  async update(id: string, data: UpdateEmployeeInput) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<{ employee: any }>(
      `/api/v1/hr/employees/${id}`,
      { method: 'PUT', body: JSON.stringify(toSnakeCaseUpdateBody(data)) },
    )
    return { employee: adaptEmployee(raw.employee) }
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
