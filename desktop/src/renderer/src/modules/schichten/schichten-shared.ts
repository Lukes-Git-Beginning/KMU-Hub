/**
 * Shared helpers for the Schichten module — extracted from SchichtenPage so
 * the detail modal and the export builders reuse the same shift semantics
 * (surcharges, holidays, hours, styling).
 */
import { Sun, Sunset, Moon } from 'lucide-react'
import type { ShiftTemplate } from '@/stores/schichten'

export const WEEKDAYS = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So']

export interface SchichtenEmployee {
  id: string
  name: string
  initials: string
  availability: 'available' | 'limited' | 'unavailable'
  role: string
}

export interface ArbZGViolation {
  employeeId: string
  employeeName: string
  type: 'max_hours' | 'rest_period' | 'break_missing' | 'consecutive_days'
  message: string
  severity: 'warning' | 'error'
}

export const SHIFT_STYLE_MAP: Record<string, { bg: string; text: string; border: string; icon: typeof Sun }> = {
  'tpl-1': { bg: 'bg-info-light', text: 'text-info', border: 'border-info/20', icon: Sun },
  'tpl-2': { bg: 'bg-warning-light', text: 'text-warning', border: 'border-warning/20', icon: Sunset },
  'tpl-3': { bg: 'bg-primary-light', text: 'text-primary', border: 'border-primary/20', icon: Moon },
}

// 7.7: Surcharge rules (Zuschlaege)
export const SURCHARGE_RULES: Record<string, { label: string; rate: string }> = {
  'tpl-3': { label: 'Nacht', rate: '+25%' },
}

export const WEEKEND_SURCHARGE = { label: 'WE', rate: '+50%' }
export const HOLIDAY_SURCHARGE = { label: 'Feiertag', rate: '+100%' }

// 7.10: German holidays (DE-first, configurable)
export const GERMAN_HOLIDAYS_2026: Record<string, string> = {
  '2026-01-01': 'Neujahr',
  '2026-04-03': 'Karfreitag',
  '2026-04-06': 'Ostermontag',
  '2026-05-01': 'Tag der Arbeit',
  '2026-05-14': 'Christi Himmelfahrt',
  '2026-05-25': 'Pfingstmontag',
  '2026-10-03': 'Tag der Deutschen Einheit',
  '2026-12-25': '1. Weihnachtstag',
  '2026-12-26': '2. Weihnachtstag',
}

export function isWeekend(dateStr: string): boolean {
  const d = new Date(dateStr + 'T00:00:00')
  const day = d.getDay()
  return day === 0 || day === 6
}

export function isHoliday(dateStr: string): string | null {
  return GERMAN_HOLIDAYS_2026[dateStr] ?? null
}

export function computeShiftHours(templateId: string): number {
  const hours: Record<string, number> = { 'tpl-1': 7.5, 'tpl-2': 7.5, 'tpl-3': 7.25 }
  return hours[templateId] ?? 8
}

// 7.7: Compute surcharge for a shift on a given date
export function getSurchargeLabel(templateId: string, dateStr: string): { label: string; rate: string } | null {
  const holiday = isHoliday(dateStr)
  if (holiday) return HOLIDAY_SURCHARGE
  if (isWeekend(dateStr)) return WEEKEND_SURCHARGE
  return SURCHARGE_RULES[templateId] ?? null
}

// Derive initials from a full name ("Thomas Keller" → "TK")
export function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/)
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

/** Net minutes of a template shift (span minus break, overnight-safe). */
export function templateNetMinutes(template: ShiftTemplate): number {
  const [sh, sm] = template.startTime.split(':').map(Number)
  const [eh, em] = template.endTime.split(':').map(Number)
  let span = (eh * 60 + em) - (sh * 60 + sm)
  if (span < 0) span += 24 * 60
  return span - template.breakMinutes
}
