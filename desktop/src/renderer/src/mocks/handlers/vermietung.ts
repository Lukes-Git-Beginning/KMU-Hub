import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { daysAgo, daysFromNow } from '../data/date-helpers'

const API = API_BASE_URL
const BASE = `${API}/api/v1/vermietung`

// ============================================================================
// Types (mirrors vermietung-types.ts wire shape)
// ============================================================================

type RentalStatus = 'reserved' | 'active' | 'completed' | 'cancelled'
type InspectionKind = 'handover' | 'return'

interface RentalObject {
  id: string
  tenant_id: string
  name: string
  description: string | null
  category: string
  daily_rate: number
  deposit: number
  location: string | null
  notes: string | null
  active: boolean
  created_at: string
  updated_at: string
  deleted_at: string | null
}

interface Rental {
  id: string
  tenant_id: string
  object_id: string
  contact_id: string | null
  renter_name: string
  start_date: string
  end_date: string
  status: RentalStatus
  total_price: number
  deposit_paid: boolean
  notes: string | null
  created_at: string
  updated_at: string
}

interface RentalInspection {
  id: string
  tenant_id: string
  rental_id: string
  kind: InspectionKind
  notes: string
  photo_urls: string[]
  performed_by: string | null
  created_at: string
  updated_at: string
}

// ============================================================================
// Mutable demo data — ported from stores/vermietung.ts (MOCK_OBJECTS)
// Mapped to wire-shape: camelCase → snake_case, type → category
// ============================================================================

const objects: RentalObject[] = [
  {
    id: 'vobj-001',
    tenant_id: 'tenant-001',
    name: 'Beamer Epson EB-W49',
    description: 'Full-HD Beamer mit 3800 Lumen, HDMI & USB, inkl. Fernbedienung und Tragetasche',
    category: 'gerät',
    daily_rate: 45,
    deposit: 0,
    location: 'Büro Zürich',
    notes: 'Seriennummer: EP-W49-2024-0871',
    active: true,
    created_at: daysAgo(90) + 'T08:00:00Z',
    updated_at: daysAgo(10) + 'T10:00:00Z',
    deleted_at: null,
  },
  {
    id: 'vobj-002',
    tenant_id: 'tenant-001',
    name: 'Konferenzraum A',
    description: 'Grosser Konferenzraum für bis zu 20 Personen, Beamer & Whiteboard vorhanden',
    category: 'raum',
    daily_rate: 150,
    deposit: 0,
    location: '2. OG',
    notes: null,
    active: true,
    created_at: daysAgo(180) + 'T08:00:00Z',
    updated_at: daysAgo(30) + 'T09:00:00Z',
    deleted_at: null,
  },
  {
    id: 'vobj-003',
    tenant_id: 'tenant-001',
    name: 'VW Transporter',
    description: 'VW T6.1 Transporter, Ladeflaeche 6m³, Anhängerkupplung',
    category: 'fahrzeug',
    daily_rate: 120,
    deposit: 500,
    location: 'Tiefgarage',
    notes: 'Kennzeichen: ZH-482731',
    active: true,
    created_at: daysAgo(120) + 'T08:00:00Z',
    updated_at: daysAgo(5) + 'T11:00:00Z',
    deleted_at: null,
  },
  {
    id: 'vobj-004',
    tenant_id: 'tenant-001',
    name: 'Hilti Bohrhammer',
    description: 'Hilti TE 30-A36, 36V Akku-Bohrhammer mit SDS-plus, inkl. 2 Akkus und Koffer',
    category: 'werkzeug',
    daily_rate: 35,
    deposit: 200,
    location: 'Lager Winterthur',
    notes: 'Seriennummer: HI-TE30-5523',
    active: true,
    created_at: daysAgo(60) + 'T08:00:00Z',
    updated_at: daysAgo(14) + 'T14:00:00Z',
    deleted_at: null,
  },
  {
    id: 'vobj-005',
    tenant_id: 'tenant-001',
    name: 'Hebebühne 12m',
    description: 'Selbstfahrende Scherenbuehne, max. Arbeitshoehe 12m, Tragkraft 350kg',
    category: 'gerät',
    daily_rate: 280,
    deposit: 1000,
    location: 'Aussenlager',
    notes: 'Seriennummer: HB-12M-0042 — derzeit in Wartung',
    active: false,
    created_at: daysAgo(200) + 'T08:00:00Z',
    updated_at: daysAgo(3) + 'T09:00:00Z',
    deleted_at: null,
  },
  {
    id: 'vobj-006',
    tenant_id: 'tenant-001',
    name: 'Schulungsraum B',
    description: 'Schulungsraum für bis zu 12 Personen, Flipchart & Beamer',
    category: 'raum',
    daily_rate: 100,
    deposit: 0,
    location: '1. OG',
    notes: null,
    active: true,
    created_at: daysAgo(180) + 'T08:00:00Z',
    updated_at: daysAgo(20) + 'T10:00:00Z',
    deleted_at: null,
  },
  {
    id: 'vobj-007',
    tenant_id: 'tenant-001',
    name: 'Laptop-Pool (5 Stk.)',
    description: '5x Lenovo ThinkPad T14s, 16GB RAM, 512GB SSD, Windows 11 Pro',
    category: 'gerät',
    daily_rate: 25,
    deposit: 0,
    location: 'IT-Büro',
    notes: null,
    active: true,
    created_at: daysAgo(150) + 'T08:00:00Z',
    updated_at: daysAgo(7) + 'T11:30:00Z',
    deleted_at: null,
  },
  {
    id: 'vobj-008',
    tenant_id: 'tenant-001',
    name: 'PKW-Anhänger',
    description: 'Einachser-Anhänger 750kg, Plane & Spriegel, Stuetzrad',
    category: 'fahrzeug',
    daily_rate: 55,
    deposit: 300,
    location: 'Tiefgarage',
    notes: 'Kennzeichen: ZH-ANH-1105',
    active: true,
    created_at: daysAgo(100) + 'T08:00:00Z',
    updated_at: daysAgo(21) + 'T09:00:00Z',
    deleted_at: null,
  },
]

