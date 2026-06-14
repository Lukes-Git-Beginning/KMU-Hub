/**
 * HR module TypeScript types matching proto/API responses.
 *
 * Covers leave management, time tracking (ArbZG), absence calendar,
 * employee profiles, HR documents, and settings.
 */

// ---------------------------------------------------------------------------
// Leave types
// ---------------------------------------------------------------------------

export interface LeaveType {
  id: string
  name: string
  key: string
  color: string
  deductsFromBalance: boolean
  requiresApproval: boolean
  requiresAuDocument: boolean
  isSystem: boolean
}

export type LeaveRequestStatus = 'pending' | 'approved' | 'rejected' | 'cancelled'

export interface LeaveRequest {
  id: string
  employeeId: string
  employeeName?: string
  leaveTypeId: string
  leaveType?: LeaveType
  startDate: string
  endDate: string
  isHalfDayStart: boolean
  halfDayPeriodStart?: 'morning' | 'afternoon'
  isHalfDayEnd: boolean
  halfDayPeriodEnd?: 'morning' | 'afternoon'
  totalDays: number
  reason?: string
  status: LeaveRequestStatus
  approvedBy?: string
  approvalComment?: string
  approvedAt?: string
  auDocumentRequired: boolean
  auDocumentFileId?: string
  createdAt: string
  updatedAt: string
}

export interface LeaveBalance {
  totalEntitlement: number
  proRataEntitlement: number
  carriedOver: number
  used: number
  remaining: number
  carryoverDeadline: string
  carryoverExpired: boolean
  year: number
}

// ---------------------------------------------------------------------------
// Time tracking
// ---------------------------------------------------------------------------

export type WorkTimeEntryStatus =
  | 'active'
  | 'completed'
  | 'correction_pending'
  | 'correction_approved'

export interface WorkTimeEntry {
  id: string
  employeeId: string
  clockIn: string
  clockOut?: string
  breakMinutes: number
  autoBreakDeducted: number
  netWorkMinutes?: number
  status: WorkTimeEntryStatus
  isCorrection: boolean
  originalEntryId?: string
  correctionReason?: string
  correctionApprovedBy?: string
  correctionApprovedAt?: string
  createdAt: string
  // Project / customer / service attribution (clockodo-style)
  projectId?: string
  projectName?: string
  customerName?: string
  activity?: string
  billable?: boolean
  note?: string
  isManual?: boolean
}

/** A billable project a work-time entry can be attributed to (Kunde → Projekt). */
export interface TimeProject {
  id: string
  name: string
  customerId?: string
  customerName?: string
  color: string
  billableDefault: boolean
}

export type TimeAnalyticsRange = 'week' | 'month'

/** Aggregated time-tracking analytics for the Auswertungen view. */
export interface TimeAnalytics {
  range: TimeAnalyticsRange
  periodStart: string
  periodEnd: string
  totalNetMinutes: number
  billableMinutes: number
  nonBillableMinutes: number
  overtimeMinutes: number
  targetMinutes: number
  workedDays: number
  dayTrend: { date: string; netMinutes: number; billableMinutes: number }[]
  byProject: { projectId: string; name: string; customerName: string; color: string; minutes: number }[]
}

export interface BreakEntry {
  id: string
  workTimeEntryId: string
  startTime: string
  endTime?: string
  durationMinutes?: number
}

export type ArbZGSeverity = 'none' | 'info' | 'warning' | 'error'

export interface WorkTimeStatus {
  isClockedIn: boolean
  isOnBreak: boolean
  currentShiftStart?: string
  currentBreakStart?: string
  todayTotalMinutes: number
  arbzgSeverity: ArbZGSeverity
}

export interface DailySummary {
  date: string
  totalWorkedMinutes: number
  totalBreakMinutes: number
  netWorkMinutes: number
  overtimeMinutes: number
  entryCount: number
}

export interface WeeklySummary {
  weekStart: string
  days: DailySummary[]
  totalWorkedMinutes: number
  totalBreakMinutes: number
  netWorkMinutes: number
  totalOvertimeMinutes: number
}

/**
 * Cumulative flextime / overtime balance (Gleitzeit-/Stundenkonto).
 * Positive = overtime credit, negative = deficit.
 */
export interface TimeBalance {
  balanceMinutes: number
  asOf: string
  periodStart: string
  targetWeeklyMinutes: number
}

export interface ArbZGComplianceResult {
  severity: ArbZGSeverity
  message: string
  restViolation: boolean
  restHoursActual?: number
  breakDeficit: number
}

// ---------------------------------------------------------------------------
// Absence calendar
// ---------------------------------------------------------------------------

