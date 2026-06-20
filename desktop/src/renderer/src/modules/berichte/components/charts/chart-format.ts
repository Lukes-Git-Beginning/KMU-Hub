import type { ReportColumn } from '@/api/berichte-types'

const EUR = new Intl.NumberFormat('de-DE', {
  style: 'currency',
  currency: 'EUR',
  maximumFractionDigits: 0,
})
const NUM = new Intl.NumberFormat('de-DE', { maximumFractionDigits: 2 })
const NUM0 = new Intl.NumberFormat('de-DE', { maximumFractionDigits: 0 })

/** Format a single value for display, honouring the column/measure type. */
export function formatValue(value: unknown, type?: ReportColumn['type']): string {
  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'number') {
    switch (type) {
      case 'currency':
        return EUR.format(value)
      case 'percent':
        return `${NUM.format(value)} %`
      case 'number':
        return NUM.format(value)
      default:
        return NUM.format(value)
    }
  }
  return String(value)
}

/** Compact axis-tick formatter (e.g. 12.500 → "12,5 Tsd."). */
export function formatAxisTick(value: number, type?: ReportColumn['type']): string {
  if (type === 'percent') return `${NUM0.format(value)} %`
  const abs = Math.abs(value)
  if (abs >= 1_000_000) return `${NUM.format(value / 1_000_000)} Mio.`
  if (abs >= 1_000) return `${NUM.format(value / 1_000)} Tsd.`
  return NUM0.format(value)
}
