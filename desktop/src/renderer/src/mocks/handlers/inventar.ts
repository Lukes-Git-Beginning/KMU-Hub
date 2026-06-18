import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { daysAgo } from '../data/date-helpers'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function paginate<T>(arr: T[], page = 1, pageSize = 20): T[] {
  const start = (page - 1) * pageSize
  return arr.slice(start, start + pageSize)
}

function newId(): string {
  return Math.random().toString(36).slice(2, 10)
}

// ---------------------------------------------------------------------------
// Mock IDs
// ---------------------------------------------------------------------------

const IDS = {
  loc1: 'loc-mock-1',
  loc2: 'loc-mock-2',
  loc3: 'loc-mock-3',
  inv1: 'inv-mock-1',
  inv2: 'inv-mock-2',
  inv3: 'inv-mock-3',
  inv4: 'inv-mock-4',
  inv5: 'inv-mock-5',
  inv6: 'inv-mock-6',
  inv7: 'inv-mock-7',
  inv8: 'inv-mock-8',
  inv9: 'inv-mock-9',
  inv10: 'inv-mock-10',
  inv11: 'inv-mock-11',
  inv12: 'inv-mock-12',
  inv13: 'inv-mock-13',
  inv14: 'inv-mock-14',
  inv15: 'inv-mock-15',
  ses1: 'ses-mock-1',
  ses2: 'ses-mock-2',
  mov1: 'mov-mock-1',
  mov2: 'mov-mock-2',
  mov3: 'mov-mock-3',
  mov4: 'mov-mock-4',
  mov5: 'mov-mock-5',
  mov6: 'mov-mock-6',
  warn1: 'warn-mock-1',
  warn2: 'warn-mock-2',
}

// ---------------------------------------------------------------------------
// Types (local, matching API wire shape)
// ---------------------------------------------------------------------------

interface InventarItem {
  id: string
  tenant_id: string
  name: string
  sku: string
  barcode: string | null
  quantity: number
  min_quantity: number
  unit: string
  location: string | null
  location_id: string | null
  category: string | null
  price: number | null
  currency: string | null
  batch_number: string | null
  serial_numbers: string[]
  linked_purchase_order: string | null
  created_at: string
  updated_at: string
}

interface InventarMovement {
  id: string
  tenant_id: string
  item_id: string
  movement_type: 'in' | 'out' | 'adjustment' | 'transfer'
  quantity: number
  performed_by: string | null
  reason: string
  reference: string | null
  location_from: string | null
  location_to: string | null
  notes: string | null
  created_at: string
}

interface StockWarning {
  id: string
  tenant_id: string
  item_id: string
  threshold: number
  current_quantity: number
  status: 'active' | 'acknowledged' | 'resolved'
  created_at: string
  acknowledged_at: string | null
  acknowledged_by: string | null
}

interface InventoryLocation {
  id: string
  tenant_id: string
  name: string
  address: string
  type: 'warehouse' | 'store' | 'vehicle'
  created_at: string
  updated_at: string
}

interface InventurCount {
  id: string
  tenant_id: string
  session_id: string
  item_id: string
  expected: number
  counted: number | null
  counted_at: string | null
}

interface InventurSession {
  id: string
  tenant_id: string
  name: string
  date: string
  status: 'open' | 'counting' | 'review' | 'completed'
  location_id: string | null
  created_by: string | null
  counts: InventurCount[]
  created_at: string
  updated_at: string
}

// ---------------------------------------------------------------------------
// Mutable state
// ---------------------------------------------------------------------------

let locations: InventoryLocation[] = [
  { id: IDS.loc1, tenant_id: 'tenant-001', name: 'Hauptlager', address: 'Industriestrasse 12, 8005 Zürich', type: 'warehouse', created_at: daysAgo(30) + 'T08:00:00Z', updated_at: daysAgo(1) + 'T08:00:00Z' },
  { id: IDS.loc2, tenant_id: 'tenant-001', name: 'Filiale Bern', address: 'Marktgasse 5, 3011 Bern', type: 'store', created_at: daysAgo(25) + 'T08:00:00Z', updated_at: daysAgo(2) + 'T08:00:00Z' },
  { id: IDS.loc3, tenant_id: 'tenant-001', name: 'Lieferwagen #1', address: 'Mobil', type: 'vehicle', created_at: daysAgo(20) + 'T08:00:00Z', updated_at: daysAgo(3) + 'T08:00:00Z' },
]

