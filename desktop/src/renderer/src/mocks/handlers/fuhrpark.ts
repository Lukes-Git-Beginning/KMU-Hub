import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'

const API = API_BASE_URL

// ============================================================================
// Mock Data
// ============================================================================

const MOCK_VEHICLES_API = [
  { id: 'v-1', tenant_id: 't-1', license_plate: 'M-AB 1234', make: 'Volkswagen', model: 'T6 Transporter', year: 2020, fuel_type: 'diesel', status: 'active', assigned_driver_id: null, mileage_km: 125000, tuev_due_date: '2024-06-15', created_at: '2023-01-01T00:00:00Z', updated_at: '2024-01-15T00:00:00Z' },
  { id: 'v-2', tenant_id: 't-1', license_plate: 'M-CD 5678', make: 'Mercedes-Benz', model: 'Sprinter', year: 2019, fuel_type: 'diesel', status: 'active', assigned_driver_id: null, mileage_km: 87500, tuev_due_date: '2024-03-10', created_at: '2023-01-01T00:00:00Z', updated_at: '2024-01-14T00:00:00Z' },
  { id: 'v-3', tenant_id: 't-1', license_plate: 'M-EF 9012', make: 'MAN', model: 'TGE', year: 2021, fuel_type: 'diesel', status: 'active', assigned_driver_id: null, mileage_km: 201000, tuev_due_date: '2025-09-20', created_at: '2023-01-01T00:00:00Z', updated_at: '2024-01-13T00:00:00Z' },
  { id: 'v-4', tenant_id: 't-1', license_plate: 'M-GH 3456', make: 'Tesla', model: 'Model 3', year: 2022, fuel_type: 'electric', status: 'active', assigned_driver_id: null, mileage_km: 45000, tuev_due_date: '2026-11-05', created_at: '2023-01-01T00:00:00Z', updated_at: '2024-01-12T00:00:00Z' },
  { id: 'v-5', tenant_id: 't-1', license_plate: 'M-IJ 7890', make: 'Peugeot', model: 'Expert', year: 2018, fuel_type: 'diesel', status: 'in_service', assigned_driver_id: null, mileage_km: 156000, tuev_due_date: '2024-02-28', created_at: '2023-01-01T00:00:00Z', updated_at: '2024-01-10T00:00:00Z' },
  { id: 'v-6', tenant_id: 't-1', license_plate: 'M-KL 2345', make: 'Ford', model: 'Transit', year: 2020, fuel_type: 'diesel', status: 'active', assigned_driver_id: null, mileage_km: 98000, tuev_due_date: '2025-04-12', created_at: '2023-01-01T00:00:00Z', updated_at: '2024-01-08T00:00:00Z' },
]

