/**
 * TypeScript types for the Produktion module.
 *
 * Mirrors backend/internal/produktion/models.go and service.go input types.
 * UUIDs are represented as strings; timestamps as ISO 8601 strings.
 */

// ---------------------------------------------------------------------------
// Domain models
// ---------------------------------------------------------------------------

export type OrderStatus = 'planned' | 'in_progress' | 'completed' | 'cancelled'
export type BookingStatus = 'booked' | 'in_use' | 'completed' | 'cancelled'
export type PlanStatus = 'draft' | 'active' | 'completed'

export interface ProductionOrder {
  id: string
  tenant_id: string
  order_number: string
  product_name: string
  quantity: number
  status: OrderStatus
  planned_start: string
  planned_end: string
  actual_start: string | null
  actual_end: string | null
  /** 1 (highest) – 5 (lowest), default 3 */
  priority: number
  notes: string
  created_by: string | null
  created_at: string
  updated_at: string
}

export interface MachineBooking {
  id: string
  tenant_id: string
  machine_id: string
  production_order_id: string
  starts_at: string
  ends_at: string
  status: BookingStatus
  notes: string
  created_by: string | null
  created_at: string
  updated_at: string
}

export interface ProductionPlan {
  id: string
  tenant_id: string
  name: string
  week_number: number
  year: number
  total_capacity_hours: number
  planned_capacity_hours: number
  status: PlanStatus
  notes: string
  created_by: string | null
  created_at: string
  updated_at: string
}

export interface CapacityOverview {
  machine_id: string
  total_capacity_hours: number
  booked_hours: number
  available_hours: number
}

// ---------------------------------------------------------------------------
// Request input types
// ---------------------------------------------------------------------------

export interface CreateOrderInput {
  order_number: string
  product_name: string
  quantity: number
  planned_start: string // RFC3339
  planned_end: string   // RFC3339
  priority?: number
  notes?: string
}

export interface UpdateOrderInput {
  product_name?: string
  quantity?: number
  planned_start?: string
  planned_end?: string
  priority?: number
  notes?: string
}

export interface ListOrdersParams {
  status?: OrderStatus
  priority?: number
  date_from?: string
  date_to?: string
  page?: number
  page_size?: number
}

export interface CreateBookingInput {
  machine_id: string
  production_order_id: string
  starts_at: string // RFC3339
  ends_at: string   // RFC3339
  notes?: string
}

export interface UpdateBookingInput {
  machine_id?: string
  production_order_id?: string
  starts_at?: string
  ends_at?: string
  notes?: string
}

export interface ListBookingsParams {
  machine_id?: string
  production_order_id?: string
  date_from?: string
  date_to?: string
  page?: number
  page_size?: number
}

export interface CreatePlanInput {
  name: string
  week_number: number
  year: number
  total_capacity_hours: number
  planned_capacity_hours?: number
  notes?: string
}

export interface UpdatePlanInput {
  name?: string
  week_number?: number
  year?: number
  total_capacity_hours?: number
  planned_capacity_hours?: number
  status?: PlanStatus
  notes?: string
}

// ---------------------------------------------------------------------------
// Response wrapper types
// ---------------------------------------------------------------------------

export interface ListOrdersResponse {
  orders: ProductionOrder[]
  total: number
}

export interface ListBookingsResponse {
  bookings: MachineBooking[]
  total: number
}
