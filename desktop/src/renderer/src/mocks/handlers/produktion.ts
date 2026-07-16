import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'

const BASE = `${API_BASE_URL}/api/v1/produktion`

// ---- Seed helpers -----------------------------------------------------------
// All dates are relative to "today" so the demo (order table, Gantt chart,
// overdue badges) always shows a live-looking dataset.

function dayIso(offsetDays: number, hour = 7): string {
  const d = new Date()
  d.setDate(d.getDate() + offsetDays)
  d.setHours(hour, 0, 0, 0)
  return d.toISOString()
}

const YEAR = new Date().getFullYear()

// ---- Seed data (mutable — handlers below are stateful) ----------------------

const seedBoms = [
  {
    id: 'bom-001',
    tenant_id: 'tenant-demo',
    product_name: 'Schaltschrank Typ A',
    sku: 'SSA-2026',
    version: '2.1',
    active: true,
    notes: '',
    created_by: null,
    created_at: dayIso(-160),
    updated_at: dayIso(-160),
    items: [
      { id: 'bi-001', bom_id: 'bom-001', material_name: 'Gehäuse 600x800', quantity: 1, unit: 'Stk', sort_order: 0 },
      { id: 'bi-002', bom_id: 'bom-001', material_name: 'Kabelkanal 40x60', quantity: 4, unit: 'm', sort_order: 1 },
      { id: 'bi-003', bom_id: 'bom-001', material_name: 'Klemmenblock 10pol', quantity: 3, unit: 'Set', sort_order: 2 },
      { id: 'bi-004', bom_id: 'bom-001', material_name: 'Hauptschalter 63A', quantity: 1, unit: 'Stk', sort_order: 3 },
      { id: 'bi-005', bom_id: 'bom-001', material_name: 'Verdrahtung H07V-K 1,5mm²', quantity: 25, unit: 'm', sort_order: 4 },
    ],
  },
  {
    id: 'bom-002',
    tenant_id: 'tenant-demo',
    product_name: 'Steuereinheit Kompakt',
    sku: 'SEK-2026',
    version: '1.0',
    active: true,
    notes: '',
    created_by: null,
    created_at: dayIso(-150),
    updated_at: dayIso(-150),
    items: [
      { id: 'bi-006', bom_id: 'bom-002', material_name: 'Platine STM32F4', quantity: 1, unit: 'Stk', sort_order: 0 },
      { id: 'bi-007', bom_id: 'bom-002', material_name: 'Kühlkörper passiv', quantity: 1, unit: 'Stk', sort_order: 1 },
      { id: 'bi-008', bom_id: 'bom-002', material_name: 'Kondensator 100uF', quantity: 8, unit: 'Stk', sort_order: 2 },
      { id: 'bi-009', bom_id: 'bom-002', material_name: 'RJ45 Buchse', quantity: 2, unit: 'Stk', sort_order: 3 },
    ],
  },
  {
    id: 'bom-003',
    tenant_id: 'tenant-demo',
    product_name: 'Kabelkonfektion Serie K',
    sku: 'KKS-K-2026',
    version: '1.4',
    active: true,
    notes: 'Kundenspezifische Längen laut Auftragsblatt.',
    created_by: null,
    created_at: dayIso(-120),
    updated_at: dayIso(-40),
    items: [
      { id: 'bi-010', bom_id: 'bom-003', material_name: 'Leitung NYM-J 5x2,5mm²', quantity: 12, unit: 'm', sort_order: 0 },
      { id: 'bi-011', bom_id: 'bom-003', material_name: 'Steckverbinder M12 4-pol', quantity: 2, unit: 'Stk', sort_order: 1 },
      { id: 'bi-012', bom_id: 'bom-003', material_name: 'Kabelverschraubung M20', quantity: 4, unit: 'Stk', sort_order: 2 },
      { id: 'bi-013', bom_id: 'bom-003', material_name: 'Schrumpfschlauch-Set', quantity: 1, unit: 'Set', sort_order: 3 },
    ],
  },
  {
    id: 'bom-004',
    tenant_id: 'tenant-demo',
    product_name: 'LED-Panel 40W',
    sku: 'LED-P40',
    version: '4.0',
    active: true,
    notes: '',
    created_by: null,
    created_at: dayIso(-100),
    updated_at: dayIso(-100),
    items: [
      { id: 'bi-014', bom_id: 'bom-004', material_name: 'LED-Modul 40W 4000K', quantity: 1, unit: 'Stk', sort_order: 0 },
      { id: 'bi-015', bom_id: 'bom-004', material_name: 'LED-Treiber dimmbar', quantity: 1, unit: 'Stk', sort_order: 1 },
      { id: 'bi-016', bom_id: 'bom-004', material_name: 'Aluminium-Rahmen 62x62', quantity: 1, unit: 'Stk', sort_order: 2 },
      { id: 'bi-017', bom_id: 'bom-004', material_name: 'Diffusor-Platte Opal', quantity: 1, unit: 'Stk', sort_order: 3 },
    ],
  },
  {
    id: 'bom-005',
    tenant_id: 'tenant-demo',
    product_name: 'Sensormodul Temperatur/Feuchte',
    sku: 'SM-TF-01',
    version: '1.0',
    active: false,
    notes: 'Abgelöst durch Rev. B — nicht mehr fertigen.',
    created_by: null,
    created_at: dayIso(-200),
    updated_at: dayIso(-60),
    items: [
      { id: 'bi-018', bom_id: 'bom-005', material_name: 'Sensor SHT40', quantity: 1, unit: 'Stk', sort_order: 0 },
      { id: 'bi-019', bom_id: 'bom-005', material_name: 'PCB Custom 30x20mm', quantity: 1, unit: 'Stk', sort_order: 1 },
      { id: 'bi-020', bom_id: 'bom-005', material_name: 'ESP32-C3 Modul', quantity: 1, unit: 'Stk', sort_order: 2 },
    ],
  },
]

