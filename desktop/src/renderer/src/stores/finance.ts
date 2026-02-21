/**
 * Finance UI state store (local only).
 *
 * All server data is managed by TanStack Query hooks in api/hooks/useFinance.ts.
 * This store only holds ephemeral UI state: active tab, filters, date range.
 */
import { create } from 'zustand'
import type { InvoiceStatus, QuoteStatus } from '@/types/finance-types'

// ---------------------------------------------------------------------------
// Tab keys
// ---------------------------------------------------------------------------

export type FinanceTabKey =
  | 'dashboard'
  | 'invoices'
  | 'quotes'
  | 'credit-notes'
  | 'dunning'
  | 'belegkette'
  | 'banking'
  | 'export'

// ---------------------------------------------------------------------------
// Store interface
// ---------------------------------------------------------------------------

interface FinanceUIState {
  activeTab: FinanceTabKey
  setActiveTab: (tab: FinanceTabKey) => void

  selectedInvoiceId: string | null
  setSelectedInvoiceId: (id: string | null) => void

  dateRange: { from: string; to: string }
  setDateRange: (range: { from: string; to: string }) => void

  invoiceFilter: { status?: InvoiceStatus }
  setInvoiceFilter: (filter: { status?: InvoiceStatus }) => void

  quoteFilter: { status?: QuoteStatus }
  setQuoteFilter: (filter: { status?: QuoteStatus }) => void
}

// Default to current year range
const now = new Date()
const yearStart = `${now.getFullYear()}-01-01`
const yearEnd = `${now.getFullYear()}-12-31`

export const useFinanceUIStore = create<FinanceUIState>()((set) => ({
  activeTab: 'dashboard',
  setActiveTab: (tab) => set({ activeTab: tab }),

  selectedInvoiceId: null,
  setSelectedInvoiceId: (id) => set({ selectedInvoiceId: id }),

  dateRange: { from: yearStart, to: yearEnd },
  setDateRange: (range) => set({ dateRange: range }),

  invoiceFilter: {},
  setInvoiceFilter: (filter) => set({ invoiceFilter: filter }),

  quoteFilter: {},
  setQuoteFilter: (filter) => set({ quoteFilter: filter }),
}))

// ---------------------------------------------------------------------------
// Display-only calculation helpers (parse string decimals for UI)
// ---------------------------------------------------------------------------

/**
 * Format a number as EUR with de-DE locale.
 */
export function formatEUR(value: number | string): string {
  const n = typeof value === 'string' ? Number(value) : value
  if (isNaN(n)) return '\u20AC 0,00'
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
  }).format(n)
}

/**
 * Calculate line total from string quantities.
 * Display-only -- server is authoritative.
 */
export function calcLineTotal(
  quantity: string | number,
  unitPrice: string | number,
): number {
  const qty = typeof quantity === 'string' ? Number(quantity) : quantity
  const price = typeof unitPrice === 'string' ? Number(unitPrice) : unitPrice
  return Math.round(qty * price * 100) / 100
}

/**
 * Calculate subtotal (net) from line items with string values.
 */
export function calcInvoiceSubtotal(
  items: Array<{ quantity: string | number; unit_price: string | number }>,
): number {
  return items.reduce(
    (sum, item) => sum + calcLineTotal(item.quantity, item.unit_price),
    0,
  )
}

/**
 * Calculate tax for a single line item.
 */
export function calcLineTax(
  quantity: string | number,
  unitPrice: string | number,
  taxRate: string | number,
): number {
  const lineTotal = calcLineTotal(quantity, unitPrice)
  const rate = typeof taxRate === 'string' ? Number(taxRate) : taxRate
  return Math.round(lineTotal * (rate / 100) * 100) / 100
}

/**
 * Calculate total tax across all line items.
 */
export function calcInvoiceTax(
  items: Array<{
    quantity: string | number
    unit_price: string | number
    tax_rate: string | number
  }>,
): number {
  return items.reduce(
    (sum, item) => sum + calcLineTax(item.quantity, item.unit_price, item.tax_rate),
    0,
  )
}

/**
 * Calculate gross total (subtotal + tax).
 */
export function calcInvoiceTotal(
  items: Array<{
    quantity: string | number
    unit_price: string | number
    tax_rate: string | number
  }>,
): number {
  return calcInvoiceSubtotal(items) + calcInvoiceTax(items)
}

/**
 * Calculate remaining balance on an invoice given payments.
 */
export function calcRemainingAmount(
  grossTotal: string | number,
  payments: Array<{ amount: string | number }>,
): number {
  const total = typeof grossTotal === 'string' ? Number(grossTotal) : grossTotal
  const paid = payments.reduce((sum, p) => {
    const amt = typeof p.amount === 'string' ? Number(p.amount) : p.amount
    return sum + amt
  }, 0)
  return Math.round((total - paid) * 100) / 100
}
