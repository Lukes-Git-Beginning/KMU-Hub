/**
 * Typed API client for Booking Pages endpoints.
 *
 * Field asymmetry: backend returns `availability_rules_json` (string),
 * client sends `availability_rules` (string). Parse on read, stringify on write.
 * UI components receive `BookingPage` with `availabilityRules: AvailabilityRules`.
 */
import { authenticatedRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface BookingPageService {
  id: string
  name: string
  duration_min: number
  price: string
  description?: string | null
}

export interface AvailabilitySlot {
  start: string  // "HH:MM"
  end: string    // "HH:MM"
}

export interface AvailabilityRules {
  weekdays: number[]                   // 0=So … 6=Sa
  slots_per_weekday: AvailabilitySlot[]
  slot_duration_min: number
  buffer_min: number
  lead_time_hours: number
  breaks?: AvailabilitySlot[]
}

/** Raw shape returned by the backend (availability_rules_json is a JSON string). */
interface RawBookingPage {
  id: string
  tenant_id: string
  calendar_id: string
  slug: string
  company_name: string
  logo_url?: string | null
  services: BookingPageService[]
  availability_rules_json: string
  active: boolean
  created_at: string
  updated_at: string
}

/** Client-side shape with parsed availabilityRules object. */
export interface BookingPage {
  id: string
  tenant_id: string
  calendar_id: string
  slug: string
  company_name: string
  logo_url?: string | null
  services: BookingPageService[]
  availabilityRules: AvailabilityRules
  active: boolean
  created_at: string
  updated_at: string
}

export interface CreateBookingPageRequest {
  calendar_id: string
  slug: string
  company_name: string
  logo_url?: string | null
  services: BookingPageService[]
  availability_rules: string  // JSON.stringify(AvailabilityRules)
}

export interface UpdateBookingPageRequest {
  company_name?: string
  logo_url?: string | null
  services?: BookingPageService[]
  availability_rules?: string  // JSON.stringify(AvailabilityRules)
  active?: boolean
}

// ---------------------------------------------------------------------------
// Internal fetch helper
// ---------------------------------------------------------------------------

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | string[] | undefined>
}

function request<T>(opts: RequestOptions): Promise<T> {
  return authenticatedRequest<T>(opts)
}

function get<T>(path: string, params?: RequestOptions['params']): Promise<T> {
  return request<T>({ method: 'GET', path, params })
}

function post<T>(path: string, body?: unknown): Promise<T> {
  return request<T>({ method: 'POST', path, body })
}

function put<T>(path: string, body?: unknown): Promise<T> {
  return request<T>({ method: 'PUT', path, body })
}

function del<T = void>(path: string): Promise<T> {
  return request<T>({ method: 'DELETE', path })
}

// ---------------------------------------------------------------------------
// Parse helper
// ---------------------------------------------------------------------------

function parseBookingPage(raw: RawBookingPage): BookingPage {
  const { availability_rules_json, ...rest } = raw
  return {
    ...rest,
    availabilityRules: JSON.parse(availability_rules_json) as AvailabilityRules,
  }
}

// ---------------------------------------------------------------------------
// API functions
// ---------------------------------------------------------------------------

export function listBookingPages(
  includeInactive?: boolean,
): Promise<{ pages: BookingPage[] }> {
  return get<{ pages: RawBookingPage[] }>(
    '/api/v1/calendar/booking-pages',
    includeInactive !== undefined
      ? { include_inactive: includeInactive }
      : undefined,
  ).then((res) => ({ pages: res.pages.map(parseBookingPage) }))
}

export function getBookingPage(id: string): Promise<{ page: BookingPage }> {
  return get<{ page: RawBookingPage }>(
    `/api/v1/calendar/booking-pages/${id}`,
  ).then((res) => ({ page: parseBookingPage(res.page) }))
}

export function createBookingPage(
  req: CreateBookingPageRequest,
): Promise<{ page: BookingPage }> {
  return post<{ page: RawBookingPage }>(
    '/api/v1/calendar/booking-pages',
    req,
  ).then((res) => ({ page: parseBookingPage(res.page) }))
}

export function updateBookingPage(
  id: string,
  req: UpdateBookingPageRequest,
): Promise<{ page: BookingPage }> {
  return put<{ page: RawBookingPage }>(
    `/api/v1/calendar/booking-pages/${id}`,
    req,
  ).then((res) => ({ page: parseBookingPage(res.page) }))
}

export function deleteBookingPage(id: string): Promise<{ status: string }> {
  return del<{ status: string }>(`/api/v1/calendar/booking-pages/${id}`)
}
