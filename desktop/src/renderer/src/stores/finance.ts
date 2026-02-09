import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface LineItem {
  id: string
  description: string
  quantity: number
  unitPrice: number
  vatRate: number // 0, 7.7, 8.1 (CH standard)
  discount: number // percentage
}

export interface Invoice {
  id: string
  number: string
  type: 'invoice' | 'quote'
  client: string
  clientEmail?: string
  date: string
  dueDate: string
  status: 'draft' | 'sent' | 'paid' | 'overdue' | 'cancelled'
  items: LineItem[]
  notes?: string
  paymentTerms: string
  payments: { date: string; amount: number; method: string; reference?: string }[]
  createdAt: string
}

export interface Transaction {
  id: string
  type: 'income' | 'expense'
  description: string
  amount: number // positive for income, negative for expense
  date: string
  category: string
  status: 'completed' | 'pending'
  reference?: string
  invoiceId?: string
  receipt?: string
}

export interface Expense {
  id: string
  description: string
  amount: number
  date: string
  category: string
  supplier: string
  project?: string
  receipt?: boolean
  status: 'pending' | 'approved' | 'rejected'
}

// Helpers
export function calcLineTotal(item: LineItem): number {
  const base = item.quantity * item.unitPrice
  const discounted = base * (1 - item.discount / 100)
  return discounted
}

export function calcLineTax(item: LineItem): number {
  return calcLineTotal(item) * (item.vatRate / 100)
}

export function calcInvoiceSubtotal(items: LineItem[]): number {
  return items.reduce((sum, item) => sum + calcLineTotal(item), 0)
}

export function calcInvoiceTax(items: LineItem[]): number {
  return items.reduce((sum, item) => sum + calcLineTax(item), 0)
}

export function calcInvoiceTotal(items: LineItem[]): number {
  return calcInvoiceSubtotal(items) + calcInvoiceTax(items)
}

export function calcPaidAmount(invoice: Invoice): number {
  return invoice.payments.reduce((sum, p) => sum + p.amount, 0)
}

export function calcRemainingAmount(invoice: Invoice): number {
  return calcInvoiceTotal(invoice.items) - calcPaidAmount(invoice)
}

function formatCHF(n: number): string {
  return new Intl.NumberFormat('de-CH', { style: 'currency', currency: 'CHF' }).format(n)
}

// ============================================================
// Store
// ============================================================

interface FinanceStore {
  invoices: Invoice[]
  transactions: Transaction[]
  expenses: Expense[]
  nextInvoiceNum: number

  addInvoice: (invoice: Omit<Invoice, 'id' | 'createdAt' | 'payments'>) => void
  updateInvoice: (id: string, updates: Partial<Invoice>) => void
  deleteInvoice: (id: string) => void
  duplicateInvoice: (id: string) => void
  sendInvoice: (id: string) => void
  recordPayment: (invoiceId: string, payment: { date: string; amount: number; method: string; reference?: string }) => void
  cancelInvoice: (id: string) => void

  addTransaction: (tx: Omit<Transaction, 'id'>) => void
  updateTransaction: (id: string, updates: Partial<Transaction>) => void
  deleteTransaction: (id: string) => void

  addExpense: (expense: Omit<Expense, 'id'>) => void
  approveExpense: (id: string) => void
  rejectExpense: (id: string) => void
  deleteExpense: (id: string) => void
}

let invoiceCounter = 43

