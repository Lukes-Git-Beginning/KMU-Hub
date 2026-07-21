import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS, displayUserName } from '../data/shared-ids'
import { EMPLOYEES, DEPARTMENTS, COMPANY } from '../mock-db'
import { daysAgo, daysFromNow } from '../data/date-helpers'
import { USER_ROLE_ASSIGNMENTS, getDemoSessionUserId, roleSummary } from '../data/rbac'
import { writeAuditEvent } from '../data/audit-events'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Map mock-db employees to HR API shape
// ---------------------------------------------------------------------------

/**
 * mock-db EMPLOYEES indices drifted from the canonical IDS.users mapping
 * (mock-db e4 is "Laura Neumann", but IDS says usr-e4 = Julia). Until the big
 * seed sweep aligns mock-db, resolve userId by NAME against the shared ids so
 * roles (R-2), leave balances and module grants attach to the right person.
 */
export const NAME_TO_USER_ID: Record<string, string> = {
  'Stefan Vogel': IDS.users.stefan,
  'Markus Weber': IDS.users.markus,
  'Thomas Keller': IDS.users.thomas,
  // R-4: real mock-db people mapped onto their role-assignment slots so the
  // reporting-line QA cases exist (Thomas Meier = it_admin usr-e3 via
  // fallback; Sarah Beck = the manager demo user).
  'Sarah Beck': IDS.users.sarah,
  'Julia Hofmann': IDS.users.julia,
  'Laura Neumann': IDS.users.laura,
  'Felix Krause': IDS.users.felix,
  'Sarah Müller': IDS.users.sarah,
  'Jan Schäfer': IDS.users.jan,
  'Lena Braun': IDS.users.lena,
  'David Wagner': IDS.users.david,
  'Lisa Becker': IDS.users.lisa,
  'Anna Großmann': IDS.users.anna,
  'Michael Schulz': IDS.users.michael,
  'Sophie Lang': IDS.users.sophie,
  'Nina Fischer': IDS.users.nina,
  'Max Steiner': IDS.users.max,
  'Elena Richter': IDS.users.elena,
}

