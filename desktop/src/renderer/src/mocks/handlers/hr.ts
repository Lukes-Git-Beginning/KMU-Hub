import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { daysAgo, today } from '../data/date-helpers'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Mock time entries — realistic German workday data
// ---------------------------------------------------------------------------

function todayAt(hour: number, minute = 0): string {
  const d = new Date()
  d.setHours(hour, minute, 0, 0)
  return d.toISOString()
}

function daysAgoAt(days: number, hour: number, minute = 0): string {
  const d = new Date()
  d.setDate(d.getDate() - days)
  d.setHours(hour, minute, 0, 0)
  return d.toISOString()
}

/** ISO timestamp `hours` (+ optional minutes) before now — for a live shift. */
function hoursAgo(hours: number, minutes = 0): string {
  return new Date(Date.now() - (hours * 60 + minutes) * 60_000).toISOString()
}

// Billable projects (Kunde → Projekt) for time attribution — clockodo-style.
const mockProjects = [
  { id: 'tp-01', name: 'Hub V2', customerId: 'cus-01', customerName: 'Interne Projekte', color: '#6366f1', billableDefault: false },
  { id: 'tp-02', name: 'Website-Relaunch', customerId: 'cus-02', customerName: 'Brunner AG', color: '#10b981', billableDefault: true },
  { id: 'tp-03', name: 'CRM-Migration', customerId: 'cus-03', customerName: 'Hoffmann GmbH', color: '#f59e0b', billableDefault: true },
  { id: 'tp-04', name: 'Support & Wartung', customerId: 'cus-04', customerName: 'Keller & Partner', color: '#3b82f6', billableDefault: true },
  { id: 'tp-05', name: 'Onboarding', customerId: 'cus-01', customerName: 'Interne Projekte', color: '#8b5cf6', billableDefault: false },
]

let mockEntries = [
  // Today
  {
    id: 'wte-001',
    employee_id: 'usr-001',
    date: today(),
    clockIn: todayAt(8, 0),
    clockOut: todayAt(12, 30),
    breakMinutes: 30,
    netWorkMinutes: 240,
    totalMinutes: 270,
    status: 'completed' as const,
    projectId: 'tp-02',
    projectName: 'Website-Relaunch',
    customerName: 'Brunner AG',
    activity: 'Frontend-Umsetzung',
    billable: true,
    note: 'Vormittag',
  },
  {
    id: 'wte-002',
    employee_id: 'usr-001',
    date: today(),
    clockIn: hoursAgo(2, 10),
    clockOut: null,
    breakMinutes: 0,
    totalMinutes: 0,
    status: 'active' as const,
    projectId: 'tp-03',
    projectName: 'CRM-Migration',
    customerName: 'Hoffmann GmbH',
    activity: 'Datenmigration',
    billable: true,
    note: '',
  },
  // Yesterday
  {
    id: 'wte-003',
    employee_id: 'usr-001',
    date: daysAgo(1),
    clockIn: daysAgoAt(1, 7, 45),
    clockOut: daysAgoAt(1, 12, 0),
    breakMinutes: 0,
    totalMinutes: 255,
    status: 'completed' as const,
    note: '',
  },
  {
    id: 'wte-004',
    employee_id: 'usr-001',
    date: daysAgo(1),
    clockIn: daysAgoAt(1, 12, 45),
    clockOut: daysAgoAt(1, 17, 15),
    breakMinutes: 0,
    totalMinutes: 270,
    status: 'completed' as const,
    note: '',
  },
  // 2 days ago
  {
    id: 'wte-005',
    employee_id: 'usr-001',
    date: daysAgo(2),
    clockIn: daysAgoAt(2, 8, 30),
    clockOut: daysAgoAt(2, 17, 0),
    breakMinutes: 45,
    totalMinutes: 465,
    status: 'completed' as const,
    note: 'Kundentermin vormittags',
  },
  // 3 days ago
  {
    id: 'wte-006',
    employee_id: 'usr-001',
    date: daysAgo(3),
    clockIn: daysAgoAt(3, 9, 0),
    clockOut: daysAgoAt(3, 16, 30),
    breakMinutes: 30,
    totalMinutes: 420,
    status: 'completed' as const,
    note: '',
  },
  // 4 days ago
  {
    id: 'wte-007',
    employee_id: 'usr-001',
    date: daysAgo(4),
    clockIn: daysAgoAt(4, 8, 0),
    clockOut: daysAgoAt(4, 17, 30),
    breakMinutes: 60,
    totalMinutes: 510,
    status: 'completed' as const,
    note: 'Sprint Review',
  },
]

