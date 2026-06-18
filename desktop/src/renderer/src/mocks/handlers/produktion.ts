import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'

const BASE = `${API_BASE_URL}/api/v1/produktion`

// ---- Seed data ----

const seedBoms = [
  {
    id: 'bom-001',
    tenant_id: 'tenant-demo',
    product_name: 'Schaltschrank Typ A',
    sku: 'SSA-2026',
    version: '2.1',
    active: true,
    created_at: '2026-01-10T08:00:00Z',
    updated_at: '2026-01-10T08:00:00Z',
    items: [
      { id: 'bi-001', bom_id: 'bom-001', material_name: 'Gehaeuse 600x800', quantity: 1, unit: 'Stk', sort_order: 0 },
      { id: 'bi-002', bom_id: 'bom-001', material_name: 'Kabelkanal 40x60', quantity: 4, unit: 'm', sort_order: 1 },
      { id: 'bi-003', bom_id: 'bom-001', material_name: 'Klemmenblock 10pol', quantity: 3, unit: 'Set', sort_order: 2 },
    ],
  },
  {
    id: 'bom-002',
    tenant_id: 'tenant-demo',
    product_name: 'Steuereinheit Kompakt',
    sku: 'SEK-2026',
    version: '1.0',
    active: true,
    created_at: '2026-01-15T09:00:00Z',
    updated_at: '2026-01-15T09:00:00Z',
    items: [
      { id: 'bi-004', bom_id: 'bom-002', material_name: 'Platine STM32F4', quantity: 1, unit: 'Stk', sort_order: 0 },
      { id: 'bi-005', bom_id: 'bom-002', material_name: 'Kuehlkoerper passiv', quantity: 1, unit: 'Stk', sort_order: 1 },
      { id: 'bi-006', bom_id: 'bom-002', material_name: 'Kondensator 100uF', quantity: 8, unit: 'Stk', sort_order: 2 },
      { id: 'bi-007', bom_id: 'bom-002', material_name: 'RJ45 Buchse', quantity: 2, unit: 'Stk', sort_order: 3 },
    ],
  },
]

const seedMachines = [
  {
    id: 'machine-001',
    tenant_id: 'tenant-demo',
    name: 'CNC-Fraese Alpha',
    type: 'CNC',
    status: 'in_use',
    location: 'Halle A',
    notes: null,
    created_at: '2026-01-05T07:00:00Z',
    updated_at: '2026-01-05T07:00:00Z',
  },
  {
    id: 'machine-002',
    tenant_id: 'tenant-demo',
    name: 'Schweissroboter R1',
    type: 'Roboter',
    status: 'available',
    location: 'Halle B',
    notes: null,
    created_at: '2026-01-05T07:00:00Z',
    updated_at: '2026-01-05T07:00:00Z',
  },
  {
    id: 'machine-003',
    tenant_id: 'tenant-demo',
    name: 'Drehbank DM-200',
    type: 'Drehbank',
    status: 'maintenance',
    location: 'Halle A',
    notes: 'Wartung bis 20.06.',
    created_at: '2026-01-05T07:00:00Z',
    updated_at: '2026-06-18T08:00:00Z',
  },
]

const seedWorkSteps = [
  {
    id: 'ws-001',
    tenant_id: 'tenant-demo',
    production_order_id: 'order-001',
    step_nr: 1,
    name: 'Material pruefen',
    status: 'completed',
    machine_id: null,
    duration_minutes: 30,
    started_at: '2026-06-01T07:00:00Z',
    completed_at: '2026-06-01T07:30:00Z',
    notes: null,
    created_at: '2026-06-01T06:00:00Z',
    updated_at: '2026-06-01T07:30:00Z',
  },
  {
    id: 'ws-002',
    tenant_id: 'tenant-demo',
    production_order_id: 'order-001',
    step_nr: 2,
    name: 'Fraesen',
    status: 'in_progress',
    machine_id: 'machine-001',
    duration_minutes: 120,
    started_at: '2026-06-02T08:00:00Z',
    completed_at: null,
    notes: null,
    created_at: '2026-06-01T06:00:00Z',
    updated_at: '2026-06-02T08:00:00Z',
  },
  {
    id: 'ws-003',
    tenant_id: 'tenant-demo',
    production_order_id: 'order-001',
    step_nr: 3,
    name: 'Endmontage',
    status: 'pending',
    machine_id: null,
    duration_minutes: 60,
    started_at: null,
    completed_at: null,
    notes: null,
    created_at: '2026-06-01T06:00:00Z',
    updated_at: '2026-06-01T06:00:00Z',
  },
]

const seedQualityChecks = [
  {
    id: 'qc-001',
    tenant_id: 'tenant-demo',
    order_id: 'order-001',
    inspector: 'Werner Stoeckli',
    checked_at: '2026-06-10T10:00:00Z',
    passed: true,
    defects_found: 0,
    notes: 'Alle Masse korrekt.',
    created_at: '2026-06-10T10:05:00Z',
    updated_at: '2026-06-10T10:05:00Z',
  },
  {
    id: 'qc-002',
    tenant_id: 'tenant-demo',
    order_id: 'order-002',
    inspector: 'Irene Graf',
    checked_at: '2026-06-12T14:00:00Z',
    passed: false,
    defects_found: 2,
    notes: 'Loetpunkte nacharbeiten.',
    created_at: '2026-06-12T14:10:00Z',
    updated_at: '2026-06-12T14:10:00Z',
  },
]

