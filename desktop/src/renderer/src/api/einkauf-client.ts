/**
 * Lightweight fetch wrapper for Einkauf API endpoints.
 *
 * Follows the same pattern as wiki-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry.
 */
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
import { authenticatedRequest } from './utils/authenticatedFetch'

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
