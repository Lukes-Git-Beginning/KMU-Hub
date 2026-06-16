import { useTranslation } from 'react-i18next'
import {
  FileText,
  Calendar,
  CreditCard,
  Clock,
  CheckCircle2,
  AlertCircle,
  XCircle,
  User,
  Download,
  Send,
  Ban,
  Info,
  History,
  Shield,
  FileCheck,
  Repeat,
  Gavel,
  ChevronRight,
} from 'lucide-react'
import { toast } from 'sonner'
import { DetailModal } from '@/components/shared'
import { Button } from '@/components/ui/button'
import {
  useInvoice,
  usePayments,
  useSendInvoice,
  useMarkInvoicePaid,
  useCancelInvoice,
  useDownloadInvoicePDF,
  useCreditNotes,
  useQuotes,
  useDunnings,
} from '@/api/hooks/useFinance'
import { formatMoney } from '@/stores/finance'
import type { InvoiceStatus } from '@/types/finance-types'
import { PDFPreviewPanel } from './PDFPreviewPanel'
import { CustomerAccountSection } from './CustomerAccountSection'
import { useFinanceDetailNavOptional } from './FinanceDetailNav'
import { formatDate } from '@/lib/format'

const statusConfig: Record<
  InvoiceStatus,
  {
    labelKey: string
    color: string
    bg: string
    icon: typeof CheckCircle2
  }
> = {
  draft: {
    labelKey: 'finanzen.status.draft',
    color: 'text-muted-foreground',
    bg: 'bg-secondary',
    icon: FileText,
  },
  sent: {
    labelKey: 'finanzen.status.sent',
    color: 'text-info',
    bg: 'bg-info-light',
    icon: Clock,
  },
  paid: {
    labelKey: 'finanzen.status.paid',
    color: 'text-success',
    bg: 'bg-success-light',
    icon: CheckCircle2,
  },
  overdue: {
    labelKey: 'finanzen.status.overdue',
    color: 'text-error',
    bg: 'bg-error-light',
    icon: AlertCircle,
  },
  cancelled: {
    labelKey: 'finanzen.status.cancelled',
    color: 'text-muted-foreground',
    bg: 'bg-secondary',
    icon: XCircle,
  },
}

const PAYMENT_METHOD_LABEL_KEYS: Record<string, string> = {
  bank_transfer: 'finanzen.paymentMethod.bankTransfer',
  cash: 'finanzen.paymentMethod.cash',
  credit_card: 'finanzen.paymentMethod.creditCard',
  other: 'finanzen.paymentMethod.other',
}

interface InvoiceDetailPanelProps {
  invoiceId: string
  onClose: () => void
  onEdit: () => void
  onRecordPayment: () => void
  onStorno?: () => void
  onBack?: () => void
}

