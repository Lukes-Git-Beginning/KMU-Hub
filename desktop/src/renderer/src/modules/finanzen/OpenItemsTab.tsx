import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { AlertTriangle, Clock, Wallet, CalendarClock } from 'lucide-react'
import { EmptyState } from '@/components/shared'
import { useInvoices } from '@/api/hooks/useFinance'
import { formatMoney, formatEUR, calcInvoiceTotal } from '@/stores/finance'
import { formatDate } from '@/lib/format'
import type { Invoice } from '@/types/finance-types'
import { useAmountsVisible, maskedAmount } from './lib/amounts-visibility'

type Bucket = 'current' | 'd30' | 'd60' | 'd60plus'

interface OpenItem {
  invoice: Invoice
  gross: number
  grossEUR: number
  daysOverdue: number
  bucket: Bucket
  currency: string
}

const BUCKET_ORDER: Bucket[] = ['current', 'd30', 'd60', 'd60plus']

const BUCKET_STYLES: Record<Bucket, string> = {
  current: 'bg-info-light text-info',
  d30: 'bg-warning/15 text-warning',
  d60: 'bg-warning/25 text-warning',
  d60plus: 'bg-error-light text-error',
}

function daysBetween(dueISO: string): number {
  const due = new Date(dueISO + 'T00:00:00').getTime()
  const today = new Date(new Date().toISOString().split('T')[0] + 'T00:00:00').getTime()
  return Math.round((today - due) / 86_400_000)
}

function bucketFor(daysOverdue: number): Bucket {
  if (daysOverdue <= 0) return 'current'
  if (daysOverdue <= 30) return 'd30'
  if (daysOverdue <= 60) return 'd60'
  return 'd60plus'
}

interface OpenItemsTabProps {
  onOpenInvoice?: (id: string) => void
}