let items: InventarItem[] = [
  { id: IDS.inv1, tenant_id: 'tenant-001', name: 'Kabelkanal 20x10mm', sku: 'KK-2010', barcode: '7610000000001', quantity: 450, min_quantity: 100, unit: 'Meter', location: 'Hauptlager', location_id: IDS.loc1, category: 'Elektromaterial', price: 2.50, currency: 'EUR', batch_number: 'CH-2026-0142', serial_numbers: [], linked_purchase_order: null, created_at: daysAgo(90) + 'T08:00:00Z', updated_at: daysAgo(4) + 'T09:30:00Z' },
  { id: IDS.inv2, tenant_id: 'tenant-001', name: 'LED Panel 60x60cm', sku: 'LED-6060', barcode: '7610000000002', quantity: 24, min_quantity: 10, unit: 'Stück', location: 'Hauptlager', location_id: IDS.loc1, category: 'Beleuchtung', price: 89.90, currency: 'EUR', batch_number: null, serial_numbers: ['LP-60-001', 'LP-60-002', 'LP-60-003'], linked_purchase_order: null, created_at: daysAgo(85) + 'T08:00:00Z', updated_at: daysAgo(5) + 'T10:00:00Z' },
  { id: IDS.inv3, tenant_id: 'tenant-001', name: 'Steckdose T13', sku: 'SD-T13', barcode: '7610000000003', quantity: 8, min_quantity: 50, unit: 'Stück', location: 'Hauptlager', location_id: IDS.loc1, category: 'Elektromaterial', price: 4.20, currency: 'EUR', batch_number: null, serial_numbers: [], linked_purchase_order: 'PO-2026-004', created_at: daysAgo(80) + 'T08:00:00Z', updated_at: daysAgo(6) + 'T08:15:00Z' },
  { id: IDS.inv4, tenant_id: 'tenant-001', name: 'Sicherung 16A', sku: 'SI-16A', barcode: null, quantity: 120, min_quantity: 30, unit: 'Stück', location: 'Hauptlager', location_id: IDS.loc1, category: 'Sicherungen', price: 3.80, currency: 'EUR', batch_number: 'SI-2026-0088', serial_numbers: [], linked_purchase_order: null, created_at: daysAgo(78) + 'T08:00:00Z', updated_at: daysAgo(4) + 'T09:00:00Z' },
  { id: IDS.inv5, tenant_id: 'tenant-001', name: 'NYM-J 3x1.5mm²', sku: 'NYM-315', barcode: '7610000000005', quantity: 1200, min_quantity: 500, unit: 'Meter', location: 'Hauptlager', location_id: IDS.loc1, category: 'Kabel', price: 1.10, currency: 'EUR', batch_number: null, serial_numbers: [], linked_purchase_order: null, created_at: daysAgo(75) + 'T08:00:00Z', updated_at: daysAgo(4) + 'T14:00:00Z' },
  { id: IDS.inv6, tenant_id: 'tenant-001', name: 'Verteilerdose AP', sku: 'VD-AP', barcode: null, quantity: 45, min_quantity: 20, unit: 'Stück', location: 'Filiale Bern', location_id: IDS.loc2, category: 'Elektromaterial', price: 6.50, currency: 'EUR', batch_number: null, serial_numbers: [], linked_purchase_order: null, created_at: daysAgo(70) + 'T08:00:00Z', updated_at: daysAgo(7) + 'T11:00:00Z' },
  { id: IDS.inv7, tenant_id: 'tenant-001', name: 'Bewegungsmelder', sku: 'BM-180', barcode: '7610000000007', quantity: 15, min_quantity: 5, unit: 'Stück', location: 'Filiale Bern', location_id: IDS.loc2, category: 'Sensoren', price: 34.90, currency: 'EUR', batch_number: null, serial_numbers: ['BM-180-A01', 'BM-180-A02'], linked_purchase_order: null, created_at: daysAgo(65) + 'T08:00:00Z', updated_at: daysAgo(8) + 'T10:00:00Z' },
  { id: IDS.inv8, tenant_id: 'tenant-001', name: 'Thermostat digital', sku: 'TH-DIG', barcode: null, quantity: 3, min_quantity: 5, unit: 'Stück', location: 'Filiale Bern', location_id: IDS.loc2, category: 'Heizung', price: 79.00, currency: 'EUR', batch_number: null, serial_numbers: ['TH-DIG-2025-001', 'TH-DIG-2025-002', 'TH-DIG-2025-003'], linked_purchase_order: null, created_at: daysAgo(60) + 'T08:00:00Z', updated_at: daysAgo(9) + 'T14:20:00Z' },
  { id: IDS.inv9, tenant_id: 'tenant-001', name: 'FI-Schutzschalter 30mA', sku: 'FI-30', barcode: null, quantity: 18, min_quantity: 10, unit: 'Stück', location: 'Hauptlager', location_id: IDS.loc1, category: 'Sicherungen', price: 42.50, currency: 'EUR', batch_number: 'FI-2026-0034', serial_numbers: [], linked_purchase_order: null, created_at: daysAgo(55) + 'T08:00:00Z', updated_at: daysAgo(4) + 'T09:00:00Z' },
  { id: IDS.inv10, tenant_id: 'tenant-001', name: 'Leerrohr 25mm', sku: 'LR-25', barcode: null, quantity: 350, min_quantity: 100, unit: 'Meter', location: 'Hauptlager', location_id: IDS.loc1, category: 'Rohre', price: 1.80, currency: 'EUR', batch_number: null, serial_numbers: [], linked_purchase_order: null, created_at: daysAgo(50) + 'T08:00:00Z', updated_at: daysAgo(5) + 'T10:00:00Z' },
  { id: IDS.inv11, tenant_id: 'tenant-001', name: 'Schrauben-Set M4', sku: 'SS-M4', barcode: null, quantity: 67, min_quantity: 20, unit: 'Packung', location: 'Lieferwagen #1', location_id: IDS.loc3, category: 'Befestigung', price: 8.90, currency: 'EUR', batch_number: null, serial_numbers: [], linked_purchase_order: null, created_at: daysAgo(45) + 'T08:00:00Z', updated_at: daysAgo(6) + 'T12:00:00Z' },
  { id: IDS.inv12, tenant_id: 'tenant-001', name: 'Rauchmelder EN 14604', sku: 'RM-EN14', barcode: '7610000000012', quantity: 28, min_quantity: 10, unit: 'Stück', location: 'Hauptlager', location_id: IDS.loc1, category: 'Sicherheit', price: 19.90, currency: 'EUR', batch_number: 'RM-2026-0012', serial_numbers: ['RM-EN14-001', 'RM-EN14-002'], linked_purchase_order: null, created_at: daysAgo(40) + 'T08:00:00Z', updated_at: daysAgo(7) + 'T11:30:00Z' },
  { id: IDS.inv13, tenant_id: 'tenant-001', name: 'Klebeband Isolier', sku: 'KB-ISO', barcode: null, quantity: 92, min_quantity: 30, unit: 'Rolle', location: 'Lieferwagen #1', location_id: IDS.loc3, category: 'Verbrauch', price: 3.20, currency: 'EUR', batch_number: null, serial_numbers: [], linked_purchase_order: null, created_at: daysAgo(35) + 'T08:00:00Z', updated_at: daysAgo(4) + 'T14:00:00Z' },
  { id: IDS.inv14, tenant_id: 'tenant-001', name: 'Kabelschuh 6mm²', sku: 'KS-6', barcode: null, quantity: 200, min_quantity: 100, unit: 'Stück', location: 'Hauptlager', location_id: IDS.loc1, category: 'Kabelzubehör', price: 0.35, currency: 'EUR', batch_number: 'KS-2026-0200', serial_numbers: [], linked_purchase_order: null, created_at: daysAgo(30) + 'T08:00:00Z', updated_at: daysAgo(5) + 'T09:00:00Z' },
  { id: IDS.inv15, tenant_id: 'tenant-001', name: 'Smart Switch Zigbee', sku: 'SS-ZB', barcode: '7610000000015', quantity: 12, min_quantity: 5, unit: 'Stück', location: 'Filiale Bern', location_id: IDS.loc2, category: 'Smart Home', price: 54.90, currency: 'CHF', batch_number: null, serial_numbers: ['SSZ-2026-A01'], linked_purchase_order: null, created_at: daysAgo(25) + 'T08:00:00Z', updated_at: daysAgo(8) + 'T10:00:00Z' },
]

