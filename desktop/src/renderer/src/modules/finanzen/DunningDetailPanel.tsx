/**
 * DunningDetailPanel — Mahn-Detailansicht (finanzen P2.5d).
 *
 * Öffnet beim Zeilenklick in der Mahnwesen-Liste: verknüpfte Rechnung (klickbar
 * → Rechnungs-Modal), Eskalations-Kette (1 → 2 → 3), Gebühr/Zinsen/Summe, Daten
 * und Aktionen (senden/eskalieren/PDF). Nutzt das shared `DetailModal`.
 */
import { useTranslation } from 'react-i18next'
import { AlertTriangle, Send, Download, ChevronRight, Calendar, Clock, FileText } from 'lucide-react'
import { toast } from 'sonner'
import { DetailModal } from '@/components/shared'
import { Button } from '@/components/ui/button'
import {
  useSendDunning,
  useEscalateDunning,
  useDownloadDunningPDF,
} from '@/api/hooks/useFinance'
import { formatEUR } from '@/stores/finance'
import { formatDate } from '@/lib/format'
import type { DunningRecord, Invoice } from '@/types/finance-types'
import { useFinanceDetailNavOptional } from './FinanceDetailNav'

const LEVEL_LABEL_KEYS: Record<number, string> = {
  1: 'finanzen.dunning.level1',
  2: 'finanzen.dunning.level2',
  3: 'finanzen.dunning.level3',
}

const STATUS_CONFIG: Record<DunningRecord['status'], { labelKey: string; colors: string }> = {
  draft: { labelKey: 'finanzen.status.draft', colors: 'bg-secondary text-muted-foreground' },
  sent: { labelKey: 'finanzen.status.sent', colors: 'bg-info-light text-info' },
  paid: { labelKey: 'finanzen.status.paid', colors: 'bg-success-light text-success' },
}

interface DunningDetailPanelProps {
  dunning: DunningRecord
  /** Resolved original invoice (for number/customer/amount + cross-navigation). */
  invoice?: Invoice
  onClose: () => void
  onBack?: () => void
}