// ---- BOM handlers ----

export const produktionHandlers = [
  http.get(`${BASE}/boms`, () => {
    return HttpResponse.json({ boms: seedBoms, total: seedBoms.length })
  }),

  http.post(`${BASE}/boms`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const newBom = {
      id: `bom-${Date.now()}`,
      tenant_id: 'tenant-demo',
      product_name: String(body['product_name'] ?? ''),
      sku: String(body['sku'] ?? ''),
      version: String(body['version'] ?? '1.0'),
      active: body['active'] !== false,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      items: Array.isArray(body['items'])
        ? (body['items'] as Array<Record<string, unknown>>).map((it, i) => ({
            id: `bi-new-${Date.now()}-${i}`,
            bom_id: `bom-${Date.now()}`,
            material_name: String(it['material_name'] ?? ''),
            quantity: Number(it['quantity'] ?? 1),
            unit: String(it['unit'] ?? 'Stk'),
            sort_order: i,
          }))
        : [],
    }
    return HttpResponse.json(newBom, { status: 201 })
  }),

  http.get(`${BASE}/boms/:bomId`, ({ params }) => {
    const bom = seedBoms.find((b) => b.id === params['bomId'])
    if (!bom) return new HttpResponse(null, { status: 404 })
    return HttpResponse.json(bom)
  }),

  http.patch(`${BASE}/boms/:bomId`, async ({ params, request }) => {
    const bom = seedBoms.find((b) => b.id === params['bomId'])
    if (!bom) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({ ...bom, ...body, updated_at: new Date().toISOString() })
  }),

  http.delete(`${BASE}/boms/:bomId`, () => {
    return new HttpResponse(null, { status: 204 })
  }),

  // ---- Work step handlers ----

  http.get(`${BASE}/orders/:orderId/steps`, ({ params }) => {
    const steps = seedWorkSteps.filter((s) => s.production_order_id === params['orderId'])
    return HttpResponse.json({ steps, total: steps.length })
  }),

  http.post(`${BASE}/orders/:orderId/steps`, async ({ params, request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const newStep = {
      id: `ws-new-${Date.now()}`,
      tenant_id: 'tenant-demo',
      production_order_id: String(params['orderId']),
      step_nr: Number(body['step_nr'] ?? 1),
      name: String(body['name'] ?? ''),
      status: 'pending',
      machine_id: body['machine_id'] ? String(body['machine_id']) : null,
      duration_minutes: Number(body['duration_minutes'] ?? 0),
      started_at: null,
      completed_at: null,
      notes: body['notes'] ? String(body['notes']) : null,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    return HttpResponse.json(newStep, { status: 201 })
  }),

  http.patch(`${BASE}/orders/:orderId/steps/:stepId`, async ({ params, request }) => {
    const step = seedWorkSteps.find((s) => s.id === params['stepId'])
    if (!step) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({ ...step, ...body, updated_at: new Date().toISOString() })
  }),

  http.delete(`${BASE}/orders/:orderId/steps/:stepId`, () => {
    return new HttpResponse(null, { status: 204 })
  }),

  // ---- Machine handlers ----

  http.get(`${BASE}/machines`, () => {
    return HttpResponse.json({ machines: seedMachines, total: seedMachines.length })
  }),

  http.post(`${BASE}/machines`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const newMachine = {
      id: `machine-new-${Date.now()}`,
      tenant_id: 'tenant-demo',
      name: String(body['name'] ?? ''),
      type: String(body['type'] ?? ''),
      status: String(body['status'] ?? 'available'),
      location: body['location'] ? String(body['location']) : null,
      notes: body['notes'] ? String(body['notes']) : null,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    return HttpResponse.json(newMachine, { status: 201 })
  }),

  http.get(`${BASE}/machines/:machineId`, ({ params }) => {
    const machine = seedMachines.find((m) => m.id === params['machineId'])
    if (!machine) return new HttpResponse(null, { status: 404 })
    return HttpResponse.json(machine)
  }),

  http.patch(`${BASE}/machines/:machineId`, async ({ params, request }) => {
    const machine = seedMachines.find((m) => m.id === params['machineId'])
    if (!machine) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({ ...machine, ...body, updated_at: new Date().toISOString() })
  }),

  http.delete(`${BASE}/machines/:machineId`, () => {
    return new HttpResponse(null, { status: 204 })
  }),

  // ---- Quality check handlers ----

  http.get(`${BASE}/quality`, () => {
    return HttpResponse.json({ checks: seedQualityChecks, total: seedQualityChecks.length })
  }),

  http.post(`${BASE}/quality`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const newCheck = {
      id: `qc-new-${Date.now()}`,
      tenant_id: 'tenant-demo',
      order_id: String(body['order_id'] ?? ''),
      inspector: String(body['inspector'] ?? ''),
      checked_at: String(body['checked_at'] ?? new Date().toISOString()),
      passed: body['passed'] !== false,
      defects_found: Number(body['defects_found'] ?? 0),
      notes: body['notes'] ? String(body['notes']) : null,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    return HttpResponse.json(newCheck, { status: 201 })
  }),

  http.get(`${BASE}/quality/:checkId`, ({ params }) => {
    const check = seedQualityChecks.find((c) => c.id === params['checkId'])
    if (!check) return new HttpResponse(null, { status: 404 })
    return HttpResponse.json(check)
  }),
]