/**
 * Shared status maps + helpers for the Produktion module — used by the page,
 * the detail modals, the settings panel and the exports so labels/colors stay
 * in one place.
 */
import type { ProductionOrder, QualityCheckResponse } from '@/api/produktion-types'

// ---------------------------------------------------------------------------
// Order status
// ---------------------------------------------------------------------------

export const orderStatusLabelKeys: Record<string, string> = {
  planned: 'produktion.status.planned',
  in_progress: 'produktion.status.in_progress',
  completed: 'produktion.status.completed',
  cancelled: 'produktion.status.cancelled',
}

export const orderStatusColors: Record<string, string> = {
  planned: 'bg-secondary text-muted-foreground',
  in_progress: 'bg-info-light text-info',
  completed: 'bg-success-light text-success',
  cancelled: 'bg-error-light text-error',
}

export const progressBarColors: Record<string, string> = {
  planned: 'bg-muted-foreground/40',
  in_progress: 'bg-info',
  completed: 'bg-success',
  cancelled: 'bg-error',
}

// ---------------------------------------------------------------------------
// Work step status
// ---------------------------------------------------------------------------

export const workStepStatusLabelKeys: Record<string, string> = {
  pending: 'produktion.workstep.pending',
  in_progress: 'produktion.workstep.inProgress',
  completed: 'produktion.workstep.completed',
  skipped: 'produktion.workstep.skipped',
}

export const workStepStatusColors: Record<string, string> = {
  pending: 'bg-secondary text-muted-foreground',
  in_progress: 'bg-info-light text-info',
  completed: 'bg-success-light text-success',
  skipped: 'bg-secondary text-muted-foreground/60',
}

// ---------------------------------------------------------------------------
// Machine status
// ---------------------------------------------------------------------------

export const machineStatusLabelKeys: Record<string, string> = {
  available: 'produktion.machineStatus.available',
  in_use: 'produktion.machineStatus.inUse',
  maintenance: 'produktion.machineStatus.maintenance',
}

export const machineStatusDots: Record<string, string> = {
  available: 'bg-success',
  in_use: 'bg-info',
  maintenance: 'bg-error',
}

// ---------------------------------------------------------------------------
// Priority (1 = highest … 5 = lowest, backend default 3)
// ---------------------------------------------------------------------------

export const PRIORITY_VALUES = [1, 2, 3, 4, 5] as const

export const priorityLabelKeys: Record<number, string> = {
  1: 'produktion.priority.p1',
  2: 'produktion.priority.p2',
  3: 'produktion.priority.p3',
  4: 'produktion.priority.p4',
  5: 'produktion.priority.p5',
}

export const priorityColors: Record<number, string> = {
  1: 'bg-error-light text-error',
  2: 'bg-warning-light text-warning',
  3: 'bg-secondary text-muted-foreground',
  4: 'bg-secondary text-muted-foreground',
  5: 'bg-secondary text-muted-foreground/60',
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

export function getDaysRemaining(dueDate: string): number {
  const due = new Date(dueDate)
  const now = new Date()
  return Math.ceil((due.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
}

export function formatDuration(minutes: number): string {
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  if (h === 0) return `${m}min`
  if (m === 0) return `${h}h`
  return `${h}h ${m}min`
}

/**
 * Deterministic demo material availability (no inventar wiring yet — real
 * stock lookup is a parity candidate). Same input always yields the same
 * traffic light so screenshots are stable.
 */
export function getMaterialAvailability(materialName: string, idx: number): 'green' | 'yellow' | 'red' {
  const val = (materialName.length * 7 + idx * 13) % 10
  if (val < 1) return 'red'
  if (val < 3) return 'yellow'
  return 'green'
}

/** Scrap derived from quality checks: every logged defect counts as one piece. */
export function getScrapForOrder(
  order: ProductionOrder,
  checks: QualityCheckResponse[],
): { quantity: number; rate: number } {
  const quantity = checks
    .filter((c) => c.order_id === order.id)
    .reduce((sum, c) => sum + (c.defects_found || 0), 0)
  const rate = order.quantity > 0 ? (quantity / order.quantity) * 100 : 0
  return { quantity, rate: Math.round(rate * 10) / 10 }
}

/** Map production orders by id → order_number for list/Gantt lookups. */
export function orderNumberById(orders: ProductionOrder[]): Map<string, string> {
  const map = new Map<string, string>()
  orders.forEach((o) => map.set(o.id, o.order_number))
  return map
}

/** Next PA number from the tenant prefix, e.g. PA-2026-0716-1234. */
export function generateOrderNumber(prefix: string): string {
  const now = new Date()
  const mmdd = `${String(now.getMonth() + 1).padStart(2, '0')}${String(now.getDate()).padStart(2, '0')}`
  return `${prefix}-${now.getFullYear()}-${mmdd}-${String(now.getTime()).slice(-4)}`
}
