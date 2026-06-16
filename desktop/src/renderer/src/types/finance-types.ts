/**
 * Finance module TypeScript types matching proto messages.
 *
 * All monetary values are represented as strings (server uses decimal.Decimal).
 * Parse to Number for display calculations only -- the server is authoritative.
 */

// ---------------------------------------------------------------------------
// Enums
// ---------------------------------------------------------------------------

export type QuoteStatus = 'draft' | 'sent' | 'accepted' | 'rejected' | 'expired'
export type InvoiceStatus = 'draft' | 'sent' | 'paid' | 'overdue' | 'cancelled'
export type CreditNoteStatus = 'draft' | 'sent'
export type DunningStatus = 'draft' | 'sent' | 'paid'
export type TaxMode = 'standard' | 'reverse_charge' | 'kleinunternehmer'
export type PaymentMethod = 'bank_transfer' | 'cash' | 'credit_card' | 'other'

/** Supported invoicing currencies. EUR is the base/reporting currency. */
export type Currency = 'EUR' | 'CHF' | 'USD'

/** Recurrence interval for a recurring invoice schedule. */
export type RecurringInterval = 'weekly' | 'monthly' | 'quarterly' | 'yearly'

/** Lifecycle of a recurring invoice schedule. */
export type RecurringStatus = 'active' | 'paused' | 'ended'

// ---------------------------------------------------------------------------
// Core domain types
// ---------------------------------------------------------------------------

export interface LineItem {
  id: string
  position: number
  description: string
  quantity: string
  unit_price: string
  tax_rate: string
  line_total: string
}

export interface TaxBreakdown {
  subtotal: string
  tax_by_rate: Record<string, string>
  total_tax: string
  gross_total: string
}

export interface CustomerSnapshot {
  name: string
  address: string
  email: string
  ust_id_nr?: string
}

export interface CompanySnapshot {
  name: string
  address: string
  steuernummer: string
  ust_id_nr: string
  handelsregister: string
  iban: string
  bic: string
  bank_name: string
}

export interface Quote {
  id: string
  quote_number: string
  status: QuoteStatus
  customer: CustomerSnapshot
  line_items: LineItem[]
  tax_mode: TaxMode
  tax_breakdown: TaxBreakdown
  valid_until: string
  /** Denormalised totals carried by mock/list payloads (server uses tax_breakdown). */
  total_net?: number | string
  total_gross?: number | string
  /** Document currency. Defaults to EUR when absent. */
  currency?: Currency
  /** Rate to EUR for non-EUR documents (1 unit foreign = N EUR), decimal string. */
  exchange_rate?: string
  notes?: string
  deal_id?: string
  created_at: string
  updated_at: string
}

export interface Invoice {
  id: string
  invoice_number: string
  status: InvoiceStatus
  customer: CustomerSnapshot
  company?: CompanySnapshot
  line_items: LineItem[]
  tax_mode: TaxMode
  tax_breakdown: TaxBreakdown
  invoice_date: string
  delivery_date?: string
  due_date: string
  payment_terms?: string
  source_quote_id?: string
  /** Denormalised totals carried by mock/list payloads (server uses tax_breakdown). */
  total_net?: number | string
  total_gross?: number | string
  /** Document currency. Defaults to EUR when absent. */
  currency?: Currency
  /** Rate to EUR for non-EUR documents (1 unit foreign = N EUR), decimal string. */
  exchange_rate?: string
  /** Set when this invoice was generated from a recurring schedule. */
  recurring_id?: string
  notes?: string
  created_at: string
  updated_at: string
}

export interface CreditNote {
  id: string
  credit_note_number: string
  status: CreditNoteStatus
  original_invoice_id: string
  /** Human-readable number of the original invoice (denormalised for display). */
  invoice_number?: string
  customer: CustomerSnapshot
  line_items: LineItem[]
  tax_mode: TaxMode
  tax_breakdown: TaxBreakdown
  /** Denormalised totals carried by mock/list payloads (server uses tax_breakdown). */
  total_net?: number | string
  total_gross?: number | string
  currency?: Currency
  /** True when the credit note fully cancels (storniert) the original invoice. */
  is_storno?: boolean
  reason?: string
  created_at: string
}

export interface Payment {
  id: string
  invoice_id: string
  amount: string
  payment_date: string
  method: PaymentMethod
  reference?: string
  notes?: string
}

export interface DunningRecord {
  id: string
  invoice_id: string
  level: 1 | 2 | 3
  status: DunningStatus
  fee: string
  interest: string
  sent_at?: string
  created_at: string
}

export interface DunningConfig {
  level1_days_after_due: number
  level2_days_after_level1: number
  level3_days_after_level2: number
  level1_fee: string
  level2_fee: string
  level3_fee: string
}

