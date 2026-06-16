/**
 * RecurringDetailPanel — Detailansicht einer wiederkehrenden Rechnung (finanzen P2.5b).
 *
 * Zeigt Zeitplan (Intervall/nächster Lauf/erzeugte Anzahl), Positionen + Summen,
 * Notizen + Aktionen (jetzt generieren/pausieren/bearbeiten). Nutzt shared `DetailPanel`.
 */
import { useTranslation } from 'react-i18next'
import { Repeat, CalendarClock, User, Hash, Play, Pause, Zap, FileText, ChevronRight, History, Calendar } from 'lucide-react'
import { toast } from 'sonner'
import { DetailModal } from '@/components/shared'
import { Button } from '@/components/ui/button'
import type { RecurringInvoice, RecurringInterval, RecurringStatus } from '@/types/finance-types'
import {
  usePauseRecurringInvoice,
  useGenerateRecurringInvoice,
  useInvoices,
} from '@/api/hooks/useFinance'
import { formatMoney } from '@/stores/finance'
import { formatDate } from '@/lib/format'
import { CustomerAccountSection } from './CustomerAccountSection'
import { useFinanceDetailNavOptional } from './FinanceDetailNav'

/** Nächstes Lauf-Datum nach Intervall (für die Vorschau kommender Läufe). */
function addInterval(dateStr: string, interval: RecurringInterval): Date {
  const d = new Date(dateStr)
  if (interval === 'weekly') d.setDate(d.getDate() + 7)
  else if (interval === 'monthly') d.setMonth(d.getMonth() + 1)
  else if (interval === 'quarterly') d.setMonth(d.getMonth() + 3)
  else if (interval === 'yearly') d.setFullYear(d.getFullYear() + 1)
  return d
}

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
  const { data: invoicesData } = useInvoices({ recurring_id: r.id })
  const nav = useFinanceDetailNavOptional()
  const pauseRec = usePauseRecurringInvoice()
  const generateRec = useGenerateRecurringInvoice()
  const currency = r.currency ?? 'EUR'
  const money = (v: number | string) => formatMoney(v, currency)

  // Bisher erzeugte Rechnungen dieses Abos (serverseitig nach recurring_id
  // gefiltert; neueste zuerst).
  const generatedInvoices = [...(invoicesData?.invoices ?? [])].sort((a, b) =>
    (a.invoice_date ?? '') < (b.invoice_date ?? '') ? 1 : -1,
  )

  // Vorschau der nächsten drei Läufe (nur solange aktiv und vor End-Datum).
  const upcomingRuns: string[] = []
  if (r.status === 'active' && r.next_run) {
    let cursor = new Date(r.next_run)
    for (let i = 0; i < 3; i++) {
      if (r.end_date && cursor > new Date(r.end_date)) break
      upcomingRuns.push(cursor.toISOString().split('T')[0])
      cursor = addInterval(cursor.toISOString().split('T')[0], r.interval)
    }
  }

  const info: { icon: typeof Repeat; label: string; value: string }[] = [
    { icon: Repeat, label: t('finanzen.recurring.intervalCol'), value: t(`finanzen.recurring.intervals.${r.interval}`) },
    { icon: CalendarClock, label: t('finanzen.recurring.nextRun'), value: r.status === 'ended' ? '–' : formatDate(r.next_run) },
    { icon: User, label: t('finanzen.customer'), value: r.customer?.name ?? '' },
    { icon: Hash, label: t('finanzen.recurring.generatedLabel'), value: String(r.generated_count ?? 0) },
    { icon: Calendar, label: t('finanzen.recurringDetail.endDate'), value: r.end_date ? formatDate(r.end_date) : t('finanzen.recurringDetail.noEndDate') },
  ]

  // GoBD-Verlauf — swap-ready: ein echtes Audit-Event-Stream würde diesen Block ersetzen.
  const auditEntries: { action: string; date: string; detail: string }[] = [
    {
      action: t('finanzen.recurringDetail.historyCreated'),
      date: r.start_date || new Date().toISOString().split('T')[0],
      detail: t('finanzen.recurringDetail.historyCreatedDetail', {
        title: r.title,
        interval: t(`finanzen.recurring.intervals.${r.interval}`),
        date: formatDate(r.start_date),
      }),
    },
    ...(r.status === 'paused'
      ? [{
          action: t('finanzen.recurringDetail.historyPaused'),
          date: new Date().toISOString().split('T')[0],
          detail: t('finanzen.recurringDetail.historyPausedDetail'),
        }]
      : []),
    ...((r.generated_count ?? 0) > 0
      ? [{
          action: t('finanzen.recurringDetail.historyGeneratedCount', { count: r.generated_count ?? 0 }),
          date: generatedInvoices[0]?.invoice_date || new Date().toISOString().split('T')[0],
          detail: t('finanzen.recurringDetail.historyGeneratedDetail'),
        }]
      : []),
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

        {/* Nächste Läufe */}
        {upcomingRuns.length > 0 && (
          <section>
            <h4 className="mb-2 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
              <CalendarClock className="h-3 w-3" />
              {t('finanzen.recurringDetail.upcomingRuns')}
            </h4>
            <div className="flex flex-wrap gap-1.5">
              {upcomingRuns.map((run, idx) => (
                <span
                  key={run}
                  className={`inline-flex items-center gap-1 rounded-md border px-2 py-1 text-[11px] ${
                    idx === 0
                      ? 'border-primary/40 bg-primary/5 font-medium text-primary'
                      : 'border-border bg-secondary/40 text-muted-foreground'
                  }`}
                >
                  {formatDate(run)}
                </span>
              ))}
            </div>
          </section>
        )}

        {/* Bereits erzeugte Rechnungen (klickbar) */}
        {generatedInvoices.length > 0 && (
          <section>
            <h4 className="mb-2 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
              <FileText className="h-3 w-3" />
              {t('finanzen.recurringDetail.generatedInvoices')}
            </h4>
            <div className="overflow-hidden rounded-md border border-border">
              {generatedInvoices.map((inv, idx) => (
                <button
                  key={inv.id}
                  type="button"
                  disabled={!nav}
                  onClick={() => nav?.open('invoice', inv.id)}
                  className={`group flex w-full items-center justify-between px-3 py-2 text-left text-xs transition-colors hover:bg-secondary/50 disabled:cursor-default disabled:hover:bg-transparent ${
                    idx > 0 ? 'border-t border-border-muted' : ''
                  }`}
                >
                  <span className="flex min-w-0 items-center gap-2">
                    <span className="font-mono text-primary">{inv.invoice_number}</span>
                    <span className="text-[10px] text-muted-foreground">{formatDate(inv.invoice_date)}</span>
                  </span>
                  <span className="flex shrink-0 items-center gap-1">
                    <span className="font-medium text-foreground">
                      {money(inv.tax_breakdown?.gross_total ?? inv.total_gross ?? 0)}
                    </span>
                    {nav && <ChevronRight className="h-3.5 w-3.5 text-muted-foreground transition-[color,transform] group-hover:translate-x-0.5 group-hover:text-primary" />}
                  </span>
                </button>
              ))}
            </div>
          </section>
        )}

        {/* Kundenkonto — alle Belege des Kunden + CRM-Sprung */}
        {r.customer && <CustomerAccountSection customer={r.customer} />}

        {/* GoBD-Verlauf — swap-ready: ein echtes Audit-Event-Stream würde diesen Block ersetzen. */}
        <section>
          <h4 className="mb-2 flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
            <History className="h-3 w-3" />
            {t('finanzen.recurringDetail.historyTitle')}
          </h4>
          <div className="overflow-hidden rounded-md border border-border">
            {auditEntries.map((entry, idx) => (
              <div
                key={idx}
                className={`flex items-start gap-2 px-3 py-2 text-xs ${idx > 0 ? 'border-t border-border-muted' : ''}`}
              >
                <History className="mt-0.5 h-3 w-3 shrink-0 text-muted-foreground" />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <span className="font-medium text-foreground">{entry.action}</span>
                    <span className="text-[10px] text-muted-foreground">{formatDate(entry.date)}</span>
                  </div>
                  <p className="text-[10px] text-muted-foreground">{entry.detail}</p>
                </div>
              </div>
            ))}
          </div>
        </section>

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
