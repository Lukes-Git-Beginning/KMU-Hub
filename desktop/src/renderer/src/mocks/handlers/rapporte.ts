/**
 * MSW handlers for the Rapporte module.
 *
 * Covers all endpoints used by rapporte-client.ts:
 *   GET    /api/v1/rapporte/reports
 *   POST   /api/v1/rapporte/reports
 *   GET    /api/v1/rapporte/reports/:id
 *   PATCH  /api/v1/rapporte/reports/:id
 *   DELETE /api/v1/rapporte/reports/:id
 *   POST   /api/v1/rapporte/reports/:id/submit
 *   POST   /api/v1/rapporte/reports/:id/approve
 *   POST   /api/v1/rapporte/reports/:id/reject
 *   GET    /api/v1/rapporte/reports/:id/lines
 *   POST   /api/v1/rapporte/reports/:id/lines
 *   PATCH  /api/v1/rapporte/lines/:id
 *   DELETE /api/v1/rapporte/lines/:id
 *   GET    /api/v1/rapporte/reports/:id/attachments
 *   POST   /api/v1/rapporte/reports/:id/attachments
 *   DELETE /api/v1/rapporte/attachments/:id
 *   GET    /api/v1/rapporte/stats
 *   GET    /api/v1/rapporte/pending
 */
import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { daysAgo } from '../data/date-helpers'

const API = API_BASE_URL
const BASE = `${API}/api/v1/rapporte`

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function newId(): string {
  return Math.random().toString(36).slice(2, 10)
}

function paginate<T>(arr: T[], page = 1, pageSize = 20): T[] {
  const start = (page - 1) * pageSize
  return arr.slice(start, start + pageSize)
}

// ---------------------------------------------------------------------------
// Wire-shape types (matching rapporte-types.ts)
// ---------------------------------------------------------------------------

interface WorkReport {
  id: string
  tenant_id: string
  title: string
  description: string
  status: 'draft' | 'submitted' | 'approved' | 'rejected'
  author_id: string
  reviewer_id: string | null
  reviewed_at: string | null
  review_note: string
  lat: number | null
  lon: number | null
  report_date: string
  created_at: string
  updated_at: string
  deleted_at: string | null
}

interface ReportLine {
  id: string
  tenant_id: string
  report_id: string
  position: number
  description: string
  quantity: number
  unit: string
  notes: string
  created_at: string
  updated_at: string
}

interface ReportAttachment {
  id: string
  tenant_id: string
  report_id: string
  line_id: string | null
  filename: string
  content_type: string
  size_bytes: number
  object_key: string
  uploaded_by: string
  created_at: string
}

// ---------------------------------------------------------------------------
// Mutable mock data
// ---------------------------------------------------------------------------