export interface CompanySettings {
  name: string
  street: string
  plz: string
  city: string
  country: string
  steuernummer: string
  ust_id_nr: string
  handelsregister: string
  bank_name: string
  iban: string
  bic: string
  logo_url?: string
  accent_color?: string
  is_kleinunternehmer: boolean
  default_payment_terms_days: number
  default_quote_validity_days: number
  basiszinssatz: string
}

// ---------------------------------------------------------------------------
// Dashboard
// ---------------------------------------------------------------------------

export interface FinanceDashboard {
  total_invoiced: string
  total_paid: string
  total_outstanding: string
  overdue_amount: string
  quotes_pending: number
  conversion_rate: number
  average_deal_size: string
  revenue_forecast: string
  revenue_this_month: number
  revenue_last_month: number
  status_breakdown: Record<InvoiceStatus, number>
  recent_invoices: Invoice[]
  expiring_quotes: Quote[]
  pending_dunnings: DunningRecord[]
  monthly_revenue?: Array<{ month: string; revenue: number }>
}

// ---------------------------------------------------------------------------
// Request types
// ---------------------------------------------------------------------------

export interface CreateQuoteRequest {
  customer: CustomerSnapshot
  tax_mode: TaxMode
  line_items: Omit<LineItem, 'id' | 'line_total'>[]
  valid_until?: string
  currency?: Currency
  exchange_rate?: string
  notes?: string
  deal_id?: string
}

export interface UpdateQuoteRequest {
  customer?: CustomerSnapshot
  tax_mode?: TaxMode
  line_items?: Omit<LineItem, 'id' | 'line_total'>[]
  valid_until?: string
  currency?: Currency
  exchange_rate?: string
  notes?: string
}

export interface CreateInvoiceRequest {
  customer: CustomerSnapshot
  tax_mode: TaxMode
  line_items: Omit<LineItem, 'id' | 'line_total'>[]
  invoice_date: string
  delivery_date?: string
  payment_terms_days: number
  currency?: Currency
  exchange_rate?: string
  notes?: string
  source_quote_id?: string
}

export interface UpdateInvoiceRequest {
  customer?: CustomerSnapshot
  tax_mode?: TaxMode
  line_items?: Omit<LineItem, 'id' | 'line_total'>[]
  invoice_date?: string
  delivery_date?: string
  payment_terms_days?: number
  currency?: Currency
  exchange_rate?: string
  notes?: string
}

export interface CreateCreditNoteRequest {
  original_invoice_id: string
  line_items: Omit<LineItem, 'id' | 'line_total'>[]
  tax_mode: TaxMode
  reason?: string
  /** When true, the original invoice is cancelled (storniert) on creation. */
  is_storno?: boolean
}

export interface RecordPaymentRequest {
  amount: string
  payment_date: string
  method: PaymentMethod
  reference?: string
  notes?: string
}

export interface UpdateDunningConfigRequest {
  level1_days_after_due?: number
  level2_days_after_level1?: number
  level3_days_after_level2?: number
  level1_fee?: string
  level2_fee?: string
  level3_fee?: string
}

export interface UpdateCompanySettingsRequest {
  name?: string
  street?: string
  plz?: string
  city?: string
  country?: string
  steuernummer?: string
  ust_id_nr?: string
  handelsregister?: string
  bank_name?: string
  iban?: string
  bic?: string
  logo_url?: string
  accent_color?: string
  is_kleinunternehmer?: boolean
  default_payment_terms_days?: number
  default_quote_validity_days?: number
  basiszinssatz?: string
}

// ---------------------------------------------------------------------------
// List / filter params
// ---------------------------------------------------------------------------

export interface ListQuotesParams {
  status?: QuoteStatus
  page?: number
  page_size?: number
}

export interface ListInvoicesParams {
  status?: InvoiceStatus
  recurring_id?: string
  page?: number
  page_size?: number
}

export interface ListCreditNotesParams {
  status?: CreditNoteStatus
  page?: number
  page_size?: number
}

export interface ListDunningsParams {
  invoice_id?: string
  level?: number
  status?: DunningStatus
  page?: number
  page_size?: number
}

export interface DashboardParams {
  date_from?: string
  date_to?: string
}

export interface ExportDATEVParams {
  date_from: string
  date_to: string
}

// ---------------------------------------------------------------------------
// Response wrappers
// ---------------------------------------------------------------------------

export interface ListQuotesResponse {
  quotes: Quote[]
  total: number
}

export interface ListInvoicesResponse {
  invoices: Invoice[]
  total: number
}

export interface ListCreditNotesResponse {
  credit_notes: CreditNote[]
  total: number
}

export interface ListPaymentsResponse {
  payments: Payment[]
  total: number
}

export interface ListDunningsResponse {
  dunnings: DunningRecord[]
  total: number
}

// ---------------------------------------------------------------------------
// Banking / Belegkette / Stunden→Rechnung (finanzen P2.5e)
// ---------------------------------------------------------------------------

export type BankMatchStatus = 'matched' | 'suggested' | 'unmatched'

