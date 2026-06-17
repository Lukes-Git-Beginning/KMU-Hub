import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { getUpcomingBirthdays, EMPLOYEES } from '../mock-db'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Global search results
// ---------------------------------------------------------------------------

function buildSearchResults(query: string) {
  const q = query.toLowerCase()
  const results: Array<{
    type: string
    id: string
    title: string
    subtitle: string
    url: string
    score: number
  }> = []

  // Contacts
  if ('gruber'.includes(q) || 'peter'.includes(q) || q.includes('gruber') || q.includes('peter')) {
    results.push({ type: 'contact', id: IDS.contacts.gruber, title: 'Peter Gruber', subtitle: 'Gruber Maschinenbau GmbH — Geschaeftsfuehrer', url: `/crm/contacts/${IDS.contacts.gruber}`, score: 95 })
  }
  if ('schneider'.includes(q) || q.includes('schneider')) {
    results.push({ type: 'contact', id: IDS.contacts.schneider, title: 'Anna Schneider', subtitle: 'Helvetia Software AG — CTO', url: `/crm/contacts/${IDS.contacts.schneider}`, score: 90 })
  }

  // Deals
  if ('crm'.includes(q) || 'lizenz'.includes(q) || q.includes('crm') || q.includes('lizenz')) {
    results.push({ type: 'deal', id: IDS.deals.crmLizenz, title: 'CRM Lizenz — Gruber Maschinenbau', subtitle: 'EUR 85\'000 — Verhandlung', url: `/crm/deals/${IDS.deals.crmLizenz}`, score: 88 })
  }
  if ('erp'.includes(q) || 'migration'.includes(q) || q.includes('erp') || q.includes('migration')) {
    results.push({ type: 'deal', id: IDS.deals.erpMigration, title: 'ERP Migration — Alpen Logistik', subtitle: 'EUR 120\'000 — Proposal', url: `/crm/deals/${IDS.deals.erpMigration}`, score: 85 })
  }

  // Projects
  if ('hub'.includes(q) || q.includes('hub')) {
    results.push({ type: 'project', id: IDS.projects.hubV2, title: 'Hub V2', subtitle: 'Projekt — Aktiv, 65% Fortschritt', url: `/projects/${IDS.projects.hubV2}`, score: 92 })
  }
  if ('website'.includes(q) || 'relaunch'.includes(q) || q.includes('website') || q.includes('relaunch')) {
    results.push({ type: 'project', id: IDS.projects.websiteRelaunch, title: 'Website Relaunch', subtitle: 'Projekt — Aktiv, 40% Fortschritt', url: `/projects/${IDS.projects.websiteRelaunch}`, score: 80 })
  }

  // Documents
  if ('brandbook'.includes(q) || q.includes('brandbook') || q.includes('brand')) {
    results.push({ type: 'document', id: 'file-010', title: 'Brandbook_2026.pdf', subtitle: 'Marketing — 14.9 MB', url: '/documents/file-010', score: 78 })
  }
  if ('vertrag'.includes(q) || q.includes('vertrag')) {
    results.push({ type: 'document', id: 'file-005', title: 'Vertrag_Gruber_Maschinenbau.pdf', subtitle: 'Verträge — 527 KB', url: '/documents/file-005', score: 82 })
  }

  // Employees
  if ('markus'.includes(q) || 'weber'.includes(q) || q.includes('markus') || q.includes('weber')) {
    results.push({ type: 'employee', id: IDS.users.markus, title: 'Markus Weber', subtitle: 'CTO / Technischer Leiter — Entwicklung', url: `/team/${IDS.users.markus}`, score: 93 })
  }

  // Sort by score descending
  results.sort((a, b) => b.score - a.score)
  return results.slice(0, 10)
}

// ---------------------------------------------------------------------------
// Stateful in-memory dashboard persistence (demo: survives within a session)
// ---------------------------------------------------------------------------

type StoredLayout = { layout: Array<Record<string, unknown>> | null; active_widgets: string[] }

/** User's personal layout override (null layout → client falls back to defaults). */
let userLayout: StoredLayout = { layout: null, active_widgets: [] }

/** Role-based default layouts (admin-configurable). */
const roleDefaults: Record<string, StoredLayout> = {
  admin: { layout: [], active_widgets: ['kpi-revenue', 'kpi-deals', 'kpi-tasks', 'cross-module-overview', 'team-status', 'activity-feed'] },
  manager: { layout: [], active_widgets: ['kpi-tasks', 'kpi-deals', 'team-status', 'team-worktime', 'my-tasks', 'calendar-upcoming'] },
  member: { layout: [], active_widgets: ['my-tasks', 'my-calendar', 'time-clock', 'team-chat', 'absences', 'notification-summary'] },
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const dashboardHandlers = [
  // Dashboard layout — user override (stateful; null layout → client uses defaults)
  http.get(`${API}/api/v1/dashboard/layout`, () => {
    return HttpResponse.json(userLayout)
  }),

  // Persist the user's personal layout
  http.put(`${API}/api/v1/dashboard/layout`, async ({ request }) => {
    const body = (await request.json()) as StoredLayout
    userLayout = { layout: body.layout ?? null, active_widgets: body.active_widgets ?? [] }
    return HttpResponse.json(userLayout)
  }),

  // Reset the user's layout back to role defaults
  http.delete(`${API}/api/v1/dashboard/layout`, () => {
    userLayout = { layout: null, active_widgets: [] }
    return HttpResponse.json(userLayout)
  }),

  // Role default layout — fetch (admin)
  http.get(`${API}/api/v1/dashboard/defaults/:role`, ({ params }) => {
    const role = String(params.role)
    return HttpResponse.json(roleDefaults[role] ?? { layout: [], active_widgets: [] })
  }),

  // Role default layout — save (admin)
  http.put(`${API}/api/v1/dashboard/defaults/:role`, async ({ params, request }) => {
    const role = String(params.role)
    const body = (await request.json()) as StoredLayout
    roleDefaults[role] = { layout: body.layout ?? [], active_widgets: body.active_widgets ?? [] }
    return HttpResponse.json(roleDefaults[role])
  }),

  // Upcoming team birthdays (mock-backed; backend gap: no birthday column yet)
  http.get(`${API}/api/v1/dashboard/birthdays`, () => {
    return HttpResponse.json({ birthdays: getUpcomingBirthdays() })
  }),

  // Team weekly worktime — per-member deterministic minutes (mock-backed)
  http.get(`${API}/api/v1/dashboard/team-worktime`, () => {
    const members = EMPLOYEES.slice(0, 6).map((e) => {
      // Deterministic weekly minutes from id hash → ~32h–44h range
      const hash = e.id.split('').reduce((a, c) => a + c.charCodeAt(0), 0)
      const minutes = 1920 + (hash % 13) * 60
      return { id: e.id, name: `${e.firstName} ${e.lastName}`, minutes }
    })
    return HttpResponse.json({ members })
  }),

  // Global search
  http.get(`${API}/api/v1/search`, ({ request }) => {
    const url = new URL(request.url)
    const query = url.searchParams.get('q') || ''

    if (!query || query.length < 2) {
      return HttpResponse.json({ results: [], total: 0, query })
    }

    const results = buildSearchResults(query)
    return HttpResponse.json({ results, total: results.length, query })
  }),
]
