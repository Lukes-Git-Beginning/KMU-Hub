import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { daysFromNow, daysAgo } from '../data/date-helpers'

const API = API_BASE_URL
const BASE = `${API}/api/v1/schichten`

// ============================================================================
// Types (Wire shapes — snake_case, mirrors gRPC-Gateway / schichten-types.ts)
// ============================================================================

interface Shift {
  id: string
  tenant_id: string
  title: string
  description: string
  start_time: string
  end_time: string
  status: 'draft' | 'published'
  location: string | null
  created_by: string | null
  created_at: string
  updated_at: string
}

interface ShiftAssignment {
  id: string
  tenant_id: string
  shift_id: string
  employee_id: string
  assigned_at: string
  assigned_by: string | null
}

interface ShiftTemplate {
  id: string
  tenant_id: string
  name: string
  description: string
  day_of_week: number       // 0=Sunday … 6=Saturday
  start_hour: number        // 0–23
  start_minute: number      // 0–59
  duration_minutes: number
  location: string | null
  created_at: string
  updated_at: string
}

interface SwapRequestWire {
  id: string
  tenant_id: string
  assignment_id: string
  shift_id: string
  requested_by_employee_id: string
  swap_with_employee_id: string
  status: 'pending' | 'approved' | 'rejected'
  reason?: string
  idempotency_key?: string
  created_at: string
  updated_at: string
}

// ============================================================================
// Helper — build RFC3339 timestamps for a given date string + time
// ============================================================================

function ts(dateStr: string, hour: number, minute = 0): string {
  return `${dateStr}T${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}:00Z`
}

// ============================================================================
// Mutable state
// ============================================================================

// --- Templates (derived from stores/schichten.ts MOCK_TEMPLATES, in Wire shape) ---
const templates: ShiftTemplate[] = [
  {
    id: 'tpl-1',
    tenant_id: 'tenant-001',
    name: 'Frühschicht',
    description: 'Frühdienst 06:00–14:00 Uhr, inkl. 30 Min. Pause',
    day_of_week: 1,           // Monday default (applies to all days via template)
    start_hour: 6,
    start_minute: 0,
    duration_minutes: 480,    // 8h
    location: 'Hauptstandort',
    created_at: daysAgo(30) + 'T08:00:00Z',
    updated_at: daysAgo(30) + 'T08:00:00Z',
  },
  {
    id: 'tpl-2',
    tenant_id: 'tenant-001',
    name: 'Spätschicht',
    description: 'Spätdienst 14:00–22:00 Uhr, inkl. 30 Min. Pause',
    day_of_week: 1,
    start_hour: 14,
    start_minute: 0,
    duration_minutes: 480,
    location: 'Hauptstandort',
    created_at: daysAgo(30) + 'T08:00:00Z',
    updated_at: daysAgo(30) + 'T08:00:00Z',
  },
  {
    id: 'tpl-3',
    tenant_id: 'tenant-001',
    name: 'Nachtschicht',
    description: 'Nachtdienst 22:00–06:00 Uhr, inkl. 45 Min. Pause',
    day_of_week: 1,
    start_hour: 22,
    start_minute: 0,
    duration_minutes: 480,
    location: 'Hauptstandort',
    created_at: daysAgo(30) + 'T08:00:00Z',
    updated_at: daysAgo(30) + 'T08:00:00Z',
  },
  {
    id: 'tpl-4',
    tenant_id: 'tenant-001',
    name: 'Wochenend-Kurzschicht',
    description: 'Samstag/Sonntag 08:00–14:00 Uhr',
    day_of_week: 6,           // Saturday
    start_hour: 8,
    start_minute: 0,
    duration_minutes: 360,
    location: null,
    created_at: daysAgo(14) + 'T09:00:00Z',
    updated_at: daysAgo(14) + 'T09:00:00Z',
  },
]