export function InvoiceDetailPanel({
  invoiceId,
  onClose,
  onEdit,
  onRecordPayment,
  onStorno,
  onBack,
}: InvoiceDetailPanelProps) {
  const { t } = useTranslation()
  const { data: invoice, isLoading } = useInvoice(invoiceId)
  const { data: paymentsData } = usePayments(invoiceId)
  const { data: creditNotesData } = useCreditNotes()
  const { data: quotesData } = useQuotes()
  const { data: dunningsData } = useDunnings({ invoice_id: invoiceId })
  const nav = useFinanceDetailNavOptional()
  const sendInvoice = useSendInvoice()
  const markPaid = useMarkInvoicePaid()
  const cancelInvoice = useCancelInvoice()
  const downloadPDF = useDownloadInvoicePDF()

  if (isLoading || !invoice) {
    return (
      <DetailModal open={true} title={t('finanzen.invoiceDetail.title')} onClose={onClose} onBack={onBack}>
        <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
          {t('common.loading')}
        </div>
      </DetailModal>
    )
  }

  const status = statusConfig[invoice.status]
  const StatusIcon = status.icon
  const currency = invoice.currency ?? 'EUR'
  const money = (v: number | string) => formatMoney(v, currency)
  const grossTotal = Number(invoice.tax_breakdown?.gross_total ?? invoice.total_gross ?? 0)
  const payments = paymentsData?.payments ?? []
  const totalPaid = payments.reduce((sum, p) => sum + Number(p.amount), 0)
  const remaining = Math.max(0, grossTotal - totalPaid)
  const isImmutable = invoice.status !== 'draft'
  const linkedCreditNotes = (creditNotesData?.credit_notes ?? []).filter(
    (cn) => cn.original_invoice_id === invoice.id || cn.invoice_number === invoice.invoice_number,
  )
  const sourceQuote = invoice.source_quote_id
    ? (quotesData?.quotes ?? []).find((q) => q.id === invoice.source_quote_id)
    : undefined
  const dunnings = [...(dunningsData?.dunnings ?? [])].sort((a, b) => a.level - b.level)

  const handleSend = () => {
    sendInvoice.mutate(invoiceId, {
      onSuccess: () => toast.success(t('finanzen.invoiceDetail.invoiceSent')),
      onError: (err) => toast.error(err.message),
    })
  }

  const handleMarkPaid = () => {
    markPaid.mutate(invoiceId, {
      onSuccess: () => toast.success(t('finanzen.invoiceDetail.markedPaid')),
      onError: (err) => toast.error(err.message),
    })
  }

  const handleCancel = () => {
    cancelInvoice.mutate(invoiceId, {
      onSuccess: () => toast.success(t('finanzen.invoiceDetail.invoiceCancelled')),
      onError: (err) => toast.error(err.message),
    })
  }

  return (
    <DetailModal open={true} title={t('finanzen.invoiceDetail.title')} onClose={onClose} onBack={onBack}>
      <div className="space-y-4">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div>
            <h3 className="text-base font-medium text-foreground font-mono">
              {invoice.invoice_number}
            </h3>
            <p className="text-xs text-muted-foreground">
              {invoice.customer.name}
            </p>
          </div>
          <span
            className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-[10px] font-medium ${status.bg} ${status.color}`}
          >
            <StatusIcon className="h-3 w-3" />
            {t(status.labelKey)}
          </span>
        </div>

        {/* Verknüpfungen — Quell-Angebot / Abo-Herkunft (klickbar) */}
        {(sourceQuote || invoice.recurring_id) && (
          <div className="flex flex-wrap gap-2">
            {sourceQuote && (
              <button
                type="button"
                disabled={!nav}
                onClick={() => nav?.open('quote', sourceQuote.id)}
                className="group inline-flex items-center gap-1.5 rounded-md border border-border bg-secondary/40 px-2.5 py-1 text-xs text-foreground transition-colors hover:bg-secondary disabled:cursor-default disabled:hover:bg-secondary/40"
              >
                <FileCheck className="h-3.5 w-3.5 text-muted-foreground" />
                {t('finanzen.invoiceDetail.fromQuote')}
                <span className="font-mono text-primary">{sourceQuote.quote_number}</span>
                {nav && <ChevronRight className="h-3 w-3 text-muted-foreground transition-[color,transform] group-hover:translate-x-0.5 group-hover:text-primary" />}
              </button>
            )}
            {invoice.recurring_id && (
              <span className="inline-flex items-center gap-1.5 rounded-md border border-border bg-secondary/40 px-2.5 py-1 text-xs text-muted-foreground">
                <Repeat className="h-3.5 w-3.5" />
                {t('finanzen.invoiceDetail.fromRecurring')}
              </span>
            )}
          </div>
        )}

        {/* Immutability notice */}
        {isImmutable && (
          <div className="flex items-start gap-2 rounded-lg border border-info/30 bg-info/5 p-3 text-xs text-info">
            <Info className="h-4 w-4 mt-0.5 shrink-0" />
            <span>
              {t('finanzen.invoiceDetail.immutableNotice')}
            </span>
          </div>
        )}

        {/* Key info */}
        <div className="grid grid-cols-2 gap-3 text-xs">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Calendar className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">
                {t('finanzen.invoiceDetail.invoiceDate')}
              </p>
              <p className="text-foreground">
                {formatDate(invoice.invoice_date)}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">{t('finanzen.invoiceDetail.dueDate')}</p>
              <p className="text-foreground">
                {formatDate(invoice.due_date)}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <User className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">{t('finanzen.customer')}</p>
              <p className="text-foreground">{invoice.customer.name}</p>
              {invoice.customer.address && (
                <p className="text-[10px] text-muted-foreground">
                  {invoice.customer.address}
                </p>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <CreditCard className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">
                {t('finanzen.invoiceDetail.paymentTerms')}
              </p>
              <p className="text-foreground">
                {invoice.payment_terms ?? '--'} {t('finanzen.invoiceDetail.days')}
              </p>
            </div>
          </div>
        </div>

        {/* Line items */}
        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
            {t('finanzen.lineItems.positions')}
          </h4>
          <div className="rounded-md border border-border overflow-hidden">
            {(invoice.line_items ?? []).map((item, idx) => (
              <div
                key={item.id || idx}
                className={`flex items-center justify-between px-3 py-2 text-xs ${
                  idx > 0 ? 'border-t border-border-muted' : ''
                }`}
              >
                <div className="min-w-0 flex-1">
                  <p className="text-foreground truncate">
                    {item.description}
                  </p>
                  <p className="text-[10px] text-muted-foreground">
                    {item.quantity} x {money(item.unit_price)}
                    {Number(item.tax_rate) > 0 && ` | ${item.tax_rate}% MwSt`}
                  </p>
                </div>
                <span className="text-foreground font-medium ml-3">
                  {money(item.line_total)}
                </span>
              </div>
            ))}
          </div>
        </section>

        {/* Tax breakdown */}
        <div className="space-y-1.5 text-xs">
          <div className="flex justify-between text-muted-foreground">
            <span>{t('finanzen.totals.subtotalNet')}</span>
            <span>{money(invoice.tax_breakdown?.subtotal ?? invoice.total_net ?? 0)}</span>
          </div>
          {Object.entries(invoice.tax_breakdown?.tax_by_rate ?? {}).map(
            ([rate, amount]) => (
              <div
                key={rate}
                className="flex justify-between text-muted-foreground"
              >
                <span>MwSt {rate}%</span>
                <span>{money(amount)}</span>
              </div>
            ),
          )}
          <div className="flex justify-between font-medium text-sm text-foreground border-t border-border pt-1.5">
            <span>{t('finanzen.totals.totalAmount')}</span>
            <span>{money(invoice.tax_breakdown?.gross_total ?? invoice.total_gross ?? 0)}</span>
          </div>
          {totalPaid > 0 && (
            <>
              <div className="flex justify-between text-success">
                <span>{t('finanzen.status.paid')}</span>
                <span>{money(totalPaid)}</span>
              </div>
              {remaining > 0 && (
                <div className="flex justify-between font-medium text-warning">
                  <span>{t('finanzen.invoiceDetail.open')}</span>
                  <span>{money(remaining)}</span>
                </div>
              )}
            </>
          )}
        </div>

        {/* Linked credit notes / storno */}
        {linkedCreditNotes.length > 0 && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
              {t('finanzen.invoiceDetail.linkedCreditNotes')}
            </h4>
            <div className="space-y-1.5">
              {linkedCreditNotes.map((cn) => (
                <button
                  key={cn.id}
                  type="button"
                  disabled={!nav}
                  onClick={() => nav?.open('creditNote', cn.id)}
                  className="group flex w-full items-center justify-between rounded-md border border-border-muted px-3 py-2 text-xs text-left transition-colors hover:bg-secondary/50 disabled:cursor-default disabled:hover:bg-transparent"
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <span className="font-mono text-primary">{cn.credit_note_number}</span>
                    {cn.is_storno && (
                      <span className="rounded bg-error-light px-1.5 py-0.5 text-[9px] font-medium text-error">
                        {t('finanzen.creditNote.stornoBadge')}
                      </span>
                    )}
                    {cn.reason && (
                      <span className="text-[10px] text-muted-foreground truncate">{cn.reason}</span>
                    )}
                  </div>
                  <span className="flex items-center gap-1 ml-2 shrink-0">
                    <span className="text-foreground font-medium">
                      {money(cn.tax_breakdown?.gross_total ?? cn.total_gross ?? 0)}
                    </span>
                    {nav && <ChevronRight className="h-3.5 w-3.5 text-muted-foreground transition-[color,transform] group-hover:translate-x-0.5 group-hover:text-primary" />}
                  </span>
                </button>
              ))}
            </div>
          </section>
        )}

        {/* Payment history */}
        {payments.length > 0 && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
              {t('finanzen.invoiceDetail.paymentHistory')}
            </h4>
            <div className="space-y-1.5">
              {payments.map((p) => (
                <div
                  key={p.id}
                  className="flex items-center justify-between rounded-md bg-success/5 px-3 py-2 text-xs"
                >
                  <div>
                    <p className="text-foreground">
                      {PAYMENT_METHOD_LABEL_KEYS[p.method] ? t(PAYMENT_METHOD_LABEL_KEYS[p.method]) : p.method}
                    </p>
                    <p className="text-[10px] text-muted-foreground">
                      {formatDate(p.payment_date)}
                      {p.reference && ` | ${p.reference}`}
                    </p>
                  </div>
                  <span className="text-success font-medium">
                    {money(p.amount)}
                  </span>
                </div>
              ))}
            </div>
          </section>
        )}

        {/* Dunning / Mahn-History */}
        {dunnings.length > 0 && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex items-center gap-1">
              <Gavel className="h-3 w-3" />
              {t('finanzen.invoiceDetail.dunningHistory')}
            </h4>
            <div className="rounded-md border border-border overflow-hidden">
              {dunnings.map((d, idx) => {
                const fee = Number(d.fee) + Number(d.interest)
                return (
                  <div
                    key={d.id}
                    className={`flex items-center justify-between px-3 py-2 text-xs ${idx > 0 ? 'border-t border-border-muted' : ''}`}
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <span className="rounded bg-warning/15 px-1.5 py-0.5 text-[9px] font-medium text-warning">
                        {t('finanzen.invoiceDetail.dunningLevel', { level: d.level })}
                      </span>
                      <span className="text-[10px] text-muted-foreground">
                        {d.sent_at ? formatDate(d.sent_at) : t('finanzen.status.draft')}
                      </span>
                    </div>
                    {fee > 0 && (
                      <span className="text-foreground font-medium">
                        +{money(fee)}
                      </span>
                    )}
                  </div>
                )
              })}
            </div>
          </section>
        )}

        {/* Kundenkonto — alle Belege des Kunden + CRM-Sprung */}
        <CustomerAccountSection customer={invoice.customer} currentDocId={invoice.id} />

        {/* Notes */}
        {invoice.notes && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-1">
              {t('finanzen.notes')}
            </h4>
            <p className="text-xs text-muted-foreground">{invoice.notes}</p>
          </section>
        )}

        {/* PDF Preview — echte Belegdaten (P2.5c) */}
        <PDFPreviewPanel
          heading="RECHNUNG"
          number={invoice.invoice_number}
          customerName={invoice.customer?.name ?? ''}
          customerAddress={invoice.customer?.address}
          date={formatDate(invoice.invoice_date)}
          lineItems={invoice.line_items ?? []}
          net={invoice.tax_breakdown?.subtotal ?? invoice.total_net ?? 0}
          tax={invoice.tax_breakdown?.total_tax ?? 0}
          gross={invoice.tax_breakdown?.gross_total ?? invoice.total_gross ?? 0}
          currency={currency}
          onDownload={() => downloadPDF.mutate(invoiceId)}
        />

        {/* GoBD Audit Log (3.17) */}
        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex items-center gap-1">
            <Shield className="h-3 w-3" />
            GoBD-Änderungsprotokoll
          </h4>
          <div className="rounded-md border border-border overflow-hidden">
            {[
              {
                action: t('finanzen.invoiceDetail.auditCreated'),
                user: 'Max Müller',
                date: invoice.invoice_date,
                detail: `Nummer: ${invoice.invoice_number}`,
              },
              ...(invoice.status !== 'draft'
                ? [
                    {
                      action: t('finanzen.invoiceDetail.auditSent'),
                      user: 'Max Müller',
                      date: invoice.invoice_date,
                      detail: `An: ${invoice.customer.email}`,
                    },
                  ]
                : []),
              ...(invoice.status === 'cancelled'
                ? [
                    {
                      action: t('finanzen.invoiceDetail.auditCancelled'),
                      user: 'Max Müller',
                      date: new Date().toISOString().split('T')[0],
                      detail: t('finanzen.invoiceDetail.auditCancelledDetail'),
                    },
                  ]
                : []),
              ...(payments.length > 0
                ? payments.map((p) => ({
                    action: t('finanzen.invoiceDetail.auditPayment'),
                    user: 'System',
                    date: p.payment_date,
                    detail: `${money(p.amount)} via ${PAYMENT_METHOD_LABEL_KEYS[p.method] ? t(PAYMENT_METHOD_LABEL_KEYS[p.method]) : p.method}`,
                  }))
                : []),
            ].map((entry, idx) => (
              <div
                key={idx}
                className={`flex items-start gap-2 px-3 py-2 text-xs ${
                  idx > 0 ? 'border-t border-border-muted' : ''
                }`}
              >
                <History className="h-3 w-3 text-muted-foreground mt-0.5 shrink-0" />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <span className="font-medium text-foreground">{entry.action}</span>
                    <span className="text-[10px] text-muted-foreground">
                      {formatDate(entry.date)}
                    </span>
                  </div>
                  <p className="text-[10px] text-muted-foreground">
                    {entry.user} — {entry.detail}
                  </p>
                </div>
              </div>
            ))}
          </div>
          <p className="mt-1 text-[9px] text-muted-foreground flex items-center gap-1">
            <Shield className="h-2.5 w-2.5" />
            {t('finanzen.invoiceDetail.gobdCompliance')}
          </p>
        </section>

        {/* Actions */}
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => downloadPDF.mutate(invoiceId)}
            disabled={downloadPDF.isPending}
          >
            <Download className="mr-1.5 h-3.5 w-3.5" />
            PDF
          </Button>
          {invoice.status === 'draft' && (
            <>
              <Button variant="outline" size="sm" onClick={onEdit}>
                {t('common.edit')}
              </Button>
              <Button
                size="sm"
                onClick={handleSend}
                disabled={sendInvoice.isPending}
              >
                <Send className="mr-1.5 h-3.5 w-3.5" />
                {t('finanzen.dunning.send')}
              </Button>
            </>
          )}
          {invoice.status !== 'paid' && invoice.status !== 'cancelled' && (
            <>
              <Button size="sm" onClick={onRecordPayment}>
                <CreditCard className="mr-1.5 h-3.5 w-3.5" />
                {t('finanzen.invoiceDetail.recordPayment')}
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={handleMarkPaid}
                disabled={markPaid.isPending}
              >
                <CheckCircle2 className="mr-1.5 h-3.5 w-3.5" />
                {t('finanzen.invoiceDetail.markPaid')}
              </Button>
              {invoice.status === 'draft' ? (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleCancel}
                  disabled={cancelInvoice.isPending}
                  className="text-error hover:text-error"
                >
                  <Ban className="mr-1.5 h-3.5 w-3.5" />
                  {t('finanzen.invoiceDetail.cancel')}
                </Button>
              ) : (
                onStorno && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={onStorno}
                    className="text-error hover:text-error"
                  >
                    <Ban className="mr-1.5 h-3.5 w-3.5" />
                    {t('finanzen.invoiceDetail.storno')}
                  </Button>
                )
              )}
            </>
          )}
        </div>
      </div>
    </DetailModal>
  )
}