const seedMachines = [
  {
    id: 'machine-001',
    tenant_id: 'tenant-demo',
    name: 'CNC-Fräse Alpha',
    type: 'CNC',
    status: 'in_use',
    notes: '',
    created_by: null,
    created_at: dayIso(-300),
    updated_at: dayIso(-3),
  },
  {
    id: 'machine-002',
    tenant_id: 'tenant-demo',
    name: 'Schweißroboter R1',
    type: 'Roboter',
    status: 'in_use',
    notes: '',
    created_by: null,
    created_at: dayIso(-300),
    updated_at: dayIso(-1),
  },
  {
    id: 'machine-003',
    tenant_id: 'tenant-demo',
    name: 'Drehbank DM-200',
    type: 'Drehbank',
    status: 'maintenance',
    notes: 'Spindellager-Wartung, voraussichtlich wieder verfügbar in 4 Tagen.',
    created_by: null,
    created_at: dayIso(-300),
    updated_at: dayIso(-2),
  },
  {
    id: 'machine-004',
    tenant_id: 'tenant-demo',
    name: 'Montageband 1',
    type: 'Montage',
    status: 'in_use',
    notes: '',
    created_by: null,
    created_at: dayIso(-280),
    updated_at: dayIso(-1),
  },
  {
    id: 'machine-005',
    tenant_id: 'tenant-demo',
    name: 'Prüfstand Elektro',
    type: 'Prüfung',
    status: 'available',
    notes: '',
    created_by: null,
    created_at: dayIso(-280),
    updated_at: dayIso(-7),
  },
]