// --- Shifts (published this week + next week, plus a few drafts) ---
const shifts: Shift[] = [
  // Published — past (this week, already done)
  {
    id: 'sft-001',
    tenant_id: 'tenant-001',
    title: 'Frühschicht Mo',
    description: 'Reguläre Frühschicht Montag',
    start_time: ts(daysAgo(5), 6),
    end_time:   ts(daysAgo(5), 14),
    status: 'published',
    location: 'Hauptstandort',
    created_by: IDS.users.stefan,
    created_at: daysAgo(10) + 'T08:00:00Z',
    updated_at: daysAgo(10) + 'T08:00:00Z',
  },
  {
    id: 'sft-002',
    tenant_id: 'tenant-001',
    title: 'Spätschicht Mo',
    description: 'Reguläre Spätschicht Montag',
    start_time: ts(daysAgo(5), 14),
    end_time:   ts(daysAgo(5), 22),
    status: 'published',
    location: 'Hauptstandort',
    created_by: IDS.users.stefan,
    created_at: daysAgo(10) + 'T08:00:00Z',
    updated_at: daysAgo(10) + 'T08:00:00Z',
  },
  {
    id: 'sft-003',
    tenant_id: 'tenant-001',
    title: 'Nachtschicht Mo/Di',
    description: 'Nachtschicht Montag auf Dienstag',
    start_time: ts(daysAgo(5), 22),
    end_time:   ts(daysAgo(4), 6),
    status: 'published',
    location: 'Hauptstandort',
    created_by: IDS.users.stefan,
    created_at: daysAgo(10) + 'T08:00:00Z',
    updated_at: daysAgo(10) + 'T08:00:00Z',
  },
  // Published — current / upcoming (today + next few days)
  {
    id: 'sft-004',
    tenant_id: 'tenant-001',
    title: 'Frühschicht heute',
    description: 'Reguläre Frühschicht',
    start_time: ts(daysFromNow(0), 6),
    end_time:   ts(daysFromNow(0), 14),
    status: 'published',
    location: 'Hauptstandort',
    created_by: IDS.users.stefan,
    created_at: daysAgo(7) + 'T08:00:00Z',
    updated_at: daysAgo(7) + 'T08:00:00Z',
  },
  {
    id: 'sft-005',
    tenant_id: 'tenant-001',
    title: 'Spätschicht heute',
    description: 'Reguläre Spätschicht',
    start_time: ts(daysFromNow(0), 14),
    end_time:   ts(daysFromNow(0), 22),
    status: 'published',
    location: 'Hauptstandort',
    created_by: IDS.users.stefan,
    created_at: daysAgo(7) + 'T08:00:00Z',
    updated_at: daysAgo(7) + 'T08:00:00Z',
  },
  {
    id: 'sft-006',
    tenant_id: 'tenant-001',
    title: 'Frühschicht morgen',
    description: 'Reguläre Frühschicht',
    start_time: ts(daysFromNow(1), 6),
    end_time:   ts(daysFromNow(1), 14),
    status: 'published',
    location: 'Hauptstandort',
    created_by: IDS.users.stefan,
    created_at: daysAgo(7) + 'T08:00:00Z',
    updated_at: daysAgo(7) + 'T08:00:00Z',
  },
  {
    id: 'sft-007',
    tenant_id: 'tenant-001',
    title: 'Spätschicht morgen',
    description: 'Reguläre Spätschicht',
    start_time: ts(daysFromNow(1), 14),
    end_time:   ts(daysFromNow(1), 22),
    status: 'published',
    location: 'Hauptstandort',
    created_by: IDS.users.stefan,
    created_at: daysAgo(7) + 'T08:00:00Z',
    updated_at: daysAgo(7) + 'T08:00:00Z',
  },
  {
    id: 'sft-008',
    tenant_id: 'tenant-001',
    title: 'Nachtschicht Di/Mi',
    description: 'Nachtschicht Dienstag auf Mittwoch',
    start_time: ts(daysFromNow(1), 22),
    end_time:   ts(daysFromNow(2), 6),
    status: 'published',
    location: 'Hauptstandort',
    created_by: IDS.users.stefan,
    created_at: daysAgo(7) + 'T08:00:00Z',
    updated_at: daysAgo(7) + 'T08:00:00Z',
  },
  {
    id: 'sft-009',
    tenant_id: 'tenant-001',
    title: 'Frühschicht übermorgen',
    description: 'Reguläre Frühschicht',
    start_time: ts(daysFromNow(2), 6),
    end_time:   ts(daysFromNow(2), 14),
    status: 'published',
    location: 'Hauptstandort',
    created_by: IDS.users.markus,
    created_at: daysAgo(5) + 'T09:00:00Z',
    updated_at: daysAgo(5) + 'T09:00:00Z',
  },
  {
    id: 'sft-010',
    tenant_id: 'tenant-001',
    title: 'Wochenend-Kurzschicht Sa',
    description: 'Wochenenddienst Samstag',
    start_time: ts(daysFromNow(4), 8),
    end_time:   ts(daysFromNow(4), 14),
    status: 'published',
    location: null,
    created_by: IDS.users.stefan,
    created_at: daysAgo(5) + 'T09:00:00Z',
    updated_at: daysAgo(5) + 'T09:00:00Z',
  },
  // Drafts — next week (not yet published)
  {
    id: 'sft-011',
    tenant_id: 'tenant-001',
    title: 'Frühschicht nächste Woche Mo',
    description: 'Frühdienst nächste Woche',
    start_time: ts(daysFromNow(7), 6),
    end_time:   ts(daysFromNow(7), 14),
    status: 'draft',
    location: 'Hauptstandort',
    created_by: IDS.users.markus,
    created_at: daysAgo(2) + 'T10:00:00Z',
    updated_at: daysAgo(2) + 'T10:00:00Z',
  },
  {
    id: 'sft-012',
    tenant_id: 'tenant-001',
    title: 'Spätschicht nächste Woche Mo',
    description: 'Spätdienst nächste Woche',
    start_time: ts(daysFromNow(7), 14),
    end_time:   ts(daysFromNow(7), 22),
    status: 'draft',
    location: 'Hauptstandort',
    created_by: IDS.users.markus,
    created_at: daysAgo(2) + 'T10:00:00Z',
    updated_at: daysAgo(2) + 'T10:00:00Z',
  },
]

