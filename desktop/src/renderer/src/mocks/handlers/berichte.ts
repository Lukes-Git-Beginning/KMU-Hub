import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import type { DashboardKPI } from '@/api/berichte-types'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Dashboard KPIs
//
// The berichte (Reports/BI) backend is not yet available; the renderer fetches
// `/api/v1/berichte/kpis` directly. Without this handler the dashboard renders
// its empty state and no KPI cards (so the sparkline feature would be invisible
// in demo mode). We serve a stable, representative set of KPIs per module.
// Backend gap (real time-series + KPI service) tracked in backend-gaps.md.
// ---------------------------------------------------------------------------

const DEMO_KPIS: DashboardKPI[] = [
  { id: 'kpi-umsatz-mtd', label: 'Umsatz (MTD)', value: '84.532', unit: '€', change_percent: 12.4, module_id: 'finanzen' },
  { id: 'kpi-offene-rechnungen', label: 'Offene Rechnungen', value: '23', unit: '', change_percent: -8.1, module_id: 'finanzen' },
  { id: 'kpi-neue-leads', label: 'Neue Leads', value: '47', unit: '', change_percent: 23, module_id: 'crm' },
  { id: 'kpi-gewinnrate', label: 'Gewinnrate', value: '32', unit: '%', change_percent: 4.2, module_id: 'crm' },
  { id: 'kpi-reaktionszeit', label: 'Ø Reaktionszeit', value: '2,4', unit: 'Std.', change_percent: -15, module_id: 'helpdesk' },
  { id: 'kpi-offene-tickets', label: 'Offene Tickets', value: '18', unit: '', change_percent: 6, module_id: 'helpdesk' },
  { id: 'kpi-lagerwert', label: 'Lagerwert', value: '412.900', unit: '€', change_percent: 1.8, module_id: 'inventar' },
  { id: 'kpi-ausschussquote', label: 'Ausschussquote', value: '1,2', unit: '%', change_percent: -0.4, module_id: 'produktion' },
  { id: 'kpi-aktive-projekte', label: 'Aktive Projekte', value: '9', unit: '', change_percent: null, module_id: 'cross' },
]

export const berichteHandlers = [
  http.get(`${API}/api/v1/berichte/kpis`, ({ request }) => {
    const url = new URL(request.url)
    const modules = url.searchParams.getAll('modules').filter(Boolean)
    const kpis =
      modules.length > 0
        ? DEMO_KPIS.filter((k) => modules.includes(String(k.module_id)))
        : DEMO_KPIS
    return HttpResponse.json({
      kpis,
      generated_at: new Date().toISOString(),
    })
  }),
]
