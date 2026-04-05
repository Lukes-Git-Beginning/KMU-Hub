/**
 * Locale-aware date/number/time formatting hooks.
 *
 * Replaces react-intl's FormattedDate, FormattedNumber, and intl.formatDate().
 * Uses native Intl API with locale from the Zustand store.
 */
import { useCallback } from 'react'
import { useLocale } from '@/hooks/useLocale'
import type { SupportedLocale } from '@/stores/locale'

// ---------------------------------------------------------------------------
// Named format presets (migrated from formats.ts)
// ---------------------------------------------------------------------------

const DATE_PRESETS: Record<SupportedLocale, Record<string, Intl.DateTimeFormatOptions>> = {
  de: {
    short: { day: '2-digit', month: '2-digit', year: 'numeric' },
    medium: { day: 'numeric', month: 'long', year: 'numeric' },
    long: { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' },
  },
  en: {
    short: { month: '2-digit', day: '2-digit', year: 'numeric' },
    medium: { month: 'long', day: 'numeric', year: 'numeric' },
    long: { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' },
  },
  fr: {
    short: { day: '2-digit', month: '2-digit', year: 'numeric' },
    medium: { day: 'numeric', month: 'long', year: 'numeric' },
    long: { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' },
  },
  it: {
    short: { day: '2-digit', month: '2-digit', year: 'numeric' },
    medium: { day: 'numeric', month: 'long', year: 'numeric' },
    long: { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' },
  },
}

const NUMBER_PRESETS: Record<SupportedLocale, Record<string, Intl.NumberFormatOptions>> = {
  de: {
    decimal: { style: 'decimal', minimumFractionDigits: 2, maximumFractionDigits: 2 },
    chf: { style: 'currency', currency: 'CHF', currencyDisplay: 'symbol' },
    eur: { style: 'currency', currency: 'EUR', currencyDisplay: 'symbol' },
  },
  en: {
    decimal: { style: 'decimal', minimumFractionDigits: 2, maximumFractionDigits: 2 },
    chf: { style: 'currency', currency: 'CHF', currencyDisplay: 'symbol' },
    eur: { style: 'currency', currency: 'EUR', currencyDisplay: 'symbol' },
  },
  fr: {
    decimal: { style: 'decimal', minimumFractionDigits: 2, maximumFractionDigits: 2 },
    chf: { style: 'currency', currency: 'CHF', currencyDisplay: 'symbol' },
    eur: { style: 'currency', currency: 'EUR', currencyDisplay: 'symbol' },
  },
  it: {
    decimal: { style: 'decimal', minimumFractionDigits: 2, maximumFractionDigits: 2 },
    chf: { style: 'currency', currency: 'CHF', currencyDisplay: 'symbol' },
    eur: { style: 'currency', currency: 'EUR', currencyDisplay: 'symbol' },
  },
}

const TIME_PRESETS: Record<SupportedLocale, Record<string, Intl.DateTimeFormatOptions>> = {
  de: {
    short: { hour: '2-digit', minute: '2-digit', hour12: false },
    medium: { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false },
  },
  en: {
    short: { hour: 'numeric', minute: '2-digit', hour12: true },
    medium: { hour: 'numeric', minute: '2-digit', second: '2-digit', hour12: true },
  },
  fr: {
    short: { hour: '2-digit', minute: '2-digit', hour12: false },
    medium: { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false },
  },
  it: {
    short: { hour: '2-digit', minute: '2-digit', hour12: false },
    medium: { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false },
  },
}

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------

/**
 * Returns a locale-aware date formatter.
 * Accepts either Intl.DateTimeFormatOptions or a named preset ("short", "medium", "long").
 */
export function useFormatDate() {
  const { locale } = useLocale()

  return useCallback(
    (value: Date | string | number, options?: Intl.DateTimeFormatOptions | string): string => {
      const date = value instanceof Date ? value : new Date(value)
      const opts = typeof options === 'string'
        ? DATE_PRESETS[locale]?.[options] ?? {}
        : options ?? {}
      return new Intl.DateTimeFormat(locale, opts).format(date)
    },
    [locale],
  )
}

/**
 * Returns a locale-aware number formatter.
 * Accepts either Intl.NumberFormatOptions or a named preset ("decimal", "chf", "eur").
 */
export function useFormatNumber() {
  const { locale } = useLocale()

  return useCallback(
    (value: number, options?: Intl.NumberFormatOptions | string): string => {
      const opts = typeof options === 'string'
        ? NUMBER_PRESETS[locale]?.[options] ?? {}
        : options ?? {}
      return new Intl.NumberFormat(locale, opts).format(value)
    },
    [locale],
  )
}

/**
 * Returns a locale-aware time formatter.
 * Accepts either Intl.DateTimeFormatOptions or a named preset ("short", "medium").
 */
export function useFormatTime() {
  const { locale } = useLocale()

  return useCallback(
    (value: Date | string | number, options?: Intl.DateTimeFormatOptions | string): string => {
      const date = value instanceof Date ? value : new Date(value)
      const opts = typeof options === 'string'
        ? TIME_PRESETS[locale]?.[options] ?? {}
        : options ?? {}
      return new Intl.DateTimeFormat(locale, opts).format(date)
    },
    [locale],
  )
}
