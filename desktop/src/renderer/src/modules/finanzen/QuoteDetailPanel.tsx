/**
 * QuoteDetailPanel — Angebots-Detailansicht (finanzen P2.5a).
 *
 * Spiegelt InvoiceDetailPanel: öffnet beim Zeilenklick, zeigt alle Infos
 * (Positionen, Summen, Gültigkeit, verknüpfte Rechnung) + Lifecycle-Aktionen
 * (senden/annehmen/ablehnen/umwandeln/PDF). Nutzt das shared `DetailPanel`.
 */
import { useTranslation } from 'react-i18next'
import {
  FileText,
  Calendar,
  Clock,
  CheckCircle2,
  XCircle,
  User,
  Send,
  Download,
  FileCheck,
  ChevronRight,
} from 'lucide-react'
import { toast } from 'sonner'
import { DetailModal } from '@/components/shared'
import { Button } from '@/components/ui/button'
import {
  useQuote,
  useSendQuote,
  useAcceptQuote,
  useRejectQuote,
  useConvertQuoteToInvoice,
  useDownloadQuotePDF,
  useInvoices,
} from '@/api/hooks/useFinance'
import { formatMoney } from '@/stores/finance'
import type { QuoteStatus } from '@/types/finance-types'
import { formatDate } from '@/lib/format'
import { PDFPreviewPanel } from './PDFPreviewPanel'
import { CustomerAccountSection } from './CustomerAccountSection'
import { useFinanceDetailNavOptional } from './FinanceDetailNav'

const statusConfig: Record<QuoteStatus, { labelKey: string; color: string; bg: string; icon: typeof CheckCircle2 }> = {
  draft: { labelKey: 'finanzen.status.draft', color: 'text-muted-foreground', bg: 'bg-secondary', icon: FileText },
  sent: { labelKey: 'finanzen.status.sent', color: 'text-info', bg: 'bg-info-light', icon: Clock },
  accepted: { labelKey: 'finanzen.quoteStatus.accepted', color: 'text-success', bg: 'bg-success-light', icon: CheckCircle2 },
  rejected: { labelKey: 'finanzen.quoteStatus.rejected', color: 'text-error', bg: 'bg-error-light', icon: XCircle },
  expired: { labelKey: 'finanzen.quoteStatus.expired', color: 'text-muted-foreground', bg: 'bg-secondary', icon: Clock },
}

interface QuoteDetailPanelProps {
  quoteId: string
  onClose: () => void
  onEdit: () => void
  /** Called after a successful convert-to-invoice so the parent can refocus. */
  onConverted?: () => void
  onBack?: () => void
}

