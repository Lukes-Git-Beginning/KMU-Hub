/**
 * TanStack Query hooks for Finance module operations.
 *
 * Query keys follow the pattern ['finance', domain, ...params] for consistent
 * cache invalidation. Mutations invalidate related queries automatically.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  financeSettingsApi,
  financeQuoteApi,
  financeInvoiceApi,
  financeCreditNoteApi,
  financePaymentApi,
  financeDunningApi,
  financeDashboardApi,
  financeExportApi,
  financeDealApi,
  financeGoBDApi,
} from '../finance-client'
import type {
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
  DateRangeParams,
} from '@/types/finance-types'

// ---------------------------------------------------------------------------
// Query key factory
// ---------------------------------------------------------------------------

export const financeKeys = {
  all: ['finance'] as const,
  companySettings: () => ['finance', 'company-settings'] as const,
  quotes: (params?: ListQuotesParams) => ['finance', 'quotes', params] as const,
  quote: (id: string) => ['finance', 'quote', id] as const,
  invoices: (params?: ListInvoicesParams) => ['finance', 'invoices', params] as const,
  invoice: (id: string) => ['finance', 'invoice', id] as const,
  creditNotes: (params?: ListCreditNotesParams) =>
    ['finance', 'credit-notes', params] as const,
  creditNote: (id: string) => ['finance', 'credit-note', id] as const,
  payments: (invoiceId: string) => ['finance', 'payments', invoiceId] as const,
  dunnings: (params?: ListDunningsParams) => ['finance', 'dunnings', params] as const,
  dunningConfig: () => ['finance', 'dunning-config'] as const,
  dashboard: (dateFrom?: string, dateTo?: string) =>
    ['finance', 'dashboard', dateFrom, dateTo] as const,
  // GoBD keys (Sprint 2 / Wave 1.B)
  journalSummary: (year: number) => ['finance', 'journal-summary', year] as const,
  paymentStats: (from: string, to: string) =>
    ['finance', 'payment-stats', from, to] as const,
}

// ---------------------------------------------------------------------------
// Company Settings hooks
// ---------------------------------------------------------------------------

export function useCompanySettings() {
  return useQuery({
    queryKey: financeKeys.companySettings(),
    queryFn: () => financeSettingsApi.get(),
    staleTime: 5 * 60 * 1000, // 5 minutes
    select: (data) => data.settings,
  })
}

export function useUpdateCompanySettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: UpdateCompanySettingsRequest) =>
      financeSettingsApi.update(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: financeKeys.companySettings() })
    },
  })
}

// ---------------------------------------------------------------------------
// Quote hooks
// ---------------------------------------------------------------------------

export function useQuotes(params?: ListQuotesParams) {
  return useQuery({
    queryKey: financeKeys.quotes(params),
    queryFn: () => financeQuoteApi.list(params),
  })
}

export function useQuote(id: string) {
  return useQuery({
    queryKey: financeKeys.quote(id),
    queryFn: () => financeQuoteApi.get(id),
    enabled: !!id,
    select: (data) => data.quote,
  })
}

export function useCreateQuote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateQuoteRequest) => financeQuoteApi.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['finance', 'quotes'] })
      qc.invalidateQueries({ queryKey: ['finance', 'dashboard'] })
    },
  })
}

export function useUpdateQuote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: UpdateQuoteRequest & { id: string }) =>
      financeQuoteApi.update(id, data),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['finance', 'quotes'] })
      qc.invalidateQueries({ queryKey: financeKeys.quote(vars.id) })
    },
  })
}

export function useDeleteQuote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeQuoteApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['finance', 'quotes'] })
      qc.invalidateQueries({ queryKey: ['finance', 'dashboard'] })
    },
  })
}

export function useSendQuote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeQuoteApi.send(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: ['finance', 'quotes'] })
      qc.invalidateQueries({ queryKey: financeKeys.quote(id) })
    },
  })
}

export function useAcceptQuote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeQuoteApi.accept(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: ['finance', 'quotes'] })
      qc.invalidateQueries({ queryKey: financeKeys.quote(id) })
      qc.invalidateQueries({ queryKey: ['finance', 'dashboard'] })
    },
  })
}

export function useRejectQuote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeQuoteApi.reject(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: ['finance', 'quotes'] })
      qc.invalidateQueries({ queryKey: financeKeys.quote(id) })
      qc.invalidateQueries({ queryKey: ['finance', 'dashboard'] })
    },
  })
}

export function useConvertQuoteToInvoice() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeQuoteApi.convertToInvoice(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['finance', 'quotes'] })
      qc.invalidateQueries({ queryKey: ['finance', 'invoices'] })
      qc.invalidateQueries({ queryKey: ['finance', 'dashboard'] })
    },
  })
}

export function useCreateQuoteFromDeal() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (dealId: string) => financeDealApi.createQuoteFromDeal(dealId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['finance', 'quotes'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Invoice hooks
// ---------------------------------------------------------------------------

export function useInvoices(params?: ListInvoicesParams) {
  return useQuery({
    queryKey: financeKeys.invoices(params),
    queryFn: () => financeInvoiceApi.list(params),
  })
}

export function useInvoice(id: string) {
  return useQuery({
    queryKey: financeKeys.invoice(id),
    queryFn: () => financeInvoiceApi.get(id),
    enabled: !!id,
    select: (data) => data.invoice,
  })
}

export function useCreateInvoice() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateInvoiceRequest) =>
      financeInvoiceApi.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['finance', 'invoices'] })
      qc.invalidateQueries({ queryKey: ['finance', 'dashboard'] })
    },
  })
}

export function useUpdateInvoice() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: UpdateInvoiceRequest & { id: string }) =>
      financeInvoiceApi.update(id, data),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['finance', 'invoices'] })
      qc.invalidateQueries({ queryKey: financeKeys.invoice(vars.id) })
    },
  })
}

export function useSendInvoice() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeInvoiceApi.send(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: ['finance', 'invoices'] })
      qc.invalidateQueries({ queryKey: financeKeys.invoice(id) })
      qc.invalidateQueries({ queryKey: ['finance', 'dashboard'] })
    },
  })
}

export function useMarkInvoicePaid() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeInvoiceApi.markPaid(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: ['finance', 'invoices'] })
      qc.invalidateQueries({ queryKey: financeKeys.invoice(id) })
      qc.invalidateQueries({ queryKey: ['finance', 'dashboard'] })
    },
  })
}

export function useCancelInvoice() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeInvoiceApi.cancel(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: ['finance', 'invoices'] })
      qc.invalidateQueries({ queryKey: financeKeys.invoice(id) })
      qc.invalidateQueries({ queryKey: ['finance', 'dashboard'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Credit Note hooks
// ---------------------------------------------------------------------------

export function useCreditNotes(params?: ListCreditNotesParams) {
  return useQuery({
    queryKey: financeKeys.creditNotes(params),
    queryFn: () => financeCreditNoteApi.list(params),
  })
}

export function useCreditNote(id: string) {
  return useQuery({
    queryKey: financeKeys.creditNote(id),
    queryFn: () => financeCreditNoteApi.get(id),
    enabled: !!id,
    select: (data) => data.credit_note,
  })
}

export function useCreateCreditNote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateCreditNoteRequest) =>
      financeCreditNoteApi.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['finance', 'credit-notes'] })
      qc.invalidateQueries({ queryKey: ['finance', 'dashboard'] })
    },
  })
}

export function useSendCreditNote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeCreditNoteApi.send(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: ['finance', 'credit-notes'] })
      qc.invalidateQueries({ queryKey: financeKeys.creditNote(id) })
    },
  })
}

// ---------------------------------------------------------------------------
// Payment hooks
// ---------------------------------------------------------------------------

export function usePayments(invoiceId: string) {
  return useQuery({
    queryKey: financeKeys.payments(invoiceId),
    queryFn: () => financePaymentApi.list(invoiceId),
    enabled: !!invoiceId,
  })
}

export function useRecordPayment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      invoiceId,
      ...data
    }: RecordPaymentRequest & { invoiceId: string }) =>
      financePaymentApi.record(invoiceId, data),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({
        queryKey: financeKeys.payments(vars.invoiceId),
      })
      qc.invalidateQueries({
        queryKey: financeKeys.invoice(vars.invoiceId),
      })
      qc.invalidateQueries({ queryKey: ['finance', 'invoices'] })
      qc.invalidateQueries({ queryKey: ['finance', 'dashboard'] })
    },
  })
}

export function useDeletePayment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financePaymentApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['finance', 'payments'] })
      qc.invalidateQueries({ queryKey: ['finance', 'invoices'] })
      qc.invalidateQueries({ queryKey: ['finance', 'dashboard'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Dunning hooks
// ---------------------------------------------------------------------------

export function useDunnings(params?: ListDunningsParams) {
  return useQuery({
    queryKey: financeKeys.dunnings(params),
    queryFn: () => financeDunningApi.list(params),
  })
}

export function useDunningConfig() {
  return useQuery({
    queryKey: financeKeys.dunningConfig(),
    queryFn: () => financeDunningApi.getConfig(),
    staleTime: 5 * 60 * 1000, // 5 minutes
    select: (data) => data.config,
  })
}

export function useDetectDunnings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => financeDunningApi.detect(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['finance', 'dunnings'] })
      qc.invalidateQueries({ queryKey: ['finance', 'dashboard'] })
    },
  })
}

export function useSendDunning() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeDunningApi.send(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['finance', 'dunnings'] })
    },
  })
}

export function useEscalateDunning() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeDunningApi.escalate(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['finance', 'dunnings'] })
    },
  })
}

export function useUpdateDunningConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: UpdateDunningConfigRequest) =>
      financeDunningApi.updateConfig(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: financeKeys.dunningConfig() })
    },
  })
}

// ---------------------------------------------------------------------------
// Dashboard hooks
// ---------------------------------------------------------------------------

export function useFinanceDashboard(dateFrom?: string, dateTo?: string) {
  return useQuery({
    queryKey: financeKeys.dashboard(dateFrom, dateTo),
    queryFn: () =>
      financeDashboardApi.get({
        date_from: dateFrom,
        date_to: dateTo,
      }),
    select: (data) => data.dashboard,
  })
}

// ---------------------------------------------------------------------------
// Export hooks
// ---------------------------------------------------------------------------

export function useExportDATEV() {
  return useMutation({
    mutationFn: (params: { date_from: string; date_to: string }) =>
      financeExportApi.exportDATEV(params),
    onSuccess: (blob, params) => {
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `EXTF_Buchungsstapel_${params.date_from}_${params.date_to}.csv`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    },
  })
}

// ---------------------------------------------------------------------------
// PDF download hooks
// ---------------------------------------------------------------------------

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

export function useDownloadQuotePDF() {
  return useMutation({
    mutationFn: (id: string) => financeQuoteApi.getPDF(id),
    onSuccess: (blob) => {
      downloadBlob(blob, `Angebot.pdf`)
    },
  })
}

export function useDownloadInvoicePDF() {
  return useMutation({
    mutationFn: (id: string) => financeInvoiceApi.getPDF(id),
    onSuccess: (blob) => {
      downloadBlob(blob, `Rechnung.pdf`)
    },
  })
}

export function useDownloadCreditNotePDF() {
  return useMutation({
    mutationFn: (id: string) => financeCreditNoteApi.getPDF(id),
    onSuccess: (blob) => {
      downloadBlob(blob, `Gutschrift.pdf`)
    },
  })
}

export function useDownloadDunningPDF() {
  return useMutation({
    mutationFn: (id: string) => financeDunningApi.getPDF(id),
    onSuccess: (blob) => {
      downloadBlob(blob, `Mahnung.pdf`)
    },
  })
}

// ---------------------------------------------------------------------------
// GoBD Journal & Compliance hooks (Sprint 2 / Wave 1.B)
// ---------------------------------------------------------------------------

/**
 * Fetches the GoBD journal summary for a given fiscal year.
 * gaps_detected === 0 is required for GoBD compliance.
 */