// Wire shape mirrors WorkStepResponse (order_id, assignee, description).
const seedWorkSteps = [
  // order-001 — Schaltschrank, 2/5 done, one running -> 40 %
  { id: 'ws-001', tenant_id: 'tenant-demo', order_id: 'order-001', step_nr: 1, name: 'Gehäuse vorbereiten', description: 'Entgraten, bohren, grundieren', duration_minutes: 120, status: 'completed', assignee: 'Thomas Keller', created_at: dayIso(-10), updated_at: dayIso(-9) },
  { id: 'ws-002', tenant_id: 'tenant-demo', order_id: 'order-001', step_nr: 2, name: 'Hutschienen montieren', description: 'Hutschienen und Kabelkanäle einbauen', duration_minutes: 60, status: 'completed', assignee: 'Thomas Keller', created_at: dayIso(-10), updated_at: dayIso(-7) },
  { id: 'ws-003', tenant_id: 'tenant-demo', order_id: 'order-001', step_nr: 3, name: 'Sicherungen einbauen', description: 'LSS, FI und Hauptschalter montieren', duration_minutes: 90, status: 'in_progress', assignee: 'Lukas Brunner', created_at: dayIso(-10), updated_at: dayIso(-1) },
  { id: 'ws-004', tenant_id: 'tenant-demo', order_id: 'order-001', step_nr: 4, name: 'Verdrahtung', description: 'Interne Verdrahtung nach Plan', duration_minutes: 180, status: 'pending', assignee: '', created_at: dayIso(-10), updated_at: dayIso(-10) },
  { id: 'ws-005', tenant_id: 'tenant-demo', order_id: 'order-001', step_nr: 5, name: 'Prüfung / Inbetriebnahme', description: 'Isolationsmessung, Funktionstest, Protokoll', duration_minutes: 60, status: 'pending', assignee: '', created_at: dayIso(-10), updated_at: dayIso(-10) },
  // order-002 — Steuereinheit, 1/4 done -> 25 %
  { id: 'ws-006', tenant_id: 'tenant-demo', order_id: 'order-002', step_nr: 1, name: 'Gehäuse fertigen', description: 'Aluminiumgehäuse CNC-fräsen und eloxieren', duration_minutes: 90, status: 'completed', assignee: 'Werner Stoeckli', created_at: dayIso(-6), updated_at: dayIso(-4) },
  { id: 'ws-007', tenant_id: 'tenant-demo', order_id: 'order-002', step_nr: 2, name: 'Elektronik bestücken', description: 'Platine, Display und Netzteil montieren', duration_minutes: 120, status: 'in_progress', assignee: 'Irene Graf', created_at: dayIso(-6), updated_at: dayIso(-1) },
  { id: 'ws-008', tenant_id: 'tenant-demo', order_id: 'order-002', step_nr: 3, name: 'Firmware flashen', description: 'Firmware und Konfiguration aufspielen', duration_minutes: 45, status: 'pending', assignee: '', created_at: dayIso(-6), updated_at: dayIso(-6) },
  { id: 'ws-009', tenant_id: 'tenant-demo', order_id: 'order-002', step_nr: 4, name: 'Endprüfung', description: 'Kalibrierung, Kommunikationstest, Verpackung', duration_minutes: 60, status: 'pending', assignee: '', created_at: dayIso(-6), updated_at: dayIso(-6) },
  // order-003 — planned, everything pending
  { id: 'ws-010', tenant_id: 'tenant-demo', order_id: 'order-003', step_nr: 1, name: 'Material kommissionieren', description: 'Komponenten laut Stückliste bereitstellen', duration_minutes: 45, status: 'pending', assignee: '', created_at: dayIso(-4), updated_at: dayIso(-4) },
  { id: 'ws-011', tenant_id: 'tenant-demo', order_id: 'order-003', step_nr: 2, name: 'Vormontage', description: 'Gehäuse und Schienen vorbereiten', duration_minutes: 120, status: 'pending', assignee: '', created_at: dayIso(-4), updated_at: dayIso(-4) },
  { id: 'ws-012', tenant_id: 'tenant-demo', order_id: 'order-003', step_nr: 3, name: 'Endmontage + Prüfung', description: 'Komplettieren, prüfen, dokumentieren', duration_minutes: 150, status: 'pending', assignee: '', created_at: dayIso(-4), updated_at: dayIso(-4) },
  // order-005 — overdue rush order, 3/4 done -> 75 %
  { id: 'ws-013', tenant_id: 'tenant-demo', order_id: 'order-005', step_nr: 1, name: 'Leitungen ablängen', description: 'Zuschnitt laut Längenliste', duration_minutes: 60, status: 'completed', assignee: 'Thomas Keller', created_at: dayIso(-20), updated_at: dayIso(-16) },
  { id: 'ws-014', tenant_id: 'tenant-demo', order_id: 'order-005', step_nr: 2, name: 'Crimpen + Stecker', description: 'Steckverbinder montieren', duration_minutes: 180, status: 'completed', assignee: 'Irene Graf', created_at: dayIso(-20), updated_at: dayIso(-8) },
  { id: 'ws-015', tenant_id: 'tenant-demo', order_id: 'order-005', step_nr: 3, name: 'Kennzeichnung', description: 'Adern und Bündel beschriften', duration_minutes: 45, status: 'completed', assignee: 'Irene Graf', created_at: dayIso(-20), updated_at: dayIso(-5) },
  { id: 'ws-016', tenant_id: 'tenant-demo', order_id: 'order-005', step_nr: 4, name: 'Durchgangsprüfung', description: 'Elektrische Prüfung + Protokoll', duration_minutes: 90, status: 'in_progress', assignee: 'Werner Stoeckli', created_at: dayIso(-20), updated_at: dayIso(-1) },
  // order-004 — completed order, all steps done
  { id: 'ws-017', tenant_id: 'tenant-demo', order_id: 'order-004', step_nr: 1, name: 'Gehäuse fertigen', description: '', duration_minutes: 90, status: 'completed', assignee: 'Werner Stoeckli', created_at: dayIso(-30), updated_at: dayIso(-20) },
  { id: 'ws-018', tenant_id: 'tenant-demo', order_id: 'order-004', step_nr: 2, name: 'Elektronik bestücken', description: '', duration_minutes: 120, status: 'completed', assignee: 'Irene Graf', created_at: dayIso(-30), updated_at: dayIso(-16) },
  { id: 'ws-019', tenant_id: 'tenant-demo', order_id: 'order-004', step_nr: 3, name: 'Endprüfung', description: '', duration_minutes: 60, status: 'completed', assignee: 'Werner Stoeckli', created_at: dayIso(-30), updated_at: dayIso(-13) },
]

