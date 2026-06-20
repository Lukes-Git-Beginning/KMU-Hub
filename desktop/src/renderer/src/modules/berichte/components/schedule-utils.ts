/**
 * Shared scheduling helpers for report schedules (R-4).
 * Used by both the global "Geplant" list (ScheduleList) and the per-document
 * scheduling modal (ScheduleReportModal) so cron maths lives in one place.
 */
import type { ReportSchedule } from '@/api/berichte-types'

export const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export const DOW_MAP: Record<string, number> = {
  SUN: 0,
  MON: 1,
  TUE: 2,
  WED: 3,
  THU: 4,
  FRI: 5,
  SAT: 6,
}

/** Weekday keys in display order (Mon-first, European). */
export const WEEKDAYS = ['MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT', 'SUN'] as const

/**
 * Demo-grade next-run estimate from a 5-field cron expression
 * (min hour dom mon dow). Month is ignored for the demo; iterates day-by-day.
 */
export function computeNextRun(cron: string, from: Date = new Date()): Date | null {
  const parts = cron.trim().split(/\s+/)
  if (parts.length < 5) return null
  const [minF, hourF, domF, , dowF] = parts
  const min = minF === '*' ? 0 : parseInt(minF, 10)
  const hour = hourF === '*' ? null : parseInt(hourF, 10)
  const dom = domF === '*' ? null : parseInt(domF, 10)
  const dow = dowF === '*' ? null : DOW_MAP[dowF.toUpperCase()] ?? parseInt(dowF, 10)
  if (Number.isNaN(min)) return null
  for (let i = 0; i < 366; i++) {
    const d = new Date(from)
    d.setDate(from.getDate() + i)
    d.setHours(hour ?? from.getHours() + (i === 0 ? 1 : 0), min, 0, 0)
    if (dom !== null && d.getDate() !== dom) continue
    if (dow !== null && d.getDay() !== dow) continue
    if (d > from) return d
  }
  return null
}

export interface RunHistoryEntry {
  at: string
  status: 'success' | 'failed' | 'skipped'
  durationMs: number
}

/** Demo run history: last ~5 runs back from last_run_at, seeded per schedule. */
export function buildRunHistory(s: ReportSchedule): RunHistoryEntry[] {
  if (!s.last_run_at) return []
  const base = new Date(s.last_run_at).getTime()
  const cron = s.cron_expression
  const stepDays = /MON|TUE|WED|THU|FRI|SAT|SUN/i.test(cron)
    ? 7
    : cron.trim().split(/\s+/)[2] !== '*'
      ? 30
      : 1
  let seed = 0
  for (let i = 0; i < s.id.length; i++) seed = (seed * 31 + s.id.charCodeAt(i)) >>> 0
  return Array.from({ length: 5 }, (_, i) => {
    seed = (seed * 1664525 + 1013904223) >>> 0
    const r = seed / 0xffffffff
    const status: RunHistoryEntry['status'] =
      i === 0 ? (s.last_run_status ?? 'success') : r < 0.12 ? 'skipped' : r < 0.2 ? 'failed' : 'success'
    return {
      at: new Date(base - i * stepDays * 86_400_000).toISOString(),
      status,
      durationMs: 800 + Math.floor(r * 4200),
    }
  })
}

// ---------------------------------------------------------------------------
// Rhythm <-> cron — the per-document modal offers a friendly rhythm picker
// instead of a raw cron field, then round-trips through these helpers.
// ---------------------------------------------------------------------------

export type RhythmKind = 'daily' | 'weekly' | 'monthly' | 'quarterly'

export interface Rhythm {
  kind: RhythmKind
  /** Weekday key for weekly schedules (MON…SUN). */
  weekday: string
  /** Day of month for monthly/quarterly schedules (1…28). */
  day: number
  hour: number
  minute: number
}

export const DEFAULT_RHYTHM: Rhythm = {
  kind: 'weekly',
  weekday: 'MON',
  day: 1,
  hour: 8,
  minute: 0,
}

/** Build a 5-field cron expression from a friendly rhythm. */
export function rhythmToCron(r: Rhythm): string {
  const m = r.minute
  const h = r.hour
  switch (r.kind) {
    case 'daily':
      return `${m} ${h} * * *`
    case 'weekly':
      return `${m} ${h} * * ${r.weekday}`
    case 'monthly':
      return `${m} ${h} ${r.day} * *`
    case 'quarterly':
      return `${m} ${h} ${r.day} 1,4,7,10 *`
  }
}

/** Best-effort reverse: derive a rhythm from a cron expression for editing. */
export function cronToRhythm(cron: string): Rhythm {
  const parts = cron.trim().split(/\s+/)
  const [minF, hourF, domF, monF, dowF] =
    parts.length >= 5 ? parts : ['0', '8', '*', '*', '*']
  const minute = minF === '*' ? 0 : parseInt(minF, 10) || 0
  const hour = hourF === '*' ? 8 : parseInt(hourF, 10) || 0
  if (dowF && dowF !== '*') {
    return { kind: 'weekly', weekday: dowF.toUpperCase(), day: 1, hour, minute }
  }
  if (domF && domF !== '*') {
    const day = parseInt(domF, 10) || 1
    if (monF && monF !== '*') return { kind: 'quarterly', weekday: 'MON', day, hour, minute }
    return { kind: 'monthly', weekday: 'MON', day, hour, minute }
  }
  return { kind: 'daily', weekday: 'MON', day: 1, hour, minute }
}

/** Pad a clock value to two digits without locale formatting. */
export function pad2(n: number): string {
  return n < 10 ? `0${n}` : String(n)
}
