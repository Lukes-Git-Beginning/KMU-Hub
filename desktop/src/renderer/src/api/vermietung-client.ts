/**
 * Lightweight fetch wrapper for Vermietung API endpoints.
 *
 * Follows the same pattern as inventar-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry.
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  RentalObject,
  Rental,
  RentalInspection,
  AvailabilityResult,
  CalendarResult,
  CreateObjectInput,
  UpdateObjectInput,
  ListObjectsParams,
  CreateRentalInput,
  UpdateRentalInput,
  ListRentalsParams,
  CheckAvailabilityParams,
  CreateInspectionInput,
  UpdateInspectionInput,
  ListInspectionsParams,
  GetRentalCalendarParams,
  ListObjectsResponse,
  ListRentalsResponse,
  ListInspectionsResponse,
} from './vermietung-types'

// ---------------------------------------------------------------------------
// Request helper
// ---------------------------------------------------------------------------

const MUTATION_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH'])

async function getAuthToken(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken
}

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | undefined>
}

async function request<T>(opts: RequestOptions): Promise<T> {
  if (!navigator.onLine && MUTATION_METHODS.has(opts.method)) {
    throw new Error('Änderungen sind offline nicht möglich.')
  }

  const url = new URL(`${API_BASE_URL}${opts.path}`)

  if (opts.params) {
    for (const [key, value] of Object.entries(opts.params)) {
      if (value === undefined) continue
      url.searchParams.set(key, String(value))
    }
  }

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  const token = await getAuthToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const init: RequestInit = { method: opts.method, headers }

  if (opts.body !== undefined) {
    init.body = JSON.stringify(opts.body)
  }

  const response = await fetch(url.toString(), init)

  if (!response.ok) {
    if (response.status === 401) {
      const { useAuthStore } = await import('@/stores/auth')
      const store = useAuthStore.getState()
      const newToken = await store.refreshToken()

      if (newToken) {
        headers['Authorization'] = `Bearer ${newToken}`
        const retryResponse = await fetch(url.toString(), { ...init, headers })

        if (!retryResponse.ok) {
          const errBody = await retryResponse.json().catch(() => ({}))
          throw new Error(
            (errBody as Record<string, string>).error ||
              `Request failed: ${retryResponse.status}`,
          )
        }

        if (retryResponse.status === 204) return {} as T
        return retryResponse.json() as Promise<T>
      }

      store.logout()
      throw new Error('Authentication expired')
    }

    const errBody = await response.json().catch(() => ({}))
    throw new Error(
      (errBody as Record<string, string>).error ||
        `Request failed: ${response.status}`,
    )
  }

  if (response.status === 204) return {} as T
  return response.json() as Promise<T>
}

// ---------------------------------------------------------------------------
// Base path
// ---------------------------------------------------------------------------

const BASE = '/api/v1/vermietung'

// ---------------------------------------------------------------------------
// Objects
// ---------------------------------------------------------------------------

export function listObjects(params?: ListObjectsParams) {
  return request<ListObjectsResponse>({
    method: 'GET',
    path: `${BASE}/objects`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getObject(id: string) {
  return request<{ object: RentalObject }>({
    method: 'GET',
    path: `${BASE}/objects/${id}`,
  })
}

export function createObject(body: CreateObjectInput) {
  return request<{ object: RentalObject }>({
    method: 'POST',
    path: `${BASE}/objects`,
    body,
  })
}

export function updateObject(id: string, body: UpdateObjectInput) {
  return request<{ object: RentalObject }>({
    method: 'PATCH',
    path: `${BASE}/objects/${id}`,
    body,
  })
}

export function deleteObject(id: string) {
  return request<void>({
    method: 'DELETE',
    path: `${BASE}/objects/${id}`,
  })
}

export function checkAvailability(objectId: string, params: CheckAvailabilityParams) {
  return request<AvailabilityResult>({
    method: 'GET',
    path: `${BASE}/objects/${objectId}/availability`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

// ---------------------------------------------------------------------------
// Rentals
// ---------------------------------------------------------------------------

export function listRentals(params?: ListRentalsParams) {
  return request<ListRentalsResponse>({
    method: 'GET',
    path: `${BASE}/rentals`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getRental(id: string) {
  return request<{ rental: Rental }>({
    method: 'GET',
    path: `${BASE}/rentals/${id}`,
  })
}

export function createRental(body: CreateRentalInput) {
  return request<{ rental: Rental }>({
    method: 'POST',
    path: `${BASE}/rentals`,
    body,
  })
}

export function updateRental(id: string, body: UpdateRentalInput) {
  return request<{ rental: Rental }>({
    method: 'PATCH',
    path: `${BASE}/rentals/${id}`,
    body,
  })
}

export function deleteRental(id: string) {
  return request<void>({
    method: 'DELETE',
    path: `${BASE}/rentals/${id}`,
  })
}

export function startRental(id: string) {
  return request<{ rental: Rental }>({
    method: 'POST',
    path: `${BASE}/rentals/${id}/start`,
  })
}

export function endRental(id: string) {
  return request<{ rental: Rental }>({
    method: 'POST',
    path: `${BASE}/rentals/${id}/end`,
  })
}

// ---------------------------------------------------------------------------
// Inspections
// ---------------------------------------------------------------------------

export function listInspections(rentalId: string, params?: ListInspectionsParams) {
  return request<ListInspectionsResponse>({
    method: 'GET',
    path: `${BASE}/rentals/${rentalId}/inspections`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function createInspection(rentalId: string, body: CreateInspectionInput) {
  return request<{ inspection: RentalInspection }>({
    method: 'POST',
    path: `${BASE}/rentals/${rentalId}/inspections`,
    body,
  })
}

export function getInspection(id: string) {
  return request<{ inspection: RentalInspection }>({
    method: 'GET',
    path: `${BASE}/inspections/${id}`,
  })
}

export function updateInspection(id: string, body: UpdateInspectionInput) {
  return request<{ inspection: RentalInspection }>({
    method: 'PATCH',
    path: `${BASE}/inspections/${id}`,
    body,
  })
}

// ---------------------------------------------------------------------------
// Calendar
// ---------------------------------------------------------------------------

export function getRentalCalendar(params: GetRentalCalendarParams) {
  return request<CalendarResult>({
    method: 'GET',
    path: `${BASE}/calendar`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

export function getExportUrl(format: 'csv' = 'csv') {
  return `${API_BASE_URL}${BASE}/export?format=${format}`
}
