/**
 * TypeScript types for the Fuhrpark (fleet management) module.
 *
 * Mirrors backend/internal/fuhrpark/models.go and the proto contract.
 * UUIDs are represented as strings; dates as ISO 8601 strings.
 */

// ---------------------------------------------------------------------------
// Enums
// ---------------------------------------------------------------------------

export type FuelType = 'petrol' | 'diesel' | 'electric' | 'hybrid' | 'other'

export type VehicleStatus = 'active' | 'in_service' | 'inactive' | 'decommissioned'

export type ServiceStatus = 'scheduled' | 'in_progress' | 'completed' | 'cancelled'

export type DamageStatus = 'reported' | 'in_repair' | 'resolved'

export type DamageSeverity = 'minor' | 'moderate' | 'major' | 'totalled'

// ---------------------------------------------------------------------------
// Domain models
// ---------------------------------------------------------------------------

export interface Vehicle {
  id: string
  tenant_id: string
  license_plate: string
  make: string
  model: string
  year: number
  vin: string | null
  color: string | null
  fuel_type: FuelType
  status: VehicleStatus
  mileage_km: number
  tuev_due_date: string | null       // ISO date YYYY-MM-DD
  tuev_reminder_sent_at: string | null
  assigned_driver_id: string | null  // UUID stub Sprint 3
  notes: string | null
  created_at: string
  updated_at: string
  deleted_at: string | null
}

export interface VehicleService {
  id: string
  tenant_id: string
  vehicle_id: string
  service_type: string
  description: string | null
  scheduled_at: string               // ISO date YYYY-MM-DD
  completed_at: string | null
  cost_cents: number | null
  workshop: string | null
  mileage_km: number | null
  status: ServiceStatus
  notes: string | null
  created_by: string | null
  created_at: string
  updated_at: string
}

export interface VehicleDamage {
  id: string
  tenant_id: string
  vehicle_id: string
  description: string
  severity: DamageSeverity
  status: DamageStatus
  reported_by: string | null
  resolved_by: string | null
  resolved_at: string | null
  photo_keys: string[]
  cost_cents: number | null
  notes: string | null
  created_at: string
  updated_at: string
}

export interface VehicleHistoryEntry {
  kind: 'service' | 'damage'
  id: string
  summary: string
  occurred_at: string
}

// ---------------------------------------------------------------------------
// Request input types
// ---------------------------------------------------------------------------

export interface CreateVehicleInput {
  license_plate: string
  make: string
  model: string
  year: number
  vin?: string
  color?: string
  fuel_type: FuelType
  mileage_km: number
  tuev_due_date?: string
  assigned_driver_id?: string
  notes?: string
}

export interface UpdateVehicleInput {
  license_plate?: string
  make?: string
  model?: string
  year?: number
  vin?: string
  color?: string
  fuel_type?: FuelType
  status?: VehicleStatus
  mileage_km?: number
  tuev_due_date?: string
  assigned_driver_id?: string
  notes?: string
}

export interface ListVehiclesParams {
  search?: string
  status?: VehicleStatus
  fuel_type?: FuelType
  page?: number
  page_size?: number
}

export interface ScheduleServiceInput {
  service_type: string
  description?: string
  scheduled_at: string
  cost_cents?: number
  workshop?: string
  mileage_km?: number
  notes?: string
}

export interface UpdateServiceInput {
  service_type?: string
  description?: string
  scheduled_at?: string
  cost_cents?: number
  workshop?: string
  mileage_km?: number
  status?: ServiceStatus
  notes?: string
}

export interface CompleteServiceInput {
  cost_cents?: number
  mileage_km?: number
  notes?: string
}

export interface ListServicesParams {
  vehicle_id?: string
  status?: ServiceStatus
  page?: number
  page_size?: number
}

export interface ReportDamageInput {
  description: string
  severity: DamageSeverity
  reported_by?: string
  photo_keys?: string[]
  cost_cents?: number
  notes?: string
}

export interface UpdateDamageInput {
  description?: string
  severity?: DamageSeverity
  status?: DamageStatus
  photo_keys?: string[]
  cost_cents?: number
  notes?: string
}

export interface ResolveDamageInput {
  resolved_by?: string
  cost_cents?: number
  notes?: string
}

export interface ListDamagesParams {
  vehicle_id?: string
  status?: DamageStatus
  page?: number
  page_size?: number
}

export interface ListVehicleHistoryParams {
  page?: number
  page_size?: number
}

export interface CheckTuevDueParams {
  days_ahead?: number
}

export interface ListUpcomingServicesParams {
  days_ahead?: number
  page?: number
  page_size?: number
}

// ---------------------------------------------------------------------------
// Response wrapper types
// ---------------------------------------------------------------------------

export interface ListVehiclesResponse {
  vehicles: Vehicle[]
  total: number
}

export interface ListServicesResponse {
  services: VehicleService[]
  total: number
}

export interface ListDamagesResponse {
  damages: VehicleDamage[]
  total: number
}

export interface VehicleHistoryResponse {
  entries: VehicleHistoryEntry[]
  total: number
}

export interface CheckTuevDueResponse {
  vehicles: Vehicle[]
}
