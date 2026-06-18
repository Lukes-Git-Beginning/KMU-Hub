/**
 * TypeScript types for the Einkauf (purchasing) module.
 *
 * Mirrors backend/internal/einkauf/models.go and service.go input types.
 * UUIDs are represented as strings; dates as ISO 8601 strings.
 * Numeric values (quantity, unit_price, tax_rate, total_amount) are strings
 * to avoid floating-point precision issues — parse with Decimal or parseFloat
 * only when needed for arithmetic.
 */

// ---------------------------------------------------------------------------
// Domain models
// ---------------------------------------------------------------------------

export type POStatus =
  | 'draft'
  | 'submitted'
  | 'sent'
  | 'partially_received'
  | 'received'
  | 'closed'
  | 'cancelled'

export interface Supplier {
  id: string
  tenant_id: string
  name: string
  contact_id: string | null
  email: string
  phone: string
  address: string
  payment_terms: string
  notes: string
  created_at: string
  updated_at: string
}

export interface PurchaseOrder {
  id: string
  tenant_id: string
  supplier_id: string
  po_number: string
  status: POStatus
  order_date: string
  expected_delivery_date: string | null
  total_amount: string
  currency: string
  notes: string
  created_by: string | null
  created_at: string
  updated_at: string
  /** Lines are populated on GetPO — absent on list responses. */
  lines?: POLine[]
}

export interface POLine {
  id: string
  tenant_id: string
  po_id: string
  product_name: string
  sku: string
  quantity: string
  unit_price: string
  tax_rate: string
  received_quantity: string
  line_position: number
  created_at: string
  updated_at: string
}

// ---------------------------------------------------------------------------
// Request input types — Suppliers
// ---------------------------------------------------------------------------

export interface CreateSupplierInput {
  name: string
  contact_id?: string
  email?: string
  phone?: string
  address?: string
  payment_terms?: string
  notes?: string
}

export interface UpdateSupplierInput {
  name?: string
  contact_id?: string | null // null to clear
  email?: string
  phone?: string
  address?: string
  payment_terms?: string
  notes?: string
}

export interface ListSuppliersParams {
  search?: string
  active_only?: boolean
  page?: number
  page_size?: number
}

// ---------------------------------------------------------------------------
// Request input types — Purchase Orders
// ---------------------------------------------------------------------------

export interface CreatePOInput {
  supplier_id: string
  po_number: string
  order_date?: string
  expected_delivery_date?: string
  currency?: string
  notes?: string
}

export interface UpdatePOInput {
  supplier_id?: string
  po_number?: string
  order_date?: string
  expected_delivery_date?: string | null // null to clear
  currency?: string
  notes?: string
}

export interface ListPOsParams {
  supplier_id?: string
  status?: POStatus
  date_from?: string
  date_to?: string
  page?: number
  page_size?: number
}

// ---------------------------------------------------------------------------
// Request input types — PO Lines
// ---------------------------------------------------------------------------

export interface AddPOLineInput {
  product_name: string
  sku?: string
  quantity: string
  unit_price: string
  tax_rate?: string
  line_position?: number
}

export interface UpdatePOLineInput {
  product_name?: string
  sku?: string
  quantity?: string
  unit_price?: string
  tax_rate?: string
  line_position?: number
}

export interface PartialReceiveItem {
  line_id: string
  received_quantity: string
}

export interface PartialReceiveInput {
  items: PartialReceiveItem[]
}

// ---------------------------------------------------------------------------
// Response wrapper types
// ---------------------------------------------------------------------------

export interface ListSuppliersResponse {
  suppliers: Supplier[]
  total: number
}

export interface ListPOsResponse {
  pos: PurchaseOrder[]
  total: number
}

export interface ListPOLinesResponse {
  lines: POLine[]
}

// ---------------------------------------------------------------------------
// Catalog Items
// ---------------------------------------------------------------------------

export interface CatalogItem {
  id: string
  tenant_id: string
  supplier_id: string
  name: string
  sku: string
  category: string
  price: string          // numeric string
  currency: string
  unit: string
  available: boolean
  min_order_qty: string  // numeric string
  created_at: string
  updated_at: string
}

export interface CreateCatalogItemInput {
  supplier_id: string
  name: string
  sku?: string
  category?: string
  price: string
  currency?: string
  unit?: string
  available?: boolean
  min_order_qty?: string
}

export interface UpdateCatalogItemInput {
  name?: string
  sku?: string
  category?: string
  price?: string
  currency?: string
  unit?: string
  available?: boolean
  min_order_qty?: string
}

export interface ListCatalogItemsParams {
  supplier_id?: string
  category?: string
  search?: string
  available?: boolean
  page?: number
  page_size?: number
}

export interface ListCatalogItemsResponse {
  catalog_items: CatalogItem[]
  total: number
}

// ---------------------------------------------------------------------------
// Supplier Ratings
// ---------------------------------------------------------------------------

export type RatingCategory = 'quality' | 'delivery' | 'price'

export interface SupplierRating {
  id: string
  tenant_id: string
  supplier_id: string
  category: RatingCategory
  rating: number         // 1-5
  comment: string
  rated_by: string | null
  rated_at: string
  created_at: string
  updated_at: string
}

export interface CreateSupplierRatingInput {
  supplier_id: string
  category: RatingCategory
  rating: number
  comment?: string
}

export interface ListSupplierRatingsResponse {
  ratings: SupplierRating[]
}

// ---------------------------------------------------------------------------
// Framework Contracts
// ---------------------------------------------------------------------------

export type ContractStatus = 'draft' | 'active' | 'expired'

export interface FrameworkContractItem {
  id: string
  tenant_id: string
  contract_id: string
  name: string
  unit_price: string    // numeric string
  unit: string
  agreed_qty: string    // numeric string
  called_qty: string    // numeric string
  created_at: string
  updated_at: string
}

export interface FrameworkContractCall {
  id: string
  tenant_id: string
  contract_id: string
  po_id: string | null
  amount: string        // numeric string
  currency: string
  called_at: string
  notes: string
  created_at: string
  updated_at: string
}

export interface FrameworkContract {
  id: string
  tenant_id: string
  supplier_id: string
  title: string
  contract_nr: string
  start_date: string | null
  end_date: string | null
  total_value: string   // numeric string
  used_value: string    // numeric string
  currency: string
  status: ContractStatus
  created_at: string
  updated_at: string
  items?: FrameworkContractItem[]
}

export interface CreateFrameworkContractInput {
  supplier_id: string
  title: string
  contract_nr: string
  start_date?: string
  end_date?: string
  total_value?: string
  currency?: string
  status?: ContractStatus
}

export interface UpdateFrameworkContractInput {
  title?: string
  contract_nr?: string
  start_date?: string | null
  end_date?: string | null
  total_value?: string
  currency?: string
  status?: ContractStatus
}

export interface ListFrameworkContractsParams {
  supplier_id?: string
  status?: ContractStatus
  page?: number
  page_size?: number
}

export interface ListFrameworkContractsResponse {
  contracts: FrameworkContract[]
  total: number
}

export interface CreateContractItemInput {
  contract_id: string
  name: string
  unit_price?: string
  unit?: string
  agreed_qty?: string
}

export interface UpdateContractItemInput {
  name?: string
  unit_price?: string
  unit?: string
  agreed_qty?: string
}

export interface CreateContractCallInput {
  contract_id: string
  po_id?: string
  amount: string
  currency?: string
  notes?: string
}

export interface ListContractCallsResponse {
  calls: FrameworkContractCall[]
}
