/**
 * TypeScript types for the Vermietung module.
 *
 * Mirrors backend/internal/vermietung/models.go and route_vermietung.go request types.
 * UUIDs are represented as strings; dates as ISO 8601 / RFC3339 strings.
 */

// ---------------------------------------------------------------------------
// Enums
// ---------------------------------------------------------------------------

export type RentalStatus = 'reserved' | 'active' | 'completed' | 'cancelled'
export type InspectionKind = 'handover' | 'return'

// ---------------------------------------------------------------------------
// Domain models
// ---------------------------------------------------------------------------

export interface RentalObject {
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

export interface Rental {
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

export interface RentalInspection {
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

// ---------------------------------------------------------------------------
// Availability & Calendar
// ---------------------------------------------------------------------------

export interface AvailabilityResult {
  available: boolean
  conflicting_rentals?: Rental[]
}

export interface CalendarEntry {
  object_id: string
  object_name: string
  rentals: Rental[]
}

export interface CalendarResult {
  entries: CalendarEntry[]
}

// ---------------------------------------------------------------------------
// Request input types — Objects
// ---------------------------------------------------------------------------

export interface CreateObjectInput {
  name: string
  description?: string
  category: string
  daily_rate: number
  deposit: number
  location?: string
  notes?: string
}

export interface UpdateObjectInput {
  name?: string
  description?: string
  category?: string
  daily_rate?: number
  deposit?: number
  location?: string
  notes?: string
  active?: boolean
}

export interface ListObjectsParams {
  search?: string
  category?: string
  active_only?: boolean
  page?: number
  page_size?: number
}

// ---------------------------------------------------------------------------
// Request input types — Rentals
// ---------------------------------------------------------------------------

export interface CreateRentalInput {
  object_id: string
  contact_id?: string
  renter_name: string
  start_date: string // RFC3339
  end_date: string   // RFC3339
  total_price: number
  deposit_paid: boolean
  notes?: string
}

export interface UpdateRentalInput {
  renter_name?: string
  start_date?: string // RFC3339
  end_date?: string   // RFC3339
  total_price?: number
  deposit_paid?: boolean
  notes?: string
}

export interface ListRentalsParams {
  object_id?: string
  status?: RentalStatus
  from?: string // RFC3339
  to?: string   // RFC3339
  page?: number
  page_size?: number
}

export interface CheckAvailabilityParams {
  start_date: string // RFC3339
  end_date: string   // RFC3339
  exclude_rental_id?: string
}

// ---------------------------------------------------------------------------
// Request input types — Inspections
// ---------------------------------------------------------------------------

export interface CreateInspectionInput {
  kind: InspectionKind
  notes: string
  photo_urls?: string[]
  performed_by?: string
}

export interface UpdateInspectionInput {
  notes?: string
  photo_urls?: string[]
}

export interface ListInspectionsParams {
  page?: number
  page_size?: number
}

// ---------------------------------------------------------------------------
// Calendar params
// ---------------------------------------------------------------------------

export interface GetRentalCalendarParams {
  month: number
  year: number
  object_id?: string
}

// ---------------------------------------------------------------------------
// Response wrapper types
// ---------------------------------------------------------------------------

export interface ListObjectsResponse {
  objects: RentalObject[]
  total: number
}

export interface ListRentalsResponse {
  rentals: Rental[]
  total: number
}

export interface ListInspectionsResponse {
  inspections: RentalInspection[]
  total: number
}
