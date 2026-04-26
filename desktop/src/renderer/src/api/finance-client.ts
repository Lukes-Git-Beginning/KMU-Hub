/**
 * Finance API client -- typed fetch wrapper for all finance HTTP endpoints.
 *
 * Follows the same auth/refresh/offline pattern as document-client.ts.
 * Gateway routes: /api/v1/finance/*
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  Quote,
  Invoice,
  CreditNote,
  Payment,
  DunningRecord,
  DunningConfig,
  CompanySettings,
  FinanceDashboard,
  CreateQuoteRequest,
  UpdateQuoteRequest,
  CreateInvoiceRequest,
  UpdateInvoiceRequest,
  CreateCreditNoteRequest,
  RecordPaymentRequest,
  UpdateDunningConfigRequest,
  UpdateCompanySettingsRequest,
  ListQuotesParams,
  ListInvoicesParams,
  ListCreditNotesParams,
  ListDunningsParams,
  DashboardParams,
  ExportDATEVParams,
  ListQuotesResponse,
  ListInvoicesResponse,
  ListCreditNotesResponse,
  ListPaymentsResponse,
  ListDunningsResponse,
  JournalSummary,
  ValidateInvoiceNumberResult,
  PaymentStats,
  LockInvoiceResponse,
  DunningNoticeResponse,
  DateRangeParams,
} from '@/types/finance-types'

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

const MUTATION_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH'])

class OfflineError extends Error {
  constructor() {
    super('Änderungen sind offline nicht möglich.')
    this.name = 'OfflineError'
  }
}

let refreshPromise: Promise<string | null> | null = null

async function getToken(): Promise<string | undefined> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken
}

async function refreshTokenFn(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  const store = useAuthStore.getState()
  if (!refreshPromise) {
    refreshPromise = store.refreshToken().finally(() => {
      refreshPromise = null
    })
  }
  return refreshPromise
}

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const method = options.method ?? 'GET'

  if (!navigator.onLine && MUTATION_METHODS.has(method)) {
    throw new OfflineError()
  }

  const token = await getToken()
  const headers = new Headers(options.headers)
  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (options.body && typeof options.body === 'string') {
    headers.set('Content-Type', 'application/json')
  }

  let res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers })

  // Handle 401 with transparent refresh
  if (res.status === 401 && !path.includes('/auth/')) {
    const newToken = await refreshTokenFn()
    if (!newToken) {
      const { useAuthStore } = await import('@/stores/auth')
      useAuthStore.getState().logout()
      throw new Error('Session abgelaufen')
    }
    headers.set('Authorization', `Bearer ${newToken}`)
    res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers })
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error ?? body.message ?? `HTTP ${res.status}`)
  }

  // 204 No Content
  if (res.status === 204) return {} as T

  return res.json() as Promise<T>
}

async function requestBlob(
  path: string,
  options: RequestInit = {},
): Promise<Blob> {
  const method = options.method ?? 'GET'

  if (!navigator.onLine && MUTATION_METHODS.has(method)) {
    throw new OfflineError()
  }

  const token = await getToken()
  const headers = new Headers(options.headers)
  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (options.body && typeof options.body === 'string') {
    headers.set('Content-Type', 'application/json')
  }

  let res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers })

  if (res.status === 401 && !path.includes('/auth/')) {
    const newToken = await refreshTokenFn()
    if (!newToken) {
      const { useAuthStore } = await import('@/stores/auth')
      useAuthStore.getState().logout()
      throw new Error('Session abgelaufen')
    }
    headers.set('Authorization', `Bearer ${newToken}`)
    res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers })
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error ?? body.message ?? `HTTP ${res.status}`)
  }

  return res.blob()
}

function qs(params: Record<string, unknown>): string {
  const entries = Object.entries(params).filter(
    ([, v]) => v !== undefined && v !== '' && v !== null,
  )
  if (entries.length === 0) return ''
  return (
    '?' +
    new URLSearchParams(entries.map(([k, v]) => [k, String(v)])).toString()
  )
}

// ---------------------------------------------------------------------------
// Company Settings
// ---------------------------------------------------------------------------

export const financeSettingsApi = {
  get() {
    return request<{ settings: CompanySettings }>('/api/v1/finance/settings')
  },

  update(data: UpdateCompanySettingsRequest) {
    return request<{ settings: CompanySettings }>('/api/v1/finance/settings', {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  },
}

// ---------------------------------------------------------------------------
// Quotes
// ---------------------------------------------------------------------------

export const financeQuoteApi = {
  list(params: ListQuotesParams = {}) {
    return request<ListQuotesResponse>(
      `/api/v1/finance/quotes${qs(params as Record<string, unknown>)}`,
    )
  },

  get(id: string) {
    return request<{ quote: Quote }>(`/api/v1/finance/quotes/${id}`)
  },

  create(data: CreateQuoteRequest) {
    return request<{ quote: Quote }>('/api/v1/finance/quotes', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },

  update(id: string, data: UpdateQuoteRequest) {
    return request<{ quote: Quote }>(`/api/v1/finance/quotes/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  },

  delete(id: string) {
    return request<Record<string, never>>(`/api/v1/finance/quotes/${id}`, {
      method: 'DELETE',
    })
  },

  send(id: string) {
    return request<{ quote: Quote }>(`/api/v1/finance/quotes/${id}/send`, {
      method: 'POST',
    })
  },

  accept(id: string) {
    return request<{ quote: Quote }>(`/api/v1/finance/quotes/${id}/accept`, {
      method: 'POST',
    })
  },

  reject(id: string) {
    return request<{ quote: Quote }>(`/api/v1/finance/quotes/${id}/reject`, {
      method: 'POST',
    })
  },

  convertToInvoice(id: string) {
    return request<{ invoice: Invoice }>(
      `/api/v1/finance/quotes/${id}/convert`,
      { method: 'POST' },
    )
  },

  getPDF(id: string) {
    return requestBlob(`/api/v1/finance/quotes/${id}/pdf`)
  },
}

// ---------------------------------------------------------------------------
// Invoices
// ---------------------------------------------------------------------------

export const financeInvoiceApi = {
  list(params: ListInvoicesParams = {}) {
    return request<ListInvoicesResponse>(
      `/api/v1/finance/invoices${qs(params as Record<string, unknown>)}`,
    )
  },

  get(id: string) {
    return request<{ invoice: Invoice }>(`/api/v1/finance/invoices/${id}`)
  },

  create(data: CreateInvoiceRequest) {
    return request<{ invoice: Invoice }>('/api/v1/finance/invoices', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },

  update(id: string, data: UpdateInvoiceRequest) {
    return request<{ invoice: Invoice }>(`/api/v1/finance/invoices/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  },

  send(id: string) {
    return request<{ invoice: Invoice }>(
      `/api/v1/finance/invoices/${id}/send`,
      { method: 'POST' },
    )
  },

  markPaid(id: string) {
    return request<{ invoice: Invoice }>(
      `/api/v1/finance/invoices/${id}/mark-paid`,
      { method: 'POST' },
    )
  },

  cancel(id: string) {
    return request<{ invoice: Invoice }>(
      `/api/v1/finance/invoices/${id}/cancel`,
      { method: 'POST' },
    )
  },

  getPDF(id: string) {
    return requestBlob(`/api/v1/finance/invoices/${id}/pdf`)
  },
}

// ---------------------------------------------------------------------------
// Credit Notes
// ---------------------------------------------------------------------------

export const financeCreditNoteApi = {
  list(params: ListCreditNotesParams = {}) {
    return request<ListCreditNotesResponse>(
      `/api/v1/finance/credit-notes${qs(params as Record<string, unknown>)}`,
    )
  },

  get(id: string) {
    return request<{ credit_note: CreditNote }>(
      `/api/v1/finance/credit-notes/${id}`,
    )
  },

  create(data: CreateCreditNoteRequest) {
    return request<{ credit_note: CreditNote }>(
      '/api/v1/finance/credit-notes',
      {
        method: 'POST',
        body: JSON.stringify(data),
      },
    )
  },

  send(id: string) {
    return request<{ credit_note: CreditNote }>(
      `/api/v1/finance/credit-notes/${id}/send`,
      { method: 'POST' },
    )
  },

  getPDF(id: string) {
    return requestBlob(`/api/v1/finance/credit-notes/${id}/pdf`)
  },
}

// ---------------------------------------------------------------------------
// Payments
// ---------------------------------------------------------------------------

export const financePaymentApi = {
  list(invoiceId: string) {
    return request<ListPaymentsResponse>(
      `/api/v1/finance/invoices/${invoiceId}/payments`,
    )
  },

  record(invoiceId: string, data: RecordPaymentRequest) {
    return request<{ payment: Payment }>(
      `/api/v1/finance/invoices/${invoiceId}/payments`,
      {
        method: 'POST',
        body: JSON.stringify(data),
      },
    )
  },

  delete(id: string) {
    return request<Record<string, never>>(`/api/v1/finance/payments/${id}`, {
      method: 'DELETE',
    })
  },
}

// ---------------------------------------------------------------------------
// Dunning
// ---------------------------------------------------------------------------

export const financeDunningApi = {
  list(params: ListDunningsParams = {}) {
    return request<ListDunningsResponse>(
      `/api/v1/finance/dunnings${qs(params as Record<string, unknown>)}`,
    )
  },

  detect() {
    return request<{ dunnings: DunningRecord[] }>(
      '/api/v1/finance/dunnings/detect',
      { method: 'POST' },
    )
  },

  send(id: string) {
    return request<{ dunning: DunningRecord }>(
      `/api/v1/finance/dunnings/${id}/send`,
      { method: 'POST' },
    )
  },

  escalate(id: string) {
    return request<{ dunning: DunningRecord }>(
      `/api/v1/finance/dunnings/${id}/escalate`,
      { method: 'POST' },
    )
  },

  getPDF(id: string) {
    return requestBlob(`/api/v1/finance/dunnings/${id}/pdf`)
  },

  getConfig() {
    return request<{ config: DunningConfig }>('/api/v1/finance/dunning-config')
  },

  updateConfig(data: UpdateDunningConfigRequest) {
    return request<{ config: DunningConfig }>('/api/v1/finance/dunning-config', {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  },
}

// ---------------------------------------------------------------------------
// Dashboard & Export
// ---------------------------------------------------------------------------

export const financeDashboardApi = {
  get(params: DashboardParams = {}) {
    return request<{ dashboard: FinanceDashboard }>(
      `/api/v1/finance/dashboard${qs(params as Record<string, unknown>)}`,
    )
  },
}

export const financeExportApi = {
  exportDATEV(params: ExportDATEVParams) {
    return requestBlob(
      `/api/v1/finance/export/datev${qs(params as Record<string, unknown>)}`,
    )
  },
}

// ---------------------------------------------------------------------------
// Deal-to-Quote
// ---------------------------------------------------------------------------

export const financeDealApi = {
  createQuoteFromDeal(dealId: string) {
    return request<{ quote: Quote }>(
      `/api/v1/finance/deals/${dealId}/quote`,
      { method: 'POST' },
    )
  },
}

// ---------------------------------------------------------------------------
// GoBD Journal & Compliance (Sprint 2 / Wave 1.B)
// ---------------------------------------------------------------------------

export const financeGoBDApi = {
  /** GET /api/v1/finance/journal/summary?year=YYYY */
  getJournalSummary(year: number) {
    return request<JournalSummary>(
      `/api/v1/finance/journal/summary?year=${year}`,
    )
  },

  /** GET /api/v1/finance/invoices/validate-number?number=RE-2026-0001 */
  validateInvoiceNumber(number: string) {
    return request<ValidateInvoiceNumberResult>(
      `/api/v1/finance/invoices/validate-number?number=${encodeURIComponent(number)}`,
    )
  },

  /** POST /api/v1/finance/invoices/{id}/lock */
  lockInvoice(id: string) {
    return request<LockInvoiceResponse>(`/api/v1/finance/invoices/${id}/lock`, {
      method: 'POST',
    })
  },

  /** GET /api/v1/finance/stats/payments?from=YYYY-MM-DD&to=YYYY-MM-DD */
  getPaymentStats(params: DateRangeParams) {
    return request<PaymentStats>(
      `/api/v1/finance/stats/payments?from=${params.from_date}&to=${params.to_date}`,
    )
  },

  /** PUT /api/v1/finance/dunning/{id}/status */
  updateDunningStatus(id: string, status: string) {
    return request<DunningRecord>(`/api/v1/finance/dunning/${id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status }),
    })
  },

  /** POST /api/v1/finance/dunning/{id}/notice */
  sendDunningNotice(id: string) {
    return request<DunningNoticeResponse>(
      `/api/v1/finance/dunning/${id}/notice`,
      { method: 'POST' },
    )
  },

  /** POST /api/v1/finance/export/gobd — downloads CSV blob */
  generateGoBDExport(params: DateRangeParams) {
    return requestBlob('/api/v1/finance/export/gobd', {
      method: 'POST',
      body: JSON.stringify({ from_date: params.from_date, to_date: params.to_date }),
    })
  },
}