// Team weekly time + week-submission state (in-memory, mutated by handlers).
let teamRows = [
  { employeeId: 'usr-002', name: 'Stefan Klein', department: 'Entwicklung', weekMinutes: 2310, targetMinutes: 2400, overtimeMinutes: 0, clockedIn: true, weekStatus: 'submitted' as const },
  { employeeId: 'usr-003', name: 'Lena Braun', department: 'Design', weekMinutes: 2460, targetMinutes: 2400, overtimeMinutes: 60, clockedIn: false, weekStatus: 'submitted' as const },
  { employeeId: 'usr-004', name: 'Felix Krause', department: 'Vertrieb', weekMinutes: 2400, targetMinutes: 2400, overtimeMinutes: 0, clockedIn: true, weekStatus: 'approved' as const },
  { employeeId: 'usr-005', name: 'Julia Hofmann', department: 'Support', weekMinutes: 1980, targetMinutes: 2400, overtimeMinutes: 0, clockedIn: false, weekStatus: 'open' as const },
  { employeeId: 'usr-006', name: 'Sophie Lang', department: 'Entwicklung', weekMinutes: 2520, targetMinutes: 2400, overtimeMinutes: 120, clockedIn: true, weekStatus: 'submitted' as const },
]
let myWeekStatus: { status: string; submittedAt?: string; approvedBy?: string } = { status: 'open' }

// Live clock state (in-memory) — stateful so clock-in/out/break actually work
// in the demo. Starts clocked-in for a rich initial view.
let workStatus: {
  isClockedIn: boolean
  isOnBreak: boolean
  currentShiftStart?: string
  currentBreakStart?: string
  todayTotalMinutes: number
  arbzgSeverity: string
} = {
  isClockedIn: true,
  isOnBreak: false,
  currentShiftStart: hoursAgo(2, 10),
  todayTotalMinutes: 370,
  arbzgSeverity: 'none',
}

