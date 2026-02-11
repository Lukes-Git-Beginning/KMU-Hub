/**
 * Locale-specific format configurations for react-intl.
 *
 * Provides date, number, and currency formatting presets per locale.
 * These are passed to IntlProvider's `formats` prop so components can
 * reference named formats: <FormattedDate value={d} format="short" />
 */
import type { CustomFormats } from 'react-intl'
import type { SupportedLocale } from '@/stores/locale'

// ---------------------------------------------------------------------------
// Format definitions per locale
// ---------------------------------------------------------------------------

const dateFormats: Record<SupportedLocale, CustomFormats['date']> = {
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

const numberFormats: Record<SupportedLocale, CustomFormats['number']> = {
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

const timeFormats: Record<SupportedLocale, CustomFormats['time']> = {
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
// Public API
// ---------------------------------------------------------------------------

/**
 * Returns the react-intl CustomFormats object for the given locale.
 * These formats can be referenced by name in FormattedDate, FormattedNumber, etc.
 */
export function getFormats(locale: SupportedLocale): CustomFormats {
  return {
    date: dateFormats[locale],
    number: numberFormats[locale],
    time: timeFormats[locale],
  }
}
