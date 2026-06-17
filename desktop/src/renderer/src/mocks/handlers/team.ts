import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { EMPLOYEES, DEPARTMENTS, COMPANY } from '../mock-db'
import { daysAgo, daysFromNow } from '../data/date-helpers'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Map mock-db employees to HR API shape
// ---------------------------------------------------------------------------

function toHrEmployee(e: (typeof EMPLOYEES)[number]) {
  const fullName = `${e.firstName} ${e.lastName}`
  return {
    // canonical camelCase fields matching EmployeeProfile interface (hr-types.ts)
    id: `usr-${e.id}`,
    userId: `usr-${e.id}`,
    userName: fullName,
    userEmail: e.email,
    positionTitle: e.jobTitle,
    contractType: e.contractType,
    department: e.department,
    departmentId: e.departmentId,
    workDaysPerWeek: 5,
    annualLeaveDays: 30,
    startDate: e.joinDate,
    managerUserId: e.managerId ? `usr-${e.managerId}` : null,
    // additional fields used by team-module components
    firstName: e.firstName,
    lastName: e.lastName,
    initials: e.initials,
    email: e.email,
    phone: e.phone,
    mobile: e.mobile,
    role: e.role,
    workload: e.workload,
    location: e.location,
    status: e.status,
    avatarUrl: e.avatar || null,
  }
}

const hrEmployees = EMPLOYEES.map(toHrEmployee)

// ---------------------------------------------------------------------------
// Leave data
// ---------------------------------------------------------------------------

const leaveTypes = [
  { id: 'lt-001', name: 'Urlaub', color: '#3b82f6', requires_approval: true, max_days: 30 },
  { id: 'lt-002', name: 'Krankheit', color: '#ef4444', requires_approval: false, max_days: null },
  { id: 'lt-003', name: 'Sonderurlaub', color: '#f59e0b', requires_approval: true, max_days: 5 },
  { id: 'lt-004', name: 'Elternzeit', color: '#10b981', requires_approval: true, max_days: null },
  { id: 'lt-005', name: 'Homeoffice', color: '#8b5cf6', requires_approval: false, max_days: null },
]

const leaveRequests = [
  { id: 'lr-001', user_id: IDS.users.felix, user_name: 'Felix Krause', type: 'Urlaub', type_id: 'lt-001', start_date: daysFromNow(14), end_date: daysFromNow(21), days: 6, status: 'pending', note: 'Familienurlaub', created_at: daysAgo(2) },
  { id: 'lr-002', user_id: IDS.users.lena, user_name: 'Lena Braun', type: 'Krankheit', type_id: 'lt-002', start_date: daysAgo(1), end_date: daysAgo(1), days: 1, status: 'approved', note: 'Erkaeltet', created_at: daysAgo(1) },
  { id: 'lr-003', user_id: IDS.users.julia, user_name: 'Julia Hofmann', type: 'Urlaub', type_id: 'lt-001', start_date: daysFromNow(30), end_date: daysFromNow(37), days: 6, status: 'approved', note: '', created_at: daysAgo(10) },
  { id: 'lr-004', user_id: IDS.users.sophie, user_name: 'Sophie Lang', type: 'Sonderurlaub', type_id: 'lt-003', start_date: daysFromNow(5), end_date: daysFromNow(5), days: 1, status: 'approved', note: 'Umzug', created_at: daysAgo(5) },
  { id: 'lr-005', user_id: IDS.users.markus, user_name: 'Markus Weber', type: 'Homeoffice', type_id: 'lt-005', start_date: daysFromNow(1), end_date: daysFromNow(1), days: 1, status: 'approved', note: '', created_at: daysAgo(0) },
  // Self-service user (Stefan Vogel = /me) — own requests so the self-service tab is not empty.
  { id: 'lr-006', user_id: IDS.users.stefan, user_name: 'Stefan Vogel', type: 'Urlaub', type_id: 'lt-001', start_date: daysFromNow(45), end_date: daysFromNow(52), days: 6, status: 'pending', note: 'Sommerurlaub', created_at: daysAgo(1) },
  { id: 'lr-007', user_id: IDS.users.stefan, user_name: 'Stefan Vogel', type: 'Homeoffice', type_id: 'lt-005', start_date: daysAgo(7), end_date: daysAgo(7), days: 1, status: 'approved', note: 'Fokus-Tag', created_at: daysAgo(8) },
  { id: 'lr-008', user_id: IDS.users.stefan, user_name: 'Stefan Vogel', type: 'Sonderurlaub', type_id: 'lt-003', start_date: daysAgo(20), end_date: daysAgo(20), days: 1, status: 'approved', note: 'Behördengang', created_at: daysAgo(22) },
]

