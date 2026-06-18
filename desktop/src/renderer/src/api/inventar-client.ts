/**
 * Lightweight fetch wrapper for Inventar API endpoints.
 *
 * Follows the same pattern as wiki-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry.
 */
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
  InventoryLocation,
  InventurSession,
  InventurCount,
  CreateLocationInput,
  UpdateLocationInput,
  CreateInventurSessionInput,
  InventurStatus,
} from './inventar-types'
import { authenticatedRequest } from './utils/authenticatedFetch'
import { API_BASE_URL } from '@/lib/constants'

// ---------------------------------------------------------------------------
// Request helper
// ---------------------------------------------------------------------------

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | undefined>
}

function request<T>(opts: RequestOptions): Promise<T> {
  return authenticatedRequest<T>(opts)
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

// ---------------------------------------------------------------------------
// Locations
// ---------------------------------------------------------------------------
export function listLocations(params?: { page?: number; page_size?: number }) {
  return request<{ locations: InventoryLocation[]; total: number }>({
    method: 'GET',
    path: `${BASE}/locations`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getLocation(id: string) {
  return request<{ location: InventoryLocation }>({ method: 'GET', path: `${BASE}/locations/${id}` })
}

export function createLocation(body: CreateLocationInput) {
  return request<{ location: InventoryLocation }>({ method: 'POST', path: `${BASE}/locations`, body })
}

export function updateLocation(id: string, body: UpdateLocationInput) {
  return request<{ location: InventoryLocation }>({ method: 'PATCH', path: `${BASE}/locations/${id}`, body })
}

export function deleteLocation(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/locations/${id}` })
}

// ---------------------------------------------------------------------------
// Inventur Sessions
// ---------------------------------------------------------------------------
export function listInventurSessions(params?: { page?: number; page_size?: number }) {
  return request<{ sessions: InventurSession[]; total: number }>({
    method: 'GET',
    path: `${BASE}/inventur`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getInventurSession(id: string) {
  return request<{ session: InventurSession }>({ method: 'GET', path: `${BASE}/inventur/${id}` })
}

export function createInventurSession(body: CreateInventurSessionInput) {
  return request<{ session: InventurSession }>({ method: 'POST', path: `${BASE}/inventur`, body })
}

export function updateInventurSessionStatus(id: string, body: { status: InventurStatus }) {
  return request<{ session: InventurSession }>({ method: 'PATCH', path: `${BASE}/inventur/${id}/status`, body })
}

export function deleteInventurSession(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/inventur/${id}` })
}

export function upsertInventurCount(sessionId: string, body: { item_id: string; counted: number }) {
  return request<{ count: InventurCount }>({ method: 'POST', path: `${BASE}/inventur/${sessionId}/counts`, body })
}

export function bookInventurDifferences(sessionId: string, body?: { booked_by?: string }) {
  return request<{ session: InventurSession }>({ method: 'POST', path: `${BASE}/inventur/${sessionId}/book`, body })
}
