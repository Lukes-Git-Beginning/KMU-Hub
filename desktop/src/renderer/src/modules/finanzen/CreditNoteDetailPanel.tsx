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
  Link2,
  Send,
  Download,
  ChevronRight,
  History,
  Shield,
} from 'lucide-react'
import { toast } from 'sonner'
import { DetailModal } from '@/components/shared'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
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
import { useHasCapability } from '@/hooks/useCapability'
import { useAmountsVisible, maskedAmount } from './lib/amounts-visibility'

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

  // RBAC R-3
  const canSendInvoice = useHasCapability('finance:invoice:send')
  const amountsVisible = useAmountsVisible()

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

  // GoBD-Änderungsprotokoll — derived from the credit note lifecycle.
  // swap-ready: echter Backend-Audit-Stream ersetzt das später
  const auditEntries: { action: string; user: string; date: string; detail: string }[] = [
    {
      action: t('finanzen.creditNoteDetail.auditCreated'),
      user: 'Max Müller',
      date: cn.created_at,
      detail: t('finanzen.creditNoteDetail.auditCreatedDetail', { number: cn.credit_note_number }),
    },
    ...(cn.is_storno
      ? [{
          action: t('finanzen.creditNoteDetail.auditStorno'),
          user: 'Max Müller',
          date: cn.created_at,
          detail: t('finanzen.creditNoteDetail.auditStornoDetail', {
            invoice: cn.invoice_number ?? '—',
          }),
        }]
      : []),
    ...(isSent
      ? [{
          action: t('finanzen.creditNoteDetail.auditSent'),
          user: 'Max Müller',
          date: cn.created_at,
          detail: t('finanzen.creditNoteDetail.auditSentDetail', { email: cn.customer?.name ?? '—' }),
        }]
      : []),
  ]

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
            <span title={!amountsVisible ? t('rbac.gate.amountsHidden') : undefined}>
              {maskedAmount(amountsVisible, money(cn.tax_breakdown?.subtotal ?? cn.total_net ?? 0))}
            </span>
          </div>
          {Object.entries(cn.tax_breakdown?.tax_by_rate ?? {}).map(([rate, amount]) => (
            <div key={rate} className="flex justify-between text-muted-foreground">
              <span>MwSt {rate}%</span>
              <span title={!amountsVisible ? t('rbac.gate.amountsHidden') : undefined}>
                {maskedAmount(amountsVisible, money(Number(amount)))}
              </span>
            </div>
          ))}
          <div className="flex justify-between border-t border-border pt-1.5 text-sm font-medium text-foreground">
            <span>{t('finanzen.totals.totalAmount')}</span>
            <span title={!amountsVisible ? t('rbac.gate.amountsHidden') : undefined}>
              {maskedAmount(amountsVisible, money(cn.tax_breakdown?.gross_total ?? cn.total_gross ?? 0))}
            </span>
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

        {/* Notes */}
        {cn.reason && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-1">
              {t('finanzen.creditNoteDetail.reason')}
            </h4>
            <p className="text-xs text-muted-foreground">{cn.reason}</p>
          </section>
        )}

        {/* GoBD Audit Log — Gutschrift-Änderungsprotokoll */}
        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex items-center gap-1">
            <Shield className="h-3 w-3" />
            {t('finanzen.invoiceDetail.auditTitle')}
          </h4>
          <div className="rounded-md border border-border overflow-hidden">
            {auditEntries.map((entry, idx) => (
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

        {/* Kundenkonto — alle Belege des Kunden + CRM-Sprung */}
        <CustomerAccountSection customer={cn.customer} currentDocId={cn.id} />

        {/* Actions */}
        <TooltipProvider>
          <div className="flex flex-wrap gap-2 border-t border-border pt-4">
            {/* Send → AUSNAHME invoice:send: visible when draft, disabled without right */}
            {!isSent && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="inline-flex" tabIndex={0}>
                    <Button
                      size="sm"
                      onClick={() => {
                        if (!canSendInvoice) return
                        sendCreditNote.mutate(creditNoteId, {
                          onSuccess: () => toast.success(`${cn.credit_note_number} ${t('finanzen.dunning.sent')}`),
                          onError: (err) => toast.error(err.message),
                        })
                      }}
                      disabled={!canSendInvoice}
                      className={!canSendInvoice ? 'pointer-events-none' : ''}
                    >
                      <Send className="mr-1.5 h-4 w-4" />
                      {t('finanzen.dunning.send')}
                    </Button>
                  </span>
                </TooltipTrigger>
                {!canSendInvoice && (
                  <TooltipContent>{t('rbac.gate.sendDisabled')}</TooltipContent>
                )}
              </Tooltip>
            )}
            {/* PDF — always free */}
            <Button variant="outline" size="sm" onClick={() => downloadPDF.mutate(creditNoteId)}>
              <Download className="mr-1.5 h-4 w-4" />
              {t('finanzen.pdf.downloadPdf')}
            </Button>
          </div>
        </TooltipProvider>
      </div>
    </DetailModal>
  )
}
