/**
 * HR API client -- typed fetch wrapper for all HR HTTP endpoints.
 *
 * Follows the same auth/refresh/offline pattern as finance-client.ts.
 * Gateway routes: /api/v1/hr/*
 */
import type {
  LeaveRequest,
  LeaveRequestStatus,
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
  TeamTimeRow,
  MyWeekStatus,
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
  TimeCategory,
  CreateTimeCategoryInput,
  UpdateTimeCategoryInput,
  TimeTemplate,
  CreateTimeTemplateInput,
} from './hr-types'
import { authenticatedRequest } from './utils/authenticatedFetch'
import { normalizeWireTimestamps } from './wire-time'

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
    userName:              raw.user_name            ?? raw.userName            ?? ([raw.first_name ?? raw.firstName, raw.last_name ?? raw.lastName].filter(Boolean).join(' ') || undefined),
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
    phone:                 raw.phone,
    mobile:                raw.mobile,
    location:              raw.location,
    workload:              raw.workload,
    initials:              raw.initials,
    avatarUrl:             raw.avatar_url           ?? raw.avatarUrl            ?? null,
    status:                raw.status,
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
    status:                  data.status,
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
// Leave — wire adapters
//
// The gateway serialises Leave* proto messages via protojson with
// UseProtoNames:true (snake_case), UseEnumNumbers:true (status / half-day
// period as integers), EmitUnpopulated:false (zero values omitted). Decimal
// amounts (total_days, entitlement, carried_over, used, remaining) are proto
// strings, not numbers. response.Proto(msg) is a flat object, response.
// ProtoList(msgs) is a bare JSON array, ad-hoc map envelopes wrap in
// {leave_requests:[...], total} / {leave_request:{...}, au_document_required}.
// These adapters normalise all of that to the camelCase FE types.
// ---------------------------------------------------------------------------

const PROTO_INT_TO_LEAVE_STATUS: Record<number, LeaveRequestStatus> = {
  1: 'pending',
  2: 'approved',
  3: 'rejected',
  4: 'cancelled',
}

function normaliseLeaveStatus(raw: unknown): LeaveRequestStatus {
  if (typeof raw === 'number') return PROTO_INT_TO_LEAVE_STATUS[raw] ?? 'pending'
  if (raw === 'pending' || raw === 'approved' || raw === 'rejected' || raw === 'cancelled') return raw
  return 'pending'
}

const PROTO_INT_TO_HALF_DAY_PERIOD: Record<number, 'morning' | 'afternoon'> = {
  1: 'morning',
  2: 'afternoon',
}

