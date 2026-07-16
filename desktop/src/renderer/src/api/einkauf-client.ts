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
  CatalogItem,
  CreateCatalogItemInput,
  UpdateCatalogItemInput,
  ListCatalogItemsParams,
  ListCatalogItemsResponse,
  SupplierRating,
  CreateSupplierRatingInput,
  ListSupplierRatingsResponse,
  FrameworkContract,
  CreateFrameworkContractInput,
  UpdateFrameworkContractInput,
  ListFrameworkContractsParams,
  ListFrameworkContractsResponse,
  FrameworkContractItem,
  CreateContractItemInput,
  UpdateContractItemInput,
  FrameworkContractCall,
  CreateContractCallInput,
  ListContractCallsResponse,
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
  return request<{ supplier: Supplier }>({ method: 'GET', path: `${BASE}/suppliers/${id}` })
    .then(r => r.supplier)
}

export function createSupplier(body: CreateSupplierInput) {
  return request<{ supplier: Supplier }>({ method: 'POST', path: `${BASE}/suppliers`, body })
    .then(r => r.supplier)
}

export function updateSupplier(id: string, body: UpdateSupplierInput) {
  return request<{ supplier: Supplier }>({ method: 'PATCH', path: `${BASE}/suppliers/${id}`, body })
    .then(r => r.supplier)
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
  return request<{ po: PurchaseOrder }>({ method: 'GET', path: `${BASE}/pos/${id}` })
    .then(r => r.po)
}

export function createPO(body: CreatePOInput) {
  return request<{ po: PurchaseOrder }>({ method: 'POST', path: `${BASE}/pos`, body })
    .then(r => r.po)
}

export function updatePO(id: string, body: UpdatePOInput) {
  return request<{ po: PurchaseOrder }>({ method: 'PATCH', path: `${BASE}/pos/${id}`, body })
    .then(r => r.po)
}

export function deletePO(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/pos/${id}` })
}

export function submitPO(id: string) {
  return request<{ po: PurchaseOrder }>({ method: 'POST', path: `${BASE}/pos/${id}/submit` })
    .then(r => r.po)
}

export function receiveGoods(id: string) {
  return request<{ po: PurchaseOrder }>({ method: 'POST', path: `${BASE}/pos/${id}/receive` })
    .then(r => r.po)
}

export function partialReceive(id: string, body: PartialReceiveInput) {
  return request<{ po: PurchaseOrder }>({ method: 'POST', path: `${BASE}/pos/${id}/partial-receive`, body })
    .then(r => r.po)
}

// PO exports are generated client-side (modules/einkauf/einkauf-export.ts) —
// the backend ExportPO endpoint is a stub and returns no document.

/**
 * 🔒 Backend-Gap: es gibt keinen Cancel-Endpoint im einkauf-Service
 * (UpdatePO trägt kein Status-Feld). Mock-first gegen MSW; der echte
 * Endpoint ist in backend-gaps.md für Luke notiert.
 */
export function cancelPO(id: string) {
  return request<{ po: PurchaseOrder }>({ method: 'POST', path: `${BASE}/pos/${id}/cancel` })
    .then(r => r.po)
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
  return request<{ line: POLine }>({ method: 'POST', path: `${BASE}/pos/${poId}/lines`, body })
    .then(r => r.line)
}

export function updatePOLine(poId: string, lineId: string, body: UpdatePOLineInput) {
  return request<{ line: POLine }>({
    method: 'PATCH',
    path: `${BASE}/pos/${poId}/lines/${lineId}`,
    body,
  }).then(r => r.line)
}

export function deletePOLine(poId: string, lineId: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/pos/${poId}/lines/${lineId}` })
}

// ---------------------------------------------------------------------------
// Catalog Items  (BASE/catalog)
// ---------------------------------------------------------------------------

