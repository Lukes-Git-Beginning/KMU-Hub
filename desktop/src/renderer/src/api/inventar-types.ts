/**
 * TypeScript types for the Inventar module.
 *
 * Mirrors backend/internal/inventar/models.go and service.go input types.
 * UUIDs are represented as strings; dates as ISO 8601 strings.
 */

// ---------------------------------------------------------------------------
// Domain models
// ---------------------------------------------------------------------------

export type MovementType = 'in' | 'out' | 'adjustment' | 'transfer'
export type WarningStatus = 'active' | 'acknowledged' | 'resolved'

export interface InventarItem {
  id: string
  tenant_id: string
  name: string
  sku: string
  barcode: string | null
  // protojson serializes int64 as a JSON string (proto3 spec); encoding/json
  // emitted a number. Accept both and coerce (Number(...)) at the call site.
  quantity: number | string
  min_quantity: number | string
  unit: string
  location: string | null
  created_at: string
  updated_at: string
  location_id: string | null
  category: string | null
  price: number | null
  currency: string | null
  batch_number: string | null
  serial_numbers: string[]
  linked_purchase_order: string | null
}

export interface ItemAttachment {
  id: string
  tenant_id: string
  item_id: string
  name: string
  object_key: string
  file_type: string
  created_at: string
  updated_at: string
}

export interface InventarMovement {
  id: string
  tenant_id: string
  item_id: string
  movement_type: MovementType
  // protojson serializes int64 as a JSON string (proto3 spec); coerce at the call site.
  quantity: number | string
  performed_by: string | null
  reason: string
  created_at: string
  reference: string | null
  location_from: string | null
  location_to: string | null
  notes: string | null
}

export interface StockWarning {
  id: string
  tenant_id: string
  item_id: string
  // protojson serializes int64 as a JSON string (proto3 spec); coerce at the call site.
  threshold: number | string
  current_quantity: number | string
  status: WarningStatus
  created_at: string
  acknowledged_at: string | null
  acknowledged_by: string | null
}

export interface StockReport {
  total_items: number
  // protojson serializes int64 as a JSON string (proto3 spec); coerce at the call site.
  total_quantity: number | string
  low_stock_count: number
  active_warnings: number
}

// ---------------------------------------------------------------------------
// Request input types
// ---------------------------------------------------------------------------

export interface CreateItemInput {
  name: string
  sku: string
  barcode?: string
  quantity?: number
  min_quantity?: number
  unit: string
  location?: string
}

export interface UpdateItemInput {
  name?: string
  sku?: string
  barcode?: string
  min_quantity?: number
  unit?: string
  location?: string
}

export interface ListItemsParams {
  search?: string
  location?: string
  low_stock?: boolean
  page?: number
  page_size?: number
}

export interface AdjustStockInput {
  delta: number
  performed_by?: string
  reason: string
}

export interface TransferStockInput {
  from_item_id: string
  to_item_id: string
  quantity: number
  performed_by?: string
  reason: string
}

export interface RecordMovementInput {
  movement_type: MovementType
  quantity: number
  performed_by?: string
  reason: string
}

export interface ListMovementsParams {
  page?: number
  page_size?: number
}

export interface CreateWarningInput {
  item_id: string
  threshold: number
  current_quantity: number
}

export interface UpdateWarningInput {
  status: WarningStatus
}

export interface AcknowledgeWarningInput {
  acknowledged_by?: string
}

export interface ListWarningsParams {
  status?: WarningStatus
  page?: number
  page_size?: number
}

// ---------------------------------------------------------------------------
// Response wrapper types
// ---------------------------------------------------------------------------

export interface ListItemsResponse {
  items: InventarItem[]
  total: number
}

export interface ListMovementsResponse {
  movements: InventarMovement[]
  total: number
}

export interface ListWarningsResponse {
  warnings: StockWarning[]
  total: number
}

export type LocationType = 'warehouse' | 'store' | 'vehicle'
export type InventurStatus = 'open' | 'counting' | 'review' | 'completed'

export interface InventoryLocation {
  id: string
  tenant_id: string
  name: string
  address: string
  type: LocationType
  created_at: string
  updated_at: string
}

export interface InventurCount {
  id: string
  tenant_id: string
  session_id: string
  item_id: string
  // protojson serializes int64 as a JSON string (proto3 spec); coerce at the call site.
  expected: number | string
  counted: number | string | null
  counted_at: string | null
}

export interface InventurSession {
  id: string
  tenant_id: string
  name: string
  date: string
  status: InventurStatus
  location_id: string | null
  created_by: string | null
  counts: InventurCount[]
  created_at: string
  updated_at: string
}

export interface CreateLocationInput {
  name: string
  address: string
  type: LocationType
}

export interface UpdateLocationInput {
  name?: string
  address?: string
  type?: LocationType
}

export interface CreateInventurSessionInput {
  name: string
  date?: string
  location_id?: string
  item_ids?: string[]
}

export interface ListLocationsParams {
  page?: number
  page_size?: number
}

export interface ListInventurSessionsParams {
  page?: number
  page_size?: number
}

export interface ExportInventoryResponse {
  url: string
}

export interface ExportInventoryParams {
  format?: 'csv'
}