const seedQualityChecks = [
  { id: 'qc-001', tenant_id: 'tenant-demo', order_id: 'order-001', inspector: 'Werner Stoeckli', checked_at: dayIso(-3, 10), passed: true, defects_found: 0, notes: 'Stichprobe 3 von 12 Schränken geprüft, alle Maße korrekt.', created_by: null, created_at: dayIso(-3, 10), updated_at: dayIso(-3, 10) },
  { id: 'qc-002', tenant_id: 'tenant-demo', order_id: 'order-002', inspector: 'Irene Graf', checked_at: dayIso(-1, 14), passed: false, defects_found: 2, notes: 'Lötpunkte an 2 Platinen nacharbeiten.', created_by: null, created_at: dayIso(-1, 14), updated_at: dayIso(-1, 14) },
  { id: 'qc-003', tenant_id: 'tenant-demo', order_id: 'order-004', inspector: 'Werner Stoeckli', checked_at: dayIso(-14, 9), passed: true, defects_found: 0, notes: 'Endkontrolle komplett, Charge freigegeben.', created_by: null, created_at: dayIso(-14, 9), updated_at: dayIso(-14, 9) },
  { id: 'qc-004', tenant_id: 'tenant-demo', order_id: 'order-005', inspector: 'Werner Stoeckli', checked_at: dayIso(-4, 11), passed: false, defects_found: 3, notes: 'Crimpkontakte bei 3 Sätzen außerhalb Toleranz, Nacharbeit läuft.', created_by: null, created_at: dayIso(-4, 11), updated_at: dayIso(-4, 11) },
  { id: 'qc-005', tenant_id: 'tenant-demo', order_id: 'order-008', inspector: 'Irene Graf', checked_at: dayIso(-26, 15), passed: true, defects_found: 1, notes: '1 Panel mit Kratzer im Diffusor aussortiert, Rest i.O.', created_by: null, created_at: dayIso(-26, 15), updated_at: dayIso(-26, 15) },
]

