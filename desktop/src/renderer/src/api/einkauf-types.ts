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
