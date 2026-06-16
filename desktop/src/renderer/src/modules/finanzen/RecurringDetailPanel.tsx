/**
 * RecurringDetailPanel — Detailansicht einer wiederkehrenden Rechnung (finanzen P2.5b).
 *
 * Zeigt Zeitplan (Intervall/nächster Lauf/erzeugte Anzahl), Positionen + Summen,
 * Notizen + Aktionen (jetzt generieren/pausieren/bearbeiten). Nutzt shared `DetailPanel`.
 */
import { useTranslation } from 'react-i18next'
import { Repeat, CalendarClock, User, Hash, Play, Pause, Zap } from 'lucide-react'
import { toast } from 'sonner'
import { DetailModal } from '@/components/shared'
import { Button } from '@/components/ui/button'
import type { RecurringInvoice, RecurringStatus } from '@/types/finance-types'
import {
  usePauseRecurringInvoice,
  useGenerateRecurringInvoice,
} from '@/api/hooks/useFinance'
import { formatMoney } from '@/stores/finance'
import { formatDate } from '@/lib/format'

const STATUS_STYLES: Record<RecurringStatus, string> = {
  active: 'bg-success-light text-success',
  paused: 'bg-warning/15 text-warning',
  ended: 'bg-secondary text-muted-foreground',
}

interface RecurringDetailPanelProps {
  recurring: RecurringInvoice
  onClose: () => void
  onEdit: () => void
}

export function RecurringDetailPanel({ recurring: r, onClose, onEdit }: RecurringDetailPanelProps) {
  const { t } = useTranslation()
  const pauseRec = usePauseRecurringInvoice()
  const generateRec = useGenerateRecurringInvoice()
  const currency = r.currency ?? 'EUR'
  const money = (v: number | string) => formatMoney(v, currency)

  const info: { icon: typeof Repeat; label: string; value: string }[] = [
    { icon: Repeat, label: t('finanzen.recurring.intervalCol'), value: t(`finanzen.recurring.intervals.${r.interval}`) },
    { icon: CalendarClock, label: t('finanzen.recurring.nextRun'), value: r.status === 'ended' ? '–' : formatDate(r.next_run) },
    { icon: User, label: t('finanzen.customer'), value: r.customer?.name ?? '' },
    { icon: Hash, label: t('finanzen.recurring.generatedLabel'), value: String(r.generated_count ?? 0) },
  ]

  return (
    <DetailModal open={true} title={t('finanzen.recurringDetail.title')} onClose={onClose}>
      <div className="space-y-4">
        {/* Header */}
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h3 className="truncate text-base font-medium text-foreground">{r.title}</h3>
            <p className="text-xs text-muted-foreground">{r.customer?.name}</p>
          </div>
          <span className={`inline-flex shrink-0 items-center gap-1 rounded-full px-2.5 py-0.5 text-[10px] font-medium ${STATUS_STYLES[r.status]}`}>
            {r.status === 'active' && <Play className="h-2.5 w-2.5" />}
            {r.status === 'paused' && <Pause className="h-2.5 w-2.5" />}
            {t(`finanzen.recurring.status.${r.status}`)}
          </span>
        </div>

        {/* Key info */}
        <div className="grid grid-cols-2 gap-3 text-xs">
          {info.map((row) => (
            <div key={row.label} className="flex items-center gap-2 text-muted-foreground">
              <row.icon className="h-3.5 w-3.5 shrink-0" />
              <div className="min-w-0">
                <p className="text-[10px] text-muted-foreground">{row.label}</p>
                <p className="truncate text-foreground">{row.value}</p>
              </div>
            </div>
          ))}
        </div>

        {/* Line items */}
        <section>
          <h4 className="mb-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
            {t('finanzen.lineItems.positions')}
          </h4>
          <div className="overflow-hidden rounded-md border border-border">
            {(r.line_items ?? []).map((item, idx) => (
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

        {/* Total */}
        <div className="flex justify-between border-t border-border pt-1.5 text-sm font-medium text-foreground">
          <span>{t('finanzen.recurring.perInvoice')}</span>
          <span>{money(r.tax_breakdown?.gross_total ?? 0)}</span>
        </div>

        {/* Notes */}
        {r.notes && (
          <section>
            <h4 className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{t('finanzen.notes')}</h4>
            <p className="text-xs text-muted-foreground">{r.notes}</p>
          </section>
        )}

        {/* Actions */}
        <div className="flex flex-wrap gap-2 border-t border-border pt-4">
          {r.status !== 'ended' && (
            <>
              <Button size="sm" onClick={() => generateRec.mutate(r.id, {
                onSuccess: () => { toast.success(t('finanzen.recurring.generated', { title: r.title })); onClose() },
                onError: (err) => toast.error(err.message),
              })}>
                <Zap className="mr-1.5 h-4 w-4" />
                {t('finanzen.recurring.generateNow')}
              </Button>
              <Button variant="outline" size="sm" onClick={() => {
                const paused = r.status === 'active'
                pauseRec.mutate({ id: r.id, paused }, {
                  onSuccess: () => { toast.success(paused ? t('finanzen.recurring.paused') : t('finanzen.recurring.resumed')); onClose() },
                  onError: (err) => toast.error(err.message),
                })
              }}>
                {r.status === 'active' ? <Pause className="mr-1.5 h-4 w-4" /> : <Play className="mr-1.5 h-4 w-4" />}
                {r.status === 'active' ? t('finanzen.recurring.pause') : t('finanzen.recurring.resume')}
              </Button>
            </>
          )}
          <Button variant="outline" size="sm" onClick={onEdit}>{t('common.edit')}</Button>
        </div>
      </div>
    </DetailModal>
  )
}