let reports: WorkReport[] = [
  {
    id: 'rpt-api-1',
    tenant_id: 'tenant-001',
    title: 'Rohbau Mehrfamilienhaus Zürich',
    description: 'Schalung 2. OG Decke Achse A-C aufgestellt und Bewehrung verlegt.',
    status: 'approved',
    author_id: 'Marco Brunner',
    reviewer_id: 'Peter Widmer',
    reviewed_at: daysAgo(3) + 'T14:00:00Z',
    review_note: 'Betonpumpe ab 13:00 im Einsatz. Nächster Guss geplant für Mittwoch.',
    lat: null,
    lon: null,
    report_date: daysAgo(6),
    created_at: daysAgo(6) + 'T08:00:00Z',
    updated_at: daysAgo(3) + 'T14:00:00Z',
    deleted_at: null,
  },
  {
    id: 'rpt-api-2',
    tenant_id: 'tenant-001',
    title: 'Fassadensanierung Bern',
    description: 'Alte Fassadenverkleidung Südseite demontiert. Dämmplatten EPS 160mm montiert (32 m²).',
    status: 'approved',
    author_id: 'Stefan Gerber',
    reviewer_id: 'Hans Meier',
    reviewed_at: daysAgo(4) + 'T10:00:00Z',
    review_note: 'Südseite komplett entfernt. Untergrund stellenweise schadhaft — Bauherr informiert.',
    lat: null,
    lon: null,
    report_date: daysAgo(7),
    created_at: daysAgo(7) + 'T07:30:00Z',
    updated_at: daysAgo(4) + 'T10:00:00Z',
    deleted_at: null,
  },
  {
    id: 'rpt-api-3',
    tenant_id: 'tenant-001',
    title: 'Innenausbau Büro Winterthur',
    description: 'Trennwände Grossraumbüro montiert (Trockenbau). Bodenbelag Vinyl in Büro 3+4 verlegt.',
    status: 'submitted',
    author_id: 'Thomas Lehmann',
    reviewer_id: null,
    reviewed_at: null,
    review_note: '',
    lat: null,
    lon: null,
    report_date: daysAgo(7),
    created_at: daysAgo(7) + 'T08:00:00Z',
    updated_at: daysAgo(2) + 'T16:00:00Z',
    deleted_at: null,
  },
  {
    id: 'rpt-api-4',
    tenant_id: 'tenant-001',
    title: 'Dacherneuerung Luzern',
    description: 'Unterdachbahn auf Ostseite verlegt (18 m²). Arbeit wegen Starkregen um 14:00 eingestellt.',
    status: 'approved',
    author_id: 'Beat Zimmermann',
    reviewer_id: 'Beat Zimmermann',
    reviewed_at: daysAgo(5) + 'T15:00:00Z',
    review_note: 'Wetterprognose für morgen besser — Weiterarbeit geplant.',
    lat: null,
    lon: null,
    report_date: daysAgo(8),
    created_at: daysAgo(8) + 'T07:00:00Z',
    updated_at: daysAgo(5) + 'T15:00:00Z',
    deleted_at: null,
  },
  {
    id: 'rpt-api-5',
    tenant_id: 'tenant-001',
    title: 'Elektrische Installation St. Gallen',
    description: 'Leerrohrverlegung 1. OG (42 Stk). Unterverteilung UG montiert und verdrahtet.',
    status: 'rejected',
    author_id: 'Christian Sutter',
    reviewer_id: 'Peter Widmer',
    reviewed_at: daysAgo(3) + 'T09:00:00Z',
    review_note: 'Prüfprotokoll Erdung fehlt als Anlage',
    lat: null,
    lon: null,
    report_date: daysAgo(8),
    created_at: daysAgo(8) + 'T07:30:00Z',
    updated_at: daysAgo(3) + 'T09:00:00Z',
    deleted_at: null,
  },
  {
    id: 'rpt-api-6',
    tenant_id: 'tenant-001',
    title: 'Badezimmer-Renovation Thun',
    description: 'Abdichtung Duschbereich mit Flüssigfolie. Bodenplatten 60x60 Feinsteinzeug verlegt.',
    status: 'draft',
    author_id: 'René Lüthi',
    reviewer_id: null,
    reviewed_at: null,
    review_note: '',
    lat: null,
    lon: null,
    report_date: daysAgo(10),
    created_at: daysAgo(10) + 'T08:00:00Z',
    updated_at: daysAgo(10) + 'T16:00:00Z',
    deleted_at: null,
  },
]

const linesByReport: Record<string, ReportLine[]> = {
  'rpt-api-1': [
    { id: 'line-1', tenant_id: 'tenant-001', report_id: 'rpt-api-1', position: 1, description: 'Bewehrungsstahl B500B', quantity: 2.4, unit: 't', notes: '', created_at: daysAgo(6) + 'T08:00:00Z', updated_at: daysAgo(6) + 'T08:00:00Z' },
    { id: 'line-2', tenant_id: 'tenant-001', report_id: 'rpt-api-1', position: 2, description: 'Beton C25/30', quantity: 12, unit: 'm³', notes: '', created_at: daysAgo(6) + 'T08:00:00Z', updated_at: daysAgo(6) + 'T08:00:00Z' },
  ],
  'rpt-api-3': [
    { id: 'line-3', tenant_id: 'tenant-001', report_id: 'rpt-api-3', position: 1, description: 'Gipskartonplatten 12.5mm', quantity: 28, unit: 'm²', notes: '', created_at: daysAgo(7) + 'T08:00:00Z', updated_at: daysAgo(7) + 'T08:00:00Z' },
  ],
}