let movements: InventarMovement[] = [
  { id: IDS.mov1, tenant_id: 'tenant-001', item_id: IDS.inv1, movement_type: 'in', quantity: 200, performed_by: null, reason: 'Wareneingang', reference: 'PO-2024-031', location_from: null, location_to: 'Hauptlager', notes: null, created_at: daysAgo(4) + 'T09:30:00Z' },
  { id: IDS.mov2, tenant_id: 'tenant-001', item_id: IDS.inv3, movement_type: 'out', quantity: 12, performed_by: null, reason: 'Auftrag', reference: 'Auftrag #A-445', location_from: 'Hauptlager', location_to: null, notes: null, created_at: daysAgo(4) + 'T08:15:00Z' },
  { id: IDS.mov3, tenant_id: 'tenant-001', item_id: IDS.inv5, movement_type: 'transfer', quantity: 100, performed_by: null, reason: 'Transfer', reference: 'Transfer #T-12', location_from: 'Hauptlager', location_to: 'Lieferwagen #1', notes: null, created_at: daysAgo(5) + 'T16:45:00Z' },
  { id: IDS.mov4, tenant_id: 'tenant-001', item_id: IDS.inv8, movement_type: 'out', quantity: 2, performed_by: null, reason: 'Auftrag', reference: 'Auftrag #A-443', location_from: 'Filiale Bern', location_to: null, notes: null, created_at: daysAgo(5) + 'T14:20:00Z' },
  { id: IDS.mov5, tenant_id: 'tenant-001', item_id: IDS.inv2, movement_type: 'in', quantity: 10, performed_by: null, reason: 'Wareneingang', reference: 'PO-2024-030', location_from: null, location_to: 'Hauptlager', notes: null, created_at: daysAgo(5) + 'T10:00:00Z' },
  { id: IDS.mov6, tenant_id: 'tenant-001', item_id: IDS.inv12, movement_type: 'adjustment', quantity: -2, performed_by: null, reason: 'Inventur-Korrektur', reference: 'Inventur-Korrektur', location_from: 'Hauptlager', location_to: null, notes: 'Defekte Ware aussortiert', created_at: daysAgo(6) + 'T11:30:00Z' },
]

