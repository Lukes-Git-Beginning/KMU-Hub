/**
 * Lightweight fetch wrapper for Fuhrpark API endpoints.
 *
 * Follows the same pattern as inventar-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry.
 */
import { API_BASE_URL } from '@/lib/constants'
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

export function getExportUrl(format: 'csv' = 'csv') {
  return `${API_BASE_URL}${BASE}/export?format=${format}`
}
