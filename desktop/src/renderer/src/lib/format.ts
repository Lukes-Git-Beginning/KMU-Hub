/**
 * Shared currency, number & date formatting utilities.
 *
 * Currency formatters default to de-DE (EUR accounting convention is locale-independent).
 * Date formatters read the active i18n locale so Cosmi stays CH/AT/INT-compatible.
 * All formatters use Intl for consistent output.
 */
import i18next from 'i18next'

// ---------------------------------------------------------------------------
// Date / Time
// ---------------------------------------------------------------------------

/**
 * Format a date using the current i18n locale.
 * @param value ISO string, Date, or number (timestamp)
 * @param options Intl.DateTimeFormatOptions (default: dateStyle 'medium')
 */
export function formatDate(
  value: string | Date | number,
  options: Intl.DateTimeFormatOptions = { dateStyle: 'medium' },
): string {
  const date = value instanceof Date ? value : new Date(value)
  if (isNaN(date.getTime())) return '—'
  const locale = i18next.language || 'de-DE'
  return new Intl.DateTimeFormat(locale, options).format(date)
}

/**
 * Format date + time using the current i18n locale.
 * @param value ISO string, Date, or number (timestamp)
 * @param options Intl.DateTimeFormatOptions (default: dateStyle 'medium', timeStyle 'short')
 */
export function formatDateTime(
  value: string | Date | number,
  options: Intl.DateTimeFormatOptions = { dateStyle: 'medium', timeStyle: 'short' },
): string {
  const date = value instanceof Date ? value : new Date(value)
  if (isNaN(date.getTime())) return '—'
  const locale = i18next.language || 'de-DE'
  return new Intl.DateTimeFormat(locale, options).format(date)
}

/**
 * Format time-only using the current i18n locale.
 * @param value ISO string, Date, or number (timestamp)
 * @param options Intl.DateTimeFormatOptions (default: timeStyle 'short')
 */
export function formatTime(
  value: string | Date | number,
  options: Intl.DateTimeFormatOptions = { timeStyle: 'short' },
): string {
  const date = value instanceof Date ? value : new Date(value)
  if (isNaN(date.getTime())) return '—'
  const locale = i18next.language || 'de-DE'
  return new Intl.DateTimeFormat(locale, options).format(date)
}

// ---------------------------------------------------------------------------
// Currency & Numbers (de-DE intentional — EUR accounting convention)
// ---------------------------------------------------------------------------

const currencyFormatters = new Map<string, Intl.NumberFormat>()

function getFormatter(currency: string): Intl.NumberFormat {
  let fmt = currencyFormatters.get(currency)
  if (!fmt) {
    fmt = new Intl.NumberFormat('de-DE', {
      style: 'currency',
      currency,
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    })
    currencyFormatters.set(currency, fmt)
  }
  return fmt
}

/**
 * Format a number as currency with symbol.
 *
 * @example formatCurrency(1234.5)        // "1.234,50 €"
 * @example formatCurrency(1234.5, 'CHF') // "1.234,50 CHF"
 * @example formatCurrency(null)          // "–"
 */
export function formatCurrency(
  amount: number | null | undefined,
  currency = 'EUR',
): string {
  if (amount == null) return '–'
  return getFormatter(currency).format(amount)
}

const amountFormatter = new Intl.NumberFormat('de-DE', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

/**
 * Format a number without currency symbol.
 * Useful when the symbol is rendered separately in JSX.
 *
 * @example formatAmount(1234.5) // "1.234,50"
 */
export function formatAmount(amount: number): string {
  return amountFormatter.format(amount)
}

/**
 * Format an ISO timestamp as a relative time string.
 * Returns locale-aware strings like "gerade eben", "vor 2 Std.", "gestern", "vor 3 Tagen".
 *
 * @param iso  ISO 8601 date string or null
 * @param locale  BCP 47 locale tag (e.g. "de", "en", "fr", "it")
 */
export function formatRelativeTime(iso: string | null, locale: string): string {
  if (!iso) {
    return locale.startsWith('de') ? 'noch nie genutzt'
      : locale.startsWith('fr') ? 'jamais utilisé'
      : locale.startsWith('it') ? 'mai usato'
      : 'never used'
  }

  const diffMs = Date.now() - new Date(iso).getTime()
  const diffMin = Math.floor(diffMs / 60_000)
  const diffH   = Math.floor(diffMs / 3_600_000)
  const diffD   = Math.floor(diffMs / 86_400_000)
  const diffW   = Math.floor(diffD / 7)
  const diffMo  = Math.floor(diffD / 30)

  if (locale.startsWith('de')) {
    if (diffMin < 2)  return 'gerade eben'
    if (diffH   < 1)  return `vor ${diffMin} Min.`
    if (diffD   < 1)  return `vor ${diffH} Std.`
    if (diffD   === 1) return 'gestern'
    if (diffW   < 1)  return `vor ${diffD} Tagen`
    if (diffMo  < 1)  return `vor ${diffW} Wochen`
    return `vor ${diffMo} Monaten`
  }
  if (locale.startsWith('fr')) {
    if (diffMin < 2)  return 'à l\'instant'
    if (diffH   < 1)  return `il y a ${diffMin} min`
    if (diffD   < 1)  return `il y a ${diffH} h`
    if (diffD   === 1) return 'hier'
    if (diffW   < 1)  return `il y a ${diffD} jours`
    if (diffMo  < 1)  return `il y a ${diffW} semaines`
    return `il y a ${diffMo} mois`
  }
  if (locale.startsWith('it')) {
    if (diffMin < 2)  return 'poco fa'
    if (diffH   < 1)  return `${diffMin} min fa`
    if (diffD   < 1)  return `${diffH} ore fa`
    if (diffD   === 1) return 'ieri'
    if (diffW   < 1)  return `${diffD} giorni fa`
    if (diffMo  < 1)  return `${diffW} settimane fa`
    return `${diffMo} mesi fa`
  }
  // English fallback
  if (diffMin < 2)  return 'just now'
  if (diffH   < 1)  return `${diffMin} min ago`
  if (diffD   < 1)  return `${diffH}h ago`
  if (diffD   === 1) return 'yesterday'
  if (diffW   < 1)  return `${diffD} days ago`
  if (diffMo  < 1)  return `${diffW} weeks ago`
  return `${diffMo} months ago`
}