let warnings: StockWarning[] = [
  { id: IDS.warn1, tenant_id: 'tenant-001', item_id: IDS.inv3, threshold: 50, current_quantity: 8, status: 'active', created_at: daysAgo(3) + 'T08:00:00Z', acknowledged_at: null, acknowledged_by: null },
  { id: IDS.warn2, tenant_id: 'tenant-001', item_id: IDS.inv8, threshold: 5, current_quantity: 3, status: 'active', created_at: daysAgo(2) + 'T08:00:00Z', acknowledged_at: null, acknowledged_by: null },
]

let inventurSessions: InventurSession[] = [
  {
    id: IDS.ses1, tenant_id: 'tenant-001', name: 'Jahresinventur Hauptlager 2026', date: '2026-02-15', status: 'review', location_id: IDS.loc1, created_by: 'Markus Weber',
    counts: [
      { id: newId(), tenant_id: 'tenant-001', session_id: IDS.ses1, item_id: IDS.inv1, expected: 450, counted: 448, counted_at: daysAgo(3) + 'T10:00:00Z' },
      { id: newId(), tenant_id: 'tenant-001', session_id: IDS.ses1, item_id: IDS.inv4, expected: 120, counted: 120, counted_at: daysAgo(3) + 'T10:00:00Z' },
      { id: newId(), tenant_id: 'tenant-001', session_id: IDS.ses1, item_id: IDS.inv5, expected: 1200, counted: 1185, counted_at: daysAgo(3) + 'T10:00:00Z' },
      { id: newId(), tenant_id: 'tenant-001', session_id: IDS.ses1, item_id: IDS.inv9, expected: 18, counted: 18, counted_at: daysAgo(3) + 'T10:00:00Z' },
      { id: newId(), tenant_id: 'tenant-001', session_id: IDS.ses1, item_id: IDS.inv10, expected: 350, counted: 352, counted_at: daysAgo(3) + 'T10:00:00Z' },
      { id: newId(), tenant_id: 'tenant-001', session_id: IDS.ses1, item_id: IDS.inv12, expected: 28, counted: 26, counted_at: daysAgo(3) + 'T10:00:00Z' },
      { id: newId(), tenant_id: 'tenant-001', session_id: IDS.ses1, item_id: IDS.inv14, expected: 200, counted: 198, counted_at: daysAgo(3) + 'T10:00:00Z' },
      { id: newId(), tenant_id: 'tenant-001', session_id: IDS.ses1, item_id: IDS.inv3, expected: 8, counted: 8, counted_at: daysAgo(3) + 'T10:00:00Z' },
    ],
    created_at: daysAgo(10) + 'T08:00:00Z', updated_at: daysAgo(3) + 'T10:00:00Z',
  },
  {
    id: IDS.ses2, tenant_id: 'tenant-001', name: 'Stichprobe Filiale Bern', date: '2026-01-20', status: 'completed', location_id: IDS.loc2, created_by: 'Sarah Müller',
    counts: [
      { id: newId(), tenant_id: 'tenant-001', session_id: IDS.ses2, item_id: IDS.inv6, expected: 45, counted: 45, counted_at: daysAgo(28) + 'T10:00:00Z' },
      { id: newId(), tenant_id: 'tenant-001', session_id: IDS.ses2, item_id: IDS.inv7, expected: 15, counted: 15, counted_at: daysAgo(28) + 'T10:00:00Z' },
      { id: newId(), tenant_id: 'tenant-001', session_id: IDS.ses2, item_id: IDS.inv8, expected: 5, counted: 5, counted_at: daysAgo(28) + 'T10:00:00Z' },
      { id: newId(), tenant_id: 'tenant-001', session_id: IDS.ses2, item_id: IDS.inv15, expected: 12, counted: 12, counted_at: daysAgo(28) + 'T10:00:00Z' },
    ],
    created_at: daysAgo(35) + 'T08:00:00Z', updated_at: daysAgo(28) + 'T10:00:00Z',
  },
]

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const inventarHandlers = [
  // Items - List
  http.get(`${API}/api/v1/inventar/items`, ({ request }) => {
    const url = new URL(request.url)
    const page = parseInt(url.searchParams.get('page') ?? '1')
    const pageSize = parseInt(url.searchParams.get('page_size') ?? '20')
    const search = url.searchParams.get('search') ?? ''
    const location = url.searchParams.get('location') ?? ''
    const lowStock = url.searchParams.get('low_stock') === 'true'

    let result = [...items]
    if (search) {
      const q = search.toLowerCase()
      result = result.filter(i => i.name.toLowerCase().includes(q) || i.sku.toLowerCase().includes(q))
    }
    if (location) {
      result = result.filter(i => i.location === location)
    }
    if (lowStock) {
      result = result.filter(i => i.quantity <= i.min_quantity)
    }
    return HttpResponse.json({ items: paginate(result, page, pageSize), total: result.length })
  }),

  // Items - Create
  http.post(`${API}/api/v1/inventar/items`, async ({ request }) => {
    const body = await request.json() as Partial<InventarItem>
    const item: InventarItem = {
      id: newId(),
      tenant_id: 'tenant-001',
      name: body.name ?? '',
      sku: body.sku ?? '',
      barcode: body.barcode ?? null,
      quantity: body.quantity ?? 0,
      min_quantity: body.min_quantity ?? 0,
      unit: body.unit ?? 'Stück',
      location: body.location ?? null,
      location_id: body.location_id ?? null,
      category: body.category ?? null,
      price: body.price ?? null,
      currency: body.currency ?? 'EUR',
      batch_number: body.batch_number ?? null,
      serial_numbers: body.serial_numbers ?? [],
      linked_purchase_order: body.linked_purchase_order ?? null,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    items = [...items, item]
    return HttpResponse.json({ item }, { status: 201 })
  }),

  // Items - Get by ID
  http.get(`${API}/api/v1/inventar/items/:id`, ({ params }) => {
    const item = items.find(i => i.id === params.id)
    if (!item) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    return HttpResponse.json({ item })
  }),

  // Items - Update
  http.patch(`${API}/api/v1/inventar/items/:id`, async ({ params, request }) => {
    const body = await request.json() as Partial<InventarItem>
    const idx = items.findIndex(i => i.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    items[idx] = { ...items[idx], ...body, updated_at: new Date().toISOString() }
    return HttpResponse.json({ item: items[idx] })
  }),

  // Items - Delete
  http.delete(`${API}/api/v1/inventar/items/:id`, ({ params }) => {
    items = items.filter(i => i.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  // Stock - Adjust
  http.post(`${API}/api/v1/inventar/items/:id/adjust`, async ({ params, request }) => {
    const body = await request.json() as { delta: number; reason: string; performed_by?: string }
    const idx = items.findIndex(i => i.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    items[idx] = { ...items[idx], quantity: items[idx].quantity + body.delta, updated_at: new Date().toISOString() }
    // Record movement
    const mov: InventarMovement = {
      id: newId(), tenant_id: 'tenant-001', item_id: String(params.id),
      movement_type: 'adjustment', quantity: body.delta,
      performed_by: body.performed_by ?? null, reason: body.reason,
      reference: null, location_from: null, location_to: null, notes: null,
      created_at: new Date().toISOString(),
    }
    movements = [...movements, mov]
    return HttpResponse.json({ item: items[idx] })
  }),

  // Stock - Transfer
  http.post(`${API}/api/v1/inventar/transfer`, async ({ request }) => {
    const body = await request.json() as { from_item_id: string; to_item_id: string; quantity: number; reason: string }
    const fromIdx = items.findIndex(i => i.id === body.from_item_id)
    const toIdx = items.findIndex(i => i.id === body.to_item_id)
    if (fromIdx !== -1) items[fromIdx] = { ...items[fromIdx], quantity: items[fromIdx].quantity - body.quantity }
    if (toIdx !== -1) items[toIdx] = { ...items[toIdx], quantity: items[toIdx].quantity + body.quantity }
    return new HttpResponse(null, { status: 200 })
  }),

  // Movements - Record
  http.post(`${API}/api/v1/inventar/items/:id/movements`, async ({ params, request }) => {
    const body = await request.json() as Partial<InventarMovement>
    const movement: InventarMovement = {
      id: newId(), tenant_id: 'tenant-001', item_id: String(params.id),
      movement_type: body.movement_type ?? 'in',
      quantity: body.quantity ?? 0,
      performed_by: body.performed_by ?? null,
      reason: body.reason ?? '',
      reference: body.reference ?? null,
      location_from: body.location_from ?? null,
      location_to: body.location_to ?? null,
      notes: body.notes ?? null,
      created_at: new Date().toISOString(),
    }
    movements = [...movements, movement]
    return HttpResponse.json({ movement }, { status: 201 })
  }),

  // Movements - List for item
  http.get(`${API}/api/v1/inventar/items/:id/movements`, ({ params, request }) => {
    const url = new URL(request.url)
    const page = parseInt(url.searchParams.get('page') ?? '1')
    const pageSize = parseInt(url.searchParams.get('page_size') ?? '20')
    const result = movements.filter(m => m.item_id === params.id).sort((a, b) => b.created_at.localeCompare(a.created_at))
    return HttpResponse.json({ movements: paginate(result, page, pageSize), total: result.length })
  }),

  // History - List for item
  http.get(`${API}/api/v1/inventar/items/:id/history`, ({ params, request }) => {
    const url = new URL(request.url)
    const page = parseInt(url.searchParams.get('page') ?? '1')
    const pageSize = parseInt(url.searchParams.get('page_size') ?? '20')
    const result = movements.filter(m => m.item_id === params.id).sort((a, b) => b.created_at.localeCompare(a.created_at))
    return HttpResponse.json({ movements: paginate(result, page, pageSize), total: result.length })
  }),

  // Warnings - List
  http.get(`${API}/api/v1/inventar/warnings`, ({ request }) => {
    const url = new URL(request.url)
    const page = parseInt(url.searchParams.get('page') ?? '1')
    const pageSize = parseInt(url.searchParams.get('page_size') ?? '20')
    const status = url.searchParams.get('status')
    let result = [...warnings]
    if (status) result = result.filter(w => w.status === status)
    return HttpResponse.json({ warnings: paginate(result, page, pageSize), total: result.length })
  }),

  // Warnings - Create
  http.post(`${API}/api/v1/inventar/warnings`, async ({ request }) => {
    const body = await request.json() as Partial<StockWarning>
    const warning: StockWarning = {
      id: newId(), tenant_id: 'tenant-001',
      item_id: body.item_id ?? '',
      threshold: body.threshold ?? 0,
      current_quantity: body.current_quantity ?? 0,
      status: 'active',
      created_at: new Date().toISOString(),
      acknowledged_at: null, acknowledged_by: null,
    }
    warnings = [...warnings, warning]
    return HttpResponse.json({ warning }, { status: 201 })
  }),

  // Warnings - Update
  http.patch(`${API}/api/v1/inventar/warnings/:id`, async ({ params, request }) => {
    const body = await request.json() as Partial<StockWarning>
    const idx = warnings.findIndex(w => w.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    warnings[idx] = { ...warnings[idx], ...body }
    return HttpResponse.json({ warning: warnings[idx] })
  }),

  // Warnings - Acknowledge
  http.post(`${API}/api/v1/inventar/warnings/:id/acknowledge`, async ({ params, request }) => {
    const body = await request.json() as { acknowledged_by?: string } | null
    const idx = warnings.findIndex(w => w.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    warnings[idx] = {
      ...warnings[idx],
      status: 'acknowledged',
      acknowledged_at: new Date().toISOString(),
      acknowledged_by: (body && body.acknowledged_by) ? body.acknowledged_by : null,
    }
    return HttpResponse.json({ warning: warnings[idx] })
  }),

  // Report - flat shape (no wrapper key)
  http.get(`${API}/api/v1/inventar/report`, () => {
    const totalItems = items.length
    const totalQuantity = items.reduce((sum, i) => sum + i.quantity, 0)
    const lowStockCount = items.filter(i => i.quantity <= i.min_quantity).length
    const activeWarnings = warnings.filter(w => w.status === 'active').length
    return HttpResponse.json({ total_items: totalItems, total_quantity: totalQuantity, low_stock_count: lowStockCount, active_warnings: activeWarnings })
  }),

  // Export - CSV download
  http.get(`${API}/api/v1/inventar/export`, () => {
    const header = 'id,name,sku,quantity,min_quantity,unit,location,category\n'
    const rows = items.map(i => `${i.id},${i.name},${i.sku},${i.quantity},${i.min_quantity},${i.unit},${i.location ?? ''},${i.category ?? ''}`).join('\n')
    return new HttpResponse(header + rows, { status: 200, headers: { 'Content-Type': 'text/csv', 'Content-Disposition': 'attachment; filename="inventar.csv"' } })
  }),

  // Locations - List
  http.get(`${API}/api/v1/inventar/locations`, ({ request }) => {
    const url = new URL(request.url)
    const page = parseInt(url.searchParams.get('page') ?? '1')
    const pageSize = parseInt(url.searchParams.get('page_size') ?? '20')
    return HttpResponse.json({ locations: paginate(locations, page, pageSize), total: locations.length })
  }),

  // Locations - Create
  http.post(`${API}/api/v1/inventar/locations`, async ({ request }) => {
    const body = await request.json() as Partial<InventoryLocation>
    const location: InventoryLocation = {
      id: newId(), tenant_id: 'tenant-001',
      name: body.name ?? '',
      address: body.address ?? '',
      type: body.type ?? 'warehouse',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    locations = [...locations, location]
    return HttpResponse.json({ location }, { status: 201 })
  }),

  // Locations - Get by ID
  http.get(`${API}/api/v1/inventar/locations/:id`, ({ params }) => {
    const location = locations.find(l => l.id === params.id)
    if (!location) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    return HttpResponse.json({ location })
  }),

  // Locations - Update
  http.patch(`${API}/api/v1/inventar/locations/:id`, async ({ params, request }) => {
    const body = await request.json() as Partial<InventoryLocation>
    const idx = locations.findIndex(l => l.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    locations[idx] = { ...locations[idx], ...body, updated_at: new Date().toISOString() }
    return HttpResponse.json({ location: locations[idx] })
  }),

  // Locations - Delete
  http.delete(`${API}/api/v1/inventar/locations/:id`, ({ params }) => {
    locations = locations.filter(l => l.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  // Inventur Sessions - List
  http.get(`${API}/api/v1/inventar/inventur`, ({ request }) => {
    const url = new URL(request.url)
    const page = parseInt(url.searchParams.get('page') ?? '1')
    const pageSize = parseInt(url.searchParams.get('page_size') ?? '20')
    return HttpResponse.json({ sessions: paginate(inventurSessions, page, pageSize), total: inventurSessions.length })
  }),

  // Inventur Sessions - Create
  http.post(`${API}/api/v1/inventar/inventur`, async ({ request }) => {
    const body = await request.json() as { name: string; date?: string; location_id?: string; item_ids?: string[] }
    const sessionId = newId()
    const session: InventurSession = {
      id: sessionId, tenant_id: 'tenant-001',
      name: body.name,
      date: body.date ?? new Date().toISOString().split('T')[0],
      status: 'open',
      location_id: body.location_id ?? null,
      created_by: null,
      counts: (body.item_ids ?? []).map(itemId => ({
        id: newId(), tenant_id: 'tenant-001',
        session_id: sessionId,
        item_id: itemId,
        expected: items.find(i => i.id === itemId)?.quantity ?? 0,
        counted: null, counted_at: null,
      })),
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    inventurSessions = [...inventurSessions, session]
    return HttpResponse.json({ session }, { status: 201 })
  }),

  // Inventur Sessions - Get by ID
  http.get(`${API}/api/v1/inventar/inventur/:id`, ({ params }) => {
    const session = inventurSessions.find(s => s.id === params.id)
    if (!session) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    return HttpResponse.json({ session })
  }),

  // Inventur Sessions - Update status
  http.patch(`${API}/api/v1/inventar/inventur/:id/status`, async ({ params, request }) => {
    const body = await request.json() as { status: InventurSession['status'] }
    const idx = inventurSessions.findIndex(s => s.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    inventurSessions[idx] = { ...inventurSessions[idx], status: body.status, updated_at: new Date().toISOString() }
    return HttpResponse.json({ session: inventurSessions[idx] })
  }),

  // Inventur Sessions - Delete
  http.delete(`${API}/api/v1/inventar/inventur/:id`, ({ params }) => {
    inventurSessions = inventurSessions.filter(s => s.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  // Inventur Counts - Upsert
  http.post(`${API}/api/v1/inventar/inventur/:id/counts`, async ({ params, request }) => {
    const body = await request.json() as { item_id: string; counted: number }
    const sessionIdx = inventurSessions.findIndex(s => s.id === params.id)
    if (sessionIdx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    const countIdx = inventurSessions[sessionIdx].counts.findIndex(c => c.item_id === body.item_id)
    let count: InventurCount
    if (countIdx !== -1) {
      inventurSessions[sessionIdx].counts[countIdx] = {
        ...inventurSessions[sessionIdx].counts[countIdx],
        counted: body.counted,
        counted_at: new Date().toISOString(),
      }
      count = inventurSessions[sessionIdx].counts[countIdx]
    } else {
      count = {
        id: newId(), tenant_id: 'tenant-001',
        session_id: String(params.id),
        item_id: body.item_id,
        expected: items.find(i => i.id === body.item_id)?.quantity ?? 0,
        counted: body.counted,
        counted_at: new Date().toISOString(),
      }
      inventurSessions[sessionIdx].counts = [...inventurSessions[sessionIdx].counts, count]
    }
    return HttpResponse.json({ count }, { status: 201 })
  }),

  // Inventur Sessions - Book differences
  http.post(`${API}/api/v1/inventar/inventur/:id/book`, async ({ params, request }) => {
    const body = await request.json() as { booked_by?: string } | null
    void body
    const sessionIdx = inventurSessions.findIndex(s => s.id === params.id)
    if (sessionIdx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    const session = inventurSessions[sessionIdx]
    // Apply differences to item quantities
    for (const count of session.counts) {
      if (count.counted !== null && count.counted !== count.expected) {
        const itemIdx = items.findIndex(i => i.id === count.item_id)
        if (itemIdx !== -1) {
          const diff = count.counted - count.expected
          items[itemIdx] = { ...items[itemIdx], quantity: items[itemIdx].quantity + diff, updated_at: new Date().toISOString() }
        }
      }
    }
    inventurSessions[sessionIdx] = { ...session, status: 'completed', updated_at: new Date().toISOString() }
    return HttpResponse.json({ session: inventurSessions[sessionIdx] })
  }),
]