// --- Assignments (shift_id → employees assigned) ---
// Derived from stores/schichten.ts MOCK_ASSIGNMENTS, mapped to API wire shape
const assignments: ShiftAssignment[] = [
  // sft-001 (Frühschicht Mo past)
  { id: 'asgn-001', tenant_id: 'tenant-001', shift_id: 'sft-001', employee_id: IDS.users.markus,  assigned_at: daysAgo(9) + 'T08:00:00Z', assigned_by: IDS.users.stefan },
  { id: 'asgn-002', tenant_id: 'tenant-001', shift_id: 'sft-001', employee_id: IDS.users.julia,   assigned_at: daysAgo(9) + 'T08:00:00Z', assigned_by: IDS.users.stefan },
  // sft-002 (Spätschicht Mo past)
  { id: 'asgn-003', tenant_id: 'tenant-001', shift_id: 'sft-002', employee_id: IDS.users.laura,   assigned_at: daysAgo(9) + 'T08:00:00Z', assigned_by: IDS.users.stefan },
  { id: 'asgn-004', tenant_id: 'tenant-001', shift_id: 'sft-002', employee_id: IDS.users.felix,   assigned_at: daysAgo(9) + 'T08:00:00Z', assigned_by: IDS.users.stefan },
  // sft-003 (Nachtschicht Mo/Di past)
  { id: 'asgn-005', tenant_id: 'tenant-001', shift_id: 'sft-003', employee_id: IDS.users.thomas,  assigned_at: daysAgo(9) + 'T08:00:00Z', assigned_by: IDS.users.stefan },
  // sft-004 (Frühschicht heute)
  { id: 'asgn-006', tenant_id: 'tenant-001', shift_id: 'sft-004', employee_id: IDS.users.markus,  assigned_at: daysAgo(6) + 'T08:00:00Z', assigned_by: IDS.users.stefan },
  { id: 'asgn-007', tenant_id: 'tenant-001', shift_id: 'sft-004', employee_id: IDS.users.sarah,   assigned_at: daysAgo(6) + 'T08:00:00Z', assigned_by: IDS.users.stefan },
  // sft-005 (Spätschicht heute)
  { id: 'asgn-008', tenant_id: 'tenant-001', shift_id: 'sft-005', employee_id: IDS.users.laura,   assigned_at: daysAgo(6) + 'T08:00:00Z', assigned_by: IDS.users.stefan },
  { id: 'asgn-009', tenant_id: 'tenant-001', shift_id: 'sft-005', employee_id: IDS.users.david,   assigned_at: daysAgo(6) + 'T08:00:00Z', assigned_by: IDS.users.stefan },
  // sft-006 (Frühschicht morgen) — the swap-request target
  { id: 'asgn-010', tenant_id: 'tenant-001', shift_id: 'sft-006', employee_id: IDS.users.jan,     assigned_at: daysAgo(5) + 'T08:00:00Z', assigned_by: IDS.users.stefan },
  { id: 'asgn-011', tenant_id: 'tenant-001', shift_id: 'sft-006', employee_id: IDS.users.julia,   assigned_at: daysAgo(5) + 'T08:00:00Z', assigned_by: IDS.users.stefan },
  // sft-007 (Spätschicht morgen) — the approved swap source
  { id: 'asgn-012', tenant_id: 'tenant-001', shift_id: 'sft-007', employee_id: IDS.users.lena,    assigned_at: daysAgo(5) + 'T08:00:00Z', assigned_by: IDS.users.stefan },
  { id: 'asgn-013', tenant_id: 'tenant-001', shift_id: 'sft-007', employee_id: IDS.users.felix,   assigned_at: daysAgo(5) + 'T08:00:00Z', assigned_by: IDS.users.stefan },
  // sft-008 (Nachtschicht Di/Mi)
  { id: 'asgn-014', tenant_id: 'tenant-001', shift_id: 'sft-008', employee_id: IDS.users.thomas,  assigned_at: daysAgo(4) + 'T09:00:00Z', assigned_by: IDS.users.markus },
  // sft-009 (Frühschicht übermorgen)
  { id: 'asgn-015', tenant_id: 'tenant-001', shift_id: 'sft-009', employee_id: IDS.users.markus,  assigned_at: daysAgo(4) + 'T09:00:00Z', assigned_by: IDS.users.markus },
  { id: 'asgn-016', tenant_id: 'tenant-001', shift_id: 'sft-009', employee_id: IDS.users.anna,    assigned_at: daysAgo(4) + 'T09:00:00Z', assigned_by: IDS.users.markus },
  // sft-010 (Wochenend-Kurzschicht Sa)
  { id: 'asgn-017', tenant_id: 'tenant-001', shift_id: 'sft-010', employee_id: IDS.users.sarah,   assigned_at: daysAgo(3) + 'T10:00:00Z', assigned_by: IDS.users.stefan },
  // sft-011 draft — no assignments yet
  // sft-012 draft — no assignments yet
]