function normaliseHalfDayPeriod(raw: unknown): 'morning' | 'afternoon' | undefined {
  if (typeof raw === 'number') return PROTO_INT_TO_HALF_DAY_PERIOD[raw]
  if (raw === 'morning' || raw === 'afternoon') return raw
  return undefined
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function adaptLeaveRequest(raw: Record<string, any>): LeaveRequest {
  return {
    id:                 (raw.id                                            ?? '') as string,
    employeeId:         (raw.employee_id         ?? raw.employeeId         ?? '') as string,
    employeeName:       (raw.employee_name       ?? raw.employeeName       ?? undefined) as string | undefined,
    leaveTypeId:        (raw.leave_type_id       ?? raw.leaveTypeId        ?? '') as string,
    leaveTypeName:      (raw.leave_type_name     ?? raw.leaveTypeName      ?? undefined) as string | undefined,
    leaveTypeColor:     (raw.leave_type_color    ?? raw.leaveTypeColor     ?? undefined) as string | undefined,
    startDate:          (raw.start_date          ?? raw.startDate          ?? '') as string,
    endDate:            (raw.end_date            ?? raw.endDate            ?? '') as string,
    isHalfDayStart:     (raw.is_half_day_start   ?? raw.isHalfDayStart     ?? false) as boolean,
    halfDayPeriodStart: normaliseHalfDayPeriod(raw.half_day_period_start ?? raw.halfDayPeriodStart),
    isHalfDayEnd:       (raw.is_half_day_end     ?? raw.isHalfDayEnd       ?? false) as boolean,
    halfDayPeriodEnd:   normaliseHalfDayPeriod(raw.half_day_period_end ?? raw.halfDayPeriodEnd),
    totalDays:          Number(raw.total_days ?? raw.totalDays ?? 0),
    reason:             (raw.reason                                        ?? undefined) as string | undefined,
    status:             normaliseLeaveStatus(raw.status),
    approvedBy:         (raw.approved_by         ?? raw.approvedBy         ?? undefined) as string | undefined,
    approvalComment:    (raw.approval_comment    ?? raw.approvalComment    ?? undefined) as string | undefined,
    approvedAt:         (raw.approved_at         ?? raw.approvedAt         ?? undefined) as string | undefined,
    auDocumentRequired: (raw.au_document_required ?? raw.auDocumentRequired ?? false) as boolean,
    auDocumentFileId:   (raw.au_document_file_id ?? raw.auDocumentFileId   ?? undefined) as string | undefined,
    createdAt:          (raw.created_at          ?? raw.createdAt          ?? '') as string,
    updatedAt:          (raw.updated_at          ?? raw.updatedAt          ?? '') as string,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function adaptLeaveBalance(raw: Record<string, any>): LeaveBalance {
  const totalEntitlement = Number(raw.entitlement ?? raw.totalEntitlement ?? 0)
  // Backend gap: pro-rata entitlement is not modelled server-side — fall back
  // to the full entitlement (no consumer computes a distinct value today).
  const proRataEntitlement = Number(raw.proRataEntitlement ?? totalEntitlement)
  const carryoverDeadline = (raw.carryover_expires_at ?? raw.carryoverDeadline ?? '') as string
  return {
    totalEntitlement,
    proRataEntitlement,
    carriedOver:      Number(raw.carried_over ?? raw.carriedOver ?? 0),
    used:              Number(raw.used ?? 0),
    remaining:         Number(raw.remaining ?? 0),
    carryoverDeadline,
    // Backend gap: no expiry flag on the wire — derive it from the deadline.
    carryoverExpired:  carryoverDeadline ? new Date(carryoverDeadline) < new Date() : false,
    year:              Number(raw.year ?? new Date().getFullYear()),
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function adaptLeaveType(raw: Record<string, any>): LeaveType {
  return {
    id:                 (raw.id                                              ?? '') as string,
    name:               (raw.name                                            ?? '') as string,
    key:                (raw.key                                             ?? '') as string,
    color:              (raw.color                                           ?? '#6b7280') as string,
    deductsFromBalance: (raw.deducts_from_balance ?? raw.deductsFromBalance   ?? false) as boolean,
    requiresApproval:   (raw.requires_approval    ?? raw.requiresApproval     ?? false) as boolean,
    requiresAuDocument: (raw.requires_au_document ?? raw.requiresAuDocument   ?? false) as boolean,
    isSystem:           (raw.is_system            ?? raw.isSystem            ?? false) as boolean,
  }
}

/**
 * Converts a CreateLeaveRequestInput to the snake_case body
 * createLeaveRequestHTTPReq expects. half_day_period_* stay the FE's
 * 'morning'/'afternoon' strings — the gateway itself maps them to the Proto
 * enum, unlike every other field on the wire.
 */
function toSnakeCaseLeaveRequestBody(data: CreateLeaveRequestInput): Record<string, unknown> {
  return {
    leave_type_id:          data.leaveTypeId,
    start_date:              data.startDate,
    end_date:                data.endDate,
    is_half_day_start:       data.isHalfDayStart ?? false,
    half_day_period_start:   data.halfDayPeriodStart,
    is_half_day_end:         data.isHalfDayEnd ?? false,
    half_day_period_end:     data.halfDayPeriodEnd,
    reason:                  data.reason,
  }
}

/** Converts a RecordSickLeaveInput to the snake_case body recordSickLeaveHTTPReq
 *  expects. Note: the backend field is `notes`, not `reason`. */
function toSnakeCaseSickLeaveBody(data: RecordSickLeaveInput): Record<string, unknown> {
  return {
    start_date: data.startDate,
    end_date:   data.endDate,
    notes:      data.reason,
  }
}

export const hrLeaveApi = {
  async createRequest(data: CreateLeaveRequestInput): Promise<{ request: LeaveRequest }> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>('/api/v1/hr/leave/requests', {
      method: 'POST',
      body: JSON.stringify(toSnakeCaseLeaveRequestBody(data)),
    })
    return { request: adaptLeaveRequest(raw) }
  },

  async listRequests(params: Record<string, unknown> = {}): Promise<{ requests: LeaveRequest[]; total: number }> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>(`/api/v1/hr/leave/requests${qs(params)}`)
    const list = (raw.leave_requests ?? raw.requests ?? []) as Record<string, unknown>[]
    const requests = list.map((r) => adaptLeaveRequest(r as Record<string, any>))
    return { requests, total: (raw.total ?? requests.length) as number }
  },

  async getRequest(id: string): Promise<{ request: LeaveRequest }> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>(`/api/v1/hr/leave/requests/${id}`)
    return { request: adaptLeaveRequest(raw) }
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

  async getBalance(): Promise<{ balance: LeaveBalance }> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>('/api/v1/hr/leave/balance')
    return { balance: adaptLeaveBalance(raw) }
  },

  async getEmployeeBalance(userId: string): Promise<{ balance: LeaveBalance }> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>(`/api/v1/hr/leave/balance/${userId}`)
    return { balance: adaptLeaveBalance(raw) }
  },

  async listTypes(): Promise<{ types: LeaveType[] }> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>('/api/v1/hr/leave/types')
    const list = (Array.isArray(raw) ? raw : (raw.types ?? [])) as Record<string, unknown>[]
    return { types: list.map((t) => adaptLeaveType(t as Record<string, any>)) }
  },

  async recordSickLeave(data: RecordSickLeaveInput): Promise<{ request: LeaveRequest }> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>('/api/v1/hr/leave/sick', {
      method: 'POST',
      body: JSON.stringify(toSnakeCaseSickLeaveBody(data)),
    })
    return { request: adaptLeaveRequest((raw.leave_request ?? raw) as Record<string, any>) }
  },
}

