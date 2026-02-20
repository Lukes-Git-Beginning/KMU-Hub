/**
 * Shared currency & number formatting utilities.
 *
 * Default locale is de-CH (Swiss German) with CHF.
 * All formatters use Intl.NumberFormat for consistent output.
 */

const currencyFormatters = new Map<string, Intl.NumberFormat>()

function getFormatter(currency: string): Intl.NumberFormat {
  let fmt = currencyFormatters.get(currency)
  if (!fmt) {
    fmt = new Intl.NumberFormat('de-CH', {
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
 * @example formatCurrency(1234.5)        // "CHF 1'234.50"
 * @example formatCurrency(1234.5, 'EUR') // "EUR 1'234.50"
 * @example formatCurrency(null)          // "–"
 */
export function formatCurrency(
  amount: number | null | undefined,
  currency = 'CHF',
): string {
  if (amount == null) return '–'
  return getFormatter(currency).format(amount)
}

const amountFormatter = new Intl.NumberFormat('de-CH', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

/**
 * Format a number without currency symbol.
 * Useful when the symbol is rendered separately in JSX.
 *
 * @example formatAmount(1234.5) // "1'234.50"
 */
export function formatAmount(amount: number): string {
  return amountFormatter.format(amount)
}