const LEAVE_TYPE_NAMES: Record<string, string> = {
  'lt-001': 'Urlaub',
  'lt-002': 'Krankheit',
  'lt-003': 'Sonderurlaub',
  'lt-004': 'Elternzeit',
  'lt-005': 'Homeoffice',
}

const leaveBalance = {
  user_id: IDS.users.stefan,
  year: new Date().getFullYear(),
  entitlement: 30,
  taken: 8,
  planned: 5,
  remaining: 17,
  sick_days: 2,
  special_leave: 1,
}

// ---------------------------------------------------------------------------
// Absence calendar
// ---------------------------------------------------------------------------

const absences = [
  { user_id: IDS.users.lena, user_name: 'Lena Braun', type: 'Krankheit', start_date: daysAgo(1), end_date: daysFromNow(1), color: '#ef4444' },
  { user_id: IDS.users.markus, user_name: 'Markus Weber', type: 'Homeoffice', start_date: daysFromNow(0), end_date: daysFromNow(0), color: '#8b5cf6' },
  { user_id: IDS.users.felix, user_name: 'Felix Krause', type: 'Urlaub', start_date: daysFromNow(14), end_date: daysFromNow(21), color: '#3b82f6' },
  { user_id: IDS.users.julia, user_name: 'Julia Hofmann', type: 'Urlaub', start_date: daysFromNow(30), end_date: daysFromNow(37), color: '#3b82f6' },
  { user_id: IDS.users.sophie, user_name: 'Sophie Lang', type: 'Sonderurlaub', start_date: daysFromNow(5), end_date: daysFromNow(5), color: '#f59e0b' },
]

