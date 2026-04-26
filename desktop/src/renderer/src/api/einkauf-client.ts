/**
 * Lightweight fetch wrapper for Einkauf API endpoints.
 *
 * Follows the same pattern as wiki-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry.
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  Supplier,
  PurchaseOrder,
  POLine,
  CreateSupplierInput,
  UpdateSupplierInput,
  ListSuppliersParams,
  CreatePOInput,
  UpdatePOInput,
  ListPOsParams,
  AddPOLineInput,
  UpdatePOLineInput,
  PartialReceiveInput,
  ListSuppliersResponse,
  ListPOsResponse,
  ListPOLinesResponse,
} from './einkauf-types'

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

const BASE = '/api/v1/einkauf'

// ---------------------------------------------------------------------------
// Suppliers
// ---------------------------------------------------------------------------

export function listSuppliers(params?: ListSuppliersParams) {
  return request<ListSuppliersResponse>({
    method: 'GET',
    path: `${BASE}/suppliers`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getSupplier(id: string) {
  return request<Supplier>({ method: 'GET', path: `${BASE}/suppliers/${id}` })
}

export function createSupplier(body: CreateSupplierInput) {
  return request<Supplier>({ method: 'POST', path: `${BASE}/suppliers`, body })
}

export function updateSupplier(id: string, body: UpdateSupplierInput) {
  return request<Supplier>({ method: 'PATCH', path: `${BASE}/suppliers/${id}`, body })
}

export function deleteSupplier(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/suppliers/${id}` })
}

// ---------------------------------------------------------------------------
// Purchase Orders
// ---------------------------------------------------------------------------

export function listPOs(params?: ListPOsParams) {
  return request<ListPOsResponse>({
    method: 'GET',
    path: `${BASE}/pos`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getPO(id: string) {
  return request<PurchaseOrder>({ method: 'GET', path: `${BASE}/pos/${id}` })
}

export function createPO(body: CreatePOInput) {
  return request<PurchaseOrder>({ method: 'POST', path: `${BASE}/pos`, body })
}

export function updatePO(id: string, body: UpdatePOInput) {
  return request<PurchaseOrder>({ method: 'PATCH', path: `${BASE}/pos/${id}`, body })
}

export function deletePO(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/pos/${id}` })
}

export function submitPO(id: string) {
  return request<PurchaseOrder>({ method: 'POST', path: `${BASE}/pos/${id}/submit` })
}

export function receiveGoods(id: string) {
  return request<PurchaseOrder>({ method: 'POST', path: `${BASE}/pos/${id}/receive` })
}

export function partialReceive(id: string, body: PartialReceiveInput) {
  return request<PurchaseOrder>({ method: 'POST', path: `${BASE}/pos/${id}/partial-receive`, body })
}

export function exportPO(id: string, format: 'pdf' | 'csv' = 'pdf') {
  return request<Blob>({
    method: 'POST',
    path: `${BASE}/pos/${id}/export`,
    params: { format },
  })
}

// ---------------------------------------------------------------------------
// PO Lines
// ---------------------------------------------------------------------------

export function listPOLines(poId: string) {
  return request<ListPOLinesResponse>({
    method: 'GET',
    path: `${BASE}/pos/${poId}/lines`,
  })
}

export function addPOLine(poId: string, body: AddPOLineInput) {
  return request<POLine>({ method: 'POST', path: `${BASE}/pos/${poId}/lines`, body })
}

export function updatePOLine(poId: string, lineId: string, body: UpdatePOLineInput) {
  return request<POLine>({
    method: 'PATCH',
    path: `${BASE}/pos/${poId}/lines/${lineId}`,
    body,
  })
}

export function deletePOLine(poId: string, lineId: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/pos/${poId}/lines/${lineId}` })
}
