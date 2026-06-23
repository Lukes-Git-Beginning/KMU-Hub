import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import {
  mockInvoices,
  mockQuotes,
  mockCreditNotes,
  mockFinanceDashboard,
  mockFinanceSettings,
} from '../data/invoices'
import { mockRecurringInvoices, advanceByInterval } from '../data/finance-recurring'
import { mockExpenses, mockTransactions } from '../data/finance-ledger'
import { mockBanking } from '../data/finance-banking'
import { mockDocumentChains } from '../data/finance-chains'
import { mockFinanceTimeEntries } from '../data/finance-time-entries'
import { buildSimplePdf, type PdfLine } from '@/modules/finanzen/lib/mini-pdf'
import { buildDatevCsv } from '@/modules/finanzen/lib/finance-export'
import type { Payment, DunningRecord, DunningStatus, DunningConfig } from '@/types/finance-types'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

interface RawLineItem {
  description?: string
  quantity?: string | number
  unit_price?: string | number
  tax_rate?: string | number
}

/** Compute net/tax/gross totals + tax_breakdown from request line items. */
function computeTotals(lineItems: RawLineItem[]) {
  const tax_by_rate: Record<string, number> = {}
  let net = 0
  const line_items = lineItems.map((it, idx) => {
    const qty = Number(it.quantity ?? 0)
    const price = Number(it.unit_price ?? 0)
    const rate = Number(it.tax_rate ?? 0)
    const lineNet = qty * price
    net += lineNet
    tax_by_rate[rate] = (tax_by_rate[rate] ?? 0) + lineNet * (rate / 100)
    return {
      id: `li-${idx}`,
      position: idx + 1,
      description: String(it.description ?? ''),
      quantity: String(it.quantity ?? '0'),
      unit_price: String(it.unit_price ?? '0'),
      tax_rate: String(it.tax_rate ?? '0'),
      line_total: lineNet.toFixed(2),
    }
  })
  const totalTax = Object.values(tax_by_rate).reduce((s, v) => s + v, 0)
  return {
    line_items,
    total_net: net,
    total_gross: net + totalTax,
    tax_breakdown: {
      subtotal: net.toFixed(2),
      tax_by_rate: Object.fromEntries(
        Object.entries(tax_by_rate).map(([k, v]) => [k, v.toFixed(2)]),
      ),
      total_tax: totalTax.toFixed(2),
      gross_total: (net + totalTax).toFixed(2),
    },
  }
}

function addDays(dateISO: string, days: number): string {
  const d = new Date((dateISO || new Date().toISOString().split('T')[0]) + 'T00:00:00')
  d.setDate(d.getDate() + days)
  return d.toISOString().split('T')[0]
}

/**
 * Normalise a list-shaped finance doc (quote/credit-note seed with `items` +
 * denormalised totals) into the detail shape the *DetailPanel components expect
 * (`line_items` + `tax_breakdown` + structured `customer`). Mirrors the invoice
 * detail normalisation so quotes/credit-notes get the same rich view.
 */
function normalizeDoc(raw: Record<string, unknown>) {
  const rawItems = (raw.items ?? raw.line_items ?? []) as Array<Record<string, unknown>>
  const line_items = rawItems.map((it, idx) => {
    const qty = Number(it.quantity ?? 1)
    const price = Number(it.unit_price ?? 0)
    return {
      id: (it.id as string) ?? `li-${idx}`,
      position: idx + 1,
      description: String(it.description ?? ''),
      quantity: qty,
      unit_price: price,
      tax_rate: Number(it.tax_rate ?? raw.tax_rate ?? 19),
      line_total: Number(it.line_total ?? it.total ?? qty * price),
    }
  })
  let tax_breakdown = raw.tax_breakdown as Record<string, unknown> | undefined
  if (!tax_breakdown) {
    const subtotal = Number(raw.total_net ?? line_items.reduce((s, it) => s + it.line_total, 0))
    const gross = Number(raw.total_gross ?? subtotal * 1.19)
    const totalTax = Math.max(0, gross - subtotal)
    const rate = subtotal > 0 ? Math.round((totalTax / subtotal) * 100) : 19
    tax_breakdown = {
      subtotal: subtotal.toFixed(2),
      tax_by_rate: { [rate]: totalTax.toFixed(2) },
      total_tax: totalTax.toFixed(2),
      gross_total: gross.toFixed(2),
    }
  }
  const customer = (raw.customer ?? { name: raw.customer_name ?? '', address: '', email: '' }) as Record<string, unknown>
  return {
    ...raw,
    line_items,
    tax_breakdown,
    customer: {
      name: customer.name ?? raw.customer_name ?? '',
      address: customer.address ?? '',
      email: customer.email ?? '',
      ...customer,
    },
  }
}

/** Build a one-page PDF (bytes) for an invoice/quote/credit-note. */
function buildDocPdf(rawDoc: Record<string, unknown>, heading: string, numberLabel: string): Uint8Array {
  const doc = normalizeDoc(rawDoc)
  const customer = doc.customer as { name?: string; address?: string }
  const tb = doc.tax_breakdown as { subtotal?: string; total_tax?: string; gross_total?: string }
  const currency = (doc.currency as string) ?? 'EUR'
  const fmt = (v: unknown) => `${Number(v ?? 0).toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} ${currency}`
  const lines: PdfLine[] = [
    { text: 'Zentria GmbH · Cosmi', size: 9 },
    { text: heading, size: 20, gap: 6 },
    { text: `${numberLabel}: ${doc[numberLabel === 'Gutschrift-Nr' ? 'credit_note_number' : numberLabel === 'Angebots-Nr' ? 'quote_number' : 'invoice_number'] ?? ''}`, gap: 8 },
    { text: `Kunde: ${customer?.name ?? ''}` },
    ...(customer?.address ? [{ text: customer.address, size: 9 }] : []),
    { text: '', gap: 6 },
    { text: 'Positionen:', size: 12 },
  ]
  for (const it of (doc.line_items as Array<Record<string, unknown>>) ?? []) {
    lines.push({ text: `${it.quantity} x ${it.description}    ${fmt(it.line_total)}`, size: 10 })
  }
  lines.push({ text: '', gap: 6 })
  lines.push({ text: `Zwischensumme (netto): ${fmt(tb?.subtotal)}`, size: 10 })
  lines.push({ text: `MwSt: ${fmt(tb?.total_tax)}`, size: 10 })
  lines.push({ text: `Gesamtbetrag: ${fmt(tb?.gross_total)}`, size: 13, gap: 4 })
  lines.push({ text: '', gap: 10 })
  lines.push({ text: 'Demo-Beleg (Cosmi) — finales PDF via ZUGFeRD/XRechnung folgt.', size: 8 })
  return buildSimplePdf(lines)
}