const attachmentsByReport: Record<string, ReportAttachment[]> = {}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const rapporteHandlers = [
  // Stats
  http.get(`${BASE}/stats`, () => {
    const total_reports = reports.length
    const draft_count = reports.filter(r => r.status === 'draft').length
    const submitted_count = reports.filter(r => r.status === 'submitted').length
    const approved_count = reports.filter(r => r.status === 'approved').length
    const rejected_count = reports.filter(r => r.status === 'rejected').length
    return HttpResponse.json({ total_reports, draft_count, submitted_count, approved_count, rejected_count })
  }),

  // Pending approvals
  http.get(`${BASE}/pending`, ({ request }) => {
    const url = new URL(request.url)
    const page = parseInt(url.searchParams.get('page') ?? '1')
    const pageSize = parseInt(url.searchParams.get('page_size') ?? '20')
    const pending = reports.filter(r => r.status === 'submitted')
    return HttpResponse.json({ reports: paginate(pending, page, pageSize), total: pending.length })
  }),

  // Reports — List
  http.get(`${BASE}/reports`, ({ request }) => {
    const url = new URL(request.url)
    const page = parseInt(url.searchParams.get('page') ?? '1')
    const pageSize = parseInt(url.searchParams.get('page_size') ?? '50')
    const search = url.searchParams.get('search') ?? ''
    const status = url.searchParams.get('status') ?? ''

    let result = reports.filter(r => r.deleted_at === null)
    if (search) {
      const q = search.toLowerCase()
      result = result.filter(r =>
        r.title.toLowerCase().includes(q) ||
        r.description.toLowerCase().includes(q) ||
        r.author_id.toLowerCase().includes(q),
      )
    }
    if (status) {
      result = result.filter(r => r.status === status)
    }
    return HttpResponse.json({ reports: paginate(result, page, pageSize), total: result.length })
  }),

  // Reports — Create
  http.post(`${BASE}/reports`, async ({ request }) => {
    const body = await request.json() as Partial<WorkReport>
    const report: WorkReport = {
      id: newId(),
      tenant_id: 'tenant-001',
      title: body.title ?? '',
      description: body.description ?? '',
      status: 'draft',
      author_id: body.author_id ?? 'current-user',
      reviewer_id: null,
      reviewed_at: null,
      review_note: '',
      lat: body.lat ?? null,
      lon: body.lon ?? null,
      report_date: body.report_date ?? new Date().toISOString().split('T')[0],
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      deleted_at: null,
    }
    reports = [report, ...reports]
    return HttpResponse.json({ report }, { status: 201 })
  }),

  // Reports — Get by ID
  http.get(`${BASE}/reports/:id`, ({ params }) => {
    const report = reports.find(r => r.id === params.id && r.deleted_at === null)
    if (!report) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    return HttpResponse.json({ report })
  }),

  // Reports — Update
  http.patch(`${BASE}/reports/:id`, async ({ params, request }) => {
    const body = await request.json() as Partial<WorkReport>
    const idx = reports.findIndex(r => r.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    reports[idx] = { ...reports[idx], ...body, updated_at: new Date().toISOString() }
    return HttpResponse.json({ report: reports[idx] })
  }),

  // Reports — Delete
  http.delete(`${BASE}/reports/:id`, ({ params }) => {
    const idx = reports.findIndex(r => r.id === params.id)
    if (idx !== -1) {
      reports[idx] = { ...reports[idx], deleted_at: new Date().toISOString() }
    }
    return new HttpResponse(null, { status: 204 })
  }),

  // Reports — Submit
  http.post(`${BASE}/reports/:id/submit`, ({ params }) => {
    const idx = reports.findIndex(r => r.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    if (reports[idx].status !== 'draft') {
      return HttpResponse.json({ error: 'can only submit draft reports' }, { status: 409 })
    }
    reports[idx] = { ...reports[idx], status: 'submitted', updated_at: new Date().toISOString() }
    return HttpResponse.json({ report: reports[idx] })
  }),

  // Reports — Approve
  http.post(`${BASE}/reports/:id/approve`, async ({ params, request }) => {
    const body = await request.json() as { reviewer_id: string; review_note?: string }
    const idx = reports.findIndex(r => r.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    reports[idx] = {
      ...reports[idx],
      status: 'approved',
      reviewer_id: body.reviewer_id,
      reviewed_at: new Date().toISOString(),
      review_note: body.review_note ?? '',
      updated_at: new Date().toISOString(),
    }
    return HttpResponse.json({ report: reports[idx] })
  }),

  // Reports — Reject
  http.post(`${BASE}/reports/:id/reject`, async ({ params, request }) => {
    const body = await request.json() as { reviewer_id: string; review_note?: string }
    const idx = reports.findIndex(r => r.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    reports[idx] = {
      ...reports[idx],
      status: 'rejected',
      reviewer_id: body.reviewer_id,
      reviewed_at: new Date().toISOString(),
      review_note: body.review_note ?? '',
      updated_at: new Date().toISOString(),
    }
    return HttpResponse.json({ report: reports[idx] })
  }),

  // Lines — List
  http.get(`${BASE}/reports/:id/lines`, ({ params }) => {
    const lines = linesByReport[String(params.id)] ?? []
    return HttpResponse.json({ lines })
  }),

  // Lines — Add
  http.post(`${BASE}/reports/:id/lines`, async ({ params, request }) => {
    const body = await request.json() as Partial<ReportLine>
    const reportId = String(params.id)
    const line: ReportLine = {
      id: newId(),
      tenant_id: 'tenant-001',
      report_id: reportId,
      position: body.position ?? ((linesByReport[reportId]?.length ?? 0) + 1),
      description: body.description ?? '',
      quantity: body.quantity ?? 0,
      unit: body.unit ?? 'Stk',
      notes: body.notes ?? '',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    linesByReport[reportId] = [...(linesByReport[reportId] ?? []), line]
    return HttpResponse.json({ line }, { status: 201 })
  }),

  // Lines — Update
  http.patch(`${BASE}/lines/:id`, async ({ params, request }) => {
    const body = await request.json() as Partial<ReportLine>
    for (const reportId of Object.keys(linesByReport)) {
      const idx = linesByReport[reportId].findIndex(l => l.id === params.id)
      if (idx !== -1) {
        linesByReport[reportId][idx] = { ...linesByReport[reportId][idx], ...body, updated_at: new Date().toISOString() }
        return HttpResponse.json({ line: linesByReport[reportId][idx] })
      }
    }
    return HttpResponse.json({ error: 'not found' }, { status: 404 })
  }),

  // Lines — Delete
  http.delete(`${BASE}/lines/:id`, ({ params }) => {
    for (const reportId of Object.keys(linesByReport)) {
      const before = linesByReport[reportId].length
      linesByReport[reportId] = linesByReport[reportId].filter(l => l.id !== params.id)
      if (linesByReport[reportId].length < before) break
    }
    return new HttpResponse(null, { status: 204 })
  }),

  // Attachments — List
  http.get(`${BASE}/reports/:id/attachments`, ({ params }) => {
    const attachments = attachmentsByReport[String(params.id)] ?? []
    return HttpResponse.json({ attachments })
  }),

  // Attachments — Upload (mock: no actual file transfer)
  http.post(`${BASE}/reports/:id/attachments`, async ({ params, request }) => {
    const body = await request.json() as Partial<ReportAttachment>
    const reportId = String(params.id)
    const attachment: ReportAttachment = {
      id: newId(),
      tenant_id: 'tenant-001',
      report_id: reportId,
      line_id: body.line_id ?? null,
      filename: body.filename ?? 'upload.bin',
      content_type: body.content_type ?? 'application/octet-stream',
      size_bytes: body.size_bytes ?? 0,
      object_key: body.object_key ?? newId(),
      uploaded_by: body.uploaded_by ?? 'current-user',
      created_at: new Date().toISOString(),
    }
    attachmentsByReport[reportId] = [...(attachmentsByReport[reportId] ?? []), attachment]
    return HttpResponse.json({ attachment }, { status: 201 })
  }),

  // Attachments — Delete
  http.delete(`${BASE}/attachments/:id`, ({ params }) => {
    for (const reportId of Object.keys(attachmentsByReport)) {
      attachmentsByReport[reportId] = attachmentsByReport[reportId].filter(a => a.id !== params.id)
    }
    return new HttpResponse(null, { status: 204 })
  }),
]