// Wire-Shape wie fuhrpark-types.ts VehicleService: scheduled_at + service_type-Enums
// (der alte Seed nutzte scheduled_date + deutschen Freitext → Datum "—", alles als "Service" gemappt).
const MOCK_SERVICES_API = [
  { id: 's-1', tenant_id: 't-1', vehicle_id: 'v-1', service_type: 'oil_change', scheduled_at: '2024-02-15', status: 'scheduled', mileage_km: 130000, cost_cents: 18000, notes: 'Ölwechsel mit Filter', created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z' },
  { id: 's-2', tenant_id: 't-1', vehicle_id: 'v-2', service_type: 'inspection', scheduled_at: '2024-03-10', status: 'scheduled', mileage_km: 90000, cost_cents: 12000, notes: 'TÜV-Hauptuntersuchung', created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z' },
  { id: 's-3', tenant_id: 't-1', vehicle_id: 'v-1', service_type: 'inspection', scheduled_at: '2023-12-01', status: 'completed', mileage_km: 120000, cost_cents: 45000, notes: 'Alles OK', created_at: '2023-11-01T00:00:00Z', updated_at: '2023-12-01T00:00:00Z' },
  { id: 's-4', tenant_id: 't-1', vehicle_id: 'v-5', service_type: 'inspection', scheduled_at: '2024-02-28', status: 'scheduled', mileage_km: 158000, cost_cents: 12000, notes: 'TÜV-Hauptuntersuchung', created_at: '2024-01-10T00:00:00Z', updated_at: '2024-01-10T00:00:00Z' },
  { id: 's-5', tenant_id: 't-1', vehicle_id: 'v-3', service_type: 'repair', scheduled_at: '2024-01-20', status: 'completed', mileage_km: 199500, cost_cents: 62000, notes: 'Bremsbeläge vorne erneuert', created_at: '2024-01-18T00:00:00Z', updated_at: '2024-01-20T00:00:00Z' },
  { id: 's-6', tenant_id: 't-1', vehicle_id: 'v-2', service_type: 'tire_change', scheduled_at: '2024-04-05', status: 'scheduled', mileage_km: 91000, cost_cents: 9500, notes: 'Sommerreifen aufziehen', created_at: '2024-01-12T00:00:00Z', updated_at: '2024-01-12T00:00:00Z' },
]

const MOCK_DAMAGES_API = [
  { id: 'd-1', tenant_id: 't-1', vehicle_id: 'v-5', license_plate: 'M-IJ 7890', description: 'Delle in der Seitentür', severity: 'minor', status: 'reported', reported_by: null, photo_keys: [], cost_cents: null, notes: '', created_at: '2024-01-08T00:00:00Z', updated_at: '2024-01-08T00:00:00Z' },
  { id: 'd-2', tenant_id: 't-1', vehicle_id: 'v-3', license_plate: 'M-EF 9012', description: 'Windschutzscheibe gesprungen', severity: 'moderate', status: 'in_repair', reported_by: null, photo_keys: ['t-1/fuhrpark/dmg-1.jpg'], cost_cents: 80000, notes: 'Werkstatt beauftragt', created_at: '2024-01-05T00:00:00Z', updated_at: '2024-01-06T00:00:00Z' },
  { id: 'd-3', tenant_id: 't-1', vehicle_id: 'v-2', license_plate: 'M-CD 5678', description: 'Kratzer am Heck', severity: 'minor', status: 'resolved', reported_by: null, photo_keys: [], cost_cents: 15000, notes: '', created_at: '2023-12-01T00:00:00Z', updated_at: '2023-12-10T00:00:00Z' },
]

const MOCK_FUEL_LOGS = [
  { id: 'fl-1', tenant_id: 't-1', vehicle_id: 'v-1', date: '2024-01-15', liters: 45.2, cost_cents: 8090, mileage_km: 125000, fuel_type: 'diesel', notes: '', created_at: '2024-01-15T10:00:00Z', updated_at: '2024-01-15T10:00:00Z' },
  { id: 'fl-2', tenant_id: 't-1', vehicle_id: 'v-2', date: '2024-01-14', liters: 38.5, cost_cents: 6893, mileage_km: 87500, fuel_type: 'diesel', notes: '', created_at: '2024-01-14T14:00:00Z', updated_at: '2024-01-14T14:00:00Z' },
  { id: 'fl-3', tenant_id: 't-1', vehicle_id: 'v-1', date: '2024-01-10', liters: 42.0, cost_cents: 7518, mileage_km: 124200, fuel_type: 'diesel', notes: 'Autobahn', created_at: '2024-01-10T08:00:00Z', updated_at: '2024-01-10T08:00:00Z' },
  { id: 'fl-4', tenant_id: 't-1', vehicle_id: 'v-3', date: '2024-01-13', liters: 55.0, cost_cents: 9845, mileage_km: 201000, fuel_type: 'diesel', notes: '', created_at: '2024-01-13T09:00:00Z', updated_at: '2024-01-13T09:00:00Z' },
  { id: 'fl-5', tenant_id: 't-1', vehicle_id: 'v-4', date: '2024-01-12', liters: 0.0, cost_cents: 1200, mileage_km: 45000, fuel_type: 'electric', notes: 'Ladesäule Büro', created_at: '2024-01-12T07:00:00Z', updated_at: '2024-01-12T07:00:00Z' },
]

const MOCK_TRIP_LOGS = [
  { id: 'tl-1', tenant_id: 't-1', vehicle_id: 'v-1', date: '2024-01-15', start_location: 'München Zentrale', end_location: 'Augsburg Kunde', purpose: 'Kundengespräch', start_km: 124500, end_km: 124580, km: 80, is_private: false, driver_name: 'Max Mustermann', notes: '', created_at: '2024-01-15T09:00:00Z', updated_at: '2024-01-15T09:00:00Z' },
  { id: 'tl-2', tenant_id: 't-1', vehicle_id: 'v-2', date: '2024-01-14', start_location: 'Büro', end_location: 'Messe München', purpose: 'Messebesuch', start_km: 87200, end_km: 87240, km: 40, is_private: false, driver_name: 'Anna Schmidt', notes: '', created_at: '2024-01-14T08:00:00Z', updated_at: '2024-01-14T08:00:00Z' },
  { id: 'tl-3', tenant_id: 't-1', vehicle_id: 'v-1', date: '2024-01-13', start_location: 'Privatadresse', end_location: 'Einkauf', purpose: 'Privat', start_km: 124200, end_km: 124215, km: 15, is_private: true, driver_name: 'Max Mustermann', notes: '', created_at: '2024-01-13T17:00:00Z', updated_at: '2024-01-13T17:00:00Z' },
  { id: 'tl-4', tenant_id: 't-1', vehicle_id: 'v-3', date: '2024-01-12', start_location: 'Lager', end_location: 'Baustelle Nord', purpose: 'Materiallieferung', start_km: 200800, end_km: 200950, km: 150, is_private: false, driver_name: 'Klaus Weber', notes: 'Überstunden', created_at: '2024-01-12T06:00:00Z', updated_at: '2024-01-12T06:00:00Z' },
]

const MOCK_VEHICLE_DOCUMENTS = [
  { id: 'doc-1', tenant_id: 't-1', vehicle_id: 'v-1', doc_type: 'registration', name: 'Fahrzeugschein VW Transporter', object_key: 't-1/fuhrpark/doc-1.pdf', upload_date: '2023-01-10', expiry_date: null, created_at: '2023-01-10T10:00:00Z', updated_at: '2023-01-10T10:00:00Z' },
  { id: 'doc-2', tenant_id: 't-1', vehicle_id: 'v-1', doc_type: 'insurance', name: 'KFZ-Versicherung 2024', object_key: 't-1/fuhrpark/doc-2.pdf', upload_date: '2024-01-01', expiry_date: '2024-12-31', created_at: '2024-01-01T10:00:00Z', updated_at: '2024-01-01T10:00:00Z' },
  { id: 'doc-3', tenant_id: 't-1', vehicle_id: 'v-1', doc_type: 'tuev', name: 'TÜV-Bericht Jan 2024', object_key: 't-1/fuhrpark/doc-3.pdf', upload_date: '2024-01-20', expiry_date: '2026-01-20', created_at: '2024-01-20T10:00:00Z', updated_at: '2024-01-20T10:00:00Z' },
  { id: 'doc-4', tenant_id: 't-1', vehicle_id: 'v-2', doc_type: 'registration', name: 'Fahrzeugschein Sprinter', object_key: 't-1/fuhrpark/doc-4.pdf', upload_date: '2022-06-15', expiry_date: null, created_at: '2022-06-15T10:00:00Z', updated_at: '2022-06-15T10:00:00Z' },
]

const MOCK_GPS_ROUTES = [
  {
    vehicle_id: 'v-1',
    vehicle_name: 'VW T6 Transporter (M-AB 1234)',
    date: '2024-01-15',
    positions: [
      { id: 'gps-1', vehicle_id: 'v-1', lat: 48.1374, lng: 11.5755, speed_kmh: 0, recorded_at: '2024-01-15T08:00:00Z' },
      { id: 'gps-2', vehicle_id: 'v-1', lat: 48.2, lng: 11.6, speed_kmh: 80, recorded_at: '2024-01-15T08:30:00Z' },
      { id: 'gps-3', vehicle_id: 'v-1', lat: 48.3651, lng: 10.8986, speed_kmh: 0, recorded_at: '2024-01-15T09:15:00Z' },
    ],
    daily_km: 82,
    status: 'parked',
    driver: 'Max Mustermann',
  },
  {
    vehicle_id: 'v-2',
    vehicle_name: 'Mercedes Sprinter (M-CD 5678)',
    date: '2024-01-15',
    positions: [
      { id: 'gps-4', vehicle_id: 'v-2', lat: 48.1374, lng: 11.5755, speed_kmh: 0, recorded_at: '2024-01-15T07:00:00Z' },
      { id: 'gps-5', vehicle_id: 'v-2', lat: 48.15, lng: 11.5, speed_kmh: 60, recorded_at: '2024-01-15T14:30:00Z' },
    ],
    daily_km: 45,
    status: 'driving',
    driver: 'Anna Schmidt',
  },
]

// ============================================================================
// Mutable state
// ============================================================================

let vehicles = [...MOCK_VEHICLES_API]
let services = [...MOCK_SERVICES_API]
let damages = [...MOCK_DAMAGES_API]
let fuelLogs = [...MOCK_FUEL_LOGS]
let tripLogs = [...MOCK_TRIP_LOGS]
let vehicleDocuments = [...MOCK_VEHICLE_DOCUMENTS]
const gpsRoutes = [...MOCK_GPS_ROUTES]

// ============================================================================
// Helpers
// ============================================================================

function newId(): string {
  return Math.random().toString(36).slice(2, 10)
}

// ============================================================================
// Handlers
// ============================================================================

export const fuhrparkHandlers = [
  // --------------------------------------------------------------------------
  // Vehicles
  // --------------------------------------------------------------------------

  http.get(`${API}/api/v1/fuhrpark/vehicles`, () => {
    return HttpResponse.json({ vehicles, total: vehicles.length })
  }),

  http.get(`${API}/api/v1/fuhrpark/vehicles/:id`, ({ params }) => {
    const vehicle = vehicles.find(v => v.id === params.id)
    if (!vehicle) return HttpResponse.json({ error: 'vehicle not found' }, { status: 404 })
    return HttpResponse.json({ vehicle })
  }),

  http.post(`${API}/api/v1/fuhrpark/vehicles`, async ({ request }) => {
    const body = await request.json() as Record<string, unknown>
    const vehicle = {
      id: `v-${newId()}`,
      tenant_id: 't-1',
      ...body,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    vehicles = [...vehicles, vehicle as typeof MOCK_VEHICLES_API[0]]
    return HttpResponse.json({ vehicle }, { status: 201 })
  }),

  http.patch(`${API}/api/v1/fuhrpark/vehicles/:id`, async ({ params, request }) => {
    const body = await request.json() as Record<string, unknown>
    const idx = vehicles.findIndex(v => v.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'vehicle not found' }, { status: 404 })
    vehicles[idx] = { ...vehicles[idx], ...body, updated_at: new Date().toISOString() }
    return HttpResponse.json({ vehicle: vehicles[idx] })
  }),

  http.delete(`${API}/api/v1/fuhrpark/vehicles/:id`, ({ params }) => {
    vehicles = vehicles.filter(v => v.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  // --------------------------------------------------------------------------
  // Vehicle Services (Wartung)
  // --------------------------------------------------------------------------

  http.get(`${API}/api/v1/fuhrpark/vehicles/:id/services`, ({ params }) => {
    const result = services.filter(s => s.vehicle_id === params.id)
    return HttpResponse.json({ services: result, total: result.length })
  }),

  http.post(`${API}/api/v1/fuhrpark/vehicles/:id/services`, async ({ params, request }) => {
    const body = await request.json() as Record<string, unknown>
    const service = {
      id: `s-${newId()}`,
      tenant_id: 't-1',
      vehicle_id: String(params.id),
      ...body,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    services = [...services, service as typeof MOCK_SERVICES_API[0]]
    return HttpResponse.json({ service }, { status: 201 })
  }),

  // Top-level list — was missing entirely, which left the Wartung tab
  // permanently empty in demo mode (useServicesList hits GET /services).
  http.get(`${API}/api/v1/fuhrpark/services`, ({ request }) => {
    const url = new URL(request.url)
    const vehicleId = url.searchParams.get('vehicle_id')
    const result = vehicleId ? services.filter(s => s.vehicle_id === vehicleId) : services
    return HttpResponse.json({ services: result, total: result.length })
  }),

  http.get(`${API}/api/v1/fuhrpark/services/upcoming`, () => {
    const upcoming = services.filter(s => s.status === 'scheduled')
    return HttpResponse.json({ services: upcoming, total: upcoming.length })
  }),

  http.patch(`${API}/api/v1/fuhrpark/services/:id`, async ({ params, request }) => {
    const body = await request.json() as Record<string, unknown>
    const idx = services.findIndex(s => s.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'service not found' }, { status: 404 })
    services[idx] = { ...services[idx], ...body, updated_at: new Date().toISOString() }
    return HttpResponse.json({ service: services[idx] })
  }),

  http.delete(`${API}/api/v1/fuhrpark/services/:id`, ({ params }) => {
    services = services.filter(s => s.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  http.post(`${API}/api/v1/fuhrpark/services/:id/complete`, ({ params }) => {
    const idx = services.findIndex(s => s.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'service not found' }, { status: 404 })
    services[idx] = { ...services[idx], status: 'completed', updated_at: new Date().toISOString() }
    return HttpResponse.json({ service: services[idx] })
  }),

  // --------------------------------------------------------------------------
  // Damages (Schadensmeldungen)
  // --------------------------------------------------------------------------

  http.get(`${API}/api/v1/fuhrpark/vehicles/:id/damages`, ({ params }) => {
    const result = damages.filter(d => d.vehicle_id === params.id)
    return HttpResponse.json({ damages: result, total: result.length })
  }),

  http.post(`${API}/api/v1/fuhrpark/vehicles/:id/damages`, async ({ params, request }) => {
    const body = await request.json() as Record<string, unknown>
    const damage = {
      id: `d-${newId()}`,
      tenant_id: 't-1',
      vehicle_id: String(params.id),
      photo_keys: [],
      ...body,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    damages = [...damages, damage as typeof MOCK_DAMAGES_API[0]]
    return HttpResponse.json({ damage }, { status: 201 })
  }),

  http.get(`${API}/api/v1/fuhrpark/damages`, ({ request }) => {
    const url = new URL(request.url)
    const vehicleId = url.searchParams.get('vehicle_id')
    const result = vehicleId ? damages.filter(d => d.vehicle_id === vehicleId) : damages
    return HttpResponse.json({ damages: result, total: result.length })
  }),

  http.patch(`${API}/api/v1/fuhrpark/damages/:id`, async ({ params, request }) => {
    const body = await request.json() as Record<string, unknown>
    const idx = damages.findIndex(d => d.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'damage not found' }, { status: 404 })
    damages[idx] = { ...damages[idx], ...body, updated_at: new Date().toISOString() }
    return HttpResponse.json({ damage: damages[idx] })
  }),

  http.post(`${API}/api/v1/fuhrpark/damages/:id/resolve`, ({ params }) => {
    const idx = damages.findIndex(d => d.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'damage not found' }, { status: 404 })
    damages[idx] = { ...damages[idx], status: 'resolved', updated_at: new Date().toISOString() }
    return HttpResponse.json({ damage: damages[idx] })
  }),

  // --------------------------------------------------------------------------
  // TÜV due
  // --------------------------------------------------------------------------

  http.get(`${API}/api/v1/fuhrpark/tuev-due`, () => {
    const today = new Date().toISOString().split('T')[0]
    const due = vehicles.filter(v => v.tuev_due_date && v.tuev_due_date <= today)
    return HttpResponse.json({ vehicles: due, total: due.length })
  }),

  // --------------------------------------------------------------------------
  // Fuel Logs (Tankprotokoll)
  // --------------------------------------------------------------------------

  http.get(`${API}/api/v1/fuhrpark/fuel-logs`, ({ request }) => {
    const url = new URL(request.url)
    const vehicleId = url.searchParams.get('vehicle_id')
    const result = vehicleId ? fuelLogs.filter(f => f.vehicle_id === vehicleId) : fuelLogs
    return HttpResponse.json({ fuel_logs: result, total: result.length })
  }),

  http.get(`${API}/api/v1/fuhrpark/vehicles/:id/fuel-logs`, ({ params }) => {
    const result = fuelLogs.filter(f => f.vehicle_id === params.id)
    return HttpResponse.json({ fuel_logs: result, total: result.length })
  }),

  http.post(`${API}/api/v1/fuhrpark/vehicles/:id/fuel-logs`, async ({ params, request }) => {
    const body = await request.json() as Record<string, unknown>
    const fuelLog = {
      id: `fl-${newId()}`,
      tenant_id: 't-1',
      vehicle_id: String(params.id),
      ...body,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    fuelLogs = [...fuelLogs, fuelLog as typeof MOCK_FUEL_LOGS[0]]
    return HttpResponse.json({ fuel_log: fuelLog }, { status: 201 })
  }),

  http.patch(`${API}/api/v1/fuhrpark/fuel-logs/:id`, async ({ params, request }) => {
    const body = await request.json() as Record<string, unknown>
    const idx = fuelLogs.findIndex(f => f.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'fuel log not found' }, { status: 404 })
    fuelLogs[idx] = { ...fuelLogs[idx], ...body, updated_at: new Date().toISOString() }
    return HttpResponse.json({ fuel_log: fuelLogs[idx] })
  }),

  http.delete(`${API}/api/v1/fuhrpark/fuel-logs/:id`, ({ params }) => {
    fuelLogs = fuelLogs.filter(f => f.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  // --------------------------------------------------------------------------
  // Trip Logs (Fahrtenbuch)
  // --------------------------------------------------------------------------

  http.get(`${API}/api/v1/fuhrpark/trip-logs`, ({ request }) => {
    const url = new URL(request.url)
    const vehicleId = url.searchParams.get('vehicle_id')
    const result = vehicleId ? tripLogs.filter(t => t.vehicle_id === vehicleId) : tripLogs
    return HttpResponse.json({ trip_logs: result, total: result.length })
  }),

  http.get(`${API}/api/v1/fuhrpark/vehicles/:id/trip-logs`, ({ params }) => {
    const result = tripLogs.filter(t => t.vehicle_id === params.id)
    return HttpResponse.json({ trip_logs: result, total: result.length })
  }),

  http.post(`${API}/api/v1/fuhrpark/vehicles/:id/trip-logs`, async ({ params, request }) => {
    const body = await request.json() as Record<string, unknown>
    const tripLog = {
      id: `tl-${newId()}`,
      tenant_id: 't-1',
      vehicle_id: String(params.id),
      ...body,
      // the real backend computes km from the odometer readings — mirror that
      // here, otherwise the fresh row renders "NaN km"
      km: Number(body.end_km ?? 0) - Number(body.start_km ?? 0),
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    tripLogs = [...tripLogs, tripLog as typeof MOCK_TRIP_LOGS[0]]
    return HttpResponse.json({ trip_log: tripLog }, { status: 201 })
  }),

  http.patch(`${API}/api/v1/fuhrpark/trip-logs/:id`, async ({ params, request }) => {
    const body = await request.json() as Record<string, unknown>
    const idx = tripLogs.findIndex(t => t.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'trip log not found' }, { status: 404 })
    tripLogs[idx] = { ...tripLogs[idx], ...body, updated_at: new Date().toISOString() }
    return HttpResponse.json({ trip_log: tripLogs[idx] })
  }),

  http.delete(`${API}/api/v1/fuhrpark/trip-logs/:id`, ({ params }) => {
    tripLogs = tripLogs.filter(t => t.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  // --------------------------------------------------------------------------
  // Vehicle Documents
  // --------------------------------------------------------------------------

  http.get(`${API}/api/v1/fuhrpark/vehicles/:id/documents`, ({ params }) => {
    const result = vehicleDocuments.filter(d => d.vehicle_id === params.id)
    return HttpResponse.json({ documents: result, total: result.length })
  }),

  http.post(`${API}/api/v1/fuhrpark/vehicles/:id/documents`, async ({ params, request }) => {
    const body = await request.json() as Record<string, unknown>
    const doc = {
      id: `doc-${newId()}`,
      tenant_id: 't-1',
      vehicle_id: String(params.id),
      ...body,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    vehicleDocuments = [...vehicleDocuments, doc as typeof MOCK_VEHICLE_DOCUMENTS[0]]
    return HttpResponse.json({ document: doc }, { status: 201 })
  }),

  http.delete(`${API}/api/v1/fuhrpark/documents/:id`, ({ params }) => {
    vehicleDocuments = vehicleDocuments.filter(d => d.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  // --------------------------------------------------------------------------
  // GPS
  // --------------------------------------------------------------------------

  http.post(`${API}/api/v1/fuhrpark/gps/ingest`, async ({ request }) => {
    const body = await request.json() as { positions?: unknown[] }
    const positions = body.positions ?? []
    return HttpResponse.json({ ingested: Array.isArray(positions) ? positions.length : 0 })
  }),

  http.get(`${API}/api/v1/fuhrpark/gps/routes`, ({ request }) => {
    const url = new URL(request.url)
    const vehicleId = url.searchParams.get('vehicle_id')
    const result = vehicleId ? gpsRoutes.filter(r => r.vehicle_id === vehicleId) : gpsRoutes
    return HttpResponse.json({ routes: result })
  }),

  http.get(`${API}/api/v1/fuhrpark/gps/positions`, ({ request }) => {
    const url = new URL(request.url)
    const vehicleId = url.searchParams.get('vehicle_id')
    const allPositions = gpsRoutes.flatMap(r => r.positions)
    const result = vehicleId ? allPositions.filter(p => p.vehicle_id === vehicleId) : allPositions
    return HttpResponse.json({ positions: result })
  }),
]