export function listCatalogItems(params?: ListCatalogItemsParams) {
  return request<ListCatalogItemsResponse>({
    method: 'GET',
    path: `${BASE}/catalog`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getCatalogItem(id: string) {
  return request<{ catalog_item: CatalogItem }>({ method: 'GET', path: `${BASE}/catalog/${id}` })
    .then(r => r.catalog_item)
}

export function createCatalogItem(body: CreateCatalogItemInput) {
  return request<{ catalog_item: CatalogItem }>({ method: 'POST', path: `${BASE}/catalog`, body })
    .then(r => r.catalog_item)
}

export function updateCatalogItem(id: string, body: UpdateCatalogItemInput) {
  return request<{ catalog_item: CatalogItem }>({ method: 'PATCH', path: `${BASE}/catalog/${id}`, body })
    .then(r => r.catalog_item)
}

export function deleteCatalogItem(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/catalog/${id}` })
}

// ---------------------------------------------------------------------------
// Supplier Ratings (BASE/suppliers/:id/ratings)
// ---------------------------------------------------------------------------

export function listSupplierRatings(supplierId: string) {
  return request<ListSupplierRatingsResponse>({
    method: 'GET',
    path: `${BASE}/suppliers/${supplierId}/ratings`,
  })
}

export function createSupplierRating(supplierId: string, body: Omit<CreateSupplierRatingInput, 'supplier_id'>) {
  return request<{ rating: SupplierRating }>({
    method: 'POST',
    path: `${BASE}/suppliers/${supplierId}/ratings`,
    body,
  }).then(r => r.rating)
}

export function deleteSupplierRating(supplierId: string, ratingId: string) {
  return request<void>({
    method: 'DELETE',
    path: `${BASE}/suppliers/${supplierId}/ratings/${ratingId}`,
  })
}

// ---------------------------------------------------------------------------
// Framework Contracts (BASE/contracts)
// ---------------------------------------------------------------------------

export function listFrameworkContracts(params?: ListFrameworkContractsParams) {
  return request<ListFrameworkContractsResponse>({
    method: 'GET',
    path: `${BASE}/contracts`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getFrameworkContract(id: string) {
  return request<{ contract: FrameworkContract }>({ method: 'GET', path: `${BASE}/contracts/${id}` })
    .then(r => r.contract)
}

export function createFrameworkContract(body: CreateFrameworkContractInput) {
  return request<{ contract: FrameworkContract }>({ method: 'POST', path: `${BASE}/contracts`, body })
    .then(r => r.contract)
}

export function updateFrameworkContract(id: string, body: UpdateFrameworkContractInput) {
  return request<{ contract: FrameworkContract }>({ method: 'PATCH', path: `${BASE}/contracts/${id}`, body })
    .then(r => r.contract)
}

export function deleteFrameworkContract(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/contracts/${id}` })
}

export function createContractItem(contractId: string, body: Omit<CreateContractItemInput, 'contract_id'>) {
  return request<{ item: FrameworkContractItem }>({
    method: 'POST',
    path: `${BASE}/contracts/${contractId}/items`,
    body,
  }).then(r => r.item)
}

export function updateContractItem(contractId: string, itemId: string, body: UpdateContractItemInput) {
  return request<{ item: FrameworkContractItem }>({
    method: 'PATCH',
    path: `${BASE}/contracts/${contractId}/items/${itemId}`,
    body,
  }).then(r => r.item)
}

export function deleteContractItem(contractId: string, itemId: string) {
  return request<void>({
    method: 'DELETE',
    path: `${BASE}/contracts/${contractId}/items/${itemId}`,
  })
}

export function createContractCall(contractId: string, body: Omit<CreateContractCallInput, 'contract_id'>) {
  return request<{ call: FrameworkContractCall }>({
    method: 'POST',
    path: `${BASE}/contracts/${contractId}/calls`,
    body,
  }).then(r => r.call)
}

export function listContractCalls(contractId: string) {
  return request<ListContractCallsResponse>({
    method: 'GET',
    path: `${BASE}/contracts/${contractId}/calls`,
  })
}