// Map German leave-type names to API keys understood by the calendar + dashboard widget.
const LEAVE_TYPE_KEYS: Record<string, string> = {
  Urlaub: 'vacation',
  Krankheit: 'sick',
  Sonderurlaub: 'special',
  Homeoffice: 'home',
  Elternzeit: 'parent',
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const teamHandlers = [
  // Employee list
  http.get(`${API}/api/v1/hr/employees`, ({ request }) => {
    const url = new URL(request.url)
    const dept = url.searchParams.get('department_id')
    const search = url.searchParams.get('q')

    let filtered = [...hrEmployees]
    if (dept) {
      filtered = filtered.filter((e) => e.departmentId === dept)
    }
    if (search) {
      const q = search.toLowerCase()
      filtered = filtered.filter(
        (e) =>
          (e.firstName ?? '').toLowerCase().includes(q) ||
          (e.lastName ?? '').toLowerCase().includes(q) ||
          (e.email ?? '').toLowerCase().includes(q),
      )
    }

    return HttpResponse.json({ employees: filtered, total: filtered.length })
  }),

  // Current user profile
  http.get(`${API}/api/v1/hr/employees/me`, () => {
    const me = hrEmployees.find((e) => e.id === IDS.users.stefan)
    return HttpResponse.json({ employee: me })
  }),

  // Employee detail
  http.get(`${API}/api/v1/hr/employees/:id`, ({ params }) => {
    const emp = hrEmployees.find((e) => e.id === params.id)
    if (!emp) {
      return HttpResponse.json({ error: 'Employee not found' }, { status: 404 })
    }
    return HttpResponse.json({ employee: emp })
  }),

  // Leave requests
  http.get(`${API}/api/v1/hr/leave/requests`, ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('status')
    const userId = url.searchParams.get('user_id')

    let filtered = [...leaveRequests]
    if (status) {
      filtered = filtered.filter((r) => r.status === status)
    }
    if (userId) {
      filtered = filtered.filter((r) => r.user_id === userId)
    }

    return HttpResponse.json({ requests: filtered, total: filtered.length })
  }),

  // Create leave request — pushes into the same in-memory list the GET reads,
  // so a newly submitted request shows up immediately in the self-service tab.
  http.post(`${API}/api/v1/hr/leave/requests`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const typeId = String(body.leave_type_id ?? body.leaveTypeId ?? 'lt-001')
    const start = String(body.start_date ?? body.startDate ?? '')
    const end = String(body.end_date ?? body.endDate ?? start)
    const dayMs = 86400000
    const days =
      start && end ? Math.max(1, Math.round((new Date(end).getTime() - new Date(start).getTime()) / dayMs) + 1) : 1
    const newReq = {
      id: `lr-${Date.now()}`,
      user_id: IDS.users.stefan,
      user_name: 'Stefan Vogel',
      type: LEAVE_TYPE_NAMES[typeId] ?? 'Urlaub',
      type_id: typeId,
      start_date: start,
      end_date: end,
      days,
      status: 'pending',
      note: String(body.reason ?? ''),
      created_at: new Date().toISOString().slice(0, 10),
    }
    leaveRequests.unshift(newReq)
    return HttpResponse.json({ request: newReq }, { status: 201 })
  }),

  // Leave balance
  http.get(`${API}/api/v1/hr/leave/balance`, () => {
    return HttpResponse.json({ balance: leaveBalance })
  }),

  // Leave types
  http.get(`${API}/api/v1/hr/leave/types`, () => {
    return HttpResponse.json({ types: leaveTypes })
  }),

  // hr/time/* (status, active, entries, summaries) are owned by the hr.ts
  // mock handler — single source of truth for the zeiterfassung module.
  // Removed the duplicate idle/zero handlers here so hr.ts's richer demo
  // data (clocked-in day, real entries) is served everywhere.

  // Absence calendar — returns AbsenceEntry[] under `entries`
  // (matches hr-client `{ entries }` + useAbsenceCalendar `select: data => data.entries`).
  http.get(`${API}/api/v1/hr/absences/calendar`, ({ request }) => {
    const url = new URL(request.url)
    const start = url.searchParams.get('start_date')
    const end = url.searchParams.get('end_date')
    const dept = url.searchParams.get('department')
    let entries = absences.map((a) => {
      const emp = hrEmployees.find((e) => e.id === a.user_id)
      return {
        employeeId: a.user_id,
        employeeName: a.user_name,
        department: emp?.department ?? '',
        leaveTypeName: a.type,
        leaveTypeKey: LEAVE_TYPE_KEYS[a.type] ?? 'vacation',
        color: a.color,
        startDate: a.start_date,
        endDate: a.end_date,
        isHalfDayStart: false,
        isHalfDayEnd: false,
      }
    })
    // Overlap filter: keep entries that intersect the requested [start, end] window.
    if (start && end) entries = entries.filter((e) => e.startDate <= end && e.endDate >= start)
    if (dept) entries = entries.filter((e) => e.department === dept)
    return HttpResponse.json({ entries })
  }),

  // HR settings
  http.get(`${API}/api/v1/hr/settings`, () => {
    return HttpResponse.json({
      settings: {
        work_hours_per_day: 8,
        work_days_per_week: 5,
        default_vacation_days: 30,
        overtime_enabled: true,
        time_tracking_mandatory: true,
        break_rules: [
          { after_hours: 6, break_min: 30 },
          { after_hours: 9, break_min: 45 },
        ],
        company: {
          name: COMPANY.name,
          address: COMPANY.address,
        },
        departments: DEPARTMENTS.map((d) => ({ id: d.id, name: d.name, color: d.color })),
      },
    })
  }),
]
