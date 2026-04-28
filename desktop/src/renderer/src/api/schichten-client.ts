/**
 * Lightweight fetch wrapper for Schichten API endpoints.
 *
 * Follows the same pattern as inventar-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry.
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  Shift,
  ShiftAssignment,
  ShiftTemplate,
  ShiftStats,
  ArbzgCheckResult,
  CreateShiftInput,
  UpdateShiftInput,
  ListShiftsParams,
  PublishShiftsInput,
  AssignEmployeeInput,
  CreateTemplateInput,
  UpdateTemplateInput,
  ListTemplatesParams,
  ApplyTemplateInput,
  ArbzgCheckParams,
  ListShiftsResponse,
  ListAssignmentsResponse,
  ListTemplatesResponse,
  PublishShiftsResponse,
  ApplyTemplateResponse,
} from './schichten-types'

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

const BASE = '/api/v1/schichten'

// ---------------------------------------------------------------------------
// Shifts
// ---------------------------------------------------------------------------

export function listShifts(params?: ListShiftsParams) {
  return request<ListShiftsResponse>({
    method: 'GET',
    path: `${BASE}/shifts`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getShift(id: string) {
  return request<{ shift: Shift }>({ method: 'GET', path: `${BASE}/shifts/${id}` })
}

export function createShift(body: CreateShiftInput) {
  return request<{ shift: Shift }>({ method: 'POST', path: `${BASE}/shifts`, body })
}

export function updateShift(id: string, body: UpdateShiftInput) {
  return request<{ shift: Shift }>({ method: 'PATCH', path: `${BASE}/shifts/${id}`, body })
}

export function deleteShift(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/shifts/${id}` })
}

export function publishShifts(body: PublishShiftsInput) {
  return request<PublishShiftsResponse>({ method: 'POST', path: `${BASE}/shifts/publish`, body })
}

// ---------------------------------------------------------------------------
// Assignments
// ---------------------------------------------------------------------------

export function listAssignments(shiftId: string) {
  return request<ListAssignmentsResponse>({
    method: 'GET',
    path: `${BASE}/shifts/${shiftId}/assignments`,
  })
}

export function assignEmployee(shiftId: string, body: AssignEmployeeInput) {
  return request<{ assignment: ShiftAssignment }>({
    method: 'POST',
    path: `${BASE}/shifts/${shiftId}/assignments`,
    body,
  })
}

export function unassignEmployee(shiftId: string, employeeId: string) {
  return request<void>({
    method: 'DELETE',
    path: `${BASE}/shifts/${shiftId}/assignments/${employeeId}`,
  })
}

// ---------------------------------------------------------------------------
// Templates
// ---------------------------------------------------------------------------

export function listTemplates(params?: ListTemplatesParams) {
  return request<ListTemplatesResponse>({
    method: 'GET',
    path: `${BASE}/templates`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function createTemplate(body: CreateTemplateInput) {
  return request<{ template: ShiftTemplate }>({ method: 'POST', path: `${BASE}/templates`, body })
}

export function updateTemplate(id: string, body: UpdateTemplateInput) {
  return request<{ template: ShiftTemplate }>({
    method: 'PATCH',
    path: `${BASE}/templates/${id}`,
    body,
  })
}

export function deleteTemplate(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/templates/${id}` })
}

export function applyTemplate(id: string, body: ApplyTemplateInput) {
  return request<ApplyTemplateResponse>({
    method: 'POST',
    path: `${BASE}/templates/${id}/apply`,
    body,
  })
}

// ---------------------------------------------------------------------------
// Compliance & Stats
// ---------------------------------------------------------------------------

export function checkArbzgCompliance(params: ArbzgCheckParams) {
  return request<ArbzgCheckResult>({
    method: 'GET',
    path: `${BASE}/compliance`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getShiftStats(params?: { from?: string; to?: string }) {
  return request<ShiftStats>({
    method: 'GET',
    path: `${BASE}/stats`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}
