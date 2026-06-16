import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// In-memory shape (raw, matches server responses)
// ---------------------------------------------------------------------------

interface RawBookingPage {
  id: string
  tenant_id: string
  calendar_id: string
  slug: string
  company_name: string
  logo_url: string | null
  services: {
    id: string
    name: string
    duration_min: number
    price: string
    description: string | null
  }[]
  availability_rules_json: string
  active: boolean
  created_at: string
  updated_at: string
}

// ---------------------------------------------------------------------------
// Seed data
// ---------------------------------------------------------------------------

let store: RawBookingPage[] = [
  {
    id: 'bp-demo-001',
    tenant_id: 'tenant-001',
    calendar_id: 'cal-1',
    slug: 'muster-dienstleister',
    company_name: 'Zentria Muster GmbH',
    logo_url: null,
    services: [
      {
        id: 'svc-1',
        name: 'Beratungsgespräch 30 Min.',
        duration_min: 30,
        price: '0.00',
        description: 'Kostenloses Erstgespräch',
      },
      {
        id: 'svc-2',
        name: 'Strategie-Workshop 60 Min.',
        duration_min: 60,
        price: '120.00',
        description: null,
      },
      {
        id: 'svc-3',
        name: 'Technische Analyse 45 Min.',
        duration_min: 45,
        price: '80.00',
        description: 'Detaillierte technische Beratung',
      },
    ],
    availability_rules_json: JSON.stringify({
      weekdays: [1, 2, 3, 4, 5],
      slots_per_weekday: [
        { start: '09:00', end: '12:00' },
        { start: '13:00', end: '17:00' },
      ],
      slot_duration_min: 30,
      buffer_min: 0,
      lead_time_hours: 24,
      breaks: [],
    }),
    active: true,
    created_at: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
    updated_at: new Date().toISOString(),
  },
]

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const bookingHandlers = [
  // GET /api/v1/calendar/booking-pages
  http.get(`${API}/api/v1/calendar/booking-pages`, ({ request }) => {
    const url = new URL(request.url)
    const includeInactive = url.searchParams.get('include_inactive') === 'true'
    const pages = includeInactive ? store : store.filter((p) => p.active)
    return HttpResponse.json({ pages })
  }),

  // POST /api/v1/calendar/booking-pages
  http.post(`${API}/api/v1/calendar/booking-pages`, async ({ request }) => {
    const body = (await request.json()) as {
      calendar_id: string
      slug: string
      company_name: string
      logo_url?: string | null
      services: RawBookingPage['services']
      availability_rules: string
    }

    if (store.some((p) => p.slug === body.slug)) {
      return HttpResponse.json({ error: 'slug already taken' }, { status: 409 })
    }

    const now = new Date().toISOString()
    const newPage: RawBookingPage = {
      id: `bp-${crypto.randomUUID().slice(0, 8)}`,
      tenant_id: 'tenant-001',
      calendar_id: body.calendar_id,
      slug: body.slug,
      company_name: body.company_name,
      logo_url: body.logo_url ?? null,
      services: body.services ?? [],
      availability_rules_json: body.availability_rules,
      active: true,
      created_at: now,
      updated_at: now,
    }
    store = [...store, newPage]
    return HttpResponse.json({ page: newPage }, { status: 201 })
  }),

  // GET /api/v1/calendar/booking-pages/:id
  http.get(`${API}/api/v1/calendar/booking-pages/:id`, ({ params }) => {
    const page = store.find((p) => p.id === params.id)
    if (!page) {
      return HttpResponse.json({ error: 'not found' }, { status: 404 })
    }
    return HttpResponse.json({ page })
  }),

  // PUT /api/v1/calendar/booking-pages/:id
  http.put(`${API}/api/v1/calendar/booking-pages/:id`, async ({ params, request }) => {
    const idx = store.findIndex((p) => p.id === params.id)
    if (idx === -1) {
      return HttpResponse.json({ error: 'not found' }, { status: 404 })
    }
    const body = (await request.json()) as {
      company_name?: string
      logo_url?: string | null
      services?: RawBookingPage['services']
      availability_rules?: string
      active?: boolean
    }
    const existing = store[idx]
    const updated: RawBookingPage = {
      ...existing,
      company_name: body.company_name ?? existing.company_name,
      logo_url: body.logo_url !== undefined ? body.logo_url : existing.logo_url,
      services: body.services ?? existing.services,
      availability_rules_json: body.availability_rules ?? existing.availability_rules_json,
      active: body.active !== undefined ? body.active : existing.active,
      updated_at: new Date().toISOString(),
    }
    store = store.map((p, i) => (i === idx ? updated : p))
    return HttpResponse.json({ page: updated })
  }),

  // DELETE /api/v1/calendar/booking-pages/:id
  http.delete(`${API}/api/v1/calendar/booking-pages/:id`, ({ params }) => {
    const exists = store.some((p) => p.id === params.id)
    if (!exists) {
      return HttpResponse.json({ error: 'not found' }, { status: 404 })
    }
    store = store.filter((p) => p.id !== params.id)
    return HttpResponse.json({ status: 'deleted' })
  }),
]