export interface AbsenceEntry {
  employeeId: string
  employeeName: string
  department: string
  leaveTypeName: string
  leaveTypeKey: string
  color: string
  startDate: string
  endDate: string
  isHalfDayStart: boolean
  halfDayPeriodStart?: string
  isHalfDayEnd: boolean
  halfDayPeriodEnd?: string
}

// ---------------------------------------------------------------------------
// Employee profiles
// ---------------------------------------------------------------------------

export type ContractType = 'full_time' | 'part_time' | 'mini_job' | 'intern' | 'temporary'

export interface EmployeeProfile {
  id: string
  userId: string
  userName?: string
  userEmail?: string
  department?: string
  positionTitle?: string
  contractType: ContractType
  workDaysPerWeek: number
  annualLeaveDays: number
  managerUserId?: string
  managerName?: string
  startDate: string
  emergencyContactName?: string
  emergencyContactPhone?: string
  addressStreet?: string
  addressCity?: string
  addressPostalCode?: string
  addressCountry: string
  createdAt: string
  updatedAt: string
}

export interface EmployeeDocument {
  id: string
  employeeId: string
  categoryId: string
  categoryName?: string
  fileId: string
  fileName?: string
  uploadedBy: string
  uploaderName?: string
  notes?: string
  createdAt: string
}

export interface HRDocumentCategory {
  id: string
  name: string
  key: string
  visibility: 'hr_only' | 'manager' | 'employee'
  isSystem: boolean
}

// ---------------------------------------------------------------------------
// HR settings
// ---------------------------------------------------------------------------

export interface HRSettings {
  id: string
  auThresholdDays: number
  showAbsenceReason: boolean
  defaultAnnualLeaveDays: number
  timezone: string
}

// ---------------------------------------------------------------------------
// Input types for mutations
// ---------------------------------------------------------------------------

export interface CreateLeaveRequestInput {
  leaveTypeId: string
  startDate: string
  endDate: string
  isHalfDayStart?: boolean
  halfDayPeriodStart?: 'morning' | 'afternoon'
  isHalfDayEnd?: boolean
  halfDayPeriodEnd?: 'morning' | 'afternoon'
  reason?: string
}

export interface ApproveRejectInput {
  comment?: string
}

export interface RecordSickLeaveInput {
  startDate: string
  endDate: string
  reason?: string
}

export interface SubmitCorrectionInput {
  originalEntryId: string
  correctedClockIn: string
  correctedClockOut: string
  correctedBreakMinutes: number
  reason: string
}

export interface CreateManualEntryInput {
  clockIn: string
  clockOut: string
  breakMinutes: number
  projectId?: string
  activity?: string
  billable?: boolean
  note?: string
}

export interface CreateEmployeeInput {
  firstName: string
  lastName: string
  email: string
  phone?: string
  temporaryPassword: string
  roles: string[]
  department?: string
  positionTitle?: string
  contractType: ContractType
  workDaysPerWeek: number
  annualLeaveDays: number
  workloadPercent: number
  managerUserId?: string
  startDate: string
  location?: string
  addressStreet?: string
  addressCity?: string
  addressPostalCode?: string
  addressCountry: string
  emergencyContactName?: string
  emergencyContactPhone?: string
  sendInviteEmail?: boolean
}

export interface UpdateEmployeeInput {
  department?: string
  positionTitle?: string
  contractType?: string
  workDaysPerWeek?: number
  annualLeaveDays?: number
  managerUserId?: string
  startDate?: string
  emergencyContactName?: string
  emergencyContactPhone?: string
  addressStreet?: string
  addressCity?: string
  addressPostalCode?: string
  addressCountry?: string
}

export interface UpdateSelfProfileInput {
  emergencyContactName?: string
  emergencyContactPhone?: string
  addressStreet?: string
  addressCity?: string
  addressPostalCode?: string
  addressCountry?: string
}

export interface UploadDocumentInput {
  categoryId: string
  fileId: string
  notes?: string
}

// ---------------------------------------------------------------------------
// List params (query filters)
// ---------------------------------------------------------------------------

export interface ListLeaveRequestsParams {
  status?: LeaveRequestStatus
  employee_id?: string
  leave_type_id?: string
  start_date?: string
  end_date?: string
  limit?: number
  offset?: number
}

export interface ListWorkTimeEntriesParams {
  employee_id?: string
  status?: WorkTimeEntryStatus
  start_date?: string
  end_date?: string
  limit?: number
  offset?: number
}

export interface ListEmployeesParams {
  department?: string
  search?: string
  limit?: number
  offset?: number
}

export interface AbsenceCalendarParams {
  start_date: string
  end_date: string
  department?: string
}
