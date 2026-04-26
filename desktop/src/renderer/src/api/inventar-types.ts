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
  quantity: number
  min_quantity: number
  unit: string
  location: string | null
  created_at: string
  updated_at: string
}

export interface InventarMovement {
  id: string
  tenant_id: string
  item_id: string
  movement_type: MovementType
  quantity: number
  performed_by: string | null
  reason: string
  created_at: string
}

export interface StockWarning {
  id: string
  tenant_id: string
  item_id: string
  threshold: number
  current_quantity: number
  status: WarningStatus
  created_at: string
  acknowledged_at: string | null
  acknowledged_by: string | null
}

export interface StockReport {
  total_items: number
  total_quantity: number
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

export interface ExportInventoryParams {
  format?: 'csv'
}