// Ported from MOCK_RESERVATIONS — mapped to wire-shape Rental
// Status mapping: upcoming → reserved, active → active, completed → completed, cancelled → cancelled
// Dates shifted to be relative to today so the calendar always shows current data

const rentals: Rental[] = [
  {
    id: 'vren-001',
    tenant_id: 'tenant-001',
    object_id: 'vobj-002',
    contact_id: IDS.contacts.mueller,
    renter_name: 'Thomas Keller',
    start_date: daysAgo(1) + 'T08:00:00Z',
    end_date: daysAgo(1) + 'T18:00:00Z',
    status: 'active',
    total_price: 150,
    deposit_paid: false,
    notes: 'Kundenpräsentation Q2',
    created_at: daysAgo(5) + 'T09:00:00Z',
    updated_at: daysAgo(1) + 'T08:00:00Z',
  },
  {
    id: 'vren-002',
    tenant_id: 'tenant-001',
    object_id: 'vobj-007',
    contact_id: IDS.contacts.weber,
    renter_name: 'Sandra Müller',
    start_date: daysAgo(3) + 'T09:00:00Z',
    end_date: daysFromNow(2) + 'T18:00:00Z',
    status: 'active',
    total_price: 125,
    deposit_paid: false,
    notes: 'Workshop-Woche Digitalisierung',
    created_at: daysAgo(7) + 'T10:00:00Z',
    updated_at: daysAgo(3) + 'T09:00:00Z',
  },
  {
    id: 'vren-003',
    tenant_id: 'tenant-001',
    object_id: 'vobj-003',
    contact_id: IDS.contacts.gruber,
    renter_name: 'Meier Elektro AG',
    start_date: daysFromNow(2) + 'T08:00:00Z',
    end_date: daysFromNow(4) + 'T18:00:00Z',
    status: 'reserved',
    total_price: 360,
    deposit_paid: true,
    notes: 'Materialtransport Baustelle Winterthur',
    created_at: daysAgo(4) + 'T11:00:00Z',
    updated_at: daysAgo(4) + 'T11:00:00Z',
  },
  {
    id: 'vren-004',
    tenant_id: 'tenant-001',
    object_id: 'vobj-001',
    contact_id: IDS.contacts.schneider,
    renter_name: 'Lukas Brunner',
    start_date: daysFromNow(3) + 'T09:00:00Z',
    end_date: daysFromNow(3) + 'T18:00:00Z',
    status: 'reserved',
    total_price: 45,
    deposit_paid: false,
    notes: 'Präsentation Teammeeting',
    created_at: daysAgo(2) + 'T13:00:00Z',
    updated_at: daysAgo(2) + 'T13:00:00Z',
  },
  {
    id: 'vren-005',
    tenant_id: 'tenant-001',
    object_id: 'vobj-004',
    contact_id: null,
    renter_name: 'Reto Aeschlimann',
    start_date: daysFromNow(4) + 'T07:00:00Z',
    end_date: daysFromNow(6) + 'T18:00:00Z',
    status: 'reserved',
    total_price: 105,
    deposit_paid: true,
    notes: 'Montage Kabelkanäle Neubau',
    created_at: daysAgo(3) + 'T09:30:00Z',
    updated_at: daysAgo(3) + 'T09:30:00Z',
  },
  {
    id: 'vren-006',
    tenant_id: 'tenant-001',
    object_id: 'vobj-006',
    contact_id: IDS.contacts.huber,
    renter_name: 'Nicole Berger',
    start_date: daysAgo(8) + 'T08:00:00Z',
    end_date: daysAgo(8) + 'T17:00:00Z',
    status: 'completed',
    total_price: 100,
    deposit_paid: false,
    notes: 'Erste-Hilfe-Kurs',
    created_at: daysAgo(15) + 'T10:00:00Z',
    updated_at: daysAgo(8) + 'T17:00:00Z',
  },
  {
    id: 'vren-007',
    tenant_id: 'tenant-001',
    object_id: 'vobj-008',
    contact_id: IDS.contacts.bauer,
    renter_name: 'Baumann GmbH',
    start_date: daysAgo(13) + 'T08:00:00Z',
    end_date: daysAgo(11) + 'T18:00:00Z',
    status: 'completed',
    total_price: 165,
    deposit_paid: true,
    notes: 'Entsorgung Altmaterial',
    created_at: daysAgo(20) + 'T09:00:00Z',
    updated_at: daysAgo(11) + 'T18:00:00Z',
  },
  {
    id: 'vren-008',
    tenant_id: 'tenant-001',
    object_id: 'vobj-001',
    contact_id: IDS.contacts.wagner,
    renter_name: 'Marco Hartmann',
    start_date: daysAgo(15) + 'T09:00:00Z',
    end_date: daysAgo(15) + 'T17:00:00Z',
    status: 'completed',
    total_price: 45,
    deposit_paid: false,
    notes: 'Kundenschulung',
    created_at: daysAgo(20) + 'T11:00:00Z',
    updated_at: daysAgo(15) + 'T17:00:00Z',
  },
  {
    id: 'vren-009',
    tenant_id: 'tenant-001',
    object_id: 'vobj-003',
    contact_id: null,
    renter_name: 'Daniel Frei',
    start_date: daysAgo(10) + 'T07:00:00Z',
    end_date: daysAgo(9) + 'T18:00:00Z',
    status: 'completed',
    total_price: 240,
    deposit_paid: true,
    notes: 'Umzug Lagermaterial',
    created_at: daysAgo(16) + 'T10:00:00Z',
    updated_at: daysAgo(9) + 'T18:00:00Z',
  },
  {
    id: 'vren-010',
    tenant_id: 'tenant-001',
    object_id: 'vobj-002',
    contact_id: IDS.contacts.berger,
    renter_name: 'Irene Graf',
    start_date: daysFromNow(5) + 'T09:00:00Z',
    end_date: daysFromNow(5) + 'T17:00:00Z',
    status: 'reserved',
    total_price: 150,
    deposit_paid: false,
    notes: 'Budget-Review Meeting',
    created_at: daysAgo(2) + 'T14:00:00Z',
    updated_at: daysAgo(2) + 'T14:00:00Z',
  },
  {
    id: 'vren-011',
    tenant_id: 'tenant-001',
    object_id: 'vobj-004',
    contact_id: IDS.contacts.weber,
    renter_name: 'Weber Bau AG',
    start_date: daysAgo(17) + 'T08:00:00Z',
    end_date: daysAgo(16) + 'T18:00:00Z',
    status: 'cancelled',
    total_price: 70,
    deposit_paid: false,
    notes: null,
    created_at: daysAgo(25) + 'T10:00:00Z',
    updated_at: daysAgo(17) + 'T07:00:00Z',
  },
  {
    id: 'vren-012',
    tenant_id: 'tenant-001',
    object_id: 'vobj-006',
    contact_id: IDS.contacts.mueller,
    renter_name: 'Thomas Keller',
    start_date: daysFromNow(7) + 'T08:00:00Z',
    end_date: daysFromNow(8) + 'T17:00:00Z',
    status: 'reserved',
    total_price: 200,
    deposit_paid: false,
    notes: 'Sicherheitsschulung neues Personal',
    created_at: daysAgo(1) + 'T16:00:00Z',
    updated_at: daysAgo(1) + 'T16:00:00Z',
  },
]

