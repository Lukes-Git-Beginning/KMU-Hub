/**
 * MSW handler — HR Self-Service Change Requests (R-4 P3).
 *
 * Stateful in-memory registry (CHANGE_REQUESTS Map). Approve mutates the
 * hrEmployees array in handlers/team.ts via the same Object.assign path as
 * PUT /hr/employees/:id (team handler lines 219-227).
 *
 * Endpoints:
 *   GET    /api/v1/hr/change-requests          ?status= &scope=mine
 *   POST   /api/v1/hr/change-requests
 *   POST   /api/v1/hr/change-requests/:id/approve
 *   POST   /api/v1/hr/change-requests/:id/reject   {reason}
 *   POST   /api/v1/hr/change-requests/:id/cancel
 *
 *   GET    /api/v1/hr/employees/me/salary-statements
 */
import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS, CURRENT_USER, USER_DISPLAY_NAMES } from '../data/shared-ids'
import { getDemoSessionUserId } from '../data/rbac'
import { NAME_TO_USER_ID } from './team'
import { EMPLOYEES } from '../mock-db'
import { daysAgo } from '../data/date-helpers'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type ChangeRequestStatus = 'pending' | 'approved' | 'rejected' | 'cancelled'
type Drawer = 'personal' | 'job'

interface ChangeRequest {
  id: string
  userId: string
  userName: string
  drawer: Drawer
  field: string
  fieldLabel: string
  oldValue: string
  newValue: string
  status: ChangeRequestStatus
  reason?: string
  createdAt: string
  decidedAt?: string
  decidedByName?: string
}

// ---------------------------------------------------------------------------
// Stateful registry — single source of truth
// ---------------------------------------------------------------------------

const CHANGE_REQUESTS = new Map<string, ChangeRequest>()

// Seed: 1 pending Antrag von Markus Weber (usr-e2), 1 approved example
const seed: ChangeRequest[] = [
  {
    id: 'cr-seed-001',
    userId: IDS.users.markus,
    userName: USER_DISPLAY_NAMES[IDS.users.markus] ?? 'Markus Weber',
    drawer: 'personal',
    field: 'mobile',
    fieldLabel: 'Mobilnummer',
    oldValue: '+49 172 345 6789',
    newValue: '+49 172 999 8877',
    status: 'pending',
    createdAt: daysAgo(2),
  },
  {
    id: 'cr-seed-002',
    userId: IDS.users.julia,
    userName: USER_DISPLAY_NAMES[IDS.users.julia] ?? 'Julia Hofmann',
    drawer: 'personal',
    field: 'addressCity',
    fieldLabel: 'Wohnort',
    oldValue: 'München',
    newValue: 'Schwabing-West',
    status: 'approved',
    createdAt: daysAgo(15),
    decidedAt: daysAgo(12),
    decidedByName: USER_DISPLAY_NAMES[IDS.users.stefan] ?? 'Stefan Vogel',
  },
]

for (const req of seed) {
  CHANGE_REQUESTS.set(req.id, req)
}

// ---------------------------------------------------------------------------
// Helper: apply an approved change request to the in-memory EMPLOYEES record.
// Uses same mutation strategy as team.ts PUT /hr/employees/:id (Object.assign).
// ---------------------------------------------------------------------------

function applyApprovedChange(req: ChangeRequest): void {
  // Same canonical resolution as toHrEmployee in handlers/team.ts: name-based
  // map (mock-db indices drifted from IDS.users) with usr-<id> fallback.
  const emp = EMPLOYEES.find((e) => {
    const canonicalUserId = NAME_TO_USER_ID[`${e.firstName} ${e.lastName}`] ?? `usr-${e.id}`
    return canonicalUserId === req.userId
  })
  if (!emp) return
  // Mutate the mock-db record in-place (same as Object.assign in team.ts handler)
  ;(emp as unknown as Record<string, unknown>)[req.field] = req.newValue
}

// ---------------------------------------------------------------------------
// Salary-statement generator (from EMPLOYEES salary for session user)
// ---------------------------------------------------------------------------

