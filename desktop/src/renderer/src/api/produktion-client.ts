/**
 * Lightweight fetch wrapper for Produktion API endpoints.
 *
 * Follows the same pattern as wiki-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry.
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  ProductionOrder,
  MachineBooking,
  ProductionPlan,
  CapacityOverview,
  CreateOrderInput,
  UpdateOrderInput,
  ListOrdersParams,
  CreateBookingInput,
  UpdateBookingInput,
  ListBookingsParams,
  CreatePlanInput,
  UpdatePlanInput,
  ListOrdersResponse,
  ListBookingsResponse,
} from './produktion-types'

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

const BASE = '/api/v1/produktion'

// ---------------------------------------------------------------------------
// Production Orders
// ---------------------------------------------------------------------------

export function listOrders(params?: ListOrdersParams) {
  return request<ListOrdersResponse>({
    method: 'GET',
    path: `${BASE}/orders`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getOrder(id: string) {
  return request<ProductionOrder>({ method: 'GET', path: `${BASE}/orders/${id}` })
}

export function createOrder(body: CreateOrderInput) {
  return request<ProductionOrder>({ method: 'POST', path: `${BASE}/orders`, body })
}

export function updateOrder(id: string, body: UpdateOrderInput) {
  return request<ProductionOrder>({ method: 'PATCH', path: `${BASE}/orders/${id}`, body })
}

export function deleteOrder(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/orders/${id}` })
}

export function startOrder(id: string) {
  return request<ProductionOrder>({ method: 'POST', path: `${BASE}/orders/${id}/start` })
}

export function completeOrder(id: string) {
  return request<ProductionOrder>({ method: 'POST', path: `${BASE}/orders/${id}/complete` })
}

export function cancelOrder(id: string) {
  return request<ProductionOrder>({ method: 'POST', path: `${BASE}/orders/${id}/cancel` })
}

// ---------------------------------------------------------------------------
// Machine Bookings
// ---------------------------------------------------------------------------

export function listMachineBookings(params?: ListBookingsParams) {
  return request<ListBookingsResponse>({
    method: 'GET',
    path: `${BASE}/bookings`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function createMachineBooking(body: CreateBookingInput) {
  return request<MachineBooking>({ method: 'POST', path: `${BASE}/bookings`, body })
}

export function updateMachineBooking(id: string, body: UpdateBookingInput) {
  return request<MachineBooking>({ method: 'PATCH', path: `${BASE}/bookings/${id}`, body })
}

export function deleteMachineBooking(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/bookings/${id}` })
}

// ---------------------------------------------------------------------------
// Production Plans
// ---------------------------------------------------------------------------

export function createPlan(body: CreatePlanInput) {
  return request<ProductionPlan>({ method: 'POST', path: `${BASE}/plans`, body })
}

export function getPlan(id: string) {
  return request<ProductionPlan>({ method: 'GET', path: `${BASE}/plans/${id}` })
}

export function updatePlan(id: string, body: UpdatePlanInput) {
  return request<ProductionPlan>({ method: 'PATCH', path: `${BASE}/plans/${id}`, body })
}

export function getCapacityOverview(planId: string, machineId: string) {
  return request<CapacityOverview>({
    method: 'GET',
    path: `${BASE}/plans/${planId}/capacity`,
    params: { machine_id: machineId },
  })
}
