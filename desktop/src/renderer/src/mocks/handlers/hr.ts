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

const mockEntries = [
  // Today
  {
    id: 'wte-001',
    employee_id: 'usr-001',
    date: today(),
    clockIn: todayAt(8, 0),
    clockOut: todayAt(12, 30),
    breakMinutes: 0,
    totalMinutes: 270,
    status: 'completed' as const,
    note: '',
  },
  {
    id: 'wte-002',
    employee_id: 'usr-001',
    date: today(),
    clockIn: todayAt(13, 15),
    clockOut: null,
    breakMinutes: 0,
    totalMinutes: 0,
    status: 'active' as const,
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

export const hrHandlers = [
  // ── Work Time Status ───────────────────────────────────────────────────

  http.get(`${API}/api/v1/hr/time/status`, () => {
    return HttpResponse.json({
      isClockedIn: true,
      isOnBreak: false,
      currentShiftStart: todayAt(8, 0),
      todayTotalMinutes: 310,
      arbzgSeverity: 'none',
    })
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
    return HttpResponse.json({
      entry: { id: `wte-${Date.now()}`, clockIn: new Date().toISOString() },
      compliance: { severity: 'none', message: '' },
    })
  }),

  http.post(`${API}/api/v1/hr/time/clock-out`, () => {
    return HttpResponse.json({
      entry: { id: 'wte-002', clockOut: new Date().toISOString(), totalMinutes: 480 },
      compliance: { severity: 'none', message: '' },
    })
  }),

  // ── Break ──────────────────────────────────────────────────────────────

  http.post(`${API}/api/v1/hr/time/break/start`, () => {
    return HttpResponse.json({
      break_entry: { start: new Date().toISOString(), end: null, minutes: 0 },
    })
  }),

  http.post(`${API}/api/v1/hr/time/break/end`, () => {
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
        totalWorkedMinutes: 310,
        totalBreakMinutes: 45,
        netWorkMinutes: 265,
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
          { date: daysAgo(4), totalWorkedMinutes: 510, netWorkMinutes: 450, overtimeMinutes: 30 },
          { date: daysAgo(3), totalWorkedMinutes: 420, netWorkMinutes: 390, overtimeMinutes: 0 },
          { date: daysAgo(2), totalWorkedMinutes: 465, netWorkMinutes: 420, overtimeMinutes: 0 },
          { date: daysAgo(1), totalWorkedMinutes: 525, netWorkMinutes: 480, overtimeMinutes: 60 },
          { date: today(), totalWorkedMinutes: 310, netWorkMinutes: 265, overtimeMinutes: 0 },
        ],
        totalWorkedMinutes: 2230,
        netWorkMinutes: 2005,
        totalOvertimeMinutes: 90,
      },
    })
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
        first_name: 'Markus',
        last_name: 'Weber',
        email: 'markus.weber@techvision.de',
        department: 'Engineering',
        position: 'Senior Developer',
        join_date: '2024-03-15',
        work_hours_per_week: 40,
      },
    })
  }),

  http.get(`${API}/api/v1/hr/employees`, () => {
    return HttpResponse.json({ employees: [], total: 0 })
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
