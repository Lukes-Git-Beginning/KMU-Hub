/**
 * Shared Rapporte display maps + helpers.
 *
 * Extracted from RapportePage so the detail modal and the page render
 * weather/status badges identically.
 */
import { Sun, Cloud, CloudRain, Snowflake } from 'lucide-react'
import type { FieldReport, WeatherType } from '@/stores/rapporte'

export const weatherIcons: Record<WeatherType, typeof Sun> = {
  sunny: Sun,
  cloudy: Cloud,
  rainy: CloudRain,
  snowy: Snowflake,
}

export const weatherLabelKeys: Record<WeatherType, string> = {
  sunny: 'rapporte.weather.sunny',
  cloudy: 'rapporte.weather.cloudy',
  rainy: 'rapporte.weather.rainy',
  snowy: 'rapporte.weather.snowy',
}

export const projectColors: Record<string, string> = {
  'prj-1': 'bg-info-light text-info',
  'prj-2': 'bg-warning-light text-warning',
  'prj-3': 'bg-success-light text-success',
  'prj-4': 'bg-error-light text-error',
  'prj-5': 'bg-primary-light text-primary',
  'prj-6': 'bg-info-light text-info',
  'prj-7': 'bg-warning-light text-warning',
  'prj-8': 'bg-error-light text-error',
}

export const approvalBadgeStyles: Record<FieldReport['approvalStatus'], string> = {
  draft: 'bg-secondary text-muted-foreground',
  submitted: 'bg-info-light text-info',
  approved: 'bg-success-light text-success',
  rejected: 'bg-error-light text-error',
}

export const approvalLabelKeys: Record<FieldReport['approvalStatus'], string> = {
  draft: 'rapporte.approval.draft',
  submitted: 'rapporte.approval.submitted',
  approved: 'rapporte.approval.approved',
  rejected: 'rapporte.approval.rejected',
}

export function formatDate(dateStr: string): string {
  return new Date(dateStr + 'T00:00:00').toLocaleDateString('de-DE', {
    weekday: 'short',
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

export function calcNetHours(start: string, end: string, breakMin: number): string {
  const [sh, sm] = start.split(':').map(Number)
  const [eh, em] = end.split(':').map(Number)
  const totalMin = (eh * 60 + em) - (sh * 60 + sm) - breakMin
  const h = Math.floor(totalMin / 60)
  const m = totalMin % 60
  return `${h}h ${m > 0 ? m + 'min' : ''}`
}

export function calcNetMinutes(start: string, end: string, breakMin: number): number {
  const [sh, sm] = start.split(':').map(Number)
  const [eh, em] = end.split(':').map(Number)
  return (eh * 60 + em) - (sh * 60 + sm) - breakMin
}