// --- SwapRequests —
// swap-req-001: pending (Jan requests to swap sft-006 with Lena's sft-007 slot)
// swap-req-002: approved (Felix ↔ Markus swap on sft-007 — Markus as counterpart
// keeps the member scope-own demo non-empty: he sees exactly this one request)
// swap-req-003: rejected (Thomas rejected a swap on sft-008)
const swapRequests: SwapRequestWire[] = [
  {
    id: 'swap-req-001',
    tenant_id: 'tenant-001',
    assignment_id: 'asgn-010',        // Jan on sft-006 (Frühschicht morgen)
    shift_id: 'sft-006',
    requested_by_employee_id: IDS.users.jan,
    swap_with_employee_id: IDS.users.lena,
    status: 'pending',
    reason: 'Arzttermin am frühen Morgen — kann erst ab 14:00 Uhr',
    idempotency_key: 'swap-idem-001',
    created_at: daysAgo(1) + 'T10:30:00Z',
    updated_at: daysAgo(1) + 'T10:30:00Z',
  },
  {
    id: 'swap-req-002',
    tenant_id: 'tenant-001',
    assignment_id: 'asgn-013',        // Felix on sft-007 (Spätschicht morgen)
    shift_id: 'sft-007',
    requested_by_employee_id: IDS.users.felix,
    swap_with_employee_id: IDS.users.markus,
    status: 'approved',
    reason: 'Familienfeier am Abend, Tausch bereits abgesprochen',
    idempotency_key: 'swap-idem-002',
    created_at: daysAgo(3) + 'T16:15:00Z',
    updated_at: daysAgo(2) + 'T09:00:00Z',
  },
  {
    id: 'swap-req-003',
    tenant_id: 'tenant-001',
    assignment_id: 'asgn-014',        // Thomas on sft-008 (Nachtschicht Di/Mi)
    shift_id: 'sft-008',
    requested_by_employee_id: IDS.users.thomas,
    swap_with_employee_id: IDS.users.david,
    status: 'rejected',
    reason: 'Kurzer Urlaub, David ist einziger verfügbarer Nachtschichtler',
    idempotency_key: 'swap-idem-003',
    created_at: daysAgo(4) + 'T14:00:00Z',
    updated_at: daysAgo(3) + 'T11:00:00Z',
  },
]

// ============================================================================
// Helpers
// ============================================================================

function paginate<T>(items: T[], page: number, pageSize: number): { items: T[]; total: number } {
  const start = (page - 1) * pageSize
  return { items: items.slice(start, start + pageSize), total: items.length }
}

// ============================================================================
// Handlers
// ============================================================================