export function DunningDetailPanel({ dunning: d, invoice, onClose, onBack }: DunningDetailPanelProps) {
  const { t } = useTranslation()
  const nav = useFinanceDetailNavOptional()
  const sendDunning = useSendDunning()
  const escalateDunning = useEscalateDunning()
  const downloadPDF = useDownloadDunningPDF()

  const sc = STATUS_CONFIG[d.status]
  const fee = Number(d.fee)
  const interest = Number(d.interest)
  const gross = Number(invoice?.tax_breakdown?.gross_total ?? invoice?.total_gross ?? 0)
  const totalDue = gross + fee + interest

  const handleSend = () =>
    sendDunning.mutate(d.id, {
      onSuccess: () => toast.success(`${t(LEVEL_LABEL_KEYS[d.level])} ${t('finanzen.dunning.sent')}`),
      onError: (err) => toast.error(err.message),
    })

  const handleEscalate = () =>
    escalateDunning.mutate(d.id, {
      onSuccess: () =>
        toast.success(
          t('finanzen.dunning.escalatedTo', {
            level: t(LEVEL_LABEL_KEYS[Math.min(d.level + 1, 3) as 1 | 2 | 3]),
          }),
        ),
      onError: (err) => toast.error(err.message),
    })

  return (
    <DetailModal open={true} title={t('finanzen.dunningDetail.title')} onClose={onClose} onBack={onBack}>
      <div className="space-y-4">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-2">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-warning-light">
              <AlertTriangle className="h-4 w-4 text-warning" />
            </div>
            <div>
              <h3 className="text-base font-medium text-foreground">{t(LEVEL_LABEL_KEYS[d.level])}</h3>
              {invoice && <p className="font-mono text-xs text-muted-foreground">{invoice.invoice_number}</p>}
            </div>
          </div>
          <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-[10px] font-medium ${sc.colors}`}>
            {t(sc.labelKey)}
          </span>
        </div>

        {/* Escalation chain */}
        <div className="flex items-center gap-1.5 rounded-lg border border-border bg-secondary/30 px-3 py-2.5">
          {[1, 2, 3].map((step) => (
            <div key={step} className="flex items-center gap-1.5">
              <span
                className={`inline-flex h-6 w-6 items-center justify-center rounded-full text-[11px] font-semibold ${
                  step <= d.level
                    ? step === 3
                      ? 'bg-error text-white'
                      : step === 2
                        ? 'bg-orange-500 text-white'
                        : 'bg-warning text-white'
                    : 'bg-secondary text-muted-foreground'
                }`}
              >
                {step}
              </span>
              {step < 3 && (
                <ChevronRight className={`h-3.5 w-3.5 ${step < d.level ? 'text-foreground' : 'text-muted-foreground/40'}`} />
              )}
            </div>
          ))}
          <span className="ml-2 text-[11px] text-muted-foreground">{t('finanzen.dunningDetail.escalationChain')}</span>
        </div>

        {/* Linked invoice (clickable) */}
        {invoice && (
          <button
            type="button"
            disabled={!nav}
            onClick={() => nav?.open('invoice', invoice.id)}
            className="group flex w-full items-center justify-between gap-2 rounded-lg border border-border bg-secondary/40 p-3 text-xs text-left transition-colors hover:bg-secondary disabled:cursor-default disabled:hover:bg-secondary/40"
          >
            <span className="flex min-w-0 items-center gap-2">
              <FileText className="h-4 w-4 shrink-0 text-muted-foreground" />
              <span className="min-w-0">
                <span className="block font-mono text-primary">{invoice.invoice_number}</span>
                <span className="block truncate text-[10px] text-muted-foreground">{invoice.customer?.name}</span>
              </span>
            </span>
            <span className="flex shrink-0 items-center gap-1">
              <span className="font-medium text-foreground">{formatEUR(gross)}</span>
              {nav && <ChevronRight className="h-3.5 w-3.5 text-muted-foreground transition-[color,transform] group-hover:translate-x-0.5 group-hover:text-primary" />}
            </span>
          </button>
        )}

        {/* Key info */}
        <div className="grid grid-cols-2 gap-3 text-xs">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Calendar className="h-3.5 w-3.5 shrink-0" />
            <div>
              <p className="text-[10px] text-muted-foreground">{t('finanzen.dunningDetail.created')}</p>
              <p className="text-foreground">{formatDate(d.created_at)}</p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <Clock className="h-3.5 w-3.5 shrink-0" />
            <div>
              <p className="text-[10px] text-muted-foreground">{t('finanzen.status.sent')}</p>
              <p className="text-foreground">{d.sent_at ? formatDate(d.sent_at) : '--'}</p>
            </div>
          </div>
        </div>

        {/* Amounts */}
        <div className="space-y-1.5 rounded-lg border border-border p-3 text-xs">
          {invoice && (
            <div className="flex justify-between text-muted-foreground">
              <span>{t('finanzen.dunningDetail.invoiceAmount')}</span>
              <span>{formatEUR(gross)}</span>
            </div>
          )}
          <div className="flex justify-between text-muted-foreground">
            <span>{t('finanzen.dunning.fee')}</span>
            <span>{formatEUR(fee)}</span>
          </div>
          <div className="flex justify-between text-muted-foreground">
            <span>{t('finanzen.dunning.interest')}</span>
            <span>{formatEUR(interest)}</span>
          </div>
          <div className="flex justify-between border-t border-border pt-1.5 text-sm font-medium text-foreground">
            <span>{t('finanzen.dunningDetail.totalDue')}</span>
            <span>{formatEUR(totalDue)}</span>
          </div>
        </div>

        {/* Actions */}
        <div className="flex flex-wrap gap-2 border-t border-border pt-4">
          {d.status === 'draft' && (
            <Button size="sm" onClick={handleSend} disabled={sendDunning.isPending}>
              <Send className="mr-1.5 h-4 w-4" />
              {t('finanzen.dunning.send')}
            </Button>
          )}
          {d.level < 3 && d.status !== 'paid' && (
            <Button variant="outline" size="sm" onClick={handleEscalate} disabled={escalateDunning.isPending}>
              <AlertTriangle className="mr-1.5 h-4 w-4" />
              {t('finanzen.dunning.escalate')}
            </Button>
          )}
          <Button variant="outline" size="sm" onClick={() => downloadPDF.mutate(d.id)} disabled={downloadPDF.isPending}>
            <Download className="mr-1.5 h-4 w-4" />
            PDF
          </Button>
        </div>
      </div>
    </DetailModal>
  )
}