export function useJournalSummary(year: number) {
  return useQuery({
    queryKey: financeKeys.journalSummary(year),
    queryFn: () => financeGoBDApi.getJournalSummary(year),
    enabled: year >= 2000 && year <= 2100,
    staleTime: 10 * 60 * 1000, // 10 min — journal state changes infrequently
  })
}

/**
 * Checks whether a candidate invoice number is format-valid and not yet assigned.
 * enabled=false by default — set enabled to true when the user stops typing.
 */
export function useValidateInvoiceNumber(number: string, enabled = false) {
  return useQuery({
    queryKey: ['finance', 'validate-invoice-number', number] as const,
    queryFn: () => financeGoBDApi.validateInvoiceNumber(number),
    enabled: enabled && number.length > 0,
    staleTime: 30 * 1000, // 30 s — invalidated by any invoice creation
  })
}

/**
 * Locks an invoice (GoBD: administrative immutability).
 * Invalidates the invoice cache on success.
 */
export function useLockInvoice() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeGoBDApi.lockInvoice(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: financeKeys.invoice(id) })
      qc.invalidateQueries({ queryKey: ['finance', 'invoices'] })
    },
  })
}

/**
 * Fetches aggregated payment statistics for a date range.
 */
export function usePaymentStats(params: DateRangeParams) {
  return useQuery({
    queryKey: financeKeys.paymentStats(params.from_date, params.to_date),
    queryFn: () => financeGoBDApi.getPaymentStats(params),
    enabled: !!params.from_date && !!params.to_date,
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * Directly overrides the status of a dunning record (admin use).
 */
export function useUpdateDunningStatus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      financeGoBDApi.updateDunningStatus(id, status),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['finance', 'dunnings'] })
    },
  })
}

/**
 * Marks a dunning record as sent and (Sprint 3) queues the email notification.
 */
export function useSendDunningNotice() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeGoBDApi.sendDunningNotice(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['finance', 'dunnings'] })
    },
  })
}

/**
 * Generates and downloads a GoBD-compliant CSV export for the given date range.
 * The file is downloaded directly via blob URL.
 */
export function useGoBDExport() {
  return useMutation({
    mutationFn: (params: DateRangeParams) =>
      financeGoBDApi.generateGoBDExport(params),
    onSuccess: (blob, params) => {
      downloadBlob(
        blob,
        `gobd-export-${params.from_date}-${params.to_date}.csv`,
      )
    },
  })
}