function pdfResponse(bytes: Uint8Array, filename: string): Response {
  return new HttpResponse(bytes, {
    headers: { 'Content-Type': 'application/pdf', 'Content-Disposition': `attachment; filename="${filename}"` },
  })
}

// ---------------------------------------------------------------------------
// Stateful demo stores: payments + dunnings (finanzen P2.5d).
// Session-persistent (in-memory), reset on reload — same pattern as the
// invoice/quote mutations above.
// ---------------------------------------------------------------------------

const todayStr = () => new Date().toISOString().split('T')[0]
const grossOf = (inv: Record<string, unknown>) =>
  Number((inv.tax_breakdown as { gross_total?: string })?.gross_total ?? inv.total_gross ?? 0)

/** invoiceId -> recorded payments. */
const paymentsStore: Record<string, Payment[]> = {}

/** Mutable dunning config in the flat shape the UI expects. */
let dunningConfig: DunningConfig = {
  level1_days_after_due: 7,
  level2_days_after_level1: 7,
  level3_days_after_level2: 14,
  level1_fee: '0.00',
  level2_fee: '5.00',
  level3_fee: '10.00',
}

const feeForLevel = (level: number) =>
  level === 1 ? dunningConfig.level1_fee : level === 2 ? dunningConfig.level2_fee : dunningConfig.level3_fee

/** Plausible Verzugszinsen for the demo: 0.5% of gross per dunning level. */
const interestFor = (gross: number, level: number) => (gross * 0.005 * level).toFixed(2)

let dunningSeq = 0
function makeDunning(invoice: Record<string, unknown>, level: 1 | 2 | 3, status: DunningStatus): DunningRecord {
  dunningSeq += 1
  return {
    id: `dun-${dunningSeq}`,
    invoice_id: String(invoice.id),
    level,
    status,
    fee: feeForLevel(level),
    interest: interestFor(grossOf(invoice), level),
    sent_at: status === 'sent' ? todayStr() : undefined,
    created_at: todayStr(),
  }
}

const paidTotal = (invoiceId: string) =>
  (paymentsStore[invoiceId] ?? []).reduce((s, p) => s + Number(p.amount), 0)

/** Overdue = issued (sent/overdue), past due date, not fully paid. */
function isOverdue(inv: Record<string, unknown>): boolean {
  if (inv.status !== 'sent' && inv.status !== 'overdue') return false
  const due = inv.due_date as string | undefined
  return !!due && due < todayStr() && paidTotal(String(inv.id)) < grossOf(inv)
}

// Seed payments for already-paid invoices so the payment history is populated.
for (const inv of mockInvoices.invoices as Record<string, unknown>[]) {
  if (inv.status === 'paid') {
    paymentsStore[String(inv.id)] = [
      {
        id: `pay-seed-${inv.id}`,
        invoice_id: String(inv.id),
        amount: grossOf(inv).toFixed(2),
        payment_date: (inv.due_date as string) ?? todayStr(),
        method: 'bank_transfer',
        reference: `SEPA-${String(inv.invoice_number ?? '').replace(/\D/g, '')}`,
      },
    ]
  }
}

// Seed a few dunnings for currently-overdue invoices so the panel shows data.
const dunningsStore: DunningRecord[] = []
for (const inv of mockInvoices.invoices as Record<string, unknown>[]) {
  if (dunningsStore.length >= 3 || !isOverdue(inv)) continue
  const first = dunningsStore.length === 0
  dunningsStore.push(makeDunning(inv, first ? 2 : 1, first ? 'sent' : 'draft'))
}

