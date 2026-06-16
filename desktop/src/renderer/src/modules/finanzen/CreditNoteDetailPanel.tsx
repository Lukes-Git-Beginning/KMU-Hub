/**
 * CreditNoteDetailPanel — Gutschrift-/Storno-Detailansicht (finanzen P2.5a).
 *
 * Öffnet beim Zeilenklick: Positionen, Summen, verknüpfte Originalrechnung,
 * Storno-Hinweis + Grund, Aktionen (senden/PDF). Nutzt das shared `DetailPanel`.
 */
import { useTranslation } from 'react-i18next'
import {
  FileText,
  Calendar,
  Clock,
  CheckCircle2,
  Link2,
  Send,
  Download,
  ChevronRight,
} from 'lucide-react'
import { toast } from 'sonner'
import { DetailModal } from '@/components/shared'
import { Button } from '@/components/ui/button'
import {
  useCreditNote,
  useSendCreditNote,
  useDownloadCreditNotePDF,
} from '@/api/hooks/useFinance'
import { formatMoney } from '@/stores/finance'
import { formatDate } from '@/lib/format'
import { PDFPreviewPanel } from './PDFPreviewPanel'
import { CustomerAccountSection } from './CustomerAccountSection'
import { useFinanceDetailNavOptional } from './FinanceDetailNav'

interface CreditNoteDetailPanelProps {
  creditNoteId: string
  onClose: () => void
  onBack?: () => void
}

export function CreditNoteDetailPanel({ creditNoteId, onClose, onBack }: CreditNoteDetailPanelProps) {
  const { t } = useTranslation()
  const { data: cn, isLoading } = useCreditNote(creditNoteId)
  const nav = useFinanceDetailNavOptional()
  const sendCreditNote = useSendCreditNote()
  const downloadPDF = useDownloadCreditNotePDF()

  if (isLoading || !cn) {
    return (
      <DetailModal open={true} title={t('finanzen.creditNoteDetail.title')} onClose={onClose} onBack={onBack}>
        <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
          {t('common.loading')}
        </div>
      </DetailModal>
    )
  }

  const currency = cn.currency ?? 'EUR'
  const money = (v: number | string) => formatMoney(v, currency)
  const isSent = cn.status === 'sent'

  return (
    <DetailModal open={true} title={t('finanzen.creditNoteDetail.title')} onClose={onClose} onBack={onBack}>
      <div className="space-y-4">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-2">
            <div>
              <h3 className="font-mono text-base font-medium text-foreground">{cn.credit_note_number}</h3>
              <p className="text-xs text-muted-foreground">{cn.customer?.name}</p>
            </div>
            {cn.is_storno && (
              <span className="rounded bg-error-light px-1.5 py-0.5 text-[9px] font-medium text-error">
                {t('finanzen.creditNote.stornoBadge')}
              </span>
            )}
          </div>
          <span className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-[10px] font-medium ${isSent ? 'bg-info-light text-info' : 'bg-secondary text-muted-foreground'}`}>
            {isSent ? <Clock className="h-3 w-3" /> : <FileText className="h-3 w-3" />}
            {t(isSent ? 'finanzen.status.sent' : 'finanzen.status.draft')}
          </span>
        </div>

        {/* Linked original invoice (klickbar → Original-Rechnung öffnen) */}
        {cn.invoice_number && (
          <button
            type="button"
            disabled={!nav || !cn.original_invoice_id}
            onClick={() => cn.original_invoice_id && nav?.open('invoice', cn.original_invoice_id)}
            className="group flex w-full items-center gap-2 rounded-lg border border-border bg-secondary/40 p-3 text-xs text-left transition-colors hover:bg-secondary disabled:cursor-default disabled:hover:bg-secondary/40"
          >
            <Link2 className="h-4 w-4 shrink-0 text-muted-foreground" />
            <span className="flex-1 text-muted-foreground">
              {t('finanzen.creditNoteDetail.originalInvoice')}:{' '}
              <span className="font-mono text-foreground">{cn.invoice_number}</span>
            </span>
            {nav && cn.original_invoice_id && <ChevronRight className="h-3.5 w-3.5 shrink-0 text-muted-foreground transition-[color,transform] group-hover:translate-x-0.5 group-hover:text-primary" />}
          </button>
        )}

        {/* Key info */}
        <div className="grid grid-cols-2 gap-3 text-xs">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Calendar className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">{t('finanzen.creditNoteDetail.date')}</p>
              <p className="text-foreground">{formatDate(cn.created_at)}</p>
            </div>
          </div>
          {cn.reason && (
            <div className="flex items-center gap-2 text-muted-foreground">
              <FileText className="h-3.5 w-3.5" />
              <div className="min-w-0">
                <p className="text-[10px] text-muted-foreground">{t('finanzen.creditNoteDetail.reason')}</p>
                <p className="truncate text-foreground">{cn.reason}</p>
              </div>
            </div>
          )}
        </div>

        {/* Line items */}
        <section>
          <h4 className="mb-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
            {t('finanzen.lineItems.positions')}
          </h4>
          <div className="overflow-hidden rounded-md border border-border">
            {(cn.line_items ?? []).map((item, idx) => (
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
            <span>{money(cn.tax_breakdown?.subtotal ?? cn.total_net ?? 0)}</span>
          </div>
          {Object.entries(cn.tax_breakdown?.tax_by_rate ?? {}).map(([rate, amount]) => (
            <div key={rate} className="flex justify-between text-muted-foreground">
              <span>MwSt {rate}%</span>
              <span>{money(amount as number)}</span>
            </div>
          ))}
          <div className="flex justify-between border-t border-border pt-1.5 text-sm font-medium text-foreground">
            <span>{t('finanzen.totals.totalAmount')}</span>
            <span>{money(cn.tax_breakdown?.gross_total ?? cn.total_gross ?? 0)}</span>
          </div>
        </div>

        {/* PDF-Vorschau — echte Belegdaten */}
        <PDFPreviewPanel
          heading="GUTSCHRIFT"
          number={cn.credit_note_number}
          customerName={cn.customer?.name ?? ''}
          customerAddress={cn.customer?.address}
          date={formatDate(cn.created_at)}
          lineItems={cn.line_items ?? []}
          net={cn.tax_breakdown?.subtotal ?? cn.total_net ?? 0}
          tax={cn.tax_breakdown?.total_tax ?? 0}
          gross={cn.tax_breakdown?.gross_total ?? cn.total_gross ?? 0}
          currency={currency}
          onDownload={() => downloadPDF.mutate(creditNoteId)}
        />

        {/* Kundenkonto — alle Belege des Kunden + CRM-Sprung */}
        <CustomerAccountSection customer={cn.customer} currentDocId={cn.id} />

        {/* Actions */}
        <div className="flex flex-wrap gap-2 border-t border-border pt-4">
          {!isSent && (
            <Button
              size="sm"
              onClick={() =>
                sendCreditNote.mutate(creditNoteId, {
                  onSuccess: () => toast.success(`${cn.credit_note_number} ${t('finanzen.dunning.sent')}`),
                  onError: (err) => toast.error(err.message),
                })
              }
            >
              <Send className="mr-1.5 h-4 w-4" />
              {t('finanzen.dunning.send')}
            </Button>
          )}
          <Button variant="outline" size="sm" onClick={() => downloadPDF.mutate(creditNoteId)}>
            <Download className="mr-1.5 h-4 w-4" />
            {t('finanzen.pdf.downloadPdf')}
          </Button>
        </div>
      </div>
    </DetailModal>
  )
}
