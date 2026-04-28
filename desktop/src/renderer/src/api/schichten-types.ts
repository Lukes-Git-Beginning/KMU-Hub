/**
 * TypeScript types for the Schichten (shift planning) module.
 *
 * Mirrors backend/internal/schichten/models.go and the proto schema.
 * UUIDs are represented as strings; timestamps as ISO 8601 / RFC3339 strings.
 */

// ---------------------------------------------------------------------------
// Enums / Literals
// ---------------------------------------------------------------------------

export type ShiftStatus = 'draft' | 'published'

// ---------------------------------------------------------------------------
// Domain models
// ---------------------------------------------------------------------------

export interface Shift {
  id: string
  tenant_id: string
  title: string
  description: string
  start_time: string  // RFC3339
  end_time: string    // RFC3339
  status: ShiftStatus
  location: string | null
  created_by: string | null
  created_at: string
  updated_at: string
}

export interface ShiftAssignment {
  id: string
  tenant_id: string
  shift_id: string
  employee_id: string
  assigned_at: string  // RFC3339
  assigned_by: string | null
}

export interface ShiftTemplate {
  id: string
  tenant_id: string
  name: string
  description: string
  day_of_week: number     // 0=Sunday … 6=Saturday
  start_hour: number      // 0–23
  start_minute: number    // 0–59
  duration_minutes: number
  location: string | null
  created_at: string
  updated_at: string
}

export interface ShiftStats {
  total_shifts: number
  published_shifts: number
  draft_shifts: number
  total_assignments: number
  unique_employees: number
}

export interface ArbzgCheckResult {
  compliant: boolean
  violation_reason: string  // empty string when compliant
}

// ---------------------------------------------------------------------------
// Request input types
// ---------------------------------------------------------------------------

export interface CreateShiftInput {
  title: string
  description?: string
  start_time: string  // RFC3339
  end_time: string    // RFC3339
  location?: string
  created_by?: string
}

export interface UpdateShiftInput {
  title?: string
  description?: string
  start_time?: string  // RFC3339
  end_time?: string    // RFC3339
  location?: string
}

export interface ListShiftsParams {
  status?: ShiftStatus | 'all'
  from?: string   // RFC3339
  to?: string     // RFC3339
  page?: number
  page_size?: number
}

export interface PublishShiftsInput {
  from: string  // RFC3339
  to: string    // RFC3339
}

export interface AssignEmployeeInput {
  employee_id: string
  assigned_by?: string
}

export interface CreateTemplateInput {
  name: string
  description?: string
  day_of_week: number
  start_hour: number
  start_minute: number
  duration_minutes: number
  location?: string
}

export interface UpdateTemplateInput {
  name?: string
  description?: string
  day_of_week?: number
  start_hour?: number
  start_minute?: number
  duration_minutes?: number
  location?: string
}

export interface ListTemplatesParams {
  page?: number
  page_size?: number
}

export interface ApplyTemplateInput {
  range_start: string  // RFC3339
  range_end: string    // RFC3339
  created_by?: string
}

export interface ArbzgCheckParams {
  employee_id: string
  new_shift_start: string  // RFC3339
  new_shift_end: string    // RFC3339
}

export interface ShiftStatsParams {
  from?: string  // RFC3339
  to?: string    // RFC3339
}

// ---------------------------------------------------------------------------
// Response wrapper types
// ---------------------------------------------------------------------------

export interface ListShiftsResponse {
  shifts: Shift[]
  total: number
}

export interface ListAssignmentsResponse {
  assignments: ShiftAssignment[]
}

export interface ListTemplatesResponse {
  templates: ShiftTemplate[]
  total: number
}

export interface PublishShiftsResponse {
  published_count: number
}

export interface ApplyTemplateResponse {
  shifts: Shift[]
  created_count: number
}