const INITIAL_INVOICES: Invoice[] = [
  {
    id: 'inv1', number: 'INV-2026-042', type: 'invoice', client: 'ABC GmbH', clientEmail: 'buchhaltung@abc-gmbh.ch',
    date: '2026-02-01', dueDate: '2026-02-28', status: 'paid',
    items: [
      { id: 'li1', description: 'Webentwicklung (Frontend)', quantity: 40, unitPrice: 180, vatRate: 8.1, discount: 0 },
      { id: 'li2', description: 'Backend API Integration', quantity: 24, unitPrice: 200, vatRate: 8.1, discount: 0 },
      { id: 'li3', description: 'UI/UX Design', quantity: 16, unitPrice: 160, vatRate: 8.1, discount: 5 },
    ],
    notes: 'Projekt: Website Relaunch Phase 1', paymentTerms: '30 Tage netto',
    payments: [{ date: '2026-02-08', amount: 15400, method: 'Bankueberweisung', reference: 'BELEG-042' }],
    createdAt: '2026-02-01',
  },
  {
    id: 'inv2', number: 'INV-2026-041', type: 'invoice', client: 'XYZ AG', clientEmail: 'rechnungen@xyz-ag.ch',
    date: '2026-01-28', dueDate: '2026-02-28', status: 'sent',
    items: [
      { id: 'li4', description: 'CRM Beratung', quantity: 20, unitPrice: 220, vatRate: 8.1, discount: 0 },
      { id: 'li5', description: 'Datenanalyse & Report', quantity: 12, unitPrice: 180, vatRate: 8.1, discount: 10 },
    ],
    notes: '', paymentTerms: '30 Tage netto', payments: [], createdAt: '2026-01-28',
  },
  {
    id: 'inv3', number: 'INV-2026-040', type: 'invoice', client: 'Swiss Hosting AG', clientEmail: 'finance@swiss-hosting.ch',
    date: '2026-01-20', dueDate: '2026-02-20', status: 'paid',
    items: [
      { id: 'li6', description: 'Infrastruktur-Beratung', quantity: 16, unitPrice: 200, vatRate: 8.1, discount: 0 },
    ],
    notes: 'Partnerschafts-Rabatt 5% auf naechste Rechnung', paymentTerms: '30 Tage netto',
    payments: [{ date: '2026-02-03', amount: 3459.20, method: 'Bankueberweisung' }],
    createdAt: '2026-01-20',
  },
  {
    id: 'inv4', number: 'INV-2026-039', type: 'invoice', client: 'Meier Consulting', clientEmail: 'office@meier-consulting.ch',
    date: '2026-01-10', dueDate: '2026-02-10', status: 'overdue',
    items: [
      { id: 'li7', description: 'Projektmanagement Setup', quantity: 8, unitPrice: 180, vatRate: 8.1, discount: 0 },
      { id: 'li8', description: 'Schulung CRM System (1 Tag)', quantity: 1, unitPrice: 2200, vatRate: 8.1, discount: 0 },
      { id: 'li9', description: 'Dokumentation', quantity: 6, unitPrice: 150, vatRate: 8.1, discount: 0 },
      { id: 'li10', description: 'Support-Pauschale (3 Monate)', quantity: 3, unitPrice: 400, vatRate: 8.1, discount: 0 },
    ],
    notes: '2. Mahnung gesendet am 2026-02-05', paymentTerms: '30 Tage netto', payments: [],
    createdAt: '2026-01-10',
  },
  {
    id: 'inv5', number: 'QUO-2026-005', type: 'quote', client: 'Brunner & Partner', clientEmail: 'eva@brunner-partner.ch',
    date: '2026-02-07', dueDate: '2026-03-07', status: 'sent',
    items: [
      { id: 'li11', description: 'CRM Komplett-Paket (Setup + Anpassung)', quantity: 1, unitPrice: 12000, vatRate: 8.1, discount: 0 },
      { id: 'li12', description: 'Onboarding & Schulung', quantity: 2, unitPrice: 2200, vatRate: 8.1, discount: 0 },
    ],
    notes: 'Angebot gueltig bis 07.03.2026', paymentTerms: '50% Anzahlung, 50% bei Abnahme', payments: [],
    createdAt: '2026-02-07',
  },
]

const INITIAL_TRANSACTIONS: Transaction[] = [
  { id: 'tx1', type: 'income', description: 'Zahlung ABC GmbH — INV-042', amount: 15400, date: '2026-02-08', category: 'Dienstleistung', status: 'completed', reference: 'INV-042', invoiceId: 'inv1' },
  { id: 'tx2', type: 'expense', description: 'Hetzner Cloud Server (Februar)', amount: -890.50, date: '2026-02-07', category: 'Hosting', status: 'completed', reference: 'EXP-128' },
  { id: 'tx3', type: 'income', description: 'Zahlung Swiss Hosting AG — INV-040', amount: 3459.20, date: '2026-02-03', category: 'Beratung', status: 'completed', reference: 'INV-040', invoiceId: 'inv3' },
  { id: 'tx4', type: 'expense', description: 'Figma Team (Jahreslizenz)', amount: -540, date: '2026-02-05', category: 'Software', status: 'completed', reference: 'EXP-127' },
  { id: 'tx5', type: 'expense', description: 'Team Lunch — Retrospektive', amount: -186.30, date: '2026-02-04', category: 'Verpflegung', status: 'completed', reference: 'EXP-126' },
  { id: 'tx6', type: 'expense', description: 'GitHub Enterprise', amount: -252, date: '2026-02-01', category: 'Software', status: 'completed', reference: 'EXP-125' },
  { id: 'tx7', type: 'expense', description: 'Bueromoebel Steiner GmbH', amount: -3200, date: '2026-01-28', category: 'Buero', status: 'completed', reference: 'EXP-124' },
  { id: 'tx8', type: 'income', description: 'Partnerschaft Q1 — Marketing AG', amount: 4500, date: '2026-01-25', category: 'Partnerschaft', status: 'completed', reference: 'PART-Q1' },
  { id: 'tx9', type: 'expense', description: 'SBB Halbtax alle Mitarbeiter', amount: -1080, date: '2026-01-20', category: 'Reise', status: 'completed', reference: 'EXP-122' },
  { id: 'tx10', type: 'expense', description: 'Strom & Internet Januar', amount: -320, date: '2026-01-15', category: 'Buero', status: 'completed', reference: 'EXP-121' },
]

