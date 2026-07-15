/**
 * Lightweight fetch wrapper for Fuhrpark API endpoints.
 *
 * Follows the same pattern as inventar-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry.
 */
import type {
  Vehicle,
  VehicleService,
  VehicleDamage,
  CreateVehicleInput,
  UpdateVehicleInput,
  ListVehiclesParams,
  ScheduleServiceInput,
  UpdateServiceInput,
  CompleteServiceInput,
  ListServicesParams,
  ReportDamageInput,
  UpdateDamageInput,
  ResolveDamageInput,
  ListDamagesParams,
  ListVehicleHistoryParams,
  CheckTuevDueParams,
  ListUpcomingServicesParams,
  ListVehiclesResponse,
  ListServicesResponse,
  ListDamagesResponse,
  VehicleHistoryResponse,
  CheckTuevDueResponse,
} from './fuhrpark-types'
import { authenticatedRequest } from './utils/authenticatedFetch'

// Re-export the wire types — useFuhrpark.ts and FuhrparkPage import them from
// this module (they were only imported here before, which tsc rejects).
export type {
  Vehicle,
  VehicleService,
  VehicleDamage,
  CreateVehicleInput,
  UpdateVehicleInput,
  ListVehiclesParams,
  ScheduleServiceInput,
  UpdateServiceInput,
  CompleteServiceInput,
  ListServicesParams,
  ReportDamageInput,
  UpdateDamageInput,
  ResolveDamageInput,
  ListDamagesParams,
  ListVehicleHistoryParams,
  CheckTuevDueParams,
  ListUpcomingServicesParams,
} from './fuhrpark-types'

// ---------------------------------------------------------------------------
// Request helper
// ---------------------------------------------------------------------------

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | undefined>
}

function request<T>(opts: RequestOptions): Promise<T> {
  return authenticatedRequest<T>(opts)
}

// ---------------------------------------------------------------------------
// Base path
// ---------------------------------------------------------------------------

const BASE = '/api/v1/fuhrpark'

// ---------------------------------------------------------------------------
// Vehicles
// ---------------------------------------------------------------------------

