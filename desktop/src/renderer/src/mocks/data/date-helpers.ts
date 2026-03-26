/**
 * Date helpers for mock data — all dates relative to "today"
 * so screenshots always show current/relevant dates.
 */

const NOW = new Date()

export function today(): string {
  return formatDate(NOW)
}

export function now(): string {
  return NOW.toISOString()
}

export function daysFromNow(days: number): string {
  const d = new Date(NOW)
  d.setDate(d.getDate() + days)
  return formatDate(d)
}

export function daysAgo(days: number): string {
  return daysFromNow(-days)
}

export function hoursFromNow(hours: number): string {
  const d = new Date(NOW)
  d.setHours(d.getHours() + hours)
  return d.toISOString()
}

export function hoursAgo(hours: number): string {
  return hoursFromNow(-hours)
}

export function minutesAgo(minutes: number): string {
  const d = new Date(NOW)
  d.setMinutes(d.getMinutes() - minutes)
  return d.toISOString()
}

export function thisWeekday(dayOfWeek: number, hour = 9, minute = 0): string {
  const d = new Date(NOW)
  const currentDay = d.getDay()
  const diff = dayOfWeek - currentDay
  d.setDate(d.getDate() + diff)
  d.setHours(hour, minute, 0, 0)
  return d.toISOString()
}

export function nextWeekday(dayOfWeek: number, hour = 9, minute = 0): string {
  const d = new Date(NOW)
  const currentDay = d.getDay()
  let diff = dayOfWeek - currentDay
  if (diff <= 0) diff += 7
  d.setDate(d.getDate() + diff)
  d.setHours(hour, minute, 0, 0)
  return d.toISOString()
}

export function startOfMonth(): string {
  const d = new Date(NOW.getFullYear(), NOW.getMonth(), 1)
  return formatDate(d)
}

export function endOfMonth(): string {
  const d = new Date(NOW.getFullYear(), NOW.getMonth() + 1, 0)
  return formatDate(d)
}

export function monthsAgo(months: number): string {
  const d = new Date(NOW)
  d.setMonth(d.getMonth() - months)
  return formatDate(d)
}

function formatDate(d: Date): string {
  return d.toISOString().split('T')[0]
}
