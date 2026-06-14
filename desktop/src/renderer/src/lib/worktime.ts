/**
 * Shared work-time formatting helpers (zeiterfassung module + header widget).
 */

/**
 * Contractual full-time daily norm (hours). The daily target is derived from
 * the employment contract, not a free user setting — weekly target =
 * workDaysPerWeek × this. Backend-gap: real per-contract weekly hours for
 * part-time accuracy.
 */
export const STANDARD_DAILY_HOURS = 8

/** Format minutes as "Xh Ym" / "Xh" (absolute value, no sign). */
export function formatWorkMinutes(total: number): string {
  const abs = Math.abs(total)
  const h = Math.floor(abs / 60)
  const m = abs % 60
  if (m === 0) return `${h}h`
  return `${h}h ${String(m).padStart(2, '0')}m`
}

/** Format a signed balance: "+12h 32m" / "−3h 15m" / "±0h" (U+2212 minus). */
export function formatSignedMinutes(total: number): string {
  if (total === 0) return '±0h'
  const sign = total > 0 ? '+' : '−'
  return `${sign}${formatWorkMinutes(total)}`
}