const INITIAL_EXPENSES: Expense[] = [
  { id: 'exp1', description: 'Neue Tastatur Logitech MX Keys', amount: 129, date: '2026-02-08', category: 'Hardware', supplier: 'Digitec', project: 'IT-Ausstattung', receipt: true, status: 'pending' },
  { id: 'exp2', description: 'Bahnticket Zuerich-Bern (Kundentermin)', amount: 51, date: '2026-02-07', category: 'Reise', supplier: 'SBB', project: 'CRM Integration', receipt: true, status: 'approved' },
  { id: 'exp3', description: 'PostFinance Kontogebuehren', amount: 25, date: '2026-02-01', category: 'Bankgebuehren', supplier: 'PostFinance', receipt: false, status: 'approved' },
  { id: 'exp4', description: 'Rechtsberatung Vertragsvorlage', amount: 480, date: '2026-01-30', category: 'Beratung', supplier: 'Kanzlei Huber', receipt: true, status: 'pending' },
  { id: 'exp5', description: 'Google Workspace (12 Lizenzen)', amount: 168, date: '2026-02-01', category: 'Software', supplier: 'Google', receipt: true, status: 'approved' },
]

export const useFinanceStore = create<FinanceStore>()(
  persist(
    (set, get) => ({
      invoices: INITIAL_INVOICES,
      transactions: INITIAL_TRANSACTIONS,
      expenses: INITIAL_EXPENSES,
      nextInvoiceNum: invoiceCounter,

      addInvoice: (invoice) =>
        set((state) => {
          const num = state.nextInvoiceNum + 1
          return {
            invoices: [
              ...state.invoices,
              { ...invoice, id: `inv${Date.now()}`, createdAt: new Date().toISOString().split('T')[0], payments: [] },
            ],
            nextInvoiceNum: num,
          }
        }),

      updateInvoice: (id, updates) =>
        set((state) => ({
          invoices: state.invoices.map((inv) => (inv.id === id ? { ...inv, ...updates } : inv)),
        })),

      deleteInvoice: (id) =>
        set((state) => ({
          invoices: state.invoices.filter((inv) => inv.id !== id),
        })),

      duplicateInvoice: (id) =>
        set((state) => {
          const orig = state.invoices.find((inv) => inv.id === id)
          if (!orig) return state
          const num = state.nextInvoiceNum + 1
          const prefix = orig.type === 'quote' ? 'QUO' : 'INV'
          return {
            invoices: [
              ...state.invoices,
              {
                ...orig,
                id: `inv${Date.now()}`,
                number: `${prefix}-2026-${String(num).padStart(3, '0')}`,
                status: 'draft',
                date: new Date().toISOString().split('T')[0],
                payments: [],
                createdAt: new Date().toISOString().split('T')[0],
              },
            ],
            nextInvoiceNum: num,
          }
        }),

      sendInvoice: (id) =>
        set((state) => ({
          invoices: state.invoices.map((inv) =>
            inv.id === id && (inv.status === 'draft') ? { ...inv, status: 'sent' as const } : inv,
          ),
        })),

      recordPayment: (invoiceId, payment) =>
        set((state) => {
          const inv = state.invoices.find((i) => i.id === invoiceId)
          if (!inv) return state
          const newPayments = [...inv.payments, payment]
          const totalPaid = newPayments.reduce((s, p) => s + p.amount, 0)
          const invoiceTotal = calcInvoiceTotal(inv.items)
          const newStatus = totalPaid >= invoiceTotal ? 'paid' as const : inv.status

          return {
            invoices: state.invoices.map((i) =>
              i.id === invoiceId ? { ...i, payments: newPayments, status: newStatus } : i,
            ),
            transactions: [
              ...state.transactions,
              {
                id: `tx${Date.now()}`,
                type: 'income' as const,
                description: `Zahlung ${inv.client} — ${inv.number}`,
                amount: payment.amount,
                date: payment.date,
                category: 'Dienstleistung',
                status: 'completed' as const,
                reference: inv.number,
                invoiceId,
              },
            ],
          }
        }),

      cancelInvoice: (id) =>
        set((state) => ({
          invoices: state.invoices.map((inv) =>
            inv.id === id ? { ...inv, status: 'cancelled' as const } : inv,
          ),
        })),

      addTransaction: (tx) =>
        set((state) => ({
          transactions: [...state.transactions, { ...tx, id: `tx${Date.now()}` }],
        })),

      updateTransaction: (id, updates) =>
        set((state) => ({
          transactions: state.transactions.map((tx) => (tx.id === id ? { ...tx, ...updates } : tx)),
        })),

      deleteTransaction: (id) =>
        set((state) => ({
          transactions: state.transactions.filter((tx) => tx.id !== id),
        })),

      addExpense: (expense) =>
        set((state) => ({
          expenses: [...state.expenses, { ...expense, id: `exp${Date.now()}` }],
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
        set((state) => ({
          expenses: state.expenses.filter((e) => e.id !== id),
        })),
    }),
    { name: 'kmuhub-finance' },
  ),
)
