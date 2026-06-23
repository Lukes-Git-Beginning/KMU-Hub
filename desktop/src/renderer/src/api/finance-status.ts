/**
 * Maps biz protojson enum-integer status values back to the string unions the
 * finance UI expects.
 *
 * The biz service serializes via `response.Proto` with `UseEnumNumbers`, so
 * `status` arrives as an integer (`2`) rather than the string the FE types
 * (`'sent'`). Status-badge lookups are keyed by the string, so an integer status
 * makes `STATUS_CONFIG[status]` undefined and crashes the list. Ordinals from
 * `proto/biz/v1/biz.pb.go` (index 0 = UNSPECIFIED). Already-string values (mock
 * mode) pass through untouched.
 */

// Index by enum ordinal; [0] is the UNSPECIFIED sentinel.
const INVOICE_STATUS = ['', 'draft', 'sent', 'paid', 'overdue', 'cancelled'] as const
const QUOTE_STATUS = ['', 'draft', 'sent', 'accepted', 'rejected', 'expired'] as const

function statusFromInt(value: unknown, table: readonly string[]): string {
  if (typeof value === 'number') return table[value] ?? String(value)
  return value as string
}

/** Return a copy of an invoice with its status coerced to the string union. */
export function withInvoiceStatus<T extends { status?: unknown }>(inv: T): T {
  if (!inv || typeof inv !== 'object') return inv
  return { ...inv, status: statusFromInt(inv.status, INVOICE_STATUS) }
}

/** Return a copy of a quote with its status coerced to the string union. */
export function withQuoteStatus<T extends { status?: unknown }>(quote: T): T {
  if (!quote || typeof quote !== 'object') return quote
  return { ...quote, status: statusFromInt(quote.status, QUOTE_STATUS) }
}