const seedOrders = [
  { id: 'order-001', tenant_id: 'tenant-demo', order_number: `PA-${YEAR}-001`, product_name: 'Schaltschrank Typ A', quantity: 12, status: 'in_progress', planned_start: dayIso(-10), planned_end: dayIso(5, 16), actual_start: dayIso(-10, 8), actual_end: null, priority: 2, bom_id: 'bom-001', notes: 'Eilauftrag Kunde Meier — Liefertermin fix.', created_by: null, created_at: dayIso(-14), updated_at: dayIso(-1) },
  { id: 'order-002', tenant_id: 'tenant-demo', order_number: `PA-${YEAR}-002`, product_name: 'Steuereinheit Kompakt', quantity: 40, status: 'in_progress', planned_start: dayIso(-6), planned_end: dayIso(9, 16), actual_start: dayIso(-6, 8), actual_end: null, priority: 3, bom_id: 'bom-002', notes: '', created_by: null, created_at: dayIso(-12), updated_at: dayIso(-1) },
  { id: 'order-003', tenant_id: 'tenant-demo', order_number: `PA-${YEAR}-003`, product_name: 'Schaltschrank Typ A', quantity: 5, status: 'planned', planned_start: dayIso(6), planned_end: dayIso(16, 16), actual_start: null, actual_end: null, priority: 4, bom_id: 'bom-001', notes: '', created_by: null, created_at: dayIso(-4), updated_at: dayIso(-4) },
  { id: 'order-004', tenant_id: 'tenant-demo', order_number: `PA-${YEAR}-004`, product_name: 'Steuereinheit Kompakt', quantity: 25, status: 'completed', planned_start: dayIso(-30), planned_end: dayIso(-12, 16), actual_start: dayIso(-30, 7), actual_end: dayIso(-13, 15), priority: 3, bom_id: 'bom-002', notes: 'Termingerecht fertig.', created_by: null, created_at: dayIso(-36), updated_at: dayIso(-13) },
  { id: 'order-005', tenant_id: 'tenant-demo', order_number: `PA-${YEAR}-005`, product_name: 'Kabelkonfektion Serie K', quantity: 60, status: 'in_progress', planned_start: dayIso(-20), planned_end: dayIso(-2, 16), actual_start: dayIso(-20, 7), actual_end: null, priority: 1, bom_id: 'bom-003', notes: 'Verzögerung durch Materiallieferung — Kunde informiert.', created_by: null, created_at: dayIso(-25), updated_at: dayIso(-1) },
  { id: 'order-006', tenant_id: 'tenant-demo', order_number: `PA-${YEAR}-006`, product_name: 'Kabelkonfektion Serie K', quantity: 30, status: 'planned', planned_start: dayIso(12), planned_end: dayIso(22, 16), actual_start: null, actual_end: null, priority: 3, bom_id: 'bom-003', notes: '', created_by: null, created_at: dayIso(-2), updated_at: dayIso(-2) },
  { id: 'order-007', tenant_id: 'tenant-demo', order_number: `PA-${YEAR}-007`, product_name: 'LED-Panel 40W', quantity: 150, status: 'cancelled', planned_start: dayIso(-15), planned_end: dayIso(2, 16), actual_start: null, actual_end: null, priority: 5, bom_id: 'bom-004', notes: 'Kunde hat den Abruf storniert.', created_by: null, created_at: dayIso(-18), updated_at: dayIso(-9) },
  { id: 'order-008', tenant_id: 'tenant-demo', order_number: `PA-${YEAR}-008`, product_name: 'LED-Panel 40W', quantity: 80, status: 'completed', planned_start: dayIso(-45), planned_end: dayIso(-25, 16), actual_start: dayIso(-45, 7), actual_end: dayIso(-26, 14), priority: 2, bom_id: 'bom-004', notes: '', created_by: null, created_at: dayIso(-50), updated_at: dayIso(-26) },
]

