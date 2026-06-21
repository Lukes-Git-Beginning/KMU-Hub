/**
 * Notification gating — decides whether a live toast may be shown right now.
 *
 * The notification center exposes "Bitte nicht stören" (DND) and Ruhezeiten
 * (quiet hours), but until this helper existed the toast pipeline ignored both,
 * so the settings had no visible effect. These pure predicates let the toast
 * (and any future live surface) suppress itself when the user asked for quiet.
 */
import type { QuietHours, DNDStatus } from '@/api/notification-client'

/**
 * True if `now` falls inside the active quiet-hours window.
 *
 * Handles overnight windows (e.g. 22:00 → 07:00 wraps past midnight) and the
 * per-weekday `days` mask (0 = Sunday … 6 = Saturday). An inactive or empty
 * configuration never suppresses.
 */
export function isWithinQuietHours(qh: QuietHours | undefined | null, now: Date = new Date()): boolean {
  if (!qh || !qh.is_active) return false
  const day = now.getDay()
  if (qh.days && qh.days.length > 0 && !qh.days.includes(day)) return false

  const start = toMinutes(qh.start_time, 22 * 60)
  const end = toMinutes(qh.end_time, 7 * 60)
  if (start === end) return false

  const cur = now.getHours() * 60 + now.getMinutes()
  // start < end → same-day window; start > end → overnight window.
  return start < end ? cur >= start && cur < end : cur >= start || cur < end
}

/** True if DND is active and not past its expiry. */
export function isDNDActive(dnd: DNDStatus | undefined | null, now: Date = new Date()): boolean {
  if (!dnd || !dnd.is_active) return false
  if (dnd.expires_at && new Date(dnd.expires_at).getTime() <= now.getTime()) return false
  return true
}

/**
 * Whether new toasts should be suppressed right now — true when either DND is
 * active or the current time is inside the quiet-hours window.
 */
export function shouldSuppressToast(args: {
  dnd?: DNDStatus | null
  quietHours?: QuietHours | null
  now?: Date
}): boolean {
  const now = args.now ?? new Date()
  return isDNDActive(args.dnd, now) || isWithinQuietHours(args.quietHours, now)
}

/** Parse "HH:MM" into minutes-of-day, falling back to `fallback` on bad input. */
function toMinutes(value: string | undefined, fallback: number): number {
  if (!value) return fallback
  const [h, m] = value.split(':').map((n) => Number(n))
  if (Number.isNaN(h) || Number.isNaN(m)) return fallback
  return h * 60 + m
}