export function QuoteDetailPanel({ quoteId, onClose, onEdit, onConverted, onBack }: QuoteDetailPanelProps) {
  const { t } = useTranslation()
  const { data: quote, isLoading } = useQuote(quoteId)
  const { data: invoicesData } = useInvoices()
  const nav = useFinanceDetailNavOptional()
  const sendQuote = useSendQuote()
  const acceptQuote = useAcceptQuote()
  const rejectQuote = useRejectQuote()
  const convertQuote = useConvertQuoteToInvoice()
  const downloadPDF = useDownloadQuotePDF()

  if (isLoading || !quote) {
    return (
      <DetailModal open={true} title={t('finanzen.quoteDetail.title')} onClose={onClose} onBack={onBack}>
        <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
          {t('common.loading')}
        </div>
      </DetailModal>
    )
  }

  const status = statusConfig[quote.status] ?? statusConfig.draft
  const StatusIcon = status.icon
  const currency = quote.currency ?? 'EUR'
  const money = (v: number | string) => formatMoney(v, currency)
  const convertedNumber = (quote as { converted_invoice_number?: string }).converted_invoice_number
  const convertedInvoice = convertedNumber
    ? (invoicesData?.invoices ?? []).find((inv) => inv.invoice_number === convertedNumber)
    : undefined

  const run = (
    mut: { mutate: (id: string, opts: { onSuccess: () => void; onError: (e: Error) => void }) => void },
    successMsg: string,
    after?: () => void,
  ) => {
    mut.mutate(quoteId, {
      onSuccess: () => { toast.success(successMsg); after?.() },
      onError: (err) => toast.error(err.message),
    })
  }

  return (
    <DetailModal open={true} title={t('finanzen.quoteDetail.title')} onClose={onClose} onBack={onBack}>
      <div className="space-y-4">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div>
            <h3 className="font-mono text-base font-medium text-foreground">{quote.quote_number}</h3>
            <p className="text-xs text-muted-foreground">{quote.customer?.name}</p>
          </div>
          <span className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-[10px] font-medium ${status.bg} ${status.color}`}>
            <StatusIcon className="h-3 w-3" />
            {t(status.labelKey)}
          </span>
        </div>

        {/* Converted-to-invoice link (klickbar, wenn die Rechnung auffindbar ist) */}
        {convertedNumber && (
          <button
            type="button"
            disabled={!nav || !convertedInvoice}
            onClick={() => convertedInvoice && nav?.open('invoice', convertedInvoice.id)}
            className="group flex w-full items-center gap-2 rounded-lg border border-success/30 bg-success/5 p-3 text-xs text-success transition-colors hover:bg-success/10 disabled:cursor-default disabled:hover:bg-success/5"
          >
            <FileCheck className="h-4 w-4 shrink-0" />
            <span className="flex-1 text-left">{t('finanzen.quoteDetail.convertedTo', { number: convertedNumber })}</span>
            {nav && convertedInvoice && <ChevronRight className="h-3.5 w-3.5 shrink-0 transition-transform group-hover:translate-x-0.5" />}
          </button>
        )}

        {/* Key info */}
        <div className="grid grid-cols-2 gap-3 text-xs">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Calendar className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">{t('finanzen.quoteDetail.quoteDate')}</p>
              <p className="text-foreground">{formatDate(quote.created_at)}</p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">{t('finanzen.quoteDetail.validUntil')}</p>
              <p className="text-foreground">{formatDate(quote.valid_until)}</p>
            </div>
          </div>
          <div className="col-span-2 flex items-center gap-2 text-muted-foreground">
            <User className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">{t('finanzen.customer')}</p>
              <p className="text-foreground">{quote.customer?.name}</p>
              {quote.customer?.address && (
                <p className="text-[10px] text-muted-foreground">{quote.customer.address}</p>
              )}
            </div>
          </div>
        </div>

        {/* Line items */}
        <section>
          <h4 className="mb-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
            {t('finanzen.lineItems.positions')}
          </h4>
          <div className="overflow-hidden rounded-md border border-border">
            {(quote.line_items ?? []).map((item, idx) => (
              <div key={item.id || idx} className={`flex items-center justify-between px-3 py-2 text-xs ${idx > 0 ? 'border-t border-border-muted' : ''}`}>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-foreground">{item.description}</p>
                  <p className="text-[10px] text-muted-foreground">
                    {item.quantity} x {money(item.unit_price)}
                    {Number(item.tax_rate) > 0 && ` | ${item.tax_rate}% MwSt`}
                  </p>
                </div>
                <span className="ml-3 font-medium text-foreground">{money(item.line_total)}</span>
              </div>
            ))}
          </div>
        </section>

        {/* Tax breakdown */}
        <div className="space-y-1.5 text-xs">
          <div className="flex justify-between text-muted-foreground">
            <span>{t('finanzen.totals.subtotalNet')}</span>
            <span>{money(quote.tax_breakdown?.subtotal ?? quote.total_net ?? 0)}</span>
          </div>
          {Object.entries(quote.tax_breakdown?.tax_by_rate ?? {}).map(([rate, amount]) => (
            <div key={rate} className="flex justify-between text-muted-foreground">
              <span>MwSt {rate}%</span>
              <span>{money(amount as number)}</span>
            </div>
          ))}
          <div className="flex justify-between border-t border-border pt-1.5 text-sm font-medium text-foreground">
            <span>{t('finanzen.totals.totalAmount')}</span>
            <span>{money(quote.tax_breakdown?.gross_total ?? quote.total_gross ?? 0)}</span>
          </div>
        </div>

        {/* Notes */}
        {quote.notes && (
          <section>
            <h4 className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
              {t('finanzen.notes')}
            </h4>
            <p className="text-xs text-muted-foreground">{quote.notes}</p>
          </section>
        )}

        {/* PDF-Vorschau — echte Belegdaten */}
        <PDFPreviewPanel
          heading="ANGEBOT"
          number={quote.quote_number}
          customerName={quote.customer?.name ?? ''}
          customerAddress={quote.customer?.address}
          date={formatDate(quote.created_at)}
          lineItems={quote.line_items ?? []}
          net={quote.tax_breakdown?.subtotal ?? quote.total_net ?? 0}
          tax={quote.tax_breakdown?.total_tax ?? 0}
          gross={quote.tax_breakdown?.gross_total ?? quote.total_gross ?? 0}
          currency={currency}
          onDownload={() => downloadPDF.mutate(quoteId)}
        />

        {/* Kundenkonto — alle Belege des Kunden + CRM-Sprung */}
        <CustomerAccountSection customer={quote.customer} currentDocId={quote.id} />

        {/* Actions */}
        <div className="flex flex-wrap gap-2 border-t border-border pt-4">
          {quote.status === 'draft' && (
            <>
              <Button variant="outline" size="sm" onClick={onEdit}>
                {t('common.edit')}
              </Button>
              <Button size="sm" onClick={() => run(sendQuote, `${quote.quote_number} ${t('finanzen.dunning.sent')}`)}>
                <Send className="mr-1.5 h-4 w-4" />
                {t('finanzen.dunning.send')}
              </Button>
            </>
          )}
          {quote.status === 'sent' && (
            <>
              <Button size="sm" onClick={() => run(acceptQuote, `${quote.quote_number} ${t('finanzen.quoteStatus.accepted')}`)}>
                <CheckCircle2 className="mr-1.5 h-4 w-4" />
                {t('finanzen.page.accept')}
              </Button>
              <Button variant="outline" size="sm" onClick={() => run(rejectQuote, `${quote.quote_number} ${t('finanzen.quoteStatus.rejected')}`)}>
                <XCircle className="mr-1.5 h-4 w-4" />
                {t('finanzen.page.reject')}
              </Button>
            </>
          )}
          {quote.status === 'accepted' && !convertedNumber && (
            <Button size="sm" onClick={() => run(convertQuote, t('finanzen.page.invoiceFromQuoteCreated'), onConverted)}>
              <FileCheck className="mr-1.5 h-4 w-4" />
              {t('finanzen.page.convertToInvoice')}
            </Button>
          )}
          <Button variant="outline" size="sm" onClick={() => downloadPDF.mutate(quoteId)}>
            <Download className="mr-1.5 h-4 w-4" />
            {t('finanzen.pdf.downloadPdf')}
          </Button>
        </div>
      </div>
    </DetailModal>
  )
}