function generateSalaryStatements(userId: string): Array<{
  id: string; month: string; label: string; gross: number; net: number
}> {
  // Find salary from mock-db for this user
  const empRecord: Record<string, string> = {
    [IDS.users.stefan]: 'Stefan Vogel',
    [IDS.users.markus]: 'Markus Weber',
    [IDS.users.thomas]: 'Thomas Keller',
    [IDS.users.julia]: 'Julia Hofmann',
    [IDS.users.laura]: 'Laura Neumann',
    [IDS.users.felix]: 'Felix Krause',
  }
  const name = empRecord[userId]
  const emp = name
    ? EMPLOYEES.find((e) => `${e.firstName} ${e.lastName}` === name)
    : EMPLOYEES.find((e) => `usr-${e.id}` === userId)

  const grossSalary = emp?.salary ?? 5000

  // Simple German net approximation (~67% of gross for full-time, covers SV + LSt)
  const netRatio = 0.675
  const net = Math.round(grossSalary * netRatio)

  const months = [
    { month: '2026-06', label: 'Juni 2026' },
    { month: '2026-05', label: 'Mai 2026' },
    { month: '2026-04', label: 'April 2026' },
  ]

  return months.map((m, i) => ({
    id: `ss-${userId}-${i + 1}`,
    month: m.month,
    label: m.label,
    gross: grossSalary,
    net,
  }))
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const hrChangeRequestHandlers = [
  // GET /api/v1/hr/change-requests
  http.get(`${API}/api/v1/hr/change-requests`, ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('status') as ChangeRequestStatus | null
    const scope = url.searchParams.get('scope')
    const sessionUserId = getDemoSessionUserId()

    let requests = Array.from(CHANGE_REQUESTS.values())

    if (scope === 'mine') {
      requests = requests.filter((r) => r.userId === sessionUserId)
    }
    if (status) {
      requests = requests.filter((r) => r.status === status)
    }

    // Newest first
    requests = requests.sort(
      (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
    )

    return HttpResponse.json({ requests, total: requests.length })
  }),

  // POST /api/v1/hr/change-requests — submit a new request
  http.post(`${API}/api/v1/hr/change-requests`, async ({ request }) => {
    const body = (await request.json()) as {
      drawer?: Drawer
      field?: string
      fieldLabel?: string
      oldValue?: string
      newValue?: string
    }
    const sessionUserId = getDemoSessionUserId()

    // Block duplicate pending request for the same field
    const alreadyPending = Array.from(CHANGE_REQUESTS.values()).find(
      (r) => r.userId === sessionUserId && r.field === body.field && r.status === 'pending',
    )
    if (alreadyPending) {
      return HttpResponse.json(
        { error: 'A pending request for this field already exists.' },
        { status: 409 },
      )
    }

    const newReq: ChangeRequest = {
      id: `cr-${Date.now()}`,
      userId: sessionUserId,
      userName: USER_DISPLAY_NAMES[sessionUserId] ?? CURRENT_USER.name,
      drawer: body.drawer ?? 'personal',
      field: body.field ?? '',
      fieldLabel: body.fieldLabel ?? body.field ?? '',
      oldValue: body.oldValue ?? '',
      newValue: body.newValue ?? '',
      status: 'pending',
      createdAt: new Date().toISOString(),
    }

    CHANGE_REQUESTS.set(newReq.id, newReq)
    return HttpResponse.json({ request: newReq }, { status: 201 })
  }),

  // POST /api/v1/hr/change-requests/:id/approve
  http.post(`${API}/api/v1/hr/change-requests/:id/approve`, ({ params }) => {
    const req = CHANGE_REQUESTS.get(String(params.id))
    if (!req) {
      return HttpResponse.json({ error: 'Not found' }, { status: 404 })
    }
    const sessionUserId = getDemoSessionUserId()
    const updatedReq: ChangeRequest = {
      ...req,
      status: 'approved',
      decidedAt: new Date().toISOString(),
      decidedByName: USER_DISPLAY_NAMES[sessionUserId] ?? 'HR',
    }
    CHANGE_REQUESTS.set(req.id, updatedReq)

    // Mutate the employee record in mock-db
    applyApprovedChange(updatedReq)

    return HttpResponse.json({ request: updatedReq })
  }),

  // POST /api/v1/hr/change-requests/:id/reject  { reason }
  http.post(`${API}/api/v1/hr/change-requests/:id/reject`, async ({ params, request }) => {
    const req = CHANGE_REQUESTS.get(String(params.id))
    if (!req) {
      return HttpResponse.json({ error: 'Not found' }, { status: 404 })
    }
    const body = (await request.json()) as { reason?: string }
    const sessionUserId = getDemoSessionUserId()
    const updatedReq: ChangeRequest = {
      ...req,
      status: 'rejected',
      reason: body.reason ?? '',
      decidedAt: new Date().toISOString(),
      decidedByName: USER_DISPLAY_NAMES[sessionUserId] ?? 'HR',
    }
    CHANGE_REQUESTS.set(req.id, updatedReq)
    return HttpResponse.json({ request: updatedReq })
  }),

  // POST /api/v1/hr/change-requests/:id/cancel
  http.post(`${API}/api/v1/hr/change-requests/:id/cancel`, ({ params }) => {
    const req = CHANGE_REQUESTS.get(String(params.id))
    if (!req) {
      return HttpResponse.json({ error: 'Not found' }, { status: 404 })
    }
    const sessionUserId = getDemoSessionUserId()
    if (req.userId !== sessionUserId) {
      return HttpResponse.json({ error: 'Forbidden' }, { status: 403 })
    }
    const updatedReq: ChangeRequest = { ...req, status: 'cancelled' }
    CHANGE_REQUESTS.set(req.id, updatedReq)
    return HttpResponse.json({ request: updatedReq })
  }),

  // GET /api/v1/hr/employees/me/salary-statements
  // NOTE: registered BEFORE team.ts GET /hr/employees/me (which only returns the profile,
  // not salary-statements). MSW matches the more-specific path first when placed before
  // the wildcard in the handlers array (hrChangeRequestHandlers spreads before teamHandlers).
  http.get(`${API}/api/v1/hr/employees/me/salary-statements`, () => {
    const sessionUserId = getDemoSessionUserId()
    const statements = generateSalaryStatements(sessionUserId)
    return HttpResponse.json({ statements })
  }),
]
