import i18next from 'i18next'

/** Format minutes as "Xh Ym" */
export function formatMinutes(minutes: number): string {
  if (minutes < 0) return '0h 0m'
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  return `${h}h ${m}m`
}

/** Format minutes as decimal hours, e.g. 510 → "8.5" */
export function formatHoursDecimal(minutes: number): string {
  return (minutes / 60).toFixed(1)
}

/** Get today's date as YYYY-MM-DD */
export function todayStr(): string {
  return new Date().toISOString().split('T')[0]
}

/** Check if a YYYY-MM-DD string is today */
export function isToday(dateStr: string): boolean {
  return dateStr === todayStr()
}

/** Get the Monday–Sunday dates for a given week offset (0 = this week) */
export function getWeekDates(weekOffset: number): Date[] {
  const now = new Date()
  const day = now.getDay()
  // Monday = 1, so offset from Monday
  const mondayOffset = day === 0 ? -6 : 1 - day
  const monday = new Date(now)
  monday.setDate(now.getDate() + mondayOffset + weekOffset * 7)
  monday.setHours(0, 0, 0, 0)

  return Array.from({ length: 7 }, (_, i) => {
    const d = new Date(monday)
    d.setDate(monday.getDate() + i)
    return d
  })
}

/** Get all days in a month for a given month offset (0 = this month) */
export function getMonthDates(monthOffset: number): Date[] {
  const now = new Date()
  const year = now.getFullYear()
  const month = now.getMonth() + monthOffset
  const firstDay = new Date(year, month, 1)
  const lastDay = new Date(year, month + 1, 0)

  return Array.from({ length: lastDay.getDate() }, (_, i) => {
    return new Date(firstDay.getFullYear(), firstDay.getMonth(), i + 1)
  })
}

/** Get ISO week number for a date */
export function getWeekNumber(date: Date): number {
  const d = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()))
  const dayNum = d.getUTCDay() || 7
  d.setUTCDate(d.getUTCDate() + 4 - dayNum)
  const yearStart = new Date(Date.UTC(d.getUTCFullYear(), 0, 1))
  return Math.ceil(((d.getTime() - yearStart.getTime()) / 86400000 + 1) / 7)
}

/** Calculate minutes between two HH:mm strings */
export function minutesBetween(start: string, end: string): number {
  const [sh, sm] = start.split(':').map(Number)
  const [eh, em] = end.split(':').map(Number)
  return (eh * 60 + em) - (sh * 60 + sm)
}

/** Format a Date to YYYY-MM-DD */
export function dateToStr(date: Date): string {
  return date.toISOString().split('T')[0]
}

/** Format date range label, e.g. "10. - 16. Feb 2026" */
export function getDateRangeLabel(dates: Date[]): string {
  if (dates.length === 0) return ''
  const first = dates[0]
  const last = dates[dates.length - 1]
  const months = [
    i18next.t('profil.zeiterfassung.monthShort.jan'),
    i18next.t('profil.zeiterfassung.monthShort.feb'),
    i18next.t('profil.zeiterfassung.monthShort.mar'),
    i18next.t('profil.zeiterfassung.monthShort.apr'),
    i18next.t('profil.zeiterfassung.monthShort.may'),
    i18next.t('profil.zeiterfassung.monthShort.jun'),
    i18next.t('profil.zeiterfassung.monthShort.jul'),
    i18next.t('profil.zeiterfassung.monthShort.aug'),
    i18next.t('profil.zeiterfassung.monthShort.sep'),
    i18next.t('profil.zeiterfassung.monthShort.oct'),
    i18next.t('profil.zeiterfassung.monthShort.nov'),
    i18next.t('profil.zeiterfassung.monthShort.dec'),
  ]

  if (first.getMonth() === last.getMonth()) {
    return `${first.getDate()}. - ${last.getDate()}. ${months[first.getMonth()]} ${first.getFullYear()}`
  }
  return `${first.getDate()}. ${months[first.getMonth()]} - ${last.getDate()}. ${months[last.getMonth()]} ${last.getFullYear()}`
}

/** Format a Date to a readable German date, e.g. "Mo, 10. Feb" */
export function formatDateShort(date: Date): string {
  const days = [
    i18next.t('profil.zeiterfassung.dayShort.sun'),
    i18next.t('profil.zeiterfassung.dayShort.mon'),
    i18next.t('profil.zeiterfassung.dayShort.tue'),
    i18next.t('profil.zeiterfassung.dayShort.wed'),
    i18next.t('profil.zeiterfassung.dayShort.thu'),
    i18next.t('profil.zeiterfassung.dayShort.fri'),
    i18next.t('profil.zeiterfassung.dayShort.sat'),
  ]
  const months = [
    i18next.t('profil.zeiterfassung.monthShort.jan'),
    i18next.t('profil.zeiterfassung.monthShort.feb'),
    i18next.t('profil.zeiterfassung.monthShort.mar'),
    i18next.t('profil.zeiterfassung.monthShort.apr'),
    i18next.t('profil.zeiterfassung.monthShort.may'),
    i18next.t('profil.zeiterfassung.monthShort.jun'),
    i18next.t('profil.zeiterfassung.monthShort.jul'),
    i18next.t('profil.zeiterfassung.monthShort.aug'),
    i18next.t('profil.zeiterfassung.monthShort.sep'),
    i18next.t('profil.zeiterfassung.monthShort.oct'),
    i18next.t('profil.zeiterfassung.monthShort.nov'),
    i18next.t('profil.zeiterfassung.monthShort.dec'),
  ]
  return `${days[date.getDay()]}, ${date.getDate()}. ${months[date.getMonth()]}`
}

/** Get month label, e.g. "Februar 2026" */
export function getMonthLabel(monthOffset: number): string {
  const now = new Date()
  const d = new Date(now.getFullYear(), now.getMonth() + monthOffset, 1)
  const months = [
    i18next.t('profil.zeiterfassung.month.january'),
    i18next.t('profil.zeiterfassung.month.february'),
    i18next.t('profil.zeiterfassung.month.march'),
    i18next.t('profil.zeiterfassung.month.april'),
    i18next.t('profil.zeiterfassung.month.may'),
    i18next.t('profil.zeiterfassung.month.june'),
    i18next.t('profil.zeiterfassung.month.july'),
    i18next.t('profil.zeiterfassung.month.august'),
    i18next.t('profil.zeiterfassung.month.september'),
    i18next.t('profil.zeiterfassung.month.october'),
    i18next.t('profil.zeiterfassung.month.november'),
    i18next.t('profil.zeiterfassung.month.december'),
  ]
  return `${months[d.getMonth()]} ${d.getFullYear()}`
}

import type { TimeEntry } from '@/stores/timetracking'

/** Group entries by date string */
export function groupEntriesByDate(entries: TimeEntry[]): Record<string, TimeEntry[]> {
  const groups: Record<string, TimeEntry[]> = {}
  for (const entry of entries) {
    if (!groups[entry.date]) groups[entry.date] = []
    groups[entry.date].push(entry)
  }
  return groups
}

/** Group entries by categoryId */
export function groupEntriesByCategory(entries: TimeEntry[]): Record<string, TimeEntry[]> {
  const groups: Record<string, TimeEntry[]> = {}
  for (const entry of entries) {
    if (!groups[entry.categoryId]) groups[entry.categoryId] = []
    groups[entry.categoryId].push(entry)
  }
  return groups
}
