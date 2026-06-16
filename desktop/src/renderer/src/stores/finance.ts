/**
 * Finance UI state store (local only).
 *
 * All server data is managed by TanStack Query hooks in api/hooks/useFinance.ts.
 * This store only holds ephemeral UI state: active tab, filters, date range.
 *
 * Mock-Store-Typen fuer BuchhaltungPage.tsx (Legacy-Modul, nicht im Router).
 * Wenn Backend-Endpoints kommen: useFinanceStore loeschen, auf TanStack Query umstellen.
 */
import { create } from 'zustand'
import type { InvoiceStatus, QuoteStatus } from '@/types/finance-types'

// ---------------------------------------------------------------------------
// Tab keys
// ---------------------------------------------------------------------------

export type FinanceTabKey =
  | 'dashboard'
  | 'stammdaten'
  | 'invoices'
  | 'quotes'
  | 'recurring'
  | 'open-items'
  | 'credit-notes'
  | 'expenses'
  | 'transactions'
  | 'berichte'
  | 'dunning'
  | 'belegkette'
  | 'banking'
  | 'export'
  | 'finanz-integrationen'
  | 'settings'

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
 * Format a number as a currency amount with de-DE locale.
 * Defaults to EUR; pass 'CHF'/'USD' for foreign-currency documents.
 */
export function formatMoney(
  value: number | string,
  currency: string = 'EUR',
): string {
  const n = typeof value === 'string' ? Number(value) : value
  const cur = currency || 'EUR'
  if (isNaN(n)) {
    return new Intl.NumberFormat('de-DE', { style: 'currency', currency: cur }).format(0)
  }
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: cur,
  }).format(n)
}

/**
 * Format a number as EUR with de-DE locale.
 * Thin wrapper around formatMoney for EUR-only contexts.
 */