export interface BankAccount {
  id: string
  bankName: string
  iban: string
  bic: string
  balance: number
  currency: string
  connected: boolean
  lastSync: string | null
}

export interface BankTransaction {
  id: string
  date: string
  description: string
  amount: number
  type: 'credit' | 'debit'
  counterpart: string
  matchStatus: BankMatchStatus
  matchedInvoice?: string
}

export interface ListBankAccountsResponse {
  accounts: BankAccount[]
}

export interface ListBankTransactionsResponse {
  transactions: BankTransaction[]
}

export type ChainDocType = 'quote' | 'invoice' | 'payment' | 'credit-note' | 'dunning'
export type ChainDocStatus = 'completed' | 'active' | 'pending' | 'cancelled' | 'overdue'

export interface ChainNode {
  type: ChainDocType
  number: string
  date: string
  amount: string
  status: ChainDocStatus
}

export interface DocumentChain {
  id: string
  customer: string
  totalValue: string
  nodes: ChainNode[]
  isComplete: boolean
}

export interface ListDocumentChainsResponse {
  chains: DocumentChain[]
}

export interface FinanceTimeEntry {
  id: string
  date: string
  project: string
  task: string
  employee: string
  hours: number
  description: string
  billed: boolean
}

export interface ListTimeEntriesResponse {
  entries: FinanceTimeEntry[]
}

// ---------------------------------------------------------------------------
// GoBD Journal & Compliance (Sprint 2 / Wave 1.B)
// ---------------------------------------------------------------------------

/** GoBD journal summary for a fiscal year. gaps_detected must be 0 for compliance. */
export interface JournalSummary {
  year: number
  total_invoices_issued: number
  gaps_detected: number
  first_number: string
  last_number: string
  highest_sequence: number
}

/** Result of validating an invoice number candidate. */
export interface ValidateInvoiceNumberResult {
  valid_format: boolean
  already_used: boolean
  canonical: string
}

/** Aggregated payment statistics for a date range. */
export interface PaymentStats {
  total_invoices: number
  total_paid: number
  total_outstanding: number
  /** Decimal string, e.g. "12345.00" */
  total_paid_amount: string
  /** Decimal string, e.g. "6789.00" */
  total_outstanding_amount: string
  /** Average days from invoice_date to payment, e.g. "14.5" */
  average_days_to_pay: string
}

/** Parameters for fetching the GoBD journal summary. */
export interface JournalSummaryParams {
  year: number
}

/** Parameters for date-range queries (payments stats, GoBD export). */
export interface DateRangeParams {
  from_date: string // YYYY-MM-DD
  to_date: string   // YYYY-MM-DD
}

/** Parameters for the GoBD CSV export. */
export type GoBDExportParams = DateRangeParams

/** Response from LockInvoice. */
export interface LockInvoiceResponse {
  invoice_json?: Record<string, unknown>
  locked_at: string // RFC3339
}

/** Response from UpdateDunningStatus / SendDunningNotice. */
export interface DunningNoticeResponse {
  dunning: DunningRecord
  email_queued?: boolean
}

// ---------------------------------------------------------------------------
// Recurring invoices (finanzen P1)
// ---------------------------------------------------------------------------

/**
 * A recurring invoice schedule. Acts as a template that emits draft invoices
 * at the configured interval until the (optional) end date is reached.
 */
export interface RecurringInvoice {
  id: string
  /** Display title, e.g. "CRM-Lizenz Müller GmbH". */
  title: string
  customer: CustomerSnapshot
  line_items: LineItem[]
  tax_mode: TaxMode
  tax_breakdown: TaxBreakdown
  currency?: Currency
  interval: RecurringInterval
  status: RecurringStatus
  /** First issue date (YYYY-MM-DD). */
  start_date: string
  /** Optional last issue date (YYYY-MM-DD). */
  end_date?: string
  /** Next scheduled issue date (YYYY-MM-DD). */
  next_run: string
  payment_terms_days: number
  /** Number of invoices already emitted from this schedule. */
  generated_count: number
  /** Issue date of the most recent emitted invoice (YYYY-MM-DD). */
  last_generated_at?: string
  notes?: string
  created_at: string
}

export interface CreateRecurringInvoiceRequest {
  title: string
  customer: CustomerSnapshot
  line_items: Omit<LineItem, 'id' | 'line_total'>[]
  tax_mode: TaxMode
  currency?: Currency
  interval: RecurringInterval
  start_date: string
  end_date?: string
  payment_terms_days: number
  notes?: string
}

export type UpdateRecurringInvoiceRequest = Partial<CreateRecurringInvoiceRequest>

export interface ListRecurringInvoicesResponse {
  recurring: RecurringInvoice[]
  total: number
}

/** Response from generating the next invoice of a recurring schedule. */
export interface GenerateRecurringInvoiceResponse {
  invoice: Invoice
  recurring: RecurringInvoice
}