// Bookings sit inside the Gantt window (today −7 … +28 days).
const seedBookings = [
  { id: 'mb-001', tenant_id: 'tenant-demo', machine_id: 'machine-001', production_order_id: 'order-001', starts_at: dayIso(-3, 8), ends_at: dayIso(2, 16), status: 'in_use', notes: '', created_by: null, created_at: dayIso(-5), updated_at: dayIso(-3) },
  { id: 'mb-002', tenant_id: 'tenant-demo', machine_id: 'machine-004', production_order_id: 'order-001', starts_at: dayIso(2, 8), ends_at: dayIso(5, 16), status: 'booked', notes: '', created_by: null, created_at: dayIso(-5), updated_at: dayIso(-5) },
  { id: 'mb-003', tenant_id: 'tenant-demo', machine_id: 'machine-002', production_order_id: 'order-002', starts_at: dayIso(-1, 8), ends_at: dayIso(4, 16), status: 'in_use', notes: '', created_by: null, created_at: dayIso(-4), updated_at: dayIso(-1) },
  { id: 'mb-004', tenant_id: 'tenant-demo', machine_id: 'machine-001', production_order_id: 'order-003', starts_at: dayIso(7, 8), ends_at: dayIso(11, 16), status: 'booked', notes: '', created_by: null, created_at: dayIso(-2), updated_at: dayIso(-2) },
  { id: 'mb-005', tenant_id: 'tenant-demo', machine_id: 'machine-005', production_order_id: 'order-002', starts_at: dayIso(5, 8), ends_at: dayIso(8, 16), status: 'booked', notes: '', created_by: null, created_at: dayIso(-2), updated_at: dayIso(-2) },
  { id: 'mb-006', tenant_id: 'tenant-demo', machine_id: 'machine-004', production_order_id: 'order-005', starts_at: dayIso(-6, 8), ends_at: dayIso(-1, 16), status: 'in_use', notes: '', created_by: null, created_at: dayIso(-8), updated_at: dayIso(-6) },
]

let idCounter = 1
function nextId(prefix: string): string {
  return `${prefix}-new-${idCounter++}`
}

// ---- Handlers (stateful: create/update/delete/status mutate the seed arrays,
// so invalidate+refetch shows the change — the previous handlers returned
// transformed copies without mutating, which made every mutation a no-op in
// the demo) ----