export function listVehicles(params?: ListVehiclesParams) {
  return request<ListVehiclesResponse>({
    method: 'GET',
    path: `${BASE}/vehicles`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getVehicle(id: string) {
  return request<{ vehicle: Vehicle }>({
    method: 'GET',
    path: `${BASE}/vehicles/${id}`,
  })
}

export function createVehicle(body: CreateVehicleInput) {
  return request<{ vehicle: Vehicle }>({
    method: 'POST',
    path: `${BASE}/vehicles`,
    body,
  })
}

export function updateVehicle(id: string, body: UpdateVehicleInput) {
  return request<{ vehicle: Vehicle }>({
    method: 'PATCH',
    path: `${BASE}/vehicles/${id}`,
    body,
  })
}

export function deleteVehicle(id: string) {
  return request<void>({
    method: 'DELETE',
    path: `${BASE}/vehicles/${id}`,
  })
}

export function getVehicleHistory(id: string, params?: ListVehicleHistoryParams) {
  return request<VehicleHistoryResponse>({
    method: 'GET',
    path: `${BASE}/vehicles/${id}/history`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

// ---------------------------------------------------------------------------
// Services (vehicle sub-resource)
// ---------------------------------------------------------------------------

export function listVehicleServices(vehicleId: string, params?: ListServicesParams) {
  return request<ListServicesResponse>({
    method: 'GET',
    path: `${BASE}/vehicles/${vehicleId}/services`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function scheduleService(vehicleId: string, body: ScheduleServiceInput) {
  return request<{ service: VehicleService }>({
    method: 'POST',
    path: `${BASE}/vehicles/${vehicleId}/services`,
    body,
  })
}

// ---------------------------------------------------------------------------
// Services (top-level)
// ---------------------------------------------------------------------------

export function listServices(params?: ListServicesParams) {
  return request<ListServicesResponse>({
    method: 'GET',
    path: `${BASE}/services`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function updateService(id: string, body: UpdateServiceInput) {
  return request<{ service: VehicleService }>({
    method: 'PATCH',
    path: `${BASE}/services/${id}`,
    body,
  })
}

export function deleteService(id: string) {
  return request<void>({
    method: 'DELETE',
    path: `${BASE}/services/${id}`,
  })
}

export function completeService(id: string, body?: CompleteServiceInput) {
  return request<{ service: VehicleService }>({
    method: 'POST',
    path: `${BASE}/services/${id}/complete`,
    body,
  })
}

export function listUpcomingServices(params?: ListUpcomingServicesParams) {
  return request<ListServicesResponse>({
    method: 'GET',
    path: `${BASE}/services/upcoming`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

// ---------------------------------------------------------------------------
// Damages (vehicle sub-resource)
// ---------------------------------------------------------------------------

export function listVehicleDamages(vehicleId: string, params?: ListDamagesParams) {
  return request<ListDamagesResponse>({
    method: 'GET',
    path: `${BASE}/vehicles/${vehicleId}/damages`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function reportDamage(vehicleId: string, body: ReportDamageInput) {
  return request<{ damage: VehicleDamage }>({
    method: 'POST',
    path: `${BASE}/vehicles/${vehicleId}/damages`,
    body,
  })
}

// ---------------------------------------------------------------------------
// Damages (top-level)
// ---------------------------------------------------------------------------

export function listDamages(params?: ListDamagesParams) {
  return request<ListDamagesResponse>({
    method: 'GET',
    path: `${BASE}/damages`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function updateDamage(id: string, body: UpdateDamageInput) {
  return request<{ damage: VehicleDamage }>({
    method: 'PATCH',
    path: `${BASE}/damages/${id}`,
    body,
  })
}

export function resolveDamage(id: string, body?: ResolveDamageInput) {
  return request<{ damage: VehicleDamage }>({
    method: 'POST',
    path: `${BASE}/damages/${id}/resolve`,
    body,
  })
}

// ---------------------------------------------------------------------------
// TÜV check & export
// ---------------------------------------------------------------------------

export function checkTuevDue(params?: CheckTuevDueParams) {
  return request<CheckTuevDueResponse>({
    method: 'GET',
    path: `${BASE}/tuev-due`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

// NOTE: the former getExportUrl() helper was dead code referencing a missing
// backend endpoint (and lacked API_BASE_URL) — exports are built client-side
// in modules/fuhrpark/fuhrpark-export.ts.

// ---------------------------------------------------------------------------
// Fuel Logs
// ---------------------------------------------------------------------------

export interface FuelLog {
  id: string
  tenant_id: string
  vehicle_id: string
  date: string // ISO "2024-01-15"
  liters: number
  // protojson serializes int64 as a JSON string (proto3 spec); coerce with Number(...) at the call site.
  cost_cents: number | string
  mileage_km: number | string
  fuel_type: 'diesel' | 'petrol' | 'electric' | 'hybrid' | 'other'
  notes: string
  created_at: string
  updated_at: string
}

export interface CreateFuelLogRequest {
  vehicle_id: string
  date: string
  liters: number
  cost_cents: number
  mileage_km: number
  fuel_type: string
  notes?: string
}

export interface UpdateFuelLogRequest {
  date?: string
  liters?: number
  cost_cents?: number
  mileage_km?: number
  fuel_type?: string
  notes?: string
}

export function listFuelLogs(params: { vehicle_id?: string; page?: number; page_size?: number } = {}) {
  return request<{ fuel_logs: FuelLog[]; total: number }>({
    method: 'GET',
    path: `${BASE}/fuel-logs`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function listVehicleFuelLogs(
  vehicleId: string,
  params: { page?: number; page_size?: number } = {},
) {
  return request<{ fuel_logs: FuelLog[]; total: number }>({
    method: 'GET',
    path: `${BASE}/vehicles/${vehicleId}/fuel-logs`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function createFuelLog(vehicleId: string, body: CreateFuelLogRequest) {
  return request<{ fuel_log: FuelLog }>({
    method: 'POST',
    path: `${BASE}/vehicles/${vehicleId}/fuel-logs`,
    body,
  })
}

export function updateFuelLog(id: string, body: UpdateFuelLogRequest) {
  return request<{ fuel_log: FuelLog }>({
    method: 'PATCH',
    path: `${BASE}/fuel-logs/${id}`,
    body,
  })
}

export function deleteFuelLog(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/fuel-logs/${id}` })
}

// ---------------------------------------------------------------------------
// Trip Logs
// ---------------------------------------------------------------------------

export interface TripLog {
  id: string
  tenant_id: string
  vehicle_id: string
  date: string
  start_location: string
  end_location: string
  purpose: string
  // protojson serializes int64 as a JSON string (proto3 spec); coerce with Number(...) at the call site.
  start_km: number | string
  end_km: number | string
  km: number | string // computed by backend
  is_private: boolean
  driver_name: string
  notes: string
  created_at: string
  updated_at: string
}

export interface CreateTripLogRequest {
  vehicle_id: string
  date: string
  start_location: string
  end_location: string
  purpose: string
  start_km: number
  end_km: number
  is_private: boolean
  driver_name: string
  notes?: string
}

export interface UpdateTripLogRequest {
  date?: string
  start_location?: string
  end_location?: string
  purpose?: string
  start_km?: number
  end_km?: number
  is_private?: boolean
  driver_name?: string
  notes?: string
}

export function listTripLogs(params: { vehicle_id?: string; page?: number; page_size?: number } = {}) {
  return request<{ trip_logs: TripLog[]; total: number }>({
    method: 'GET',
    path: `${BASE}/trip-logs`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function listVehicleTripLogs(
  vehicleId: string,
  params: { page?: number; page_size?: number } = {},
) {
  return request<{ trip_logs: TripLog[]; total: number }>({
    method: 'GET',
    path: `${BASE}/vehicles/${vehicleId}/trip-logs`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function createTripLog(vehicleId: string, body: CreateTripLogRequest) {
  return request<{ trip_log: TripLog }>({
    method: 'POST',
    path: `${BASE}/vehicles/${vehicleId}/trip-logs`,
    body,
  })
}

export function updateTripLog(id: string, body: UpdateTripLogRequest) {
  return request<{ trip_log: TripLog }>({
    method: 'PATCH',
    path: `${BASE}/trip-logs/${id}`,
    body,
  })
}

export function deleteTripLog(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/trip-logs/${id}` })
}

// ---------------------------------------------------------------------------
// Vehicle Documents
// ---------------------------------------------------------------------------

export interface VehicleDocument {
  id: string
  tenant_id: string
  vehicle_id: string
  doc_type: 'registration' | 'insurance' | 'tuev' | 'other'
  name: string
  object_key: string
  upload_date: string
  expiry_date?: string
  created_at: string
  updated_at: string
}

export interface CreateVehicleDocumentRequest {
  vehicle_id: string
  doc_type: string
  name: string
  object_key: string
  expiry_date?: string
}

export function listVehicleDocuments(
  vehicleId: string,
  params: { page?: number; page_size?: number } = {},
) {
  return request<{ documents: VehicleDocument[]; total: number }>({
    method: 'GET',
    path: `${BASE}/vehicles/${vehicleId}/documents`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function createVehicleDocument(vehicleId: string, body: CreateVehicleDocumentRequest) {
  return request<{ document: VehicleDocument }>({
    method: 'POST',
    path: `${BASE}/vehicles/${vehicleId}/documents`,
    body,
  })
}

export function deleteVehicleDocument(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/documents/${id}` })
}

// ---------------------------------------------------------------------------
// GPS / Tracking
// ---------------------------------------------------------------------------

export interface GpsPosition {
  id: string
  vehicle_id: string
  lat: number
  lng: number
  speed_kmh: number
  recorded_at: string
}

export interface VehicleRoute {
  vehicle_id: string
  vehicle_name: string
  date: string
  positions: GpsPosition[]
  daily_km: number
  status: 'driving' | 'parked' | 'unknown'
  driver: string
}

export interface IngestGpsPositionsRequest {
  vehicle_id: string
  positions: Array<{
    lat: number
    lng: number
    speed_kmh?: number
    recorded_at: string
  }>
}

export function ingestGpsPositions(body: IngestGpsPositionsRequest) {
  return request<{ ingested: number }>({ method: 'POST', path: `${BASE}/gps/ingest`, body })
}

export function getVehicleRoutes(
  params: { date_from?: string; date_to?: string; vehicle_id?: string } = {},
) {
  return request<{ routes: VehicleRoute[] }>({
    method: 'GET',
    path: `${BASE}/gps/routes`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getGpsPositions(params: {
  vehicle_id: string
  from?: string
  to?: string
  limit?: number
}) {
  return request<{ positions: GpsPosition[] }>({
    method: 'GET',
    path: `${BASE}/gps/positions`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}