export function OpenItemsTab({ onOpenInvoice }: OpenItemsTabProps) {
  const { t } = useTranslation()
  const { data, isLoading } = useInvoices()
  const [activeBucket, setActiveBucket] = useState<Bucket | 'all'>('all')
  const amountsVisible = useAmountsVisible()

  const items = useMemo<OpenItem[]>(() => {
    const invoices = data?.invoices ?? []
    return invoices
      .filter((inv) => inv.status === 'sent' || inv.status === 'overdue')
      .map((inv) => {
        const gross =
          Number(inv.tax_breakdown?.gross_total) || calcInvoiceTotal(inv.line_items ?? [])
        const rate = Number(inv.exchange_rate ?? 1) || 1
        const currency = inv.currency ?? 'EUR'
        const daysOverdue = daysBetween(inv.due_date)
        return {
          invoice: inv,
          gross,
          grossEUR: currency === 'EUR' ? gross : gross * rate,
          daysOverdue,
          bucket: bucketFor(daysOverdue),
          currency,
        }
      })
      .sort((a, b) => b.daysOverdue - a.daysOverdue)
  }, [data])

  const stats = useMemo(() => {
    const totalEUR = items.reduce((s, it) => s + it.grossEUR, 0)
    const overdue = items.filter((it) => it.daysOverdue > 0)
    const overdueEUR = overdue.reduce((s, it) => s + it.grossEUR, 0)
    const avgDays = overdue.length
      ? Math.round(overdue.reduce((s, it) => s + it.daysOverdue, 0) / overdue.length)
      : 0
    const byBucket: Record<Bucket, { count: number; eur: number }> = {
      current: { count: 0, eur: 0 },
      d30: { count: 0, eur: 0 },
      d60: { count: 0, eur: 0 },
      d60plus: { count: 0, eur: 0 },
    }
    for (const it of items) {
      byBucket[it.bucket].count += 1
      byBucket[it.bucket].eur += it.grossEUR
    }
    return { totalEUR, overdueEUR, overdueCount: overdue.length, avgDays, byBucket }
  }, [items])

  const filtered = activeBucket === 'all' ? items : items.filter((it) => it.bucket === activeBucket)

  if (isLoading) {
    return (
      <div className="space-y-3 py-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="h-14 rounded-xl bg-secondary/50 animate-shimmer" />
        ))}
      </div>
    )
  }

  if (items.length === 0) {
    return (
      <EmptyState
        icon={Wallet}
        title={t('finanzen.openItems.emptyTitle')}
        description={t('finanzen.openItems.emptyDescription')}
      />
    )
  }

  return (
    <div className="space-y-5 animate-fade-up">
      {/* KPI cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <div className="rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Wallet className="h-4 w-4" />
            <span className="text-xs">{t('finanzen.openItems.totalOpen')}</span>
          </div>
          <p
            className="mt-1.5 text-xl font-semibold text-foreground stat-accent"
            title={!amountsVisible ? t('rbac.gate.amountsHidden') : undefined}
          >
            {maskedAmount(amountsVisible, formatEUR(stats.totalEUR))}
          </p>
          <p className="text-[10px] text-muted-foreground">{t('finanzen.openItems.itemCount', { count: items.length })}</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-2 text-error">
            <AlertTriangle className="h-4 w-4" />
            <span className="text-xs">{t('finanzen.openItems.totalOverdue')}</span>
          </div>
          <p
            className="mt-1.5 text-xl font-semibold text-error"
            title={!amountsVisible ? t('rbac.gate.amountsHidden') : undefined}
          >
            {maskedAmount(amountsVisible, formatEUR(stats.overdueEUR))}
          </p>
          <p className="text-[10px] text-muted-foreground">{t('finanzen.openItems.itemCount', { count: stats.overdueCount })}</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Clock className="h-4 w-4" />
            <span className="text-xs">{t('finanzen.openItems.avgOverdue')}</span>
          </div>
          <p className="mt-1.5 text-xl font-semibold text-foreground">{t('finanzen.openItems.days', { count: stats.avgDays })}</p>
          <p className="text-[10px] text-muted-foreground">{t('finanzen.openItems.avgOverdueHint')}</p>
        </div>
      </div>

      {/* Aging buckets */}
      <div className="flex flex-wrap items-center gap-2">
        <button
          onClick={() => setActiveBucket('all')}
          className={`rounded-lg border px-3 py-1.5 text-xs transition-colors ${
            activeBucket === 'all'
              ? 'border-primary bg-primary/10 text-primary'
              : 'border-border text-muted-foreground hover:text-foreground'
          }`}
        >
          {t('finanzen.openItems.bucketAll')} ({items.length})
        </button>
        {BUCKET_ORDER.map((b) => (
          <button
            key={b}
            onClick={() => setActiveBucket(b)}
            disabled={stats.byBucket[b].count === 0}
            className={`flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs transition-colors disabled:opacity-40 ${
              activeBucket === b
                ? 'border-primary bg-primary/10 text-primary'
                : 'border-border text-muted-foreground hover:text-foreground'
            }`}
          >
            <span className={`inline-block h-2 w-2 rounded-full ${BUCKET_STYLES[b].split(' ')[0]}`} />
            {t(`finanzen.openItems.bucket.${b}`)} ({stats.byBucket[b].count})
            <span
              className="text-[10px] text-muted-foreground"
              title={!amountsVisible ? t('rbac.gate.amountsHidden') : undefined}
            >
              · {maskedAmount(amountsVisible, formatEUR(stats.byBucket[b].eur))}
            </span>
          </button>
        ))}
      </div>

      {/* Table */}
      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="grid grid-cols-[110px_1fr_110px_110px_120px_90px] gap-3 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
          <span>{t('finanzen.col.nr')}</span>
          <span>{t('finanzen.col.customer')}</span>
          <span>{t('finanzen.col.dueDate')}</span>
          <span>{t('finanzen.openItems.overdueCol')}</span>
          <span className="text-right">{t('finanzen.col.amount')}</span>
          <span>{t('finanzen.openItems.agingCol')}</span>
        </div>
        {filtered.map((it) => (
          <div
            key={it.invoice.id}
            role="button"
            tabIndex={0}
            onClick={() => onOpenInvoice?.(it.invoice.id)}
            onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onOpenInvoice?.(it.invoice.id) } }}
            className="grid cursor-pointer grid-cols-[110px_1fr_110px_110px_120px_90px] gap-3 items-center px-4 py-3 border-b border-border-muted hover:bg-secondary/30 transition-colors focus-visible:bg-secondary/40 focus-visible:outline-none"
          >
            <span className="text-sm font-mono text-primary">{it.invoice.invoice_number}</span>
            <span className="text-sm text-foreground truncate">{it.invoice.customer?.name ?? ''}</span>
            <span className="text-xs text-muted-foreground">{formatDate(it.invoice.due_date)}</span>
            <span className={`text-xs font-medium ${it.daysOverdue > 0 ? 'text-error' : 'text-muted-foreground'}`}>
              {it.daysOverdue > 0
                ? t('finanzen.openItems.daysOverdue', { count: it.daysOverdue })
                : t('finanzen.openItems.notDue', { count: Math.abs(it.daysOverdue) })}
            </span>
            <span
              className="text-sm font-medium text-foreground text-right"
              title={!amountsVisible ? t('rbac.gate.amountsHidden') : undefined}
            >
              {maskedAmount(amountsVisible, formatMoney(it.gross, it.currency))}
            </span>
            <span className={`inline-flex w-fit items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${BUCKET_STYLES[it.bucket]}`}>
              {t(`finanzen.openItems.bucket.${it.bucket}`)}
            </span>
          </div>
        ))}
      </div>

      <p className="flex items-center gap-1.5 text-[10px] text-muted-foreground">
        <CalendarClock className="h-3 w-3" />
        {t('finanzen.openItems.fxNote')}
      </p>
    </div>
  )
}