// ---------------------------------------------------------------------------
// Time tracking — wire adapters
//
// The HR gRPC service serialises via encoding/json on pb.go structs, so the
// wire shape is snake_case. Timestamps arrive as `{seconds, nanos}` proto
// objects rather than ISO strings. These adapters normalise both issues so the
// frontend types (all camelCase, all ISO strings) are satisfied.
//
// Fields present in the proto but NOT in the FE type (e.g. tenant_id, breaks
// array on WorkTimeEntry) are dropped silently. Fields absent from the backend
// (project_name, customer_name, activity, billable, note, is_manual) default
// to undefined/false — they are backend gaps documented in the Backend-Lücken
// section of the echt-schaltung report.
// ---------------------------------------------------------------------------

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function adaptWorkTimeEntry(raw: Record<string, any>): WorkTimeEntry {
  // Timestamps come in as {seconds, nanos} — normalizeWireTimestamps converts them
  const normalised = normalizeWireTimestamps(raw) as Record<string, unknown>
  return {
    id:                     (normalised.id                                                ?? '') as string,
    employeeId:             (normalised.employee_id      ?? normalised.employeeId         ?? '') as string,
    clockIn:                (normalised.clock_in         ?? normalised.clockIn            ?? '') as string,
    clockOut:               (normalised.clock_out        ?? normalised.clockOut           ?? undefined) as string | undefined,
    breakMinutes:           (normalised.break_minutes    ?? normalised.breakMinutes        ?? 0) as number,
    autoBreakDeducted:      (normalised.auto_break_deducted ?? normalised.autoBreakDeducted ?? 0) as number,
    netWorkMinutes:         (normalised.net_work_minutes ?? normalised.netWorkMinutes     ?? undefined) as number | undefined,
    status:                 (normalised.status                                            ?? 'completed') as WorkTimeEntry['status'],
    isCorrection:           (normalised.is_correction    ?? normalised.isCorrection       ?? false) as boolean,
    originalEntryId:        (normalised.original_entry_id ?? normalised.originalEntryId  ?? undefined) as string | undefined,
    correctionReason:       (normalised.correction_reason ?? normalised.correctionReason  ?? undefined) as string | undefined,
    correctionApprovedBy:   (normalised.correction_approved_by ?? normalised.correctionApprovedBy ?? undefined) as string | undefined,
    correctionApprovedAt:   (normalised.correction_approved_at ?? normalised.correctionApprovedAt ?? undefined) as string | undefined,
    createdAt:              (normalised.created_at       ?? normalised.createdAt          ?? '') as string,
    // Project/customer attribution — backend gap: proto WorkTimeEntry only has
    // project_id (migration 000212). project_name / customer_name / activity /
    // billable / note / is_manual are NOT stored in the hr_work_time_entries
    // table and will always be undefined until a backend migration adds them.
    projectId:              (normalised.project_id       ?? normalised.projectId          ?? undefined) as string | undefined,
    projectName:            (normalised.project_name     ?? normalised.projectName        ?? undefined) as string | undefined,
    customerName:           (normalised.customer_name    ?? normalised.customerName       ?? undefined) as string | undefined,
    activity:               (normalised.activity                                          ?? undefined) as string | undefined,
    billable:               (normalised.billable                                          ?? undefined) as boolean | undefined,
    note:                   (normalised.note                                              ?? undefined) as string | undefined,
    isManual:               (normalised.is_manual        ?? normalised.isManual           ?? undefined) as boolean | undefined,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function adaptDailySummary(raw: Record<string, any>): DailySummary {
  return {
    date:                (raw.date                                                        ?? '') as string,
    // Proto field is total_work_minutes; mock used totalWorkedMinutes — accept both
    totalWorkedMinutes:  (raw.total_work_minutes  ?? raw.total_worked_minutes ?? raw.totalWorkedMinutes  ?? 0) as number,
    totalBreakMinutes:   (raw.total_break_minutes ?? raw.totalBreakMinutes   ?? 0) as number,
    netWorkMinutes:      (raw.net_work_minutes    ?? raw.netWorkMinutes      ?? 0) as number,
    overtimeMinutes:     (raw.overtime_minutes    ?? raw.overtimeMinutes     ?? 0) as number,
    // entryCount is not in the proto — backend gap, default 0
    entryCount:          (raw.entry_count         ?? raw.entryCount          ?? 0) as number,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function adaptWeeklySummary(raw: Record<string, any>): WeeklySummary {
  // Proto field is daily_summaries; mock used days
  const rawDays = (raw.daily_summaries ?? raw.days ?? []) as Record<string, unknown>[]
  return {
    weekStart:           (raw.week_start            ?? raw.weekStart            ?? '') as string,
    totalWorkedMinutes:  (raw.total_work_minutes    ?? raw.total_worked_minutes ?? raw.totalWorkedMinutes   ?? 0) as number,
    totalBreakMinutes:   (raw.total_break_minutes   ?? raw.totalBreakMinutes    ?? 0) as number,
    netWorkMinutes:      (raw.net_work_minutes      ?? raw.netWorkMinutes       ?? 0) as number,
    // Proto field is overtime_minutes; mock used totalOvertimeMinutes
    totalOvertimeMinutes:(raw.overtime_minutes      ?? raw.total_overtime_minutes ?? raw.totalOvertimeMinutes ?? 0) as number,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    days:                rawDays.map((d) => adaptDailySummary(d as Record<string, any>)),
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function adaptWorkTimeStatus(raw: Record<string, any>): WorkTimeStatus {
  return {
    isClockedIn:         (raw.is_clocked_in         ?? raw.isClockedIn         ?? false) as boolean,
    isOnBreak:           (raw.is_on_break            ?? raw.isOnBreak           ?? false) as boolean,
    currentShiftStart:   (raw.current_shift_start    ?? raw.currentShiftStart   ?? undefined) as string | undefined,
    currentBreakStart:   (raw.current_break_start    ?? raw.currentBreakStart   ?? undefined) as string | undefined,
    todayTotalMinutes:   (raw.today_total_minutes    ?? raw.todayTotalMinutes   ?? 0) as number,
    arbzgSeverity:       (raw.arbzg_severity         ?? raw.arbzgSeverity       ?? 'none') as WorkTimeStatus['arbzgSeverity'],
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function adaptTimeBalance(raw: Record<string, any>): TimeBalance {
  return {
    balanceMinutes:      (raw.balance_minutes        ?? raw.balanceMinutes      ?? 0) as number,
    asOf:                (raw.as_of                  ?? raw.asOf                ?? '') as string,
    periodStart:         (raw.period_start           ?? raw.periodStart         ?? '') as string,
    targetWeeklyMinutes: (raw.target_weekly_minutes  ?? raw.targetWeeklyMinutes ?? 2400) as number,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function adaptTimeProject(raw: Record<string, any>): TimeProject {
  return {
    id:              (raw.id                                        ?? '') as string,
    name:            (raw.name                                      ?? '') as string,
    customerId:      (raw.customer_id  ?? raw.customerId            ?? undefined) as string | undefined,
    customerName:    (raw.customer_name ?? raw.customerName         ?? undefined) as string | undefined,
    color:           (raw.color                                     ?? '#6b7280') as string,
    billableDefault: (raw.billable_default ?? raw.billableDefault   ?? false) as boolean,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function adaptTimeAnalytics(raw: Record<string, any>): TimeAnalytics {
  // Backend returns total_minutes / target_minutes / overtime_minutes /
  // avg_daily_minutes / day_trend / by_project (snake_case, flat).
  // The FE TimeAnalytics type uses camelCase + additional computed fields.
  const dayTrend = ((raw.day_trend ?? raw.dayTrend ?? []) as Record<string, unknown>[]).map((d) => ({
    date:            (d.date                                    ?? '') as string,
    netMinutes:      (d.net_minutes      ?? d.netMinutes        ?? 0) as number,
    billableMinutes: (d.billable_minutes ?? d.billableMinutes   ?? 0) as number,
  }))

  const byProject = ((raw.by_project ?? raw.byProject ?? []) as Record<string, unknown>[]).map((p) => ({
    projectId:    (p.project_id   ?? p.projectId    ?? '') as string,
    name:         (p.name                           ?? '') as string,
    customerName: (p.customer_name ?? p.customerName ?? '') as string,
    color:        (p.color                          ?? '#6b7280') as string,
    minutes:      (p.minutes                        ?? 0) as number,
  }))

  const totalMinutes    = (raw.total_minutes    ?? raw.totalNetMinutes    ?? 0) as number
  const billable        = (raw.billable_minutes ?? raw.billableMinutes    ?? 0) as number
  const target          = (raw.target_minutes   ?? raw.targetMinutes      ?? 0) as number
  const overtime        = (raw.overtime_minutes ?? raw.overtimeMinutes    ?? 0) as number
  const workedDays      = (raw.worked_days      ?? raw.workedDays         ?? 0) as number
  const rangeVal        = (raw.range                                      ?? 'week') as TimeAnalyticsRange
  const periodStart     = (raw.period_start     ?? raw.periodStart        ?? '') as string
  const periodEnd       = (raw.period_end       ?? raw.periodEnd          ?? '') as string

  return {
    range:              rangeVal,
    periodStart,
    periodEnd,
    totalNetMinutes:    totalMinutes,
    billableMinutes:    billable,
    nonBillableMinutes: Math.max(0, totalMinutes - billable),
    overtimeMinutes:    overtime,
    targetMinutes:      target,
    workedDays,
    dayTrend,
    byProject,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function adaptTeamTimeRow(raw: Record<string, any>): TeamTimeRow {
  return {
    employeeId:      (raw.employee_id     ?? raw.employeeId     ?? '') as string,
    name:            (raw.name                                  ?? '') as string,
    department:      (raw.department                            ?? undefined) as string | undefined,
    weekMinutes:     (raw.week_minutes    ?? raw.weekMinutes    ?? 0) as number,
    targetMinutes:   (raw.target_minutes  ?? raw.targetMinutes  ?? 0) as number,
    overtimeMinutes: (raw.overtime_minutes ?? raw.overtimeMinutes ?? 0) as number,
    clockedIn:       (raw.clocked_in      ?? raw.clockedIn      ?? false) as boolean,
    weekStatus:      (raw.week_status     ?? raw.weekStatus     ?? 'open') as TeamTimeRow['weekStatus'],
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function adaptMyWeekStatus(raw: Record<string, any>): MyWeekStatus {
  // Backend returns week_approval object + total_minutes + target_minutes at root.
  // week_approval has: week_start, status, submitted_at, approved_by, rejection_reason
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const wa = (raw.week_approval ?? raw) as Record<string, any>
  return {
    weekStart:       (wa.week_start      ?? wa.weekStart      ?? raw.week_start  ?? '') as string,
    status:          (wa.status                               ?? 'open') as MyWeekStatus['status'],
    submittedAt:     (wa.submitted_at    ?? wa.submittedAt    ?? undefined) as string | undefined,
    approvedBy:      (wa.approved_by     ?? wa.approvedBy     ?? undefined) as string | undefined,
    rejectionReason: (wa.rejection_reason ?? wa.rejectionReason ?? undefined) as string | undefined,
  }
}

// ---------------------------------------------------------------------------
// Time tracking
// ---------------------------------------------------------------------------

export const hrTimeApi = {
  async clockIn() {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>('/api/v1/hr/time/clock-in', { method: 'POST' })
    // Backend returns entry directly or wrapped; compliance may be in severity/arbzg_message
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const entryRaw = (raw.entry ?? raw) as Record<string, any>
    const entry = adaptWorkTimeEntry(entryRaw)
    const compliance: ArbZGComplianceResult = raw.compliance ?? {
      severity: raw.severity ?? 'none',
      message: raw.arbzg_message ?? '',
      restViolation: false,
      breakDeficit: 0,
    }
    return { entry, compliance }
  },

  async clockOut() {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>('/api/v1/hr/time/clock-out', { method: 'POST' })
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const entryRaw = (raw.entry ?? raw) as Record<string, any>
    const entry = adaptWorkTimeEntry(entryRaw)
    const compliance: ArbZGComplianceResult = raw.compliance ?? {
      severity: raw.severity ?? 'none',
      message: raw.arbzg_message ?? '',
      restViolation: false,
      breakDeficit: 0,
    }
    return { entry, compliance }
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

  async getStatus(): Promise<WorkTimeStatus> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>('/api/v1/hr/time/status')
    return adaptWorkTimeStatus(raw)
  },

  async listEntries(params: Record<string, unknown> = {}): Promise<{ entries: WorkTimeEntry[]; total: number }> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>(`/api/v1/hr/time/entries${qs(params)}`)
    const entries = ((raw.entries ?? []) as Record<string, unknown>[]).map(
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (e) => adaptWorkTimeEntry(e as Record<string, any>),
    )
    return { entries, total: (raw.total ?? entries.length) as number }
  },

  async getDailySummary(date: string): Promise<{ summary: DailySummary }> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>(`/api/v1/hr/time/summary/daily?date=${date}`)
    // Backend returns the DailySummary directly (response.JSON(w, 200, resp.Summary))
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const summaryRaw = (raw.summary ?? raw) as Record<string, any>
    return { summary: adaptDailySummary(summaryRaw) }
  },

  async getWeeklySummary(weekStart: string): Promise<{ summary: WeeklySummary }> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>(`/api/v1/hr/time/summary/weekly?week_start=${weekStart}`)
    // Backend returns the WeeklySummary directly (response.JSON(w, 200, resp.Summary))
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const summaryRaw = (raw.summary ?? raw) as Record<string, any>
    return { summary: adaptWeeklySummary(summaryRaw) }
  },

  async getBalance(): Promise<{ balance: TimeBalance }> {
    // Backend returns flat fields: balance_minutes, target_weekly_minutes, period_start, as_of
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>('/api/v1/hr/time/balance')
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const balanceRaw = (raw.balance ?? raw) as Record<string, any>
    return { balance: adaptTimeBalance(balanceRaw) }
  },

  async listProjects(): Promise<{ projects: TimeProject[] }> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>('/api/v1/hr/time/projects')
    const projects = ((raw.projects ?? []) as Record<string, unknown>[]).map(
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (p) => adaptTimeProject(p as Record<string, any>),
    )
    return { projects }
  },

  async getAnalytics(range: TimeAnalyticsRange): Promise<{ analytics: TimeAnalytics }> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>(`/api/v1/hr/time/analytics?range=${range}`)
    // Backend returns flat fields at root level (no wrapping key)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const analyticsRaw = (raw.analytics ?? raw) as Record<string, any>
    return { analytics: adaptTimeAnalytics(analyticsRaw) }
  },

  async getTeamTime(weekStart: string): Promise<{ rows: TeamTimeRow[] }> {
    // Backend returns `team` key (not `rows`)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>(`/api/v1/hr/time/team?week_start=${weekStart}`)
    const items = ((raw.team ?? raw.rows ?? []) as Record<string, unknown>[]).map(
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (r) => adaptTeamTimeRow(r as Record<string, any>),
    )
    return { rows: items }
  },

  async getMyWeekStatus(weekStart: string): Promise<{ status: MyWeekStatus }> {
    // Backend returns { week_approval, total_minutes, target_minutes }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>(`/api/v1/hr/time/weeks/status?week_start=${weekStart}`)
    return { status: adaptMyWeekStatus(raw) }
  },

  async submitWeek(weekStart: string): Promise<{ status: MyWeekStatus }> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>('/api/v1/hr/time/weeks/submit', {
      method: 'POST',
      body: JSON.stringify({ week_start: weekStart }),
    })
    return { status: adaptMyWeekStatus(raw) }
  },

  approveWeek(employeeId: string, weekStart: string) {
    return request<{ ok: boolean }>('/api/v1/hr/time/weeks/approve', {
      method: 'POST',
      body: JSON.stringify({ employee_id: employeeId, week_start: weekStart }),
    })
  },

  rejectWeek(employeeId: string, weekStart: string, reason: string) {
    return request<{ ok: boolean }>('/api/v1/hr/time/weeks/reject', {
      method: 'POST',
      body: JSON.stringify({ employee_id: employeeId, week_start: weekStart, rejection_reason: reason }),
    })
  },

  async createEntry(data: CreateManualEntryInput): Promise<{ entry: WorkTimeEntry }> {
    // Convert camelCase input to snake_case for the backend
    const body = {
      clock_in:      data.clockIn,
      clock_out:     data.clockOut,
      break_minutes: data.breakMinutes,
      project_id:    data.projectId,
      activity:      data.activity,
      billable:      data.billable,
      note:          data.note,
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>('/api/v1/hr/time/entries', {
      method: 'POST',
      body: JSON.stringify(body),
    })
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return { entry: adaptWorkTimeEntry((raw.entry ?? raw) as Record<string, any>) }
  },

  async submitCorrection(data: SubmitCorrectionInput): Promise<{ entry: WorkTimeEntry }> {
    // Convert camelCase input to the snake_case the backend expects
    const body = {
      original_entry_id:        data.originalEntryId,
      corrected_clock_in:       data.correctedClockIn,
      corrected_clock_out:      data.correctedClockOut,
      corrected_break_minutes:  data.correctedBreakMinutes,
      reason:                   data.reason,
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>('/api/v1/hr/time/corrections', {
      method: 'POST',
      body: JSON.stringify(body),
    })
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return { entry: adaptWorkTimeEntry((raw.correction ?? raw.entry ?? raw) as Record<string, any>) }
  },

  async approveCorrection(id: string): Promise<{ entry: WorkTimeEntry }> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<any>(
      `/api/v1/hr/time/corrections/${id}/approve`,
      { method: 'POST' },
    )
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return { entry: adaptWorkTimeEntry((raw.correction ?? raw.entry ?? raw) as Record<string, any>) }
  },
}

// ---------------------------------------------------------------------------
// Absences
// ---------------------------------------------------------------------------

/**
 * Adapts a raw backend AbsenceEntry (snake_case wire shape per OpenAPI spec)
 * to the camelCase AbsenceEntry type used by the FE.
 *
 * The backend emits:
 *   employee_id, employee_name, department, start_date, end_date,
 *   leave_type_name, leave_type_color
 *
 * The MSW demo handler emits camelCase directly (masks the mismatch in demo
 * mode). Using `?? ` chaining (snake_case first, camelCase fallback) keeps
 * both modes working without changes to mock handlers.
 *
 * Note: leave_type_key, isHalfDayStart/End, halfDayPeriodStart/End are not
 * part of the OpenAPI AbsenceEntry schema and default to safe fallback values.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function adaptAbsenceEntry(raw: Record<string, any>): AbsenceEntry {
  return {
    employeeId:         raw.employee_id          ?? raw.employeeId         ?? '',
    employeeName:       raw.employee_name        ?? raw.employeeName       ?? '',
    department:         raw.department                                     ?? '',
    leaveTypeName:      raw.leave_type_name      ?? raw.leaveTypeName      ?? '',
    leaveTypeKey:       raw.leave_type_key       ?? raw.leaveTypeKey       ?? '',
    color:              raw.leave_type_color     ?? raw.color              ?? '#6b7280',
    startDate:          raw.start_date           ?? raw.startDate          ?? '',
    endDate:            raw.end_date             ?? raw.endDate            ?? '',
    isHalfDayStart:     raw.is_half_day_start    ?? raw.isHalfDayStart    ?? false,
    halfDayPeriodStart: raw.half_day_period_start ?? raw.halfDayPeriodStart,
    isHalfDayEnd:       raw.is_half_day_end      ?? raw.isHalfDayEnd      ?? false,
    halfDayPeriodEnd:   raw.half_day_period_end  ?? raw.halfDayPeriodEnd,
  }
}

export const hrAbsenceApi = {
  async getCalendar(params: Record<string, unknown>) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await request<{ entries: any[] }>(
      `/api/v1/hr/absences/calendar${qs(params)}`,
    )
    return {
      entries: (raw.entries ?? []).map(adaptAbsenceEntry),
    }
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
// Time categories
// ---------------------------------------------------------------------------

export const hrTimeCategoryApi = {
  list() {
    return request<{ categories: TimeCategory[] }>('/api/v1/hr/time/categories')
  },

  create(data: CreateTimeCategoryInput) {
    return request<{ category: TimeCategory }>('/api/v1/hr/time/categories', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },

  update(id: string, data: UpdateTimeCategoryInput) {
    return request<{ category: TimeCategory }>(`/api/v1/hr/time/categories/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  },

  delete(id: string) {
    return request<Record<string, never>>(`/api/v1/hr/time/categories/${id}`, {
      method: 'DELETE',
    })
  },
}

// ---------------------------------------------------------------------------
// Time templates
// ---------------------------------------------------------------------------

export const hrTimeTemplateApi = {
  list() {
    return request<{ templates: TimeTemplate[] }>('/api/v1/hr/time/templates')
  },

  create(data: CreateTimeTemplateInput) {
    return request<{ template: TimeTemplate }>('/api/v1/hr/time/templates', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },

  delete(id: string) {
    return request<Record<string, never>>(`/api/v1/hr/time/templates/${id}`, {
      method: 'DELETE',
    })
  },
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

/**
 * Adapts a raw HRSettings payload to the typed, camelCase HRSettings.
 *
 * The gateway serialises the proto HRSettings via encoding/json on the pb.go
 * struct, so the wire is snake_case (au_threshold_days, work_hours_per_day…);
 * MSW demo handlers emit snake_case too. Uses `?? ` chaining: snake_case first,
 * camelCase fallback.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function adaptHRSettings(raw: Record<string, any>): HRSettings {
  return {
    id:                     raw.id                                                  ?? '',
    auThresholdDays:        raw.au_threshold_days        ?? raw.auThresholdDays      ?? 3,
    showAbsenceReason:      raw.show_absence_reason      ?? raw.showAbsenceReason    ?? false,
    defaultAnnualLeaveDays: raw.default_annual_leave_days ?? raw.defaultAnnualLeaveDays ?? 25,
    timezone:               raw.timezone                                            ?? 'Europe/Zurich',
    workHoursPerDay:        raw.work_hours_per_day       ?? raw.workHoursPerDay      ?? 8,
    maxDailyHours:          raw.max_daily_hours          ?? raw.maxDailyHours        ?? 10,
    breakAfterHours:        raw.break_after_hours        ?? raw.breakAfterHours      ?? 6,
  }
}

/** Converts a Partial<HRSettings> to the snake_case body updateHRSettingsHTTPReq expects. */
function toSnakeCaseHRSettingsBody(data: Partial<HRSettings>): Record<string, unknown> {
  const body: Record<string, unknown> = {}
  if (data.auThresholdDays !== undefined) body.au_threshold_days = data.auThresholdDays
  if (data.showAbsenceReason !== undefined) body.show_absence_reason = data.showAbsenceReason
  if (data.defaultAnnualLeaveDays !== undefined) body.default_annual_leave_days = data.defaultAnnualLeaveDays
  if (data.timezone !== undefined) body.timezone = data.timezone
  if (data.workHoursPerDay !== undefined) body.work_hours_per_day = data.workHoursPerDay
  if (data.maxDailyHours !== undefined) body.max_daily_hours = data.maxDailyHours
  if (data.breakAfterHours !== undefined) body.break_after_hours = data.breakAfterHours
  return body
}

export const hrSettingsApi = {
  async get(): Promise<{ settings: HRSettings }> {
    const res = await request<{ settings: Record<string, unknown> }>('/api/v1/hr/settings')
    return { settings: adaptHRSettings(res.settings ?? {}) }
  },

  async update(data: Partial<HRSettings>): Promise<{ settings: HRSettings }> {
    const res = await request<{ settings: Record<string, unknown> }>('/api/v1/hr/settings', {
      method: 'PUT',
      body: JSON.stringify(toSnakeCaseHRSettingsBody(data)),
    })
    return { settings: adaptHRSettings(res.settings ?? {}) }
  },
}