export const produktionHandlers = [
  http.get(`${BASE}/orders`, () => {
    return HttpResponse.json({ orders: seedOrders, total: seedOrders.length })
  }),

  http.get(`${BASE}/orders/:orderId`, ({ params }) => {
    const order = seedOrders.find((o) => o.id === params['orderId'])
    if (!order) return new HttpResponse(null, { status: 404 })
    return HttpResponse.json({ order })
  }),

  http.post(`${BASE}/orders`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const now = new Date().toISOString()
    const order = {
      id: nextId('order'),
      tenant_id: 'tenant-demo',
      status: 'planned',
      priority: 3,
      actual_start: null,
      actual_end: null,
      notes: '',
      bom_id: undefined,
      created_by: null,
      created_at: now,
      updated_at: now,
      ...body,
    }
    seedOrders.unshift(order as unknown as (typeof seedOrders)[number])
    return HttpResponse.json({ order }, { status: 201 })
  }),

  http.patch(`${BASE}/orders/:orderId`, async ({ params, request }) => {
    const order = seedOrders.find((o) => o.id === params['orderId'])
    if (!order) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as Record<string, unknown>
    Object.assign(order, body, { updated_at: new Date().toISOString() })
    return HttpResponse.json({ order })
  }),

  http.delete(`${BASE}/orders/:orderId`, ({ params }) => {
    const idx = seedOrders.findIndex((o) => o.id === params['orderId'])
    if (idx >= 0) seedOrders.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  http.post(`${BASE}/orders/:orderId/start`, ({ params }) => {
    const order = seedOrders.find((o) => o.id === params['orderId'])
    if (!order) return new HttpResponse(null, { status: 404 })
    order.status = 'in_progress'
    order.actual_start = new Date().toISOString()
    order.updated_at = order.actual_start
    return HttpResponse.json({ order })
  }),

  http.post(`${BASE}/orders/:orderId/complete`, ({ params }) => {
    const order = seedOrders.find((o) => o.id === params['orderId'])
    if (!order) return new HttpResponse(null, { status: 404 })
    order.status = 'completed'
    order.actual_end = new Date().toISOString()
    order.updated_at = order.actual_end
    return HttpResponse.json({ order })
  }),

  http.post(`${BASE}/orders/:orderId/cancel`, ({ params }) => {
    const order = seedOrders.find((o) => o.id === params['orderId'])
    if (!order) return new HttpResponse(null, { status: 404 })
    order.status = 'cancelled'
    order.updated_at = new Date().toISOString()
    return HttpResponse.json({ order })
  }),

  // ---- Machine booking handlers ----

  http.get(`${BASE}/bookings`, () => {
    return HttpResponse.json({ bookings: seedBookings, total: seedBookings.length })
  }),

  http.post(`${BASE}/bookings`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const now = new Date().toISOString()
    const booking = {
      id: nextId('mb'),
      tenant_id: 'tenant-demo',
      status: 'booked',
      notes: '',
      created_by: null,
      created_at: now,
      updated_at: now,
      ...body,
    }
    seedBookings.push(booking as (typeof seedBookings)[number])
    return HttpResponse.json(booking, { status: 201 })
  }),

  http.patch(`${BASE}/bookings/:bookingId`, async ({ params, request }) => {
    const booking = seedBookings.find((b) => b.id === params['bookingId'])
    if (!booking) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as Record<string, unknown>
    Object.assign(booking, body, { updated_at: new Date().toISOString() })
    return HttpResponse.json(booking)
  }),

  http.delete(`${BASE}/bookings/:bookingId`, ({ params }) => {
    const idx = seedBookings.findIndex((b) => b.id === params['bookingId'])
    if (idx >= 0) seedBookings.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // ---- BOM handlers (client expects the { bom } envelope on single reads) ----

  http.get(`${BASE}/boms`, () => {
    return HttpResponse.json({ boms: seedBoms, total: seedBoms.length })
  }),

  http.post(`${BASE}/boms`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const bomId = nextId('bom')
    const now = new Date().toISOString()
    const newBom = {
      id: bomId,
      tenant_id: 'tenant-demo',
      product_name: String(body['product_name'] ?? ''),
      sku: String(body['sku'] ?? ''),
      version: String(body['version'] ?? '1.0'),
      active: body['active'] !== false,
      notes: String(body['notes'] ?? ''),
      created_by: null,
      created_at: now,
      updated_at: now,
      items: Array.isArray(body['items'])
        ? (body['items'] as Array<Record<string, unknown>>).map((it, i) => ({
            id: nextId('bi'),
            bom_id: bomId,
            material_name: String(it['material_name'] ?? ''),
            quantity: Number(it['quantity'] ?? 1),
            unit: String(it['unit'] ?? 'Stk'),
            sort_order: i,
          }))
        : [],
    }
    seedBoms.push(newBom)
    return HttpResponse.json({ bom: newBom }, { status: 201 })
  }),

  http.get(`${BASE}/boms/:bomId`, ({ params }) => {
    const bom = seedBoms.find((b) => b.id === params['bomId'])
    if (!bom) return new HttpResponse(null, { status: 404 })
    return HttpResponse.json({ bom })
  }),

  http.patch(`${BASE}/boms/:bomId`, async ({ params, request }) => {
    const bom = seedBoms.find((b) => b.id === params['bomId'])
    if (!bom) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as Record<string, unknown>
    Object.assign(bom, body, { updated_at: new Date().toISOString() })
    return HttpResponse.json({ bom })
  }),

  http.delete(`${BASE}/boms/:bomId`, ({ params }) => {
    const idx = seedBoms.findIndex((b) => b.id === params['bomId'])
    if (idx >= 0) seedBoms.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // ---- Work step handlers (client expects the { step } envelope) ----

  http.get(`${BASE}/orders/:orderId/steps`, ({ params }) => {
    const steps = seedWorkSteps.filter((s) => s.order_id === params['orderId'])
    return HttpResponse.json({ steps, total: steps.length })
  }),

  http.post(`${BASE}/orders/:orderId/steps`, async ({ params, request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const now = new Date().toISOString()
    const orderSteps = seedWorkSteps.filter((s) => s.order_id === params['orderId'])
    const step = {
      id: nextId('ws'),
      tenant_id: 'tenant-demo',
      order_id: String(params['orderId']),
      step_nr: Number(body['step_nr'] ?? orderSteps.length + 1),
      name: String(body['name'] ?? ''),
      description: String(body['description'] ?? ''),
      duration_minutes: Number(body['duration_minutes'] ?? 0),
      status: 'pending',
      assignee: String(body['assignee'] ?? ''),
      created_at: now,
      updated_at: now,
    }
    seedWorkSteps.push(step)
    return HttpResponse.json({ step }, { status: 201 })
  }),

  http.patch(`${BASE}/orders/:orderId/steps/:stepId`, async ({ params, request }) => {
    const step = seedWorkSteps.find((s) => s.id === params['stepId'])
    if (!step) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as Record<string, unknown>
    Object.assign(step, body, { updated_at: new Date().toISOString() })
    return HttpResponse.json({ step })
  }),

  http.delete(`${BASE}/orders/:orderId/steps/:stepId`, ({ params }) => {
    const idx = seedWorkSteps.findIndex((s) => s.id === params['stepId'])
    if (idx >= 0) seedWorkSteps.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // ---- Machine handlers (client expects the { machine } envelope) ----

  http.get(`${BASE}/machines`, () => {
    return HttpResponse.json({ machines: seedMachines, total: seedMachines.length })
  }),

  http.post(`${BASE}/machines`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const now = new Date().toISOString()
    const machine = {
      id: nextId('machine'),
      tenant_id: 'tenant-demo',
      name: String(body['name'] ?? ''),
      type: String(body['type'] ?? ''),
      status: 'available',
      notes: String(body['notes'] ?? ''),
      created_by: null,
      created_at: now,
      updated_at: now,
    }
    seedMachines.push(machine)
    return HttpResponse.json({ machine }, { status: 201 })
  }),

  http.get(`${BASE}/machines/:machineId`, ({ params }) => {
    const machine = seedMachines.find((m) => m.id === params['machineId'])
    if (!machine) return new HttpResponse(null, { status: 404 })
    return HttpResponse.json({ machine })
  }),

  http.patch(`${BASE}/machines/:machineId`, async ({ params, request }) => {
    const machine = seedMachines.find((m) => m.id === params['machineId'])
    if (!machine) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as Record<string, unknown>
    Object.assign(machine, body, { updated_at: new Date().toISOString() })
    return HttpResponse.json({ machine })
  }),

  http.delete(`${BASE}/machines/:machineId`, ({ params }) => {
    const idx = seedMachines.findIndex((m) => m.id === params['machineId'])
    if (idx >= 0) seedMachines.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // ---- Quality check handlers (client expects the { check } envelope) ----

  http.get(`${BASE}/quality`, () => {
    return HttpResponse.json({ checks: seedQualityChecks, total: seedQualityChecks.length })
  }),

  http.post(`${BASE}/quality`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const now = new Date().toISOString()
    const check = {
      id: nextId('qc'),
      tenant_id: 'tenant-demo',
      order_id: String(body['order_id'] ?? ''),
      inspector: String(body['inspector'] ?? ''),
      checked_at: String(body['checked_at'] ?? now),
      passed: body['passed'] !== false,
      defects_found: Number(body['defects_found'] ?? 0),
      notes: String(body['notes'] ?? ''),
      created_by: null,
      created_at: now,
      updated_at: now,
    }
    seedQualityChecks.unshift(check)
    return HttpResponse.json({ check }, { status: 201 })
  }),

  http.get(`${BASE}/quality/:checkId`, ({ params }) => {
    const check = seedQualityChecks.find((c) => c.id === params['checkId'])
    if (!check) return new HttpResponse(null, { status: 404 })
    return HttpResponse.json({ check })
  }),
]