export function formatEUR(value: number | string): string {
  return formatMoney(value, 'EUR')
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
 *
 * Akzeptiert sowohl snake_case-API-Items (unit_price, tax_rate) als auch
 * camelCase-Mock-LineItems (unitPrice, vatRate).
 */
export function calcInvoiceTotal(
  items: Array<
    | { quantity: string | number; unit_price: string | number; tax_rate: string | number }
    | { quantity: string | number; unitPrice: string | number; vatRate: string | number; discount?: number }
  >,
): number {
  return items.reduce((sum, item) => {
    if ('unit_price' in item) {
      // snake_case API format
      return sum + calcLineTotal(item.quantity, item.unit_price) + calcLineTax(item.quantity, item.unit_price, item.tax_rate)
    }
    // camelCase LineItem format
    const qty = Number(item.quantity)
    const price = Number(item.unitPrice)
    const vat = Number(item.vatRate)
    const discount = item.discount ?? 0
    const lineNet = qty * price * (1 - discount / 100)
    const lineTax = lineNet * (vat / 100)
    return sum + lineNet + lineTax
  }, 0)
}

/**
 * Calculate remaining balance on an invoice.
 *
 * Overload 1: calcRemainingAmount(invoice) — akzeptiert ein Invoice-Objekt.
 * Overload 2: calcRemainingAmount(grossTotal, payments) — legacy-Signatur.
 */
export function calcRemainingAmount(invoice: Invoice): number
export function calcRemainingAmount(grossTotal: string | number, payments: Array<{ amount: string | number }>): number
export function calcRemainingAmount(
  invoiceOrTotal: Invoice | string | number,
  payments?: Array<{ amount: string | number }>,
): number {
  if (typeof invoiceOrTotal === 'object') {
    const total = calcInvoiceTotalFromInvoice(invoiceOrTotal)
    const paid = calcPaidAmount(invoiceOrTotal)
    return Math.round((total - paid) * 100) / 100
  }
  const total = typeof invoiceOrTotal === 'string' ? Number(invoiceOrTotal) : invoiceOrTotal
  const paid = (payments ?? []).reduce((sum, p) => {
    const amt = typeof p.amount === 'string' ? Number(p.amount) : p.amount
    return sum + amt
  }, 0)
  return Math.round((total - paid) * 100) / 100
}

// ---------------------------------------------------------------------------
// Mock-Store fuer BuchhaltungPage.tsx (Legacy-Modul — nicht im Router).
//
// Diese Typen und der Store wurden aus dem alten `buchhaltung`-Modul
// extrahiert. Das Modul wird sowieso nicht ueber den Router angesprochen;
// der Store muss nur compilieren, damit der TypeScript-Build sauber bleibt.
//
// Wenn Backend-Endpoints kommen: useFinanceStore loeschen, auf TanStack
// Query-Hooks umstellen (useInvoices, useExpenses, useTransactions, ...).
// ---------------------------------------------------------------------------

// --- LineItem ---------------------------------------------------------------

export interface LineItem {
  id: string
  description: string
  quantity: number
  unitPrice: number
  vatRate: number
  discount: number
}

// --- Invoice / Quote --------------------------------------------------------

export type InvoiceMockStatus = 'draft' | 'sent' | 'paid' | 'overdue' | 'cancelled'

export interface Payment {
  id: string
  date: string
  amount: number
  method: string
  reference?: string
}

export interface Invoice {
  id: string
  number: string
  type: 'invoice' | 'quote'
  client: string
  clientEmail?: string
  date: string
  dueDate: string
  status: InvoiceMockStatus
  paymentTerms: string
  notes?: string
  items: LineItem[]
  payments: Payment[]
}

// --- Dunning ----------------------------------------------------------------

export interface Dunning {
  id: string
  invoiceNumber: string
  client: string
  dueAmount: number
  level: 1 | 2 | 3
  status: 'pending' | 'sent' | 'reminded' | 'paid' | 'inkasso'
  sentAt: string
}

// ---------------------------------------------------------------------------
// calcInvoiceTotal / calcRemainingAmount — Invoice-Objekt-Overloads
// ---------------------------------------------------------------------------

/**
 * Berechnet den Bruttobetrag einer Rechnung aus ihren LineItems (camelCase).
 * Overload: akzeptiert auch ein Invoice-Objekt direkt.
 */
export function calcInvoiceTotalFromInvoice(invoice: Invoice): number {
  return invoice.items.reduce((sum, item) => {
    const lineNet = item.quantity * item.unitPrice * (1 - (item.discount ?? 0) / 100)
    const lineTax = lineNet * (item.vatRate / 100)
    return sum + lineNet + lineTax
  }, 0)
}

/**
 * Bereits bezahlter Betrag (Summe aller Payments).
 */
export function calcPaidAmount(invoice: Invoice): number {
  return invoice.payments.reduce((sum, p) => sum + p.amount, 0)
}

// Patch: calcRemainingAmount neu — akzeptiert entweder ein Invoice-Objekt
// (1 Argument) oder (grossTotal, payments[]) (2 Argumente).
// Die originale Implementierung unten wird durch diese ersetzt.

export interface Expense {
  id: string
  description: string
  amount: number
  date: string
  category: string
  supplier: string
  project?: string
  receipt?: boolean
  /** Dateiname des angehängten Belegs (für GoBD-Archiv / DATEV-Beleg-ZIP, P3/P4). */
  receiptName?: string
  /** SKR03/04-Sachkonto für den DATEV-Export (leichtes Mapping, kein Buchungssatz). */
  account?: string
  status: 'pending' | 'approved' | 'rejected'
}

export interface Transaction {
  id: string
  type: 'income' | 'expense'
  description: string
  amount: number
  date: string
  category: string
  status: 'completed' | 'pending'
  reference?: string
  invoiceId?: string
}

interface FinanceMockState {
  // --- State ---
  invoices: Invoice[]
  dunnings: Dunning[]
  expenses: Expense[]
  transactions: Transaction[]
  nextInvoiceNum: number

  // --- Invoice Actions ---
  addInvoice: (invoice: Omit<Invoice, 'id' | 'payments'>) => void
  updateInvoice: (id: string, updates: Partial<Omit<Invoice, 'id' | 'payments'>>) => void
  deleteInvoice: (id: string) => void
  duplicateInvoice: (id: string) => void
  sendInvoice: (id: string) => void
  cancelInvoice: (id: string) => void
  recordPayment: (invoiceId: string, payment: Omit<Payment, 'id'>) => void

  // --- Dunning Actions ---
  sendDunning: (id: string) => void
  escalateDunning: (id: string) => void

  // --- Expense Actions ---
  addExpense: (expense: Omit<Expense, 'id'>) => void
  approveExpense: (id: string) => void
  rejectExpense: (id: string) => void
  deleteExpense: (id: string) => void

  // --- Transaction Actions ---
  deleteTransaction: (id: string) => void
}

const today = new Date()
const daysAgo = (n: number): string => {
  const d = new Date(today)
  d.setDate(d.getDate() - n)
  return d.toISOString().slice(0, 10)
}
const daysFromNow = (n: number): string => {
  const d = new Date(today)
  d.setDate(d.getDate() + n)
  return d.toISOString().slice(0, 10)
}

const SEED_INVOICES: Invoice[] = [
  {
    id: 'inv-1',
    number: 'INV-2026-001',
    type: 'invoice',
    client: 'Müller GmbH',
    clientEmail: 'buchhaltung@mueller-gmbh.de',
    date: daysAgo(30),
    dueDate: daysAgo(0),
    status: 'sent',
    paymentTerms: '30 Tage netto',
    items: [
      { id: 'li-1', description: 'CRM-Beratung Mai', quantity: 8, unitPrice: 135, vatRate: 19, discount: 0 },
      { id: 'li-2', description: 'Einrichtung Cosmi', quantity: 4, unitPrice: 160, vatRate: 19, discount: 0 },
    ],
    payments: [],
  },
  {
    id: 'inv-2',
    number: 'INV-2026-002',
    type: 'invoice',
    client: 'Schneider & Partner',
    clientEmail: 'office@schneider-partner.de',
    date: daysAgo(60),
    dueDate: daysAgo(30),
    status: 'overdue',
    paymentTerms: '30 Tage netto',
    items: [
      { id: 'li-3', description: 'Monatliche Lizenz COSMI Pro', quantity: 1, unitPrice: 290, vatRate: 19, discount: 0 },
    ],
    payments: [],
  },
  {
    id: 'inv-3',
    number: 'INV-2026-003',
    type: 'invoice',
    client: 'Bäckerei Koch',
    date: daysAgo(10),
    dueDate: daysFromNow(20),
    status: 'draft',
    paymentTerms: '14 Tage netto',
    items: [
      { id: 'li-4', description: 'Onsite-Prozessanalyse', quantity: 2, unitPrice: 900, vatRate: 19, discount: 0 },
    ],
    payments: [],
  },
  {
    id: 'inv-4',
    number: 'INV-2026-004',
    type: 'invoice',
    client: 'TechVenture AG',
    clientEmail: 'finance@techventure.ag',
    date: daysAgo(45),
    dueDate: daysAgo(15),
    status: 'paid',
    paymentTerms: '30 Tage netto',
    items: [
      { id: 'li-5', description: 'API-Integration Bexio', quantity: 12, unitPrice: 110, vatRate: 19, discount: 0 },
    ],
    payments: [{ id: 'pay-1', date: daysAgo(14), amount: 1571.6, method: 'Banküberweisung' }],
  },
  {
    id: 'inv-5',
    number: 'QUO-2026-001',
    type: 'quote',
    client: 'Handwerk Fischer',
    clientEmail: 'info@handwerk-fischer.de',
    date: daysAgo(5),
    dueDate: daysFromNow(25),
    status: 'draft',
    paymentTerms: '14 Tage netto',
    notes: 'Angebot gültig 30 Tage',
    items: [
      { id: 'li-6', description: 'Cosmi Starter Paket inkl. Onboarding', quantity: 1, unitPrice: 1490, vatRate: 19, discount: 5 },
    ],
    payments: [],
  },
]

const SEED_DUNNINGS: Dunning[] = [
  {
    id: 'dun-1',
    invoiceNumber: 'INV-2026-002',
    client: 'Schneider & Partner',
    dueAmount: 345.1,
    level: 1,
    status: 'sent',
    sentAt: daysAgo(10),
  },
  {
    id: 'dun-2',
    invoiceNumber: 'INV-2025-089',
    client: 'Gastro Ritter KG',
    dueAmount: 890,
    level: 2,
    status: 'pending',
    sentAt: daysAgo(25),
  },
  {
    id: 'dun-3',
    invoiceNumber: 'INV-2025-071',
    client: 'Handels GmbH Weiß',
    dueAmount: 2340,
    level: 3,
    status: 'inkasso',
    sentAt: daysAgo(60),
  },
]

const SEED_EXPENSES: Expense[] = [
  { id: 'exp-1', description: 'Bürobedarf Q2', amount: 234.5, date: daysAgo(3), category: 'Büromaterial', supplier: 'Staples', status: 'pending' },
  { id: 'exp-2', description: 'Server-Hosting Hetzner', amount: 89.0, date: daysAgo(7), category: 'IT-Infrastruktur', supplier: 'Hetzner Online GmbH', status: 'approved' },
  { id: 'exp-3', description: 'Geschäftsessen Kunde Müller', amount: 142.8, date: daysAgo(12), category: 'Bewirtung', supplier: 'Restaurant Adler', project: 'Projekt Müller-CRM', status: 'pending' },
  { id: 'exp-4', description: 'Software-Lizenz JetBrains', amount: 199.0, date: daysAgo(18), category: 'Software', supplier: 'JetBrains s.r.o.', status: 'approved' },
  { id: 'exp-5', description: 'Werbeanzeigen Q2', amount: 850.0, date: daysAgo(25), category: 'Marketing', supplier: 'Google Ireland Ltd.', status: 'rejected' },
  { id: 'exp-6', description: 'Bahnfahrt Berlin-Hamburg', amount: 78.4, date: daysAgo(34), category: 'Reisekosten', supplier: 'Deutsche Bahn AG', project: 'Kundentermin DB', status: 'approved' },
  { id: 'exp-7', description: 'Fortbildung Excel-Schulung', amount: 320.0, date: daysAgo(48), category: 'Weiterbildung', supplier: 'Akademie Online GmbH', status: 'approved' },
]

const SEED_TRANSACTIONS: Transaction[] = [
  { id: 'tx-1', type: 'income', description: 'Rechnung 2026-042 Müller GmbH', amount: 4280.0, date: daysAgo(2), category: 'Umsatzerlöse', status: 'completed', reference: 'RE-2026-042' },
  { id: 'tx-2', type: 'expense', description: 'Bürobedarf Staples', amount: 234.5, date: daysAgo(3), category: 'Büromaterial', status: 'completed' },
  { id: 'tx-3', type: 'income', description: 'Anzahlung Projekt Schneider', amount: 1500.0, date: daysAgo(5), category: 'Umsatzerlöse', status: 'completed', reference: 'RE-2026-041' },
  { id: 'tx-4', type: 'expense', description: 'Server-Hosting Hetzner', amount: 89.0, date: daysAgo(7), category: 'IT-Infrastruktur', status: 'completed' },
  { id: 'tx-5', type: 'income', description: 'Beratungsleistung Mai', amount: 2400.0, date: daysAgo(10), category: 'Umsatzerlöse', status: 'completed', reference: 'RE-2026-040' },
  { id: 'tx-6', type: 'expense', description: 'Geschäftsessen Adler', amount: 142.8, date: daysAgo(12), category: 'Bewirtung', status: 'completed' },
  { id: 'tx-7', type: 'income', description: 'Wartungsvertrag Q2', amount: 890.0, date: daysAgo(15), category: 'Umsatzerlöse', status: 'completed', reference: 'RE-2026-039' },
  { id: 'tx-8', type: 'expense', description: 'JetBrains-Lizenzen', amount: 199.0, date: daysAgo(18), category: 'Software', status: 'completed' },
  { id: 'tx-9', type: 'income', description: 'Schulung WebApp-Entwicklung', amount: 3200.0, date: daysAgo(22), category: 'Umsatzerlöse', status: 'completed', reference: 'RE-2026-038' },
  { id: 'tx-10', type: 'expense', description: 'Google Ads Q2', amount: 850.0, date: daysAgo(25), category: 'Marketing', status: 'completed' },
  { id: 'tx-11', type: 'income', description: 'Lizenzgebühr Tool-X', amount: 1200.0, date: daysAgo(30), category: 'Lizenzeinnahmen', status: 'completed', reference: 'RE-2026-037' },
  { id: 'tx-12', type: 'expense', description: 'Bahnfahrt DB', amount: 78.4, date: daysAgo(34), category: 'Reisekosten', status: 'completed' },
  { id: 'tx-13', type: 'income', description: 'Beratung Schmidt-Projekt', amount: 1750.0, date: daysAgo(40), category: 'Umsatzerlöse', status: 'completed', reference: 'RE-2026-036' },
  { id: 'tx-14', type: 'expense', description: 'Fortbildung Excel', amount: 320.0, date: daysAgo(48), category: 'Weiterbildung', status: 'completed' },
]

export const useFinanceStore = create<FinanceMockState>()((set) => ({
  // --- Initial State ---
  invoices: SEED_INVOICES,
  dunnings: SEED_DUNNINGS,
  expenses: SEED_EXPENSES,
  transactions: SEED_TRANSACTIONS,
  nextInvoiceNum: SEED_INVOICES.filter((i) => i.type === 'invoice').length,

  // --- Invoice Actions ---
  addInvoice: (invoice) =>
    set((state) => ({
      invoices: [
        ...state.invoices,
        { ...invoice, id: `inv-${Date.now()}`, payments: [] },
      ],
      nextInvoiceNum: state.nextInvoiceNum + 1,
    })),

  updateInvoice: (id, updates) =>
    set((state) => ({
      invoices: state.invoices.map((inv) => (inv.id === id ? { ...inv, ...updates } : inv)),
    })),

  deleteInvoice: (id) =>
    set((state) => ({ invoices: state.invoices.filter((inv) => inv.id !== id) })),

  duplicateInvoice: (id) =>
    set((state) => {
      const original = state.invoices.find((inv) => inv.id === id)
      if (!original) return {}
      const prefix = original.type === 'quote' ? 'QUO' : 'INV'
      const newNum = `${prefix}-COPY-${String(Date.now()).slice(-4)}`
      return {
        invoices: [
          ...state.invoices,
          { ...original, id: `inv-${Date.now()}`, number: newNum, status: 'draft' as const, payments: [] },
        ],
      }
    }),

  sendInvoice: (id) =>
    set((state) => ({
      invoices: state.invoices.map((inv) =>
        inv.id === id && inv.status === 'draft' ? { ...inv, status: 'sent' as const } : inv,
      ),
    })),

  cancelInvoice: (id) =>
    set((state) => ({
      invoices: state.invoices.map((inv) =>
        inv.id === id ? { ...inv, status: 'cancelled' as const } : inv,
      ),
    })),

  recordPayment: (invoiceId, payment) =>
    set((state) => ({
      invoices: state.invoices.map((inv) => {
        if (inv.id !== invoiceId) return inv
        const newPayments = [...inv.payments, { ...payment, id: `pay-${Date.now()}` }]
        const total = calcInvoiceTotalFromInvoice({ ...inv, payments: newPayments })
        const paid = newPayments.reduce((s, p) => s + p.amount, 0)
        const newStatus: InvoiceMockStatus = paid >= total ? 'paid' : inv.status
        return { ...inv, payments: newPayments, status: newStatus }
      }),
    })),

  // --- Dunning Actions ---
  sendDunning: (id) =>
    set((state) => ({
      dunnings: state.dunnings.map((d) =>
        d.id === id && d.status === 'pending' ? { ...d, status: 'sent' as const } : d,
      ),
    })),

  escalateDunning: (id) =>
    set((state) => ({
      dunnings: state.dunnings.map((d) => {
        if (d.id !== id || d.level >= 3) return d
        const newLevel = (d.level + 1) as 1 | 2 | 3
        const newStatus = newLevel === 3 ? ('inkasso' as const) : ('pending' as const)
        return { ...d, level: newLevel, status: newStatus, sentAt: new Date().toISOString().slice(0, 10) }
      }),
    })),

  // --- Expense Actions ---
  addExpense: (expense) =>
    set((state) => ({
      expenses: [...state.expenses, { ...expense, id: `exp-${Date.now()}` }],
    })),

  approveExpense: (id) =>
    set((state) => ({
      expenses: state.expenses.map((e) => (e.id === id ? { ...e, status: 'approved' as const } : e)),
    })),

  rejectExpense: (id) =>
    set((state) => ({
      expenses: state.expenses.map((e) => (e.id === id ? { ...e, status: 'rejected' as const } : e)),
    })),

  deleteExpense: (id) =>
    set((state) => ({ expenses: state.expenses.filter((e) => e.id !== id) })),

  // --- Transaction Actions ---
  deleteTransaction: (id) =>
    set((state) => ({ transactions: state.transactions.filter((t) => t.id !== id) })),
}))
