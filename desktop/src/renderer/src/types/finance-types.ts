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
  notes?: string
  created_at: string
  updated_at: string
}

export interface CreditNote {
  id: string
  credit_note_number: string
  status: CreditNoteStatus
  original_invoice_id: string
  customer: CustomerSnapshot
  line_items: LineItem[]
  tax_mode: TaxMode
  tax_breakdown: TaxBreakdown
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
  notes?: string
  deal_id?: string
}

export interface UpdateQuoteRequest {
  customer?: CustomerSnapshot
  tax_mode?: TaxMode
  line_items?: Omit<LineItem, 'id' | 'line_total'>[]
  valid_until?: string
  notes?: string
}

export interface CreateInvoiceRequest {
  customer: CustomerSnapshot
  tax_mode: TaxMode
  line_items: Omit<LineItem, 'id' | 'line_total'>[]
  invoice_date: string
  delivery_date?: string
  payment_terms_days: number
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
  notes?: string
}

export interface CreateCreditNoteRequest {
  original_invoice_id: string
  line_items: Omit<LineItem, 'id' | 'line_total'>[]
  tax_mode: TaxMode
  reason?: string
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
