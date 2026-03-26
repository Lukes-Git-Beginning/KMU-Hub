import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'

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
    results.push({ type: 'document', id: 'file-005', title: 'Vertrag_Gruber_Maschinenbau.pdf', subtitle: 'Vertraege — 527 KB', url: '/documents/file-005', score: 82 })
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
// Handlers
// ---------------------------------------------------------------------------

export const dashboardHandlers = [
  // Dashboard layout — null (client uses defaults)
  http.get(`${API}/api/v1/dashboard/layout`, () => {
    return HttpResponse.json({ layout: null })
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