export const schichtenHandlers = [

  // --------------------------------------------------------------------------
  // Shifts — list
  // GET /api/v1/schichten/shifts
  // Response: { shifts: Shift[], total: number }
  // --------------------------------------------------------------------------
  http.get(`${BASE}/shifts`, ({ request }) => {
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 50)
    const status = url.searchParams.get('status')
    const from = url.searchParams.get('from')
    const to = url.searchParams.get('to')

    let filtered = [...shifts]
    if (status && status !== 'all') {
      filtered = filtered.filter(s => s.status === status)
    }
    if (from) {
      filtered = filtered.filter(s => s.start_time >= from)
    }
    if (to) {
      filtered = filtered.filter(s => s.start_time <= to)
    }

    const result = paginate(filtered, page, pageSize)
    return HttpResponse.json({ shifts: result.items, total: result.total })
  }),

  // --------------------------------------------------------------------------
  // Shifts — publish batch
  // POST /api/v1/schichten/shifts/publish
  // Body: { from: string, to: string }
  // Response: { published_count: number }
  // --------------------------------------------------------------------------
  http.post(`${BASE}/shifts/publish`, async ({ request }) => {
    const body = (await request.json()) as { from?: string; to?: string }
    const from = body.from ?? ''
    const to = body.to ?? ''
    let count = 0
    for (const s of shifts) {
      if (s.status === 'draft' && s.start_time >= from && s.start_time <= to) {
        s.status = 'published'
        s.updated_at = new Date().toISOString()
        count++
      }
    }
    return HttpResponse.json({ published_count: count })
  }),

  // --------------------------------------------------------------------------
  // Shifts — get single
  // GET /api/v1/schichten/shifts/:id
  // Response: { shift: Shift }
  // --------------------------------------------------------------------------
  http.get(`${BASE}/shifts/:id`, ({ params }) => {
    const shift = shifts.find(s => s.id === params.id)
    if (!shift) return HttpResponse.json({ error: 'shift not found' }, { status: 404 })
    return HttpResponse.json({ shift })
  }),

  // --------------------------------------------------------------------------
  // Shifts — create
  // POST /api/v1/schichten/shifts
  // Response: { shift: Shift }
  // --------------------------------------------------------------------------
  http.post(`${BASE}/shifts`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const shift: Shift = {
      id: `sft-${Date.now()}`,
      tenant_id: 'tenant-001',
      title: (body.title as string) ?? 'Neue Schicht',
      description: (body.description as string) ?? '',
      start_time: (body.start_time as string) ?? new Date().toISOString(),
      end_time: (body.end_time as string) ?? new Date().toISOString(),
      status: 'draft',
      location: (body.location as string) ?? null,
      created_by: (body.created_by as string) ?? IDS.users.stefan,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    shifts.push(shift)
    return HttpResponse.json({ shift }, { status: 201 })
  }),

  // --------------------------------------------------------------------------
  // Shifts — update
  // PATCH /api/v1/schichten/shifts/:id
  // Response: { shift: Shift }
  // --------------------------------------------------------------------------
  http.patch(`${BASE}/shifts/:id`, async ({ params, request }) => {
    const shift = shifts.find(s => s.id === params.id)
    if (!shift) return HttpResponse.json({ error: 'shift not found' }, { status: 404 })
    const body = (await request.json()) as Partial<Shift>
    if (body.title != null) shift.title = body.title
    if (body.description != null) shift.description = body.description
    if (body.start_time != null) shift.start_time = body.start_time
    if (body.end_time != null) shift.end_time = body.end_time
    if (body.location !== undefined) shift.location = body.location
    shift.updated_at = new Date().toISOString()
    return HttpResponse.json({ shift })
  }),

  // --------------------------------------------------------------------------
  // Shifts — delete
  // DELETE /api/v1/schichten/shifts/:id
  // Response: 204
  // --------------------------------------------------------------------------
  http.delete(`${BASE}/shifts/:id`, ({ params }) => {
    const idx = shifts.findIndex(s => s.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'shift not found' }, { status: 404 })
    shifts.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // --------------------------------------------------------------------------
  // Assignments — list for shift
  // GET /api/v1/schichten/shifts/:shiftId/assignments
  // Response: { assignments: ShiftAssignment[] }
  // --------------------------------------------------------------------------
  http.get(`${BASE}/shifts/:shiftId/assignments`, ({ params }) => {
    const shiftAssignments = assignments.filter(a => a.shift_id === params.shiftId)
    return HttpResponse.json({ assignments: shiftAssignments })
  }),

  // --------------------------------------------------------------------------
  // Assignments — assign employee
  // POST /api/v1/schichten/shifts/:shiftId/assignments
  // Body: { employee_id: string, assigned_by?: string }
  // Response: { assignment: ShiftAssignment }
  // --------------------------------------------------------------------------
  http.post(`${BASE}/shifts/:shiftId/assignments`, async ({ params, request }) => {
    const body = (await request.json()) as { employee_id: string; assigned_by?: string }
    const shiftId = params.shiftId as string

    const existing = assignments.find(
      a => a.shift_id === shiftId && a.employee_id === body.employee_id
    )
    if (existing) {
      return HttpResponse.json({ error: 'employee already assigned to this shift' }, { status: 409 })
    }

    const assignment: ShiftAssignment = {
      id: `asgn-${Date.now()}`,
      tenant_id: 'tenant-001',
      shift_id: shiftId,
      employee_id: body.employee_id,
      assigned_at: new Date().toISOString(),
      assigned_by: body.assigned_by ?? IDS.users.stefan,
    }
    assignments.push(assignment)
    return HttpResponse.json({ assignment }, { status: 201 })
  }),

  // --------------------------------------------------------------------------
  // Assignments — unassign employee
  // DELETE /api/v1/schichten/shifts/:shiftId/assignments/:employeeId
  // Response: 204
  // --------------------------------------------------------------------------
  http.delete(`${BASE}/shifts/:shiftId/assignments/:employeeId`, ({ params }) => {
    const idx = assignments.findIndex(
      a => a.shift_id === params.shiftId && a.employee_id === params.employeeId
    )
    if (idx === -1) return HttpResponse.json({ error: 'assignment not found' }, { status: 404 })
    assignments.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // --------------------------------------------------------------------------
  // Templates — list
  // GET /api/v1/schichten/templates
  // Response: { templates: ShiftTemplate[], total: number }
  // --------------------------------------------------------------------------
  http.get(`${BASE}/templates`, ({ request }) => {
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 50)
    const result = paginate(templates, page, pageSize)
    return HttpResponse.json({ templates: result.items, total: result.total })
  }),

  // --------------------------------------------------------------------------
  // Templates — create
  // POST /api/v1/schichten/templates
  // Response: { template: ShiftTemplate }
  // --------------------------------------------------------------------------
  http.post(`${BASE}/templates`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const template: ShiftTemplate = {
      id: `tpl-${Date.now()}`,
      tenant_id: 'tenant-001',
      name: (body.name as string) ?? 'Neue Vorlage',
      description: (body.description as string) ?? '',
      day_of_week: (body.day_of_week as number) ?? 1,
      start_hour: (body.start_hour as number) ?? 8,
      start_minute: (body.start_minute as number) ?? 0,
      duration_minutes: (body.duration_minutes as number) ?? 480,
      location: (body.location as string) ?? null,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    templates.push(template)
    return HttpResponse.json({ template }, { status: 201 })
  }),

  // --------------------------------------------------------------------------
  // Templates — update
  // PATCH /api/v1/schichten/templates/:id
  // Response: { template: ShiftTemplate }
  // --------------------------------------------------------------------------
  http.patch(`${BASE}/templates/:id`, async ({ params, request }) => {
    const template = templates.find(t => t.id === params.id)
    if (!template) return HttpResponse.json({ error: 'template not found' }, { status: 404 })
    const body = (await request.json()) as Partial<ShiftTemplate>
    if (body.name != null) template.name = body.name
    if (body.description != null) template.description = body.description
    if (body.day_of_week != null) template.day_of_week = body.day_of_week
    if (body.start_hour != null) template.start_hour = body.start_hour
    if (body.start_minute != null) template.start_minute = body.start_minute
    if (body.duration_minutes != null) template.duration_minutes = body.duration_minutes
    if (body.location !== undefined) template.location = body.location
    template.updated_at = new Date().toISOString()
    return HttpResponse.json({ template })
  }),

  // --------------------------------------------------------------------------
  // Templates — delete
  // DELETE /api/v1/schichten/templates/:id
  // Response: 204
  // --------------------------------------------------------------------------
  http.delete(`${BASE}/templates/:id`, ({ params }) => {
    const idx = templates.findIndex(t => t.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'template not found' }, { status: 404 })
    templates.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // --------------------------------------------------------------------------
  // Templates — apply to date range
  // POST /api/v1/schichten/templates/:id/apply
  // Body: { range_start: string, range_end: string, created_by?: string }
  // Response: { shifts: Shift[], created_count: number }
  // --------------------------------------------------------------------------
  http.post(`${BASE}/templates/:id/apply`, async ({ params, request }) => {
    const template = templates.find(t => t.id === params.id)
    if (!template) return HttpResponse.json({ error: 'template not found' }, { status: 404 })
    const body = (await request.json()) as { range_start: string; range_end: string; created_by?: string }

    // Generate one shift per matching day_of_week in the range
    const createdShifts: Shift[] = []
    const start = new Date(body.range_start)
    const end = new Date(body.range_end)
    const cur = new Date(start)

    while (cur <= end) {
      if (cur.getDay() === template.day_of_week) {
        const dateStr = cur.toISOString().split('T')[0]
        const shiftStart = ts(dateStr, template.start_hour, template.start_minute)
        const endHour = template.start_hour + Math.floor(template.duration_minutes / 60)
        const endMinute = template.start_minute + (template.duration_minutes % 60)
        const shiftEnd = ts(dateStr, endHour, endMinute)

        const shift: Shift = {
          id: `sft-tpl-${params.id}-${dateStr}`,
          tenant_id: 'tenant-001',
          title: `${template.name} (${dateStr})`,
          description: template.description,
          start_time: shiftStart,
          end_time: shiftEnd,
          status: 'draft',
          location: template.location,
          created_by: body.created_by ?? IDS.users.stefan,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }
        shifts.push(shift)
        createdShifts.push(shift)
      }
      cur.setDate(cur.getDate() + 1)
    }

    return HttpResponse.json({ shifts: createdShifts, created_count: createdShifts.length })
  }),

  // --------------------------------------------------------------------------
  // ArbZG Compliance check
  // GET /api/v1/schichten/compliance
  // Params: employee_id, new_shift_start, new_shift_end
  // Response: { compliant: boolean, violation_reason: string }
  // --------------------------------------------------------------------------
  http.get(`${BASE}/compliance`, ({ request }) => {
    const url = new URL(request.url)
    const employeeId = url.searchParams.get('employee_id') ?? ''
    const newStart = url.searchParams.get('new_shift_start') ?? ''
    const newEnd = url.searchParams.get('new_shift_end') ?? ''

    // Simplified ArbZG check: find existing assignments for this employee
    // within 11h rest-period before new_shift_start
    const employeeAssignments = assignments.filter(a => a.employee_id === employeeId)
    const newStartDate = new Date(newStart)

    let compliant = true
    let violationReason = ''

    for (const asgn of employeeAssignments) {
      const shift = shifts.find(s => s.id === asgn.shift_id)
      if (!shift) continue

      const prevEnd = new Date(shift.end_time)
      const restHours = (newStartDate.getTime() - prevEnd.getTime()) / (1000 * 60 * 60)

      // ArbZG §5: min 11h rest period between shifts
      if (restHours >= 0 && restHours < 11) {
        compliant = false
        violationReason = `Ruhezeit nach Schicht "${shift.title}" beträgt nur ${restHours.toFixed(1)}h (Minimum: 11h gemäß ArbZG §5)`
        break
      }

      // ArbZG §3: max 10h per day (simplified: check same-day double booking)
      const newStartStr = newStart.split('T')[0]
      const prevStartStr = shift.start_time.split('T')[0]
      if (newStartStr === prevStartStr) {
        const prevStartD = new Date(shift.start_time)
        const prevEndD = new Date(shift.end_time)
        const newEndD = new Date(newEnd)
        const totalMinutes =
          (prevEndD.getTime() - prevStartD.getTime()) / 60000 +
          (newEndD.getTime() - newStartDate.getTime()) / 60000
        if (totalMinutes > 600) { // > 10h
          compliant = false
          violationReason = `Tagesarbeitszeit von ${(totalMinutes / 60).toFixed(1)}h überschreitet gesetzliches Maximum von 10h (ArbZG §3)`
          break
        }
      }
    }

    return HttpResponse.json({ compliant, violation_reason: violationReason })
  }),

  // --------------------------------------------------------------------------
  // Stats
  // GET /api/v1/schichten/stats
  // Params: from?, to?
  // Response: ShiftStats
  // --------------------------------------------------------------------------
  http.get(`${BASE}/stats`, ({ request }) => {
    const url = new URL(request.url)
    const from = url.searchParams.get('from')
    const to = url.searchParams.get('to')

    let relevantShifts = [...shifts]
    if (from) relevantShifts = relevantShifts.filter(s => s.start_time >= from)
    if (to) relevantShifts = relevantShifts.filter(s => s.start_time <= to)

    const published = relevantShifts.filter(s => s.status === 'published')
    const draft = relevantShifts.filter(s => s.status === 'draft')
    const relevantIds = new Set(relevantShifts.map(s => s.id))
    const relevantAssignments = assignments.filter(a => relevantIds.has(a.shift_id))
    const uniqueEmployees = new Set(relevantAssignments.map(a => a.employee_id))

    return HttpResponse.json({
      total_shifts: relevantShifts.length,
      published_shifts: published.length,
      draft_shifts: draft.length,
      total_assignments: relevantAssignments.length,
      unique_employees: uniqueEmployees.size,
    })
  }),

  // --------------------------------------------------------------------------
  // SwapRequests — list
  // GET /api/v1/schichten/swap-requests
  // Params: shift_id?, status?, page_size?
  // Response: { swap_requests: SwapRequestWire[] }
  // --------------------------------------------------------------------------
  http.get(`${BASE}/swap-requests`, ({ request }) => {
    const url = new URL(request.url)
    const shiftId = url.searchParams.get('shift_id')
    const status = url.searchParams.get('status') as SwapRequestWire['status'] | null
    const pageSize = Number(url.searchParams.get('page_size') ?? 50)

    let filtered = [...swapRequests]
    if (shiftId) filtered = filtered.filter(r => r.shift_id === shiftId)
    if (status) filtered = filtered.filter(r => r.status === status)
    filtered = filtered.slice(0, pageSize)

    return HttpResponse.json({ swap_requests: filtered })
  }),

  // --------------------------------------------------------------------------
  // SwapRequests — create
  // POST /api/v1/schichten/swap-requests
  // Body: { assignment_id, shift_id, requested_by_employee_id, swap_with_employee_id, reason?, idempotency_key }
  // Response: { swap_request: SwapRequestWire }
  // --------------------------------------------------------------------------
  http.post(`${BASE}/swap-requests`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>

    // Idempotency check
    const idemKey = body.idempotency_key as string | undefined
    if (idemKey) {
      const existing = swapRequests.find(r => r.idempotency_key === idemKey)
      if (existing) return HttpResponse.json({ swap_request: existing }, { status: 200 })
    }

    const swapRequest: SwapRequestWire = {
      id: `swap-req-${Date.now()}`,
      tenant_id: 'tenant-001',
      assignment_id: (body.assignment_id as string) ?? '',
      shift_id: (body.shift_id as string) ?? '',
      requested_by_employee_id: (body.requested_by_employee_id as string) ?? '',
      swap_with_employee_id: (body.swap_with_employee_id as string) ?? '',
      status: 'pending',
      reason: (body.reason as string) ?? undefined,
      idempotency_key: idemKey,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    swapRequests.push(swapRequest)
    return HttpResponse.json({ swap_request: swapRequest }, { status: 201 })
  }),

  // --------------------------------------------------------------------------
  // SwapRequests — approve
  // POST /api/v1/schichten/swap-requests/:id/approve
  // Response: { swap_request: SwapRequestWire }
  // --------------------------------------------------------------------------
  http.post(`${BASE}/swap-requests/:id/approve`, ({ params }) => {
    const swapRequest = swapRequests.find(r => r.id === params.id)
    if (!swapRequest) {
      return HttpResponse.json({ error: 'swap request not found' }, { status: 404 })
    }
    if (swapRequest.status !== 'pending') {
      return HttpResponse.json(
        { error: `swap request is already ${swapRequest.status}` },
        { status: 409 }
      )
    }
    swapRequest.status = 'approved'
    swapRequest.updated_at = new Date().toISOString()

    // Reflect the swap in assignments: swap the two employees
    const sourceAsgn = assignments.find(a => a.id === swapRequest.assignment_id)
    if (sourceAsgn) {
      sourceAsgn.employee_id = swapRequest.swap_with_employee_id
      sourceAsgn.assigned_at = new Date().toISOString()
    }

    return HttpResponse.json({ swap_request: swapRequest })
  }),

  // --------------------------------------------------------------------------
  // SwapRequests — reject
  // POST /api/v1/schichten/swap-requests/:id/reject
  // Response: { swap_request: SwapRequestWire }
  // --------------------------------------------------------------------------
  http.post(`${BASE}/swap-requests/:id/reject`, ({ params }) => {
    const swapRequest = swapRequests.find(r => r.id === params.id)
    if (!swapRequest) {
      return HttpResponse.json({ error: 'swap request not found' }, { status: 404 })
    }
    if (swapRequest.status !== 'pending') {
      return HttpResponse.json(
        { error: `swap request is already ${swapRequest.status}` },
        { status: 409 }
      )
    }
    swapRequest.status = 'rejected'
    swapRequest.updated_at = new Date().toISOString()
    return HttpResponse.json({ swap_request: swapRequest })
  }),
]