export const hrHandlers = [
  // ── Work Time Status ───────────────────────────────────────────────────

  http.get(`${API}/api/v1/hr/time/status`, () => {
    return HttpResponse.json(workStatus)
  }),

  // ── Active Shift ───────────────────────────────────────────────────────

  http.get(`${API}/api/v1/hr/time/active`, () => {
    return HttpResponse.json({
      shift: {
        id: 'shift-001',
        employee_id: 'usr-001',
        clock_in: todayAt(8, 0),
        clock_out: null,
        breaks: [
          { start: todayAt(12, 30), end: todayAt(13, 15), minutes: 45 },
        ],
        total_worked_minutes: 310,
        total_break_minutes: 45,
      },
    })
  }),

  // ── Clock In/Out ───────────────────────────────────────────────────────

  http.post(`${API}/api/v1/hr/time/clock-in`, () => {
    workStatus = {
      ...workStatus,
      isClockedIn: true,
      isOnBreak: false,
      currentShiftStart: new Date().toISOString(),
      currentBreakStart: undefined,
    }
    return HttpResponse.json({
      entry: { id: `wte-${Date.now()}`, clockIn: workStatus.currentShiftStart },
      compliance: { severity: 'none', message: '' },
    })
  }),

  http.post(`${API}/api/v1/hr/time/clock-out`, () => {
    workStatus = {
      ...workStatus,
      isClockedIn: false,
      isOnBreak: false,
      currentShiftStart: undefined,
      currentBreakStart: undefined,
    }
    return HttpResponse.json({
      entry: { id: `wte-${Date.now()}`, clockOut: new Date().toISOString(), totalMinutes: 480 },
      compliance: { severity: 'none', message: '' },
    })
  }),

  // ── Break ──────────────────────────────────────────────────────────────

  http.post(`${API}/api/v1/hr/time/break/start`, () => {
    workStatus = { ...workStatus, isOnBreak: true, currentBreakStart: new Date().toISOString() }
    return HttpResponse.json({
      break_entry: { start: workStatus.currentBreakStart, end: null, minutes: 0 },
    })
  }),

  http.post(`${API}/api/v1/hr/time/break/end`, () => {
    workStatus = { ...workStatus, isOnBreak: false, currentBreakStart: undefined }
    return HttpResponse.json({
      break_entry: { start: todayAt(12, 30), end: new Date().toISOString(), minutes: 45 },
    })
  }),

  // ── Entries ────────────────────────────────────────────────────────────

  http.get(`${API}/api/v1/hr/time/entries`, ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('status')

    let filtered = [...mockEntries]
    if (status === 'correction_pending') {
      filtered = [] // no pending corrections in demo
    }

    return HttpResponse.json({
      entries: filtered,
      total: filtered.length,
    })
  }),

  // ── Daily Summary ──────────────────────────────────────────────────────

  http.get(`${API}/api/v1/hr/time/summary/daily`, ({ request }) => {
    const url = new URL(request.url)
    const date = url.searchParams.get('date') || today()

    return HttpResponse.json({
      summary: {
        date,
        totalWorkedMinutes: 400,
        totalBreakMinutes: 30,
        netWorkMinutes: 370,
        overtimeMinutes: 0,
        entryCount: 2,
      },
    })
  }),

  // ── Weekly Summary ─────────────────────────────────────────────────────

  http.get(`${API}/api/v1/hr/time/summary/weekly`, ({ request }) => {
    const url = new URL(request.url)
    const weekStart = url.searchParams.get('week_start') || daysAgo(4)

    return HttpResponse.json({
      summary: {
        weekStart,
        days: [
          { date: daysAgo(4), totalWorkedMinutes: 510, totalBreakMinutes: 60, netWorkMinutes: 450, overtimeMinutes: 30 },
          { date: daysAgo(3), totalWorkedMinutes: 420, totalBreakMinutes: 30, netWorkMinutes: 390, overtimeMinutes: 0 },
          { date: daysAgo(2), totalWorkedMinutes: 465, totalBreakMinutes: 45, netWorkMinutes: 420, overtimeMinutes: 0 },
          { date: daysAgo(1), totalWorkedMinutes: 525, totalBreakMinutes: 45, netWorkMinutes: 480, overtimeMinutes: 60 },
          { date: today(), totalWorkedMinutes: 310, totalBreakMinutes: 45, netWorkMinutes: 265, overtimeMinutes: 0 },
        ],
        totalWorkedMinutes: 2230,
        totalBreakMinutes: 225,
        netWorkMinutes: 2005,
        totalOvertimeMinutes: 90,
      },
    })
  }),

  // ── Time Balance (Stundenkonto / Gleitzeit) ───────────────────────────────

  http.get(`${API}/api/v1/hr/time/balance`, () => {
    return HttpResponse.json({
      balance: {
        balanceMinutes: 752, // +12h 32m cumulative overtime credit
        asOf: today(),
        periodStart: '2026-01-01',
        targetWeeklyMinutes: 2400, // 40h
      },
    })
  }),

  // ── Projects (Kunde → Projekt) for time attribution ───────────────────────

  http.get(`${API}/api/v1/hr/time/projects`, () => {
    return HttpResponse.json({ projects: mockProjects })
  }),

  // ── Analytics (Auswertungen: KPIs, day trend, project + billable split) ────

  http.get(`${API}/api/v1/hr/time/analytics`, ({ request }) => {
    const url = new URL(request.url)
    const range = url.searchParams.get('range') === 'month' ? 'month' : 'week'
    const days = range === 'month' ? 30 : 7
    const pad = (n: number) => String(n).padStart(2, '0')
    const perWorkday = [430, 465, 450, 480, 410, 395, 470]

    const dayTrend: { date: string; netMinutes: number; billableMinutes: number }[] = []
    let totalNet = 0
    let totalBillable = 0
    for (let i = days - 1; i >= 0; i--) {
      const d = new Date()
      d.setDate(d.getDate() - i)
      const dow = d.getDay()
      const isWorkday = dow >= 1 && dow <= 5
      const net = isWorkday ? perWorkday[(d.getDate()) % perWorkday.length] : 0
      const billable = Math.round(net * 0.72)
      dayTrend.push({
        date: `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`,
        netMinutes: net,
        billableMinutes: billable,
      })
      totalNet += net
      totalBillable += billable
    }

    const weights = [0.18, 0.3, 0.24, 0.16, 0.12]
    const byProject = mockProjects.map((p, i) => ({
      projectId: p.id,
      name: p.name,
      customerName: p.customerName,
      color: p.color,
      minutes: Math.round(totalNet * weights[i % weights.length]),
    }))

    const workedDays = dayTrend.filter((d) => d.netMinutes > 0).length
    const targetMinutes = workedDays * 8 * 60

    return HttpResponse.json({
      analytics: {
        range,
        periodStart: dayTrend[0]?.date ?? today(),
        periodEnd: dayTrend[dayTrend.length - 1]?.date ?? today(),
        totalNetMinutes: totalNet,
        billableMinutes: totalBillable,
        nonBillableMinutes: totalNet - totalBillable,
        overtimeMinutes: Math.max(0, totalNet - targetMinutes),
        targetMinutes,
        workedDays,
        dayTrend,
        byProject,
      },
    })
  }),

  // ── Team weekly time (manager view) ───────────────────────────────────────

  http.get(`${API}/api/v1/hr/time/team`, () => {
    return HttpResponse.json({ rows: teamRows })
  }),

  // ── Week submission / approval workflow ───────────────────────────────────

  http.get(`${API}/api/v1/hr/time/weeks/status`, ({ request }) => {
    const url = new URL(request.url)
    const weekStart = url.searchParams.get('week_start') || today()
    return HttpResponse.json({ status: { weekStart, ...myWeekStatus } })
  }),

  http.post(`${API}/api/v1/hr/time/weeks/submit`, async ({ request }) => {
    const body = (await request.json()) as { weekStart?: string }
    myWeekStatus = { status: 'submitted', submittedAt: new Date().toISOString() }
    return HttpResponse.json({ status: { weekStart: body.weekStart ?? today(), ...myWeekStatus } })
  }),

  http.post(`${API}/api/v1/hr/time/weeks/approve`, async ({ request }) => {
    const body = (await request.json()) as { employeeId?: string }
    teamRows = teamRows.map((r) => (r.employeeId === body.employeeId ? { ...r, weekStatus: 'approved' as const } : r))
    return HttpResponse.json({ ok: true })
  }),

  http.post(`${API}/api/v1/hr/time/weeks/reject`, async ({ request }) => {
    const body = (await request.json()) as { employeeId?: string }
    teamRows = teamRows.map((r) => (r.employeeId === body.employeeId ? { ...r, weekStatus: 'rejected' as const } : r))
    return HttpResponse.json({ ok: true })
  }),

  // ── Manual entry creation ─────────────────────────────────────────────────

  http.post(`${API}/api/v1/hr/time/entries`, async ({ request }) => {
    const body = (await request.json()) as {
      clockIn: string
      clockOut: string
      breakMinutes?: number
      projectId?: string
      activity?: string
      billable?: boolean
      note?: string
    }
    const project = mockProjects.find((p) => p.id === body.projectId)
    const grossMinutes = Math.max(
      0,
      Math.round((new Date(body.clockOut).getTime() - new Date(body.clockIn).getTime()) / 60_000),
    )
    const breakMinutes = body.breakMinutes ?? 0
    const entry = {
      id: `wte-${Date.now()}`,
      employee_id: 'usr-001',
      date: body.clockIn.slice(0, 10),
      clockIn: body.clockIn,
      clockOut: body.clockOut,
      breakMinutes,
      netWorkMinutes: Math.max(0, grossMinutes - breakMinutes),
      totalMinutes: grossMinutes,
      status: 'completed' as const,
      isManual: true,
      projectId: body.projectId,
      projectName: project?.name,
      customerName: project?.customerName,
      activity: body.activity,
      billable: body.billable ?? project?.billableDefault ?? false,
      note: body.note ?? '',
    }
    mockEntries = [entry, ...mockEntries]
    return HttpResponse.json({ entry }, { status: 201 })
  }),

  // ── Corrections ────────────────────────────────────────────────────────

  http.post(`${API}/api/v1/hr/time/corrections`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({
      entry: {
        id: `wte-corr-${Date.now()}`,
        ...body,
        status: 'correction_pending',
      },
    }, { status: 201 })
  }),

  http.post(`${API}/api/v1/hr/time/corrections/:id/approve`, ({ params }) => {
    return HttpResponse.json({
      entry: { id: params.id, status: 'completed' },
    })
  }),

  // ── Leave ──────────────────────────────────────────────────────────────

  http.get(`${API}/api/v1/hr/leave/balance`, () => {
    return HttpResponse.json({
      balance: {
        total_days: 30,
        used_days: 8,
        remaining_days: 22,
        pending_days: 2,
      },
    })
  }),

  http.get(`${API}/api/v1/hr/leave/types`, () => {
    return HttpResponse.json({
      types: [
        { id: 'lt-001', name: 'Urlaub', color: '#22c55e', paid: true },
        { id: 'lt-002', name: 'Krankheit', color: '#ef4444', paid: true },
        { id: 'lt-003', name: 'Sonderurlaub', color: '#3b82f6', paid: true },
        { id: 'lt-004', name: 'Unbezahlter Urlaub', color: '#a855f7', paid: false },
      ],
    })
  }),

  http.get(`${API}/api/v1/hr/leave/requests`, () => {
    return HttpResponse.json({ requests: [], total: 0 })
  }),

  http.post(`${API}/api/v1/hr/leave/requests`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({
      request: { id: `lr-${Date.now()}`, ...body, status: 'pending' },
    }, { status: 201 })
  }),

  // ── Absences Calendar ──────────────────────────────────────────────────

  http.get(`${API}/api/v1/hr/absences/calendar`, () => {
    return HttpResponse.json({ absences: [] })
  }),

  // ── Employee Profile ───────────────────────────────────────────────────

  http.get(`${API}/api/v1/hr/employees/me`, () => {
    return HttpResponse.json({
      employee: {
        id: 'usr-001',
        user_id: 'usr-001',
        user_name: 'Markus Weber',
        user_email: 'markus.weber@techvision.de',
        department: 'Engineering',
        position_title: 'Senior Developer',
        contract_type: 'full_time',
        work_days_per_week: 5,
        annual_leave_days: 30,
        start_date: '2024-03-15',
      },
    })
  }),

  http.post(`${API}/api/v1/hr/employees`, async ({ request }) => {
    const body = await request.json() as Record<string, unknown>
    return HttpResponse.json({
      employee: {
        id: `emp-${Date.now()}`,
        user_id: body.user_id ?? '',
        department: body.department ?? '',
        position_title: body.position_title ?? '',
        contract_type: body.contract_type ?? 'full_time',
        work_days_per_week: body.work_days_per_week ?? 5,
        annual_leave_days: body.annual_leave_days ?? 20,
        start_date: body.start_date ?? new Date().toISOString().split('T')[0],
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    }, { status: 201 })
  }),

  http.get(`${API}/api/v1/hr/employees`, () => {
    return HttpResponse.json({
      employees: [
        {
          id: 'usr-001',
          user_id: 'usr-001',
          user_name: 'Markus Weber',
          user_email: 'markus.weber@techvision.de',
          department: 'Engineering',
          position_title: 'Senior Developer',
          contract_type: 'full_time',
          work_days_per_week: 5,
          annual_leave_days: 30,
          start_date: '2024-03-15',
        },
        {
          id: 'usr-002',
          user_id: 'usr-002',
          user_name: 'Sabine Müller',
          user_email: 'sabine.mueller@techvision.de',
          department: 'Vertrieb',
          position_title: 'Account Managerin',
          contract_type: 'full_time',
          work_days_per_week: 5,
          annual_leave_days: 28,
          start_date: '2023-07-01',
        },
        {
          id: 'usr-003',
          user_id: 'usr-003',
          user_name: 'Thomas Schäfer',
          user_email: 'thomas.schaefer@techvision.de',
          department: 'Engineering',
          position_title: 'Backend Engineer',
          contract_type: 'full_time',
          work_days_per_week: 5,
          annual_leave_days: 28,
          start_date: '2023-11-15',
        },
        {
          id: 'usr-004',
          user_id: 'usr-004',
          user_name: 'Jana Köhler',
          user_email: 'jana.koehler@techvision.de',
          department: 'Design',
          position_title: 'UX Designerin',
          contract_type: 'part_time',
          work_days_per_week: 4,
          annual_leave_days: 24,
          start_date: '2024-01-08',
        },
        {
          id: 'usr-005',
          user_id: 'usr-005',
          user_name: 'Felix Bauer',
          user_email: 'felix.bauer@techvision.de',
          department: 'Vertrieb',
          position_title: 'Sales Development Rep',
          contract_type: 'full_time',
          work_days_per_week: 5,
          annual_leave_days: 25,
          start_date: '2025-02-01',
        },
        {
          id: 'usr-006',
          user_id: 'usr-006',
          user_name: 'Anna Großmann',
          user_email: 'anna.grossmann@techvision.de',
          department: 'Operations',
          position_title: 'Office Managerin',
          contract_type: 'mini_job',
          work_days_per_week: 3,
          annual_leave_days: 18,
          start_date: '2024-06-01',
        },
      ],
      total: 6,
    })
  }),

  http.get(`${API}/api/v1/hr/settings`, () => {
    return HttpResponse.json({
      settings: {
        work_hours_per_day: 8,
        break_after_hours: 6,
        min_break_minutes: 30,
        max_daily_hours: 10,
        overtime_enabled: true,
      },
    })
  }),
]