export const financeHandlers = [
  // ---- Invoices ----

  // List invoices
  http.get(`${API}/api/v1/finance/invoices`, ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('status')
    const recurringId = url.searchParams.get('recurring_id')
    const page = parseInt(url.searchParams.get('page') || '1', 10)
    const perPage = parseInt(url.searchParams.get('per_page') || '50', 10)

    let filtered = [...mockInvoices.invoices]
    if (status) {
      filtered = filtered.filter((inv) => inv.status === status)
    }
    if (recurringId) {
      filtered = filtered.filter(
        (inv) => (inv as { recurring_id?: string }).recurring_id === recurringId,
      )
    }

    const start = (page - 1) * perPage
    const paged = filtered.slice(start, start + perPage)

    return HttpResponse.json({
      invoices: paged,
      total: filtered.length,
      page,
      per_page: perPage,
    })
  }),

  // Invoice detail
  http.get(`${API}/api/v1/finance/invoices/:id`, ({ params }) => {
    const raw = mockInvoices.invoices.find((inv) => inv.id === params.id)
    if (!raw) {
      return HttpResponse.json({ error: 'Invoice not found' }, { status: 404 })
    }
    // Normalise legacy mock fields so InvoiceDetailPanel gets the right shape:
    // - `items` → `line_items` (with id/line_total filled)
    // - `issue_date` → `invoice_date`
    const inv = raw as Record<string, unknown>
    const rawItems = (inv.items ?? inv.line_items ?? []) as Array<{
      description?: string
      quantity?: number
      unit_price?: number
      total?: number
      tax_rate?: number
      line_total?: number
      id?: string
    }>
    const line_items = rawItems.map((it, idx) => ({
      id: it.id ?? `li-${idx}`,
      description: it.description ?? '',
      quantity: it.quantity ?? 1,
      unit_price: it.unit_price ?? 0,
      tax_rate: it.tax_rate ?? (inv.tax_rate as number | undefined) ?? 19,
      line_total: it.line_total ?? it.total ?? 0,
    }))
    const invoice_date = (inv.invoice_date ?? inv.issue_date ?? new Date().toISOString().split('T')[0]) as string
    const tax_breakdown = inv.tax_breakdown ?? {
      subtotal: inv.total_net ?? 0,
      tax_by_rate: { [(inv.tax_rate as number | undefined) ?? 19]: (inv.total_gross as number ?? 0) - (inv.total_net as number ?? 0) },
      gross_total: inv.total_gross ?? 0,
    }
    const customer = (inv.customer ?? { name: inv.customer_name ?? '', address: '', email: '' }) as Record<string, unknown>
    const normalised = {
      ...inv,
      line_items,
      invoice_date,
      tax_breakdown,
      customer: {
        name: customer.name ?? '',
        address: customer.address ?? '',
        email: customer.email ?? '',
        ...customer,
      },
    }
    return HttpResponse.json({ invoice: normalised })
  }),

  // Create invoice — stateful: persists into the mock list so it shows up.
  http.post(`${API}/api/v1/finance/invoices`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const totals = computeTotals((body.line_items as RawLineItem[]) ?? [])
    const invoiceDate = (body.invoice_date as string) || new Date().toISOString().split('T')[0]
    const termsDays = Number(body.payment_terms_days ?? 30)
    const seq = mockInvoices.invoices.length + 1
    const customer = (body.customer ?? { name: '', address: '', email: '' }) as Record<string, unknown>
    const newInvoice = {
      id: `inv-${Date.now()}`,
      number: `RE-2026-${String(seq).padStart(3, '0')}`,
      invoice_number: `RE-2026-${String(seq).padStart(3, '0')}`,
      status: 'draft' as const,
      customer,
      customer_name: customer.name ?? '',
      tax_mode: body.tax_mode ?? 'standard',
      currency: body.currency ?? 'EUR',
      exchange_rate: body.exchange_rate,
      issue_date: invoiceDate,
      invoice_date: invoiceDate,
      due_date: addDays(invoiceDate, termsDays),
      payment_terms: String(termsDays),
      source_quote_id: body.source_quote_id,
      notes: body.notes ?? '',
      ...totals,
      tax_rate: 19,
      created_at: new Date().toISOString(),
    }
    mockInvoices.invoices.unshift(newInvoice as never)
    mockInvoices.total = mockInvoices.invoices.length
    return HttpResponse.json({ invoice: newInvoice }, { status: 201 })
  }),

  // Update invoice — stateful: recompute totals when line items change.
  http.put(`${API}/api/v1/finance/invoices/:id`, async ({ params, request }) => {
    const existing = mockInvoices.invoices.find((inv) => inv.id === params.id) as Record<string, unknown> | undefined
    if (!existing) {
      return HttpResponse.json({ error: 'Invoice not found' }, { status: 404 })
    }
    const body = (await request.json()) as Record<string, unknown>
    Object.assign(existing, body)
    if (Array.isArray(body.line_items)) {
      Object.assign(existing, computeTotals(body.line_items as RawLineItem[]))
    }
    if (body.payment_terms_days != null) {
      existing.payment_terms = String(body.payment_terms_days)
    }
    return HttpResponse.json({ invoice: existing })
  }),

  // Send invoice — stateful
  http.post(`${API}/api/v1/finance/invoices/:id/send`, ({ params }) => {
    const existing = mockInvoices.invoices.find((inv) => inv.id === params.id) as Record<string, unknown> | undefined
    if (!existing) {
      return HttpResponse.json({ error: 'Invoice not found' }, { status: 404 })
    }
    existing.status = 'sent'
    return HttpResponse.json({ invoice: existing })
  }),

  // Mark invoice as paid — stateful
  http.post(`${API}/api/v1/finance/invoices/:id/mark-paid`, ({ params }) => {
    const existing = mockInvoices.invoices.find((inv) => inv.id === params.id) as Record<string, unknown> | undefined
    if (!existing) {
      return HttpResponse.json({ error: 'Invoice not found' }, { status: 404 })
    }
    existing.status = 'paid'
    return HttpResponse.json({ invoice: existing })
  }),

  // Cancel invoice — stateful
  http.post(`${API}/api/v1/finance/invoices/:id/cancel`, ({ params }) => {
    const existing = mockInvoices.invoices.find((inv) => inv.id === params.id) as Record<string, unknown> | undefined
    if (!existing) {
      return HttpResponse.json({ error: 'Invoice not found' }, { status: 404 })
    }
    existing.status = 'cancelled'
    return HttpResponse.json({ invoice: existing })
  }),

  // Invoice payments list — stateful (P2.5d)
  http.get(`${API}/api/v1/finance/invoices/:id/payments`, ({ params }) => {
    const list = paymentsStore[String(params.id)] ?? []
    return HttpResponse.json({ payments: list, total: list.length })
  }),

  // Record a payment — stateful. Marks the invoice paid once fully covered and
  // settles any open dunnings for it.
  http.post(`${API}/api/v1/finance/invoices/:id/payments`, async ({ params, request }) => {
    const invoiceId = String(params.id)
    const invoice = mockInvoices.invoices.find((i) => i.id === invoiceId) as Record<string, unknown> | undefined
    if (!invoice) return HttpResponse.json({ error: 'Invoice not found' }, { status: 404 })
    const body = (await request.json()) as Record<string, unknown>
    const payment: Payment = {
      id: `pay-${Date.now()}`,
      invoice_id: invoiceId,
      amount: String(body.amount ?? '0'),
      payment_date: (body.payment_date as string) || todayStr(),
      method: (body.method as Payment['method']) ?? 'bank_transfer',
      reference: (body.reference as string) || undefined,
      notes: (body.notes as string) || undefined,
    }
    paymentsStore[invoiceId] = [...(paymentsStore[invoiceId] ?? []), payment]
    // Fully covered → mark paid + settle open dunnings.
    if (paidTotal(invoiceId) >= grossOf(invoice) - 0.005) {
      invoice.status = 'paid'
      for (const d of dunningsStore) {
        if (d.invoice_id === invoiceId && d.status !== 'paid') d.status = 'paid'
      }
    }
    return HttpResponse.json({ payment }, { status: 201 })
  }),

  // Delete a payment — stateful
  http.delete(`${API}/api/v1/finance/payments/:id`, ({ params }) => {
    for (const key of Object.keys(paymentsStore)) {
      paymentsStore[key] = (paymentsStore[key] ?? []).filter((p) => p.id !== String(params.id))
    }
    return HttpResponse.json({})
  }),

  // ---- Quotes ----

  // List quotes
  http.get(`${API}/api/v1/finance/quotes`, ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('status')

    let filtered = [...mockQuotes.quotes]
    if (status) {
      filtered = filtered.filter((q) => q.status === status)
    }

    return HttpResponse.json({
      quotes: filtered,
      total: filtered.length,
      page: 1,
      per_page: 50,
    })
  }),

  // Quote detail — normalised to the detail shape (line_items + tax_breakdown)
  http.get(`${API}/api/v1/finance/quotes/:id`, ({ params }) => {
    const quote = mockQuotes.quotes.find((q) => q.id === params.id)
    if (!quote) {
      return HttpResponse.json({ error: 'Quote not found' }, { status: 404 })
    }
    return HttpResponse.json({ quote: normalizeDoc(quote as Record<string, unknown>) })
  }),

  // Create quote
  http.post(`${API}/api/v1/finance/quotes`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const newQuote = {
      id: `qt-${Date.now()}`,
      number: `AN-2026-${String(mockQuotes.quotes.length + 1).padStart(3, '0')}`,
      quote_number: `AN-2026-${String(mockQuotes.quotes.length + 1).padStart(3, '0')}`,
      status: 'draft',
      ...body,
      created_at: new Date().toISOString(),
    }
    mockQuotes.quotes.unshift(newQuote as never)
    mockQuotes.total = mockQuotes.quotes.length
    return HttpResponse.json({ quote: newQuote }, { status: 201 })
  }),

  // Update quote — stateful
  http.put(`${API}/api/v1/finance/quotes/:id`, async ({ params, request }) => {
    const quote = mockQuotes.quotes.find((q) => q.id === params.id) as Record<string, unknown> | undefined
    if (!quote) return HttpResponse.json({ error: 'Quote not found' }, { status: 404 })
    Object.assign(quote, (await request.json()) as Record<string, unknown>)
    return HttpResponse.json({ quote: normalizeDoc(quote) })
  }),

  // Delete quote — stateful
  http.delete(`${API}/api/v1/finance/quotes/:id`, ({ params }) => {
    const idx = mockQuotes.quotes.findIndex((q) => q.id === params.id)
    if (idx >= 0) mockQuotes.quotes.splice(idx, 1)
    mockQuotes.total = mockQuotes.quotes.length
    return HttpResponse.json({})
  }),

  // Quote lifecycle — stateful status transitions
  http.post(`${API}/api/v1/finance/quotes/:id/send`, ({ params }) => {
    const quote = mockQuotes.quotes.find((q) => q.id === params.id) as Record<string, unknown> | undefined
    if (!quote) return HttpResponse.json({ error: 'Quote not found' }, { status: 404 })
    quote.status = 'sent'
    return HttpResponse.json({ quote: normalizeDoc(quote) })
  }),
  http.post(`${API}/api/v1/finance/quotes/:id/accept`, ({ params }) => {
    const quote = mockQuotes.quotes.find((q) => q.id === params.id) as Record<string, unknown> | undefined
    if (!quote) return HttpResponse.json({ error: 'Quote not found' }, { status: 404 })
    quote.status = 'accepted'
    return HttpResponse.json({ quote: normalizeDoc(quote) })
  }),
  http.post(`${API}/api/v1/finance/quotes/:id/reject`, ({ params }) => {
    const quote = mockQuotes.quotes.find((q) => q.id === params.id) as Record<string, unknown> | undefined
    if (!quote) return HttpResponse.json({ error: 'Quote not found' }, { status: 404 })
    quote.status = 'rejected'
    return HttpResponse.json({ quote: normalizeDoc(quote) })
  }),

  // Convert an accepted quote into a draft invoice — emits a real invoice.
  http.post(`${API}/api/v1/finance/quotes/:id/convert`, ({ params }) => {
    const quote = mockQuotes.quotes.find((q) => q.id === params.id) as Record<string, unknown> | undefined
    if (!quote) return HttpResponse.json({ error: 'Quote not found' }, { status: 404 })
    const norm = normalizeDoc(quote)
    const today = new Date().toISOString().split('T')[0]
    const seq = mockInvoices.invoices.length + 1
    const invoiceNumber = `RE-2026-${String(seq).padStart(3, '0')}`
    const newInvoice = {
      id: `inv-${Date.now()}`,
      invoice_number: invoiceNumber,
      number: invoiceNumber,
      status: 'draft' as const,
      customer: norm.customer,
      customer_name: (norm.customer as Record<string, unknown>).name ?? '',
      invoice_date: today,
      issue_date: today,
      due_date: addDays(today, 14),
      currency: quote.currency ?? 'EUR',
      payment_terms: 14,
      line_items: norm.line_items,
      items: norm.line_items,
      tax_breakdown: norm.tax_breakdown,
      total_net: Number((norm.tax_breakdown as Record<string, unknown>).subtotal ?? quote.total_net ?? 0),
      total_gross: Number((norm.tax_breakdown as Record<string, unknown>).gross_total ?? quote.total_gross ?? 0),
      source_quote_id: quote.id,
      created_at: new Date().toISOString(),
    }
    mockInvoices.invoices.unshift(newInvoice as never)
    mockInvoices.total = mockInvoices.invoices.length
    quote.status = 'accepted'
    quote.converted_invoice_id = newInvoice.id
    quote.converted_invoice_number = invoiceNumber
    return HttpResponse.json({ invoice: newInvoice })
  }),

  // ---- Credit Notes ----

  // List credit notes — normalise so every note carries a resolvable
  // original_invoice_id + invoice_number (seed data only had invoice_number).
  http.get(`${API}/api/v1/finance/credit-notes`, () => {
    const normalised = mockCreditNotes.credit_notes.map((cn) => {
      const raw = cn as Record<string, unknown>
      const linked = mockInvoices.invoices.find(
        (inv) => inv.invoice_number === raw.invoice_number || inv.id === raw.original_invoice_id,
      )
      return {
        ...raw,
        original_invoice_id: raw.original_invoice_id ?? linked?.id ?? raw.invoice_number ?? '',
        invoice_number: raw.invoice_number ?? linked?.invoice_number ?? '',
        status: raw.status === 'issued' ? 'sent' : raw.status,
      }
    })
    return HttpResponse.json({ credit_notes: normalised, total: normalised.length })
  }),

  // Credit note detail — normalised (line_items + tax_breakdown + linked invoice)
  http.get(`${API}/api/v1/finance/credit-notes/:id`, ({ params }) => {
    const cn = mockCreditNotes.credit_notes.find((c) => c.id === params.id) as
      | Record<string, unknown>
      | undefined
    if (!cn) return HttpResponse.json({ error: 'Credit note not found' }, { status: 404 })
    const linked = mockInvoices.invoices.find(
      (inv) => inv.invoice_number === cn.invoice_number || inv.id === cn.original_invoice_id,
    )
    const normalised = {
      ...normalizeDoc(cn),
      original_invoice_id: cn.original_invoice_id ?? linked?.id ?? cn.invoice_number ?? '',
      invoice_number: cn.invoice_number ?? linked?.invoice_number ?? '',
      status: cn.status === 'issued' ? 'sent' : cn.status,
    }
    return HttpResponse.json({ credit_note: normalised })
  }),

  // Create credit note — stateful. With is_storno, cancels the linked invoice.
  http.post(`${API}/api/v1/finance/credit-notes`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const linked = mockInvoices.invoices.find((inv) => inv.id === body.original_invoice_id) as
      | Record<string, unknown>
      | undefined
    const totals = computeTotals((body.line_items as RawLineItem[]) ?? [])
    const seq = mockCreditNotes.credit_notes.length + 1
    const newNote = {
      id: `cn-${Date.now()}`,
      number: `GS-2026-${String(seq).padStart(3, '0')}`,
      credit_note_number: `GS-2026-${String(seq).padStart(3, '0')}`,
      status: 'draft' as const,
      original_invoice_id: (body.original_invoice_id as string) ?? '',
      invoice_number: (linked?.invoice_number as string) ?? '',
      customer: linked?.customer ?? { name: '', address: '', email: '' },
      customer_name: (linked?.customer as Record<string, unknown> | undefined)?.name ?? '',
      tax_mode: body.tax_mode ?? 'standard',
      currency: linked?.currency ?? 'EUR',
      is_storno: Boolean(body.is_storno),
      reason: body.reason ?? '',
      ...totals,
      created_at: new Date().toISOString(),
    }
    mockCreditNotes.credit_notes.unshift(newNote as never)
    mockCreditNotes.total = mockCreditNotes.credit_notes.length
    // Storno semantics: a full credit note cancels the original invoice.
    if (body.is_storno && linked) {
      linked.status = 'cancelled'
    }
    return HttpResponse.json({ credit_note: newNote }, { status: 201 })
  }),

  // Send credit note — stateful
  http.post(`${API}/api/v1/finance/credit-notes/:id/send`, ({ params }) => {
    const existing = mockCreditNotes.credit_notes.find((cn) => cn.id === params.id) as
      | Record<string, unknown>
      | undefined
    if (!existing) {
      return HttpResponse.json({ error: 'Credit note not found' }, { status: 404 })
    }
    existing.status = 'sent'
    return HttpResponse.json({ credit_note: existing })
  }),

  // ---- Dashboard ----

  // Finance dashboard KPIs
  http.get(`${API}/api/v1/finance/dashboard`, () => {
    return HttpResponse.json({ dashboard: mockFinanceDashboard })
  }),

  // ---- Settings ----

  // Company finance settings
  http.get(`${API}/api/v1/finance/settings`, () => {
    return HttpResponse.json(mockFinanceSettings)
  }),

  // ---- Dunning (finanzen P2.5d) — stateful ----

  // Dunning list — supports invoice_id / level / status filters.
  http.get(`${API}/api/v1/finance/dunning`, ({ request }) => {
    const url = new URL(request.url)
    const invoiceId = url.searchParams.get('invoice_id')
    const level = url.searchParams.get('level')
    const status = url.searchParams.get('status')
    let list = [...dunningsStore]
    if (invoiceId) list = list.filter((d) => d.invoice_id === invoiceId)
    if (level) list = list.filter((d) => d.level === Number(level))
    if (status) list = list.filter((d) => d.status === status)
    return HttpResponse.json({ dunnings: list, total: list.length })
  }),

  // Detect overdue invoices and create level-1 dunnings for those without one.
  http.post(`${API}/api/v1/finance/dunning/detect`, () => {
    const created: DunningRecord[] = []
    for (const inv of mockInvoices.invoices as Record<string, unknown>[]) {
      if (!isOverdue(inv)) continue
      const has = dunningsStore.some((d) => d.invoice_id === String(inv.id) && d.status !== 'paid')
      if (has) continue
      const d = makeDunning(inv, 1, 'draft')
      dunningsStore.unshift(d)
      created.push(d)
    }
    return HttpResponse.json({ dunnings: created })
  }),

  // Send a dunning — stateful
  http.post(`${API}/api/v1/finance/dunning/:id/send`, ({ params }) => {
    const d = dunningsStore.find((x) => x.id === params.id)
    if (!d) return HttpResponse.json({ error: 'Dunning not found' }, { status: 404 })
    d.status = 'sent'
    d.sent_at = todayStr()
    return HttpResponse.json({ dunning: d })
  }),

  // Escalate a dunning to the next level — recomputes fee + interest.
  http.post(`${API}/api/v1/finance/dunning/:id/escalate`, ({ params }) => {
    const d = dunningsStore.find((x) => x.id === params.id)
    if (!d) return HttpResponse.json({ error: 'Dunning not found' }, { status: 404 })
    if (d.level < 3) {
      d.level = (d.level + 1) as 1 | 2 | 3
      const inv = mockInvoices.invoices.find((i) => i.id === d.invoice_id) as Record<string, unknown> | undefined
      d.fee = feeForLevel(d.level)
      d.interest = interestFor(inv ? grossOf(inv) : 0, d.level)
      d.status = 'draft'
      d.sent_at = undefined
    }
    return HttpResponse.json({ dunning: d })
  }),

  // Dunning PDF — real downloadable file
  http.get(`${API}/api/v1/finance/dunning/:id/pdf`, ({ params }) => {
    const d = dunningsStore.find((x) => x.id === params.id)
    if (!d) return HttpResponse.json({ error: 'Dunning not found' }, { status: 404 })
    const inv = mockInvoices.invoices.find((i) => i.id === d.invoice_id) as Record<string, unknown> | undefined
    const fmt = (v: unknown) => `${Number(v ?? 0).toLocaleString('de-DE', { minimumFractionDigits: 2 })} EUR`
    const lines: PdfLine[] = [
      { text: 'Zentria GmbH · Cosmi', size: 9 },
      { text: `${d.level}. Mahnung`, size: 20, gap: 6 },
      { text: `Rechnung: ${inv?.invoice_number ?? d.invoice_id}`, gap: 4 },
      { text: `Kunde: ${(inv?.customer as { name?: string })?.name ?? ''}`, gap: 8 },
      { text: `Offener Betrag: ${fmt(inv ? grossOf(inv) : 0)}`, size: 11 },
      { text: `Mahngebühr: ${fmt(d.fee)}`, size: 11 },
      { text: `Verzugszinsen: ${fmt(d.interest)}`, size: 11, gap: 6 },
      { text: 'Demo-Beleg (Cosmi) — finales Mahnschreiben folgt.', size: 8 },
    ]
    return pdfResponse(buildSimplePdf(lines), `Mahnung-${inv?.invoice_number ?? d.invoice_id}.pdf`)
  }),

  // Dunning config — flat shape the UI expects (P2.5d fix).
  http.get(`${API}/api/v1/finance/dunning/config`, () => {
    return HttpResponse.json({ config: dunningConfig })
  }),

  // Update dunning config — stateful
  http.put(`${API}/api/v1/finance/dunning/config`, async ({ request }) => {
    const body = (await request.json()) as Partial<DunningConfig>
    dunningConfig = { ...dunningConfig, ...body }
    return HttpResponse.json({ config: dunningConfig })
  }),

  // ---- Recurring invoices (finanzen P1) ----

  // List recurring schedules
  http.get(`${API}/api/v1/finance/recurring`, () => {
    return HttpResponse.json({
      recurring: mockRecurringInvoices.recurring,
      total: mockRecurringInvoices.recurring.length,
    })
  }),

  // Create recurring schedule
  http.post(`${API}/api/v1/finance/recurring`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const totals = computeTotals((body.line_items as RawLineItem[]) ?? [])
    const startDate = (body.start_date as string) || new Date().toISOString().split('T')[0]
    const customer = (body.customer ?? { name: '', address: '', email: '' }) as Record<string, unknown>
    const newRec = {
      id: `rec-${Date.now()}`,
      title: (body.title as string) || (customer.name as string) || 'Wiederkehrende Rechnung',
      customer,
      tax_mode: body.tax_mode ?? 'standard',
      currency: body.currency ?? 'EUR',
      interval: body.interval ?? 'monthly',
      status: 'active' as const,
      start_date: startDate,
      end_date: body.end_date || undefined,
      next_run: startDate,
      payment_terms_days: Number(body.payment_terms_days ?? 14),
      generated_count: 0,
      notes: body.notes ?? '',
      line_items: totals.line_items,
      tax_breakdown: totals.tax_breakdown,
      created_at: new Date().toISOString(),
    }
    mockRecurringInvoices.recurring.unshift(newRec as never)
    mockRecurringInvoices.total = mockRecurringInvoices.recurring.length
    return HttpResponse.json({ recurring: newRec }, { status: 201 })
  }),

  // Update recurring schedule
  http.put(`${API}/api/v1/finance/recurring/:id`, async ({ params, request }) => {
    const rec = mockRecurringInvoices.recurring.find((r) => r.id === params.id) as
      | Record<string, unknown>
      | undefined
    if (!rec) return HttpResponse.json({ error: 'Recurring not found' }, { status: 404 })
    const body = (await request.json()) as Record<string, unknown>
    Object.assign(rec, body)
    if (Array.isArray(body.line_items)) {
      const totals = computeTotals(body.line_items as RawLineItem[])
      rec.line_items = totals.line_items
      rec.tax_breakdown = totals.tax_breakdown
    }
    return HttpResponse.json({ recurring: rec })
  }),

  // Delete recurring schedule
  http.delete(`${API}/api/v1/finance/recurring/:id`, ({ params }) => {
    const idx = mockRecurringInvoices.recurring.findIndex((r) => r.id === params.id)
    if (idx >= 0) mockRecurringInvoices.recurring.splice(idx, 1)
    mockRecurringInvoices.total = mockRecurringInvoices.recurring.length
    return HttpResponse.json({})
  }),

  // Pause / resume schedule — stateful
  http.post(`${API}/api/v1/finance/recurring/:id/pause`, ({ params }) => {
    const rec = mockRecurringInvoices.recurring.find((r) => r.id === params.id) as
      | Record<string, unknown>
      | undefined
    if (!rec) return HttpResponse.json({ error: 'Recurring not found' }, { status: 404 })
    rec.status = 'paused'
    return HttpResponse.json({ recurring: rec })
  }),
  http.post(`${API}/api/v1/finance/recurring/:id/resume`, ({ params }) => {
    const rec = mockRecurringInvoices.recurring.find((r) => r.id === params.id) as
      | Record<string, unknown>
      | undefined
    if (!rec) return HttpResponse.json({ error: 'Recurring not found' }, { status: 404 })
    if (rec.status !== 'ended') rec.status = 'active'
    return HttpResponse.json({ recurring: rec })
  }),

  // Generate next invoice from a schedule — stateful across resources:
  // emits a real draft invoice into the invoice list + advances next_run.
  http.post(`${API}/api/v1/finance/recurring/:id/generate`, ({ params }) => {
    const rec = mockRecurringInvoices.recurring.find((r) => r.id === params.id) as
      | (Record<string, unknown> & {
          interval: 'weekly' | 'monthly' | 'quarterly' | 'yearly'
          next_run: string
          payment_terms_days: number
        })
      | undefined
    if (!rec) return HttpResponse.json({ error: 'Recurring not found' }, { status: 404 })

    const issueDate = new Date().toISOString().split('T')[0]
    const seq = mockInvoices.invoices.length + 1
    const breakdown = rec.tax_breakdown as Record<string, unknown> | undefined
    const customer = rec.customer as Record<string, unknown>
    const newInvoice = {
      id: `inv-${Date.now()}`,
      number: `RE-2026-${String(seq).padStart(3, '0')}`,
      invoice_number: `RE-2026-${String(seq).padStart(3, '0')}`,
      status: 'draft' as const,
      customer,
      customer_name: customer.name ?? '',
      tax_mode: rec.tax_mode ?? 'standard',
      currency: rec.currency ?? 'EUR',
      issue_date: issueDate,
      invoice_date: issueDate,
      due_date: addDays(issueDate, Number(rec.payment_terms_days ?? 14)),
      payment_terms: String(rec.payment_terms_days ?? 14),
      recurring_id: rec.id,
      notes: `Automatisch erzeugt aus „${rec.title}"`,
      line_items: rec.line_items,
      tax_breakdown: breakdown,
      total_net: Number(breakdown?.subtotal ?? 0),
      total_gross: Number(breakdown?.gross_total ?? 0),
      tax_rate: 19,
      created_at: new Date().toISOString(),
    }
    mockInvoices.invoices.unshift(newInvoice as never)
    mockInvoices.total = mockInvoices.invoices.length

    rec.generated_count = Number(rec.generated_count ?? 0) + 1
    rec.last_generated_at = issueDate
    rec.next_run = advanceByInterval(rec.next_run, rec.interval)

    return HttpResponse.json({ invoice: newInvoice, recurring: rec })
  }),

  // ---- Expenses (Ausgaben) — finanzen P2 ----
  // Swap-ready Mock: kein echtes Backend (siehe backend-gaps.md §finanzen).

  http.get(`${API}/api/v1/finance/expenses`, () => {
    return HttpResponse.json({ expenses: mockExpenses, total: mockExpenses.length })
  }),

  http.post(`${API}/api/v1/finance/expenses`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const newExpense = {
      id: `exp-${Date.now()}`,
      description: String(body.description ?? ''),
      amount: Number(body.amount ?? 0),
      date: (body.date as string) || new Date().toISOString().split('T')[0],
      category: String(body.category ?? 'Sonstiges'),
      supplier: String(body.supplier ?? ''),
      project: (body.project as string) || undefined,
      account: (body.account as string) || undefined,
      receipt: Boolean(body.receiptName),
      receiptName: (body.receiptName as string) || undefined,
      status: 'pending' as const,
    }
    mockExpenses.unshift(newExpense)
    return HttpResponse.json({ expense: newExpense }, { status: 201 })
  }),

  http.put(`${API}/api/v1/finance/expenses/:id`, async ({ params, request }) => {
    const exp = mockExpenses.find((e) => e.id === params.id)
    if (!exp) return HttpResponse.json({ error: 'Expense not found' }, { status: 404 })
    const body = (await request.json()) as Record<string, unknown>
    Object.assign(exp, body)
    if ('receiptName' in body) exp.receipt = Boolean(body.receiptName)
    return HttpResponse.json({ expense: exp })
  }),

  http.post(`${API}/api/v1/finance/expenses/:id/approve`, ({ params }) => {
    const exp = mockExpenses.find((e) => e.id === params.id)
    if (!exp) return HttpResponse.json({ error: 'Expense not found' }, { status: 404 })
    exp.status = 'approved'
    return HttpResponse.json({ expense: exp })
  }),

  http.post(`${API}/api/v1/finance/expenses/:id/reject`, ({ params }) => {
    const exp = mockExpenses.find((e) => e.id === params.id)
    if (!exp) return HttpResponse.json({ error: 'Expense not found' }, { status: 404 })
    exp.status = 'rejected'
    return HttpResponse.json({ expense: exp })
  }),

  // Beleg anhängen — swap-ready: nur Metadaten (Dateiname), kein Upload-Body.
  http.post(`${API}/api/v1/finance/expenses/:id/receipt`, async ({ params, request }) => {
    const exp = mockExpenses.find((e) => e.id === params.id)
    if (!exp) return HttpResponse.json({ error: 'Expense not found' }, { status: 404 })
    const body = (await request.json()) as Record<string, unknown>
    exp.receiptName = (body.receiptName as string) || undefined
    exp.receipt = Boolean(exp.receiptName)
    return HttpResponse.json({ expense: exp })
  }),

  http.delete(`${API}/api/v1/finance/expenses/:id`, ({ params }) => {
    const idx = mockExpenses.findIndex((e) => e.id === params.id)
    if (idx >= 0) mockExpenses.splice(idx, 1)
    return HttpResponse.json({})
  }),

  // ---- Transactions (Transaktionen) — finanzen P2 ----

  http.get(`${API}/api/v1/finance/transactions`, () => {
    return HttpResponse.json({ transactions: mockTransactions, total: mockTransactions.length })
  }),

  http.delete(`${API}/api/v1/finance/transactions/:id`, ({ params }) => {
    const idx = mockTransactions.findIndex((tx) => tx.id === params.id)
    if (idx >= 0) mockTransactions.splice(idx, 1)
    return HttpResponse.json({})
  }),

  // ---- PDF + Export (finanzen P2.5c) — return real downloadable files ----

  http.get(`${API}/api/v1/finance/invoices/:id/pdf`, ({ params }) => {
    const doc = mockInvoices.invoices.find((i) => i.id === params.id) as Record<string, unknown> | undefined
    if (!doc) return HttpResponse.json({ error: 'Invoice not found' }, { status: 404 })
    return pdfResponse(buildDocPdf(doc, 'RECHNUNG', 'Rechnungs-Nr'), `${doc.invoice_number ?? 'Rechnung'}.pdf`)
  }),

  http.get(`${API}/api/v1/finance/quotes/:id/pdf`, ({ params }) => {
    const doc = mockQuotes.quotes.find((q) => q.id === params.id) as Record<string, unknown> | undefined
    if (!doc) return HttpResponse.json({ error: 'Quote not found' }, { status: 404 })
    return pdfResponse(buildDocPdf(doc, 'ANGEBOT', 'Angebots-Nr'), `${doc.quote_number ?? 'Angebot'}.pdf`)
  }),

  http.get(`${API}/api/v1/finance/credit-notes/:id/pdf`, ({ params }) => {
    const doc = mockCreditNotes.credit_notes.find((c) => c.id === params.id) as Record<string, unknown> | undefined
    if (!doc) return HttpResponse.json({ error: 'Credit note not found' }, { status: 404 })
    return pdfResponse(buildDocPdf(doc, 'GUTSCHRIFT', 'Gutschrift-Nr'), `${doc.credit_note_number ?? 'Gutschrift'}.pdf`)
  }),

  // DATEV-Buchungsstapel als echte CSV (vereinfacht; volle EXTF-Spec = P3).
  http.get(`${API}/api/v1/finance/export/datev`, () => {
    const csv = buildDatevCsv(mockInvoices.invoices as never)
    return new HttpResponse('﻿' + csv, {
      headers: { 'Content-Type': 'text/csv;charset=utf-8', 'Content-Disposition': 'attachment; filename="EXTF_Buchungsstapel.csv"' },
    })
  }),

  // ---- Banking (FinAPI placeholder) — finanzen P2.5e ----
  // Swap-ready Mock: kein echtes Backend (FinAPI-Integration = P5).

  http.get(`${API}/api/v1/finance/bank-accounts`, () => {
    return HttpResponse.json({ accounts: mockBanking.accounts })
  }),

  // Connect a bank account (FinAPI placeholder) — stateful: balance + sync show.
  http.post(`${API}/api/v1/finance/bank-accounts/:id/connect`, ({ params }) => {
    const acc = mockBanking.accounts.find((a) => a.id === params.id)
    if (!acc) return HttpResponse.json({ error: 'Account not found' }, { status: 404 })
    acc.connected = true
    acc.lastSync = new Date().toISOString().split('T')[0]
    return HttpResponse.json({ account: acc })
  }),

  http.get(`${API}/api/v1/finance/bank-transactions`, ({ request }) => {
    const status = new URL(request.url).searchParams.get('status')
    let txs = [...mockBanking.transactions]
    if (status && status !== 'all') {
      txs = txs.filter((tx) => tx.matchStatus === status)
    }
    return HttpResponse.json({ transactions: txs })
  }),

  // Confirm a match — stateful. Optional invoice_number assigns a specific
  // invoice (manual matching from the detail modal).
  http.post(`${API}/api/v1/finance/bank-transactions/:id/match`, async ({ params, request }) => {
    const tx = mockBanking.transactions.find((t) => t.id === params.id)
    if (!tx) return HttpResponse.json({ error: 'Transaction not found' }, { status: 404 })
    const body = (await request.json().catch(() => ({}))) as { invoice_number?: string }
    if (body.invoice_number) tx.matchedInvoice = body.invoice_number
    tx.matchStatus = 'matched'
    return HttpResponse.json({ transaction: tx })
  }),

  // Reject a suggested match — stateful: clears the suggested invoice.
  http.post(`${API}/api/v1/finance/bank-transactions/:id/reject-match`, ({ params }) => {
    const tx = mockBanking.transactions.find((t) => t.id === params.id)
    if (!tx) return HttpResponse.json({ error: 'Transaction not found' }, { status: 404 })
    tx.matchStatus = 'unmatched'
    delete tx.matchedInvoice
    return HttpResponse.json({ transaction: tx })
  }),

  // ---- Belegkette (document chains) — finanzen P2.5e ----
  http.get(`${API}/api/v1/finance/document-chains`, () => {
    return HttpResponse.json({ chains: mockDocumentChains.chains })
  }),

  // ---- Stunden → Rechnung — unbilled time entries (finanzen P2.5e) ----
  // Finance-internal view onto the Zeiterfassung service; ?billed=false filters.
  http.get(`${API}/api/v1/finance/time-entries`, ({ request }) => {
    const billed = new URL(request.url).searchParams.get('billed')
    let entries = [...mockFinanceTimeEntries.entries]
    if (billed === 'false') entries = entries.filter((e) => !e.billed)
    return HttpResponse.json({ entries })
  }),
]
