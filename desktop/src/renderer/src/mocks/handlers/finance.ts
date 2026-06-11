import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import {
  mockInvoices,
  mockQuotes,
  mockCreditNotes,
  mockFinanceDashboard,
  mockFinanceSettings,
} from '../data/invoices'

const API = API_BASE_URL

export const financeHandlers = [
  // ---- Invoices ----

  // List invoices
  http.get(`${API}/api/v1/finance/invoices`, ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('status')
    const page = parseInt(url.searchParams.get('page') || '1', 10)
    const perPage = parseInt(url.searchParams.get('per_page') || '50', 10)

    let filtered = [...mockInvoices.invoices]
    if (status) {
      filtered = filtered.filter((inv) => inv.status === status)
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

  // Create invoice
  http.post(`${API}/api/v1/finance/invoices`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const newInvoice = {
      id: `inv-${Date.now()}`,
      number: `RE-2026-${String(mockInvoices.invoices.length + 1).padStart(3, '0')}`,
      status: 'draft',
      ...body,
      created_at: new Date().toISOString(),
    }
    return HttpResponse.json({ invoice: newInvoice }, { status: 201 })
  }),

  // Update invoice
  http.put(`${API}/api/v1/finance/invoices/:id`, async ({ params, request }) => {
    const existing = mockInvoices.invoices.find((inv) => inv.id === params.id)
    if (!existing) {
      return HttpResponse.json({ error: 'Invoice not found' }, { status: 404 })
    }
    const body = (await request.json()) as Record<string, unknown>
    const updated = { ...existing, ...body }
    return HttpResponse.json({ invoice: updated })
  }),

  // Send invoice
  http.post(`${API}/api/v1/finance/invoices/:id/send`, ({ params }) => {
    const existing = mockInvoices.invoices.find((inv) => inv.id === params.id)
    if (!existing) {
      return HttpResponse.json({ error: 'Invoice not found' }, { status: 404 })
    }
    return HttpResponse.json({ invoice: { ...existing, status: 'sent' } })
  }),

  // Mark invoice as paid
  http.post(`${API}/api/v1/finance/invoices/:id/mark-paid`, ({ params }) => {
    const existing = mockInvoices.invoices.find((inv) => inv.id === params.id)
    if (!existing) {
      return HttpResponse.json({ error: 'Invoice not found' }, { status: 404 })
    }
    return HttpResponse.json({ invoice: { ...existing, status: 'paid' } })
  }),

  // Invoice payments list
  http.get(`${API}/api/v1/finance/invoices/:id/payments`, () => {
    return HttpResponse.json({ payments: [], total: 0 })
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

  // Quote detail
  http.get(`${API}/api/v1/finance/quotes/:id`, ({ params }) => {
    const quote = mockQuotes.quotes.find((q) => q.id === params.id)
    if (!quote) {
      return HttpResponse.json({ error: 'Quote not found' }, { status: 404 })
    }
    return HttpResponse.json({ quote })
  }),

  // Create quote
  http.post(`${API}/api/v1/finance/quotes`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const newQuote = {
      id: `qt-${Date.now()}`,
      number: `AN-2026-${String(mockQuotes.quotes.length + 1).padStart(3, '0')}`,
      status: 'draft',
      ...body,
      created_at: new Date().toISOString(),
    }
    return HttpResponse.json({ quote: newQuote }, { status: 201 })
  }),

  // ---- Credit Notes ----

  // List credit notes
  http.get(`${API}/api/v1/finance/credit-notes`, () => {
    return HttpResponse.json(mockCreditNotes)
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

  // ---- Dunning ----

  // Dunning list (empty)
  http.get(`${API}/api/v1/finance/dunnings`, () => {
    return HttpResponse.json({ dunnings: [], total: 0 })
  }),

  // Dunning config
  http.get(`${API}/api/v1/finance/dunning-config`, () => {
    return HttpResponse.json({
      config: {
        enabled: true,
        levels: [
          { level: 1, days_after_due: 7, fee: 0, template: 'Zahlungserinnerung' },
          { level: 2, days_after_due: 14, fee: 5.0, template: '1. Mahnung' },
          { level: 3, days_after_due: 28, fee: 10.0, template: '2. Mahnung' },
        ],
      },
    })
  }),
]
