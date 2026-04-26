/**
 * Lightweight fetch wrapper for Inventar API endpoints.
 *
 * Follows the same pattern as wiki-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry.
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  InventarItem,
  InventarMovement,
  StockWarning,
  StockReport,
  CreateItemInput,
  UpdateItemInput,
  ListItemsParams,
  AdjustStockInput,
  TransferStockInput,
  RecordMovementInput,
  ListMovementsParams,
  CreateWarningInput,
  UpdateWarningInput,
  AcknowledgeWarningInput,
  ListWarningsParams,
  ListItemsResponse,
  ListMovementsResponse,
  ListWarningsResponse,
} from './inventar-types'

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

const BASE = '/api/v1/inventar'

// ---------------------------------------------------------------------------
// Items
// ---------------------------------------------------------------------------

export function listItems(params?: ListItemsParams) {
  return request<ListItemsResponse>({
    method: 'GET',
    path: `${BASE}/items`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getItem(id: string) {
  return request<{ item: InventarItem }>({ method: 'GET', path: `${BASE}/items/${id}` })
}

export function createItem(body: CreateItemInput) {
  return request<{ item: InventarItem }>({ method: 'POST', path: `${BASE}/items`, body })
}

export function updateItem(id: string, body: UpdateItemInput) {
  return request<{ item: InventarItem }>({ method: 'PATCH', path: `${BASE}/items/${id}`, body })
}

export function deleteItem(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/items/${id}` })
}

// ---------------------------------------------------------------------------
// Stock operations
// ---------------------------------------------------------------------------

export function adjustStock(itemId: string, body: AdjustStockInput) {
  return request<{ item: InventarItem }>({
    method: 'POST',
    path: `${BASE}/items/${itemId}/adjust`,
    body,
  })
}

export function transferStock(body: TransferStockInput) {
  return request<void>({ method: 'POST', path: `${BASE}/transfer`, body })
}

export function recordMovement(itemId: string, body: RecordMovementInput) {
  return request<{ movement: InventarMovement }>({
    method: 'POST',
    path: `${BASE}/items/${itemId}/movements`,
    body,
  })
}

export function listMovements(itemId: string, params?: ListMovementsParams) {
  return request<ListMovementsResponse>({
    method: 'GET',
    path: `${BASE}/items/${itemId}/movements`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getStockHistory(itemId: string, params?: ListMovementsParams) {
  return request<ListMovementsResponse>({
    method: 'GET',
    path: `${BASE}/items/${itemId}/history`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

// ---------------------------------------------------------------------------
// Warnings
// ---------------------------------------------------------------------------

export function listWarnings(params?: ListWarningsParams) {
  return request<ListWarningsResponse>({
    method: 'GET',
    path: `${BASE}/warnings`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function createWarning(body: CreateWarningInput) {
  return request<{ warning: StockWarning }>({ method: 'POST', path: `${BASE}/warnings`, body })
}

export function updateWarning(id: string, body: UpdateWarningInput) {
  return request<{ warning: StockWarning }>({
    method: 'PATCH',
    path: `${BASE}/warnings/${id}`,
    body,
  })
}

export function acknowledgeWarning(id: string, body?: AcknowledgeWarningInput) {
  return request<{ warning: StockWarning }>({
    method: 'POST',
    path: `${BASE}/warnings/${id}/acknowledge`,
    body,
  })
}

// ---------------------------------------------------------------------------
// Reports
// ---------------------------------------------------------------------------

export function getStockReport() {
  return request<StockReport>({ method: 'GET', path: `${BASE}/report` })
}

export function getExportUrl(format: 'csv' = 'csv') {
  return `${API_BASE_URL}${BASE}/export?format=${format}`
}
