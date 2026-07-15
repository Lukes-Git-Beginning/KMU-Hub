/**
 * Shared Inventar display maps + stock-status helpers.
 *
 * Extracted from InventarPage so the detail modals (ItemDetailModal,
 * LocationDetailModal) and the page render status/type badges identically.
 */
import {
  ArrowDownToLine,
  ArrowUpFromLine,
  RefreshCw,
  ClipboardEdit,
  Warehouse,
  Store,
  Truck,
} from 'lucide-react'
import type { InventarItem } from '@/api/inventar-types'

export const movementTypeKeys: Record<string, string> = {
  in: 'inventar.movementType.in',
  out: 'inventar.movementType.out',
  transfer: 'inventar.movementType.transfer',
  adjustment: 'inventar.movementType.adjustment',
}

export const movementTypeColors: Record<string, string> = {
  in: 'bg-success-light text-success',
  out: 'bg-error-light text-error',
  transfer: 'bg-info-light text-info',
  adjustment: 'bg-warning-light text-warning',
}

export const movementTypeIcons: Record<string, typeof ArrowDownToLine> = {
  in: ArrowDownToLine,
  out: ArrowUpFromLine,
  transfer: RefreshCw,
  adjustment: ClipboardEdit,
}

export const locationTypeKeys: Record<string, string> = {
  warehouse: 'inventar.locationType.warehouse',
  store: 'inventar.locationType.store',
  vehicle: 'inventar.locationType.vehicle',
}

export const locationTypeIcons: Record<string, typeof Warehouse> = {
  warehouse: Warehouse,
  store: Store,
  vehicle: Truck,
}

export const inventurStatusKeys: Record<string, string> = {
  open: 'inventar.inventurStatus.open',
  counting: 'inventar.inventurStatus.counting',
  review: 'inventar.inventurStatus.review',
  completed: 'inventar.inventurStatus.completed',
}

export const inventurStatusColors: Record<string, string> = {
  open: 'bg-info-light text-info',
  counting: 'bg-warning-light text-warning',
  review: 'bg-[#fff3e0] text-[#e65100] dark:bg-[#e65100]/20 dark:text-[#ffab40]',
  completed: 'bg-success-light text-success',
}

export function getStockStatus(item: InventarItem): 'ok' | 'warning' | 'critical' {
  // protojson serializes int64 as a JSON string (proto3 spec); coerce before comparing
  // to avoid lexicographic string comparison (e.g. "9" <= "10" would be false).
  const quantity = Number(item.quantity)
  const minQuantity = Number(item.min_quantity)
  if (quantity <= minQuantity) return 'critical'
  if (quantity < minQuantity * 2) return 'warning'
  return 'ok'
}

export const stockStatusLabelKeys: Record<string, string> = {
  critical: 'inventar.status.critical',
  warning: 'inventar.status.warning',
  ok: 'inventar.status.ok',
}

export function getStockStatusDisplay(item: InventarItem): {
  color: string
  labelKey: string
  dotColor: string
} {
  const status = getStockStatus(item)
  if (status === 'critical')
    return { color: 'bg-error', labelKey: stockStatusLabelKeys.critical, dotColor: 'bg-error' }
  if (status === 'warning')
    return { color: 'bg-warning', labelKey: stockStatusLabelKeys.warning, dotColor: 'bg-warning' }
  return { color: 'bg-success', labelKey: stockStatusLabelKeys.ok, dotColor: 'bg-success' }
}

/** Badge classes for the item stock status (detail modal header). */
export function stockStatusBadgeClass(item: InventarItem): string {
  const status = getStockStatus(item)
  if (status === 'critical') return 'bg-error-light text-error'
  if (status === 'warning') return 'bg-warning-light text-warning'
  return 'bg-success-light text-success'
}