// Ported from MOCK_ZUSTANDSPROTOKOLLE — mapped to RentalInspection wire-shape
// type: pickup → handover, return → return

const inspections: RentalInspection[] = [
  {
    id: 'vinsp-001',
    tenant_id: 'tenant-001',
    rental_id: 'vren-007',
    kind: 'handover',
    notes: 'Plane hat leichten Riss, Mieter informiert. Allgemeinzustand ok, Beleuchtung ok.',
    photo_urls: [],
    performed_by: IDS.users.thomas,
    created_at: daysAgo(13) + 'T08:30:00Z',
    updated_at: daysAgo(13) + 'T08:30:00Z',
  },
  {
    id: 'vinsp-002',
    tenant_id: 'tenant-001',
    rental_id: 'vren-007',
    kind: 'return',
    notes: 'Rückgabe ohne neue Schäden. Riss an Plane unveraendert.',
    photo_urls: [],
    performed_by: IDS.users.thomas,
    created_at: daysAgo(11) + 'T18:15:00Z',
    updated_at: daysAgo(11) + 'T18:15:00Z',
  },
  {
    id: 'vinsp-003',
    tenant_id: 'tenant-001',
    rental_id: 'vren-006',
    kind: 'handover',
    notes: 'Schulungsraum in einwandfreiem Zustand übergeben.',
    photo_urls: [],
    performed_by: IDS.users.stefan,
    created_at: daysAgo(8) + 'T08:05:00Z',
    updated_at: daysAgo(8) + 'T08:05:00Z',
  },
  {
    id: 'vinsp-004',
    tenant_id: 'tenant-001',
    rental_id: 'vren-006',
    kind: 'return',
    notes: 'Raum sauber hinterlassen, keine Beanstandungen.',
    photo_urls: [],
    performed_by: IDS.users.stefan,
    created_at: daysAgo(8) + 'T17:10:00Z',
    updated_at: daysAgo(8) + 'T17:10:00Z',
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

export const vermietungHandlers = [
  // --- Objects ---

  http.get(`${BASE}/objects`, ({ request }) => {
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 20)
    const search = url.searchParams.get('search')?.toLowerCase()
    const category = url.searchParams.get('category')
    const activeOnly = url.searchParams.get('active_only') === 'true'

    let filtered = objects
    if (activeOnly) filtered = filtered.filter(o => o.active)
    if (category) filtered = filtered.filter(o => o.category === category)
    if (search) {
      filtered = filtered.filter(
        o =>
          o.name.toLowerCase().includes(search) ||
          (o.description?.toLowerCase().includes(search) ?? false) ||
          (o.location?.toLowerCase().includes(search) ?? false)
      )
    }

    const result = paginate(filtered, page, pageSize)
    return HttpResponse.json({ objects: result.items, total: result.total })
  }),

  http.post(`${BASE}/objects`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const now = new Date().toISOString()
    const obj: RentalObject = {
      id: `vobj-${Date.now()}`,
      tenant_id: 'tenant-001',
      name: (body.name as string) ?? '',
      description: (body.description as string) ?? null,
      category: (body.category as string) ?? 'gerät',
      daily_rate: (body.daily_rate as number) ?? 0,
      deposit: (body.deposit as number) ?? 0,
      location: (body.location as string) ?? null,
      notes: (body.notes as string) ?? null,
      active: true,
      created_at: now,
      updated_at: now,
      deleted_at: null,
    }
    objects.push(obj)
    return HttpResponse.json({ object: obj }, { status: 201 })
  }),

  http.get(`${BASE}/objects/:id`, ({ params }) => {
    const obj = objects.find(o => o.id === params.id)
    if (!obj) return HttpResponse.json({ error: 'object not found' }, { status: 404 })
    return HttpResponse.json({ object: obj })
  }),

  http.patch(`${BASE}/objects/:id`, async ({ params, request }) => {
    const obj = objects.find(o => o.id === params.id)
    if (!obj) return HttpResponse.json({ error: 'object not found' }, { status: 404 })
    const body = (await request.json()) as Partial<RentalObject>
    Object.assign(obj, body)
    obj.updated_at = new Date().toISOString()
    return HttpResponse.json({ object: obj })
  }),

  http.delete(`${BASE}/objects/:id`, ({ params }) => {
    const idx = objects.findIndex(o => o.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'object not found' }, { status: 404 })
    objects.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  http.get(`${BASE}/objects/:objectId/availability`, ({ params, request }) => {
    const url = new URL(request.url)
    const startDate = url.searchParams.get('start_date')
    const endDate = url.searchParams.get('end_date')
    const excludeRentalId = url.searchParams.get('exclude_rental_id')

    if (!startDate || !endDate) {
      return HttpResponse.json({ error: 'start_date and end_date required' }, { status: 400 })
    }

    const start = new Date(startDate).getTime()
    const end = new Date(endDate).getTime()

    const conflicting = rentals.filter(r => {
      if (r.object_id !== params.objectId) return false
      if (r.status === 'cancelled') return false
      if (excludeRentalId && r.id === excludeRentalId) return false
      const rs = new Date(r.start_date).getTime()
      const re = new Date(r.end_date).getTime()
      return rs < end && re > start
    })

    return HttpResponse.json({
      available: conflicting.length === 0,
      conflicting_rentals: conflicting,
    })
  }),

  // --- Rentals ---

  http.get(`${BASE}/rentals`, ({ request }) => {
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 20)
    const statusFilter = url.searchParams.get('status') as RentalStatus | null
    const objectId = url.searchParams.get('object_id')
    const from = url.searchParams.get('from')
    const to = url.searchParams.get('to')

    let filtered = rentals
    if (statusFilter) filtered = filtered.filter(r => r.status === statusFilter)
    if (objectId) filtered = filtered.filter(r => r.object_id === objectId)
    if (from) {
      const fromTs = new Date(from).getTime()
      filtered = filtered.filter(r => new Date(r.end_date).getTime() >= fromTs)
    }
    if (to) {
      const toTs = new Date(to).getTime()
      filtered = filtered.filter(r => new Date(r.start_date).getTime() <= toTs)
    }

    const result = paginate(filtered, page, pageSize)
    return HttpResponse.json({ rentals: result.items, total: result.total })
  }),

  http.post(`${BASE}/rentals`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const now = new Date().toISOString()
    const rental: Rental = {
      id: `vren-${Date.now()}`,
      tenant_id: 'tenant-001',
      object_id: (body.object_id as string) ?? '',
      contact_id: (body.contact_id as string) ?? null,
      renter_name: (body.renter_name as string) ?? '',
      start_date: (body.start_date as string) ?? '',
      end_date: (body.end_date as string) ?? '',
      status: 'reserved',
      total_price: (body.total_price as number) ?? 0,
      deposit_paid: (body.deposit_paid as boolean) ?? false,
      notes: (body.notes as string) ?? null,
      created_at: now,
      updated_at: now,
    }
    rentals.push(rental)
    return HttpResponse.json({ rental }, { status: 201 })
  }),

  http.get(`${BASE}/rentals/:id`, ({ params }) => {
    const rental = rentals.find(r => r.id === params.id)
    if (!rental) return HttpResponse.json({ error: 'rental not found' }, { status: 404 })
    return HttpResponse.json({ rental })
  }),

  http.patch(`${BASE}/rentals/:id`, async ({ params, request }) => {
    const rental = rentals.find(r => r.id === params.id)
    if (!rental) return HttpResponse.json({ error: 'rental not found' }, { status: 404 })
    const body = (await request.json()) as Partial<Rental>
    Object.assign(rental, body)
    rental.updated_at = new Date().toISOString()
    return HttpResponse.json({ rental })
  }),

  http.delete(`${BASE}/rentals/:id`, ({ params }) => {
    const idx = rentals.findIndex(r => r.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'rental not found' }, { status: 404 })
    rentals.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  http.post(`${BASE}/rentals/:id/start`, ({ params }) => {
    const rental = rentals.find(r => r.id === params.id)
    if (!rental) return HttpResponse.json({ error: 'rental not found' }, { status: 404 })
    rental.status = 'active'
    rental.updated_at = new Date().toISOString()
    return HttpResponse.json({ rental })
  }),

  http.post(`${BASE}/rentals/:id/end`, ({ params }) => {
    const rental = rentals.find(r => r.id === params.id)
    if (!rental) return HttpResponse.json({ error: 'rental not found' }, { status: 404 })
    rental.status = 'completed'
    rental.updated_at = new Date().toISOString()
    return HttpResponse.json({ rental })
  }),

  // --- Inspections ---

  http.get(`${BASE}/rentals/:rentalId/inspections`, ({ params, request }) => {
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 20)

    const filtered = inspections.filter(i => i.rental_id === params.rentalId)
    const result = paginate(filtered, page, pageSize)
    return HttpResponse.json({ inspections: result.items, total: result.total })
  }),

  http.post(`${BASE}/rentals/:rentalId/inspections`, async ({ params, request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const now = new Date().toISOString()
    const inspection: RentalInspection = {
      id: `vinsp-${Date.now()}`,
      tenant_id: 'tenant-001',
      rental_id: params.rentalId as string,
      kind: (body.kind as InspectionKind) ?? 'handover',
      notes: (body.notes as string) ?? '',
      photo_urls: (body.photo_urls as string[]) ?? [],
      performed_by: (body.performed_by as string) ?? null,
      created_at: now,
      updated_at: now,
    }
    inspections.push(inspection)
    return HttpResponse.json({ inspection }, { status: 201 })
  }),

  http.get(`${BASE}/inspections/:id`, ({ params }) => {
    const inspection = inspections.find(i => i.id === params.id)
    if (!inspection) return HttpResponse.json({ error: 'inspection not found' }, { status: 404 })
    return HttpResponse.json({ inspection })
  }),

  http.patch(`${BASE}/inspections/:id`, async ({ params, request }) => {
    const inspection = inspections.find(i => i.id === params.id)
    if (!inspection) return HttpResponse.json({ error: 'inspection not found' }, { status: 404 })
    const body = (await request.json()) as Partial<RentalInspection>
    Object.assign(inspection, body)
    inspection.updated_at = new Date().toISOString()
    return HttpResponse.json({ inspection })
  }),

  // --- Calendar ---

  http.get(`${BASE}/calendar`, ({ request }) => {
    const url = new URL(request.url)
    const month = Number(url.searchParams.get('month'))
    const year = Number(url.searchParams.get('year'))
    const objectIdFilter = url.searchParams.get('object_id')

    // Build calendar: one entry per object with its rentals in the requested month
    const monthStart = new Date(year, month - 1, 1).getTime()
    const monthEnd = new Date(year, month, 0, 23, 59, 59, 999).getTime()

    let relevantObjects = objects.filter(o => o.active)
    if (objectIdFilter) relevantObjects = relevantObjects.filter(o => o.id === objectIdFilter)

    const entries = relevantObjects.map(obj => {
      const objRentals = rentals.filter(r => {
        if (r.object_id !== obj.id) return false
        if (r.status === 'cancelled') return false
        const rs = new Date(r.start_date).getTime()
        const re = new Date(r.end_date).getTime()
        return rs <= monthEnd && re >= monthStart
      })
      return {
        object_id: obj.id,
        object_name: obj.name,
        rentals: objRentals,
      }
    })

    return HttpResponse.json({ entries })
  }),
]