function toHrEmployee(e: (typeof EMPLOYEES)[number]) {
  const fullName = `${e.firstName} ${e.lastName}`
  const canonicalUserId = NAME_TO_USER_ID[fullName] ?? `usr-${e.id}`
  // R-4: resolve the manager to the SAME canonical id space as userId —
  // an index-based `usr-<managerId>` would break the reporting-line chain
  // wherever NAME_TO_USER_ID remaps a person (e.g. Sarah Beck → usr-e7).
  const manager = e.managerId ? EMPLOYEES.find((m) => m.id === e.managerId) : undefined
  const managerUserId = manager
    ? (NAME_TO_USER_ID[`${manager.firstName} ${manager.lastName}`] ?? `usr-${manager.id}`)
    : null
  return {
    // canonical camelCase fields matching EmployeeProfile interface (hr-types.ts)
    id: `usr-${e.id}`,
    userId: canonicalUserId,
    userName: fullName,
    userEmail: e.email,
    positionTitle: e.jobTitle,
    contractType: e.contractType,
    department: e.department,
    departmentId: e.departmentId,
    workDaysPerWeek: 5,
    annualLeaveDays: 30,
    startDate: e.joinDate,
    managerUserId,
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

// ---------------------------------------------------------------------------
// Personnel documents (Personalakte tab) — rich shape with status/expiry,
// generated from the real hrEmployees so the file shows the same people as
// the rest of the team module.
// ---------------------------------------------------------------------------

function surname(full: string): string {
  const parts = full.trim().split(/\s+/)
  return parts[parts.length - 1] || full
}

const personnelDocuments: Array<Record<string, unknown>> = hrEmployees.slice(0, 9).flatMap((emp, i) => {
  const ln = surname(emp.userName)
  const docs: Array<Record<string, unknown>> = [
    {
      id: `pdoc-${emp.id}-av`, employeeId: emp.id, employeeName: emp.userName,
      title: 'Arbeitsvertrag', category: 'vertrag',
      fileName: `AV_${ln}.pdf`, fileSize: `${230 + i * 4} KB`,
      uploadedAt: daysAgo(420 + i * 12), uploadedBy: 'Laura Weber', status: 'aktuell',
    },
  ]
  if (i % 3 === 0) {
    const expired = i === 3
    const soon = i === 0
    docs.push({
      id: `pdoc-${emp.id}-cert`, employeeId: emp.id, employeeName: emp.userName,
      title: 'Zertifikat', category: 'zertifikat',
      fileName: `Cert_${ln}.pdf`, fileSize: `${95 + i * 3} KB`,
      uploadedAt: daysAgo(300), uploadedBy: emp.userName,
      expiresAt: expired ? daysAgo(40) : soon ? daysFromNow(25) : daysFromNow(700),
      status: expired ? 'abgelaufen' : soon ? 'bald_ablaufend' : 'aktuell',
      notes: expired ? 'Zertifikat abgelaufen — Erneuerung nötig!' : soon ? 'Erneuerung bald fällig' : undefined,
    })
  }
  return docs
})

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
    // Session-aware (R-4): self-service must show the signed-in demo user's
    // own file, not always Stefan's.
    const sessionUserId = getDemoSessionUserId()
    const me =
      hrEmployees.find((e) => e.userId === sessionUserId) ??
      hrEmployees.find((e) => e.id === IDS.users.stefan)
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

  // Update employee — merges the patch into the in-memory record (e.g. status
  // for deactivation). The list GET re-reads the same array, so the change shows.
  http.put(`${API}/api/v1/hr/employees/:id`, async ({ params, request }) => {
    const emp = hrEmployees.find((e) => e.id === params.id) as Record<string, unknown> | undefined
    if (!emp) {
      return HttpResponse.json({ error: 'Employee not found' }, { status: 404 })
    }
    const body = (await request.json()) as Record<string, unknown>
    Object.assign(emp, body)
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

  // Personnel documents (Personalakte tab) — all documents across employees.
  http.get(`${API}/api/v1/hr/personnel-documents`, () => {
    return HttpResponse.json({ documents: personnelDocuments })
  }),

  // Upload a personnel document — pushes into the same list the GET reads.
  http.post(`${API}/api/v1/hr/personnel-documents`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const doc = {
      id: `pdoc-${Date.now()}`,
      employeeId: String(body.employeeId ?? 'usr-e1'),
      employeeName: String(body.employeeName ?? 'Stefan Vogel'),
      title: String(body.title ?? 'Neues Dokument'),
      category: String(body.category ?? 'sonstiges'),
      fileName: String(body.fileName ?? 'dokument.pdf'),
      fileSize: String(body.fileSize ?? '120 KB'),
      uploadedAt: new Date().toISOString().slice(0, 10),
      uploadedBy: 'Stefan Vogel',
      expiresAt: body.expiresAt ? String(body.expiresAt) : undefined,
      status: 'aktuell',
    }
    personnelDocuments.unshift(doc)
    return HttpResponse.json({ document: doc }, { status: 201 })
  }),

  // ---------------------------------------------------------------------------
  // R-4: Offboard employee — full cascade
  // POST /api/v1/hr/employees/:id/offboard
  // Body: { lastWorkDay, exitDate, exitType, reason?, backfill, successorUserId? }
  // Cascade: HR-status inactive + AdminUser deactivated + USER_ROLE_ASSIGNMENTS leeren
  //           + Nachfolger als neuer managerUserId aller Betroffenen
  // ---------------------------------------------------------------------------
  http.post(`${API}/api/v1/hr/employees/:id/offboard`, async ({ params, request }) => {
    const emp = hrEmployees.find((e) => e.id === params.id) as Record<string, unknown> | undefined
    if (!emp) {
      return HttpResponse.json({ error: 'Employee not found' }, { status: 404 })
    }

    const body = (await request.json()) as {
      lastWorkDay?: string
      exitDate?: string
      exitType?: string
      reason?: string
      backfill?: boolean
      successorUserId?: string
    }

    // 1. HR-Status auf inactive setzen
    emp.status = 'inactive'
    emp.exitDate = body.exitDate ?? body.lastWorkDay
    emp.exitType = body.exitType ?? 'resignation'

    // 2. Rollen-Zuweisungen des Users entfernen (USER_ROLE_ASSIGNMENTS mutieren)
    const offboardedUserId = String(emp.userId ?? '')
    const rolesBefore = (USER_ROLE_ASSIGNMENTS[offboardedUserId] ?? []).map((r) => roleSummary(r).name)
    if (offboardedUserId && USER_ROLE_ASSIGNMENTS[offboardedUserId]) {
      USER_ROLE_ASSIGNMENTS[offboardedUserId] = []
    }
    writeAuditEvent({
      action: 'user.offboarded',
      target: displayUserName(offboardedUserId) || String(emp.name ?? params.id),
      targetType: 'user',
      oldValue: { status: 'active', roles: rolesBefore },
      newValue: { status: 'inactive', roles: [] },
      extraDetails: {
        exit_type: String(body.exitType ?? 'resignation'),
        ...(body.successorUserId ? { successor: displayUserName(body.successorUserId) } : {}),
      },
    })

    // 3. Admin-User deaktivieren (hrEmployees hat userId — suche per userId)
    const adminEmp = hrEmployees.find((e) => e.userId === offboardedUserId) as Record<string, unknown> | undefined
    if (adminEmp) {
      adminEmp.accountStatus = 'deactivated'
    }

    // 4. Nachfolger als neuer managerUserId aller Betroffenen
    if (body.successorUserId) {
      const successor = body.successorUserId
      for (const e of hrEmployees as Array<Record<string, unknown>>) {
        if (e.managerUserId === offboardedUserId || e.managerUserId === String(params.id)) {
          e.managerUserId = successor
        }
      }
    }

    return HttpResponse.json({ employee: emp })
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
