/**
 * Lightweight month-grid helpers for the work calendar view.
 *
 * No reusable month-grid primitive existed in the codebase, so this module
 * provides the minimal date math: a Monday-aligned grid of full weeks that
 * covers a given month, plus local date-key formatting used to bucket tasks
 * by their due date.
 */

/** Local YYYY-MM-DD key (timezone-safe, avoids ISO UTC shifts). */
export function dateToKey(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

/** Parse a task due_date (date or full ISO timestamp) to a local date key. */
export function dueDateToKey(due?: string): string | null {
  if (!due) return null
  const d = new Date(due)
  if (isNaN(d.getTime())) return null
  return dateToKey(d)
}

/**
 * Build a Monday-aligned grid of full weeks covering the target month.
 * monthOffset is relative to the reference "today" (0 = current month).
 * Returns the visible day cells plus the target month/year for labelling.
 */
export function getMonthGrid(today: Date, monthOffset: number): {
  days: Date[]
  monthIndex: number
  year: number
} {
  const target = new Date(today.getFullYear(), today.getMonth() + monthOffset, 1)
  const monthIndex = target.getMonth()
  const year = target.getFullYear()

  // Find the Monday on or before the 1st of the month.
  const first = new Date(year, monthIndex, 1)
  const dow = (first.getDay() + 6) % 7 // 0 = Monday … 6 = Sunday
  const gridStart = new Date(year, monthIndex, 1 - dow)

  // Render whole weeks until we pass the last day of the month.
  const days: Date[] = []
  const cursor = new Date(gridStart)
  // Always render at least enough weeks to cover the month; cap at 6 weeks.
  for (let i = 0; i < 42; i++) {
    days.push(new Date(cursor))
    cursor.setDate(cursor.getDate() + 1)
    // Stop after completing a week once we are past the month end.
    if (i >= 27 && (i + 1) % 7 === 0) {
      const last = days[days.length - 1]
      if (last.getMonth() !== monthIndex && last > new Date(year, monthIndex + 1, 0)) {
        break
      }
    }
  }
  return { days, monthIndex, year }
}

/** Month label like "Juni 2026" in the given locale. */
export function getMonthLabel(monthIndex: number, year: number, locale = 'de-DE'): string {
  return new Date(year, monthIndex, 1).toLocaleDateString(locale, {
    month: 'long',
    year: 'numeric',
  })
}

/** Short weekday headers (Mon-first) in the given locale. */
export function getWeekdayLabels(locale = 'de-DE'): string[] {
  // 2024-01-01 is a Monday — anchor for stable weekday names.
  return Array.from({ length: 7 }, (_, i) =>
    new Date(2024, 0, 1 + i).toLocaleDateString(locale, { weekday: 'short' })
  )
}

export function isSameMonth(d: Date, monthIndex: number): boolean {
  return d.getMonth() === monthIndex
}

export function isWeekend(d: Date): boolean {
  const day = d.getDay()
  return day === 0 || day === 6
}
