/**
 * Shared Vermietung display maps + helpers.
 *
 * Extracted from VermietungPage so the detail modals (ObjectDetailModal,
 * RentalDetailModal) and the page render types/status badges identically.
 */
import { Wrench, DoorOpen, Car, Hammer } from 'lucide-react'
import type { Rental, RentalObject } from '@/api/vermietung-types'

export const WEEKDAYS = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So']

export const OBJECT_TYPE_CONFIG: Record<
  string,
  { labelKey: string; icon: typeof Wrench; color: string; bg: string; badgeBg: string }
> = {
  gerät: { labelKey: 'vermietung.objectType.gerät', icon: Wrench, color: 'text-amber-600 dark:text-amber-400', bg: 'bg-amber-100 dark:bg-amber-900/30', badgeBg: 'bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300' },
  raum: { labelKey: 'vermietung.objectType.raum', icon: DoorOpen, color: 'text-blue-600 dark:text-blue-400', bg: 'bg-blue-100 dark:bg-blue-900/30', badgeBg: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300' },
  fahrzeug: { labelKey: 'vermietung.objectType.fahrzeug', icon: Car, color: 'text-green-600 dark:text-green-400', bg: 'bg-green-100 dark:bg-green-900/30', badgeBg: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300' },
  werkzeug: { labelKey: 'vermietung.objectType.werkzeug', icon: Hammer, color: 'text-orange-600 dark:text-orange-400', bg: 'bg-orange-100 dark:bg-orange-900/30', badgeBg: 'bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300' },
}

export const DEFAULT_TYPE_CONFIG = {
  labelKey: 'vermietung.objectType.gerät',
  icon: Wrench,
  color: 'text-muted-foreground',
  bg: 'bg-secondary',
  badgeBg: 'bg-secondary text-muted-foreground',
}

export function getTypeCfg(cat: string) {
  return OBJECT_TYPE_CONFIG[cat] ?? DEFAULT_TYPE_CONFIG
}

export const STATUS_CONFIG: Record<
  'available' | 'reserved' | 'maintenance',
  { labelKey: string; dot: string; text: string }
> = {
  available: { labelKey: 'vermietung.status.available', dot: 'bg-success', text: 'text-success' },
  reserved: { labelKey: 'vermietung.status.reserved', dot: 'bg-info', text: 'text-info' },
  maintenance: { labelKey: 'vermietung.status.maintenance', dot: 'bg-warning', text: 'text-warning' },
}

export const RESERVATION_STATUS_CONFIG: Record<string, { labelKey: string; bg: string }> = {
  active: { labelKey: 'vermietung.reservationStatus.active', bg: 'bg-success-light text-success' },
  reserved: { labelKey: 'vermietung.reservationStatus.upcoming', bg: 'bg-info-light text-info' },
  completed: { labelKey: 'vermietung.reservationStatus.completed', bg: 'bg-secondary text-muted-foreground' },
  cancelled: { labelKey: 'vermietung.reservationStatus.cancelled', bg: 'bg-error-light text-error' },
}

export const DEPOSIT_STATUS_CONFIG: Record<'none' | 'collected', { labelKey: string; bg: string }> = {
  none: { labelKey: 'vermietung.depositStatus.none', bg: 'bg-secondary text-muted-foreground' },
  collected: { labelKey: 'vermietung.depositStatus.collected', bg: 'bg-success-light text-success' },
}

export function computeObjectStatus(obj: RentalObject, rentals: Rental[]): 'available' | 'reserved' | 'maintenance' {
  if (!obj.active) return 'maintenance'
  const today = new Date().toISOString().slice(0, 10)
  const hasActiveRental = rentals.some(
    (r) =>
      r.object_id === obj.id &&
      (r.status === 'active' || r.status === 'reserved') &&
      r.end_date.slice(0, 10) >= today,
  )
  return hasActiveRental ? 'reserved' : 'available'
}

export function computeDepositStatus(rental: Rental): 'none' | 'collected' {
  return rental.deposit_paid ? 'collected' : 'none'
}

/** Overdue = ausgegeben (active), Rückgabedatum überschritten (Booqable "late"). */
export function isOverdue(rental: Rental): boolean {
  const today = new Date().toISOString().slice(0, 10)
  return rental.status === 'active' && rental.end_date.slice(0, 10) < today
}

export function getWeekDates(offset: number): string[] {
  const today = new Date()
  const dayOfWeek = today.getDay()
  const monday = new Date(today)
  monday.setDate(today.getDate() - ((dayOfWeek + 6) % 7) + offset * 7)
  const dates: string[] = []
  for (let i = 0; i < 7; i++) {
    const d = new Date(monday)
    d.setDate(monday.getDate() + i)
    dates.push(d.toISOString().split('T')[0])
  }
  return dates
}

export function getKW(dateStr: string): number {
  const date = new Date(dateStr + 'T00:00:00')
  const d = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()))
  const dayNum = d.getUTCDay() || 7
  d.setUTCDate(d.getUTCDate() + 4 - dayNum)
  const yearStart = new Date(Date.UTC(d.getUTCFullYear(), 0, 1))
  return Math.ceil(((d.getTime() - yearStart.getTime()) / 86400000 + 1) / 7)
}

export function formatDateRange(dates: string[]): string {
  if (dates.length === 0) return ''
  const first = new Date(dates[0] + 'T00:00:00')
  const last = new Date(dates[dates.length - 1] + 'T00:00:00')
  const opts: Intl.DateTimeFormatOptions = { day: '2-digit', month: 'short' }
  return `${first.toLocaleDateString('de-DE', opts)} – ${last.toLocaleDateString('de-DE', opts)} ${last.getFullYear()}`
}

export function formatDate(dateStr: string): string {
  return new Date(dateStr.slice(0, 10) + 'T00:00:00').toLocaleDateString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

export function isToday(dateStr: string): boolean {
  return dateStr === new Date().toISOString().split('T')[0]
}

export function daysBetween(start: string, end: string): number {
  const s = new Date(start + 'T00:00:00')
  const e = new Date(end + 'T00:00:00')
  return Math.max(1, Math.round((e.getTime() - s.getTime()) / 86400000) + 1)
}

export function dateInRange(date: string, start: string, end: string): boolean {
  return date >= start && date <= end
}

/** Shift an ISO date (yyyy-mm-dd) by n days. */
export function shiftDate(dateStr: string, days: number): string {
  const d = new Date(dateStr + 'T00:00:00')
  d.setDate(d.getDate() + days)
  return d.toISOString().slice(0, 10)
}

/** Compute rental price breakdown given days and rates. */
export function computeRentalPrice(
  totalDays: number,
  dailyRate: number,
  weeklyRate?: number,
): { weeks: number; remainingDays: number; total: number; breakdown: string } {
  if (weeklyRate && totalDays >= 7) {
    const weeks = Math.floor(totalDays / 7)
    const remainingDays = totalDays % 7
    const total = weeks * weeklyRate + remainingDays * dailyRate
    const parts: string[] = [`${weeks} Woche${weeks > 1 ? 'n' : ''}`]
    if (remainingDays > 0) {
      parts.push(`${remainingDays} Tag${remainingDays > 1 ? 'e' : ''}`)
    }
    return { weeks, remainingDays, total, breakdown: parts.join(' + ') }
  }
  return {
    weeks: 0,
    remainingDays: totalDays,
    total: totalDays * dailyRate,
    breakdown: `${totalDays} Tag${totalDays > 1 ? 'e' : ''}`,
  }
}
