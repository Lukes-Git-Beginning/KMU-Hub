/**
 * Berichte-Tab für den Buchhaltungs-Hub (finanzen P2).
 *
 * Datengetriebene Migration der alten buchhaltung-Reports: Einnahmen/Ausgaben
 * je Monat (letzte 6 Monate, aus den echten Transaktionen) + Ausgaben nach
 * Kategorie. Die vollwertige EÜR-Auswertung (recharts) folgt in P5.
 */
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { BarChart3, PieChart, EyeOff } from 'lucide-react'
import { useTransactions } from '@/api/hooks/useFinanceLedger'
import { EmptyState } from '@/components/shared'
import { formatCurrency } from '@/lib/format'
import { useAmountsVisible } from '../lib/amounts-visibility'

const MONTHS_BACK = 6

export function BerichteTab() {
  const { t, i18n } = useTranslation()
  const { data: transactions = [] } = useTransactions()
  const amountsVisible = useAmountsVisible()

  // Letzte 6 Monate als Buckets (income/expense-Summe je Monat).
  const months = useMemo(() => {
    const now = new Date()
    const buckets: { key: string; label: string; income: number; expense: number }[] = []
    for (let i = MONTHS_BACK - 1; i >= 0; i--) {
      const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
      buckets.push({
        key: `${d.getFullYear()}-${d.getMonth()}`,
        label: d.toLocaleDateString(i18n.language, { month: 'short' }),
        income: 0,
        expense: 0,
      })
    }
    const byKey = new Map(buckets.map((b) => [b.key, b]))
    for (const tx of transactions) {
      const d = new Date(tx.date)
      const bucket = byKey.get(`${d.getFullYear()}-${d.getMonth()}`)
      if (!bucket) continue
      if (tx.type === 'income') bucket.income += Math.abs(tx.amount)
      else bucket.expense += Math.abs(tx.amount)
    }
    return buckets
  }, [transactions, i18n.language])

  const maxValue = useMemo(
    () => Math.max(1, ...months.flatMap((m) => [m.income, m.expense])),
    [months],
  )

  const byCategory = useMemo(() => {
    const cats = new Map<string, number>()
    transactions
      .filter((tx) => tx.type === 'expense')
      .forEach((tx) => cats.set(tx.category, (cats.get(tx.category) ?? 0) + Math.abs(tx.amount)))
    const total = Array.from(cats.values()).reduce((s, v) => s + v, 0) || 1
    return { entries: Array.from(cats.entries()).sort((a, b) => b[1] - a[1]).slice(0, 5), total }
  }, [transactions])

  const categoryColors = ['bg-primary', 'bg-info', 'bg-warning', 'bg-error', 'bg-success']

  if (transactions.length === 0) {
    return (
      <EmptyState
        icon={BarChart3}
        title={t('buchhaltung.reports.emptyTitle')}
        description={t('buchhaltung.reports.emptyDesc')}
      />
    )
  }

  // When amounts are hidden, replace the charts with a notice block
  if (!amountsVisible) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 rounded-xl border border-border bg-card py-16 text-center">
        <EyeOff className="h-8 w-8 text-muted-foreground" />
        <p className="text-sm font-medium text-foreground">{t('rbac.gate.amountsHidden')}</p>
      </div>
    )
  }

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      {/* Einnahmen vs. Ausgaben */}
      <div className="rounded-xl border border-border bg-card p-6">
        <div className="mb-4 flex items-center gap-2">
          <BarChart3 className="h-5 w-5 text-primary" aria-hidden="true" />
          <h3 className="text-sm font-medium text-foreground">{t('buchhaltung.reports.incomeVsExpenses')}</h3>
        </div>
        <div className="flex h-40 items-end gap-3">
          {months.map((m) => (
            <div key={m.key} className="flex flex-1 flex-col items-center gap-1">
              <div className="flex h-32 w-full items-end gap-1">
                <div
                  className="flex-1 rounded-t-sm bg-primary transition-[height]"
                  style={{ height: `${Math.max(2, (m.income / maxValue) * 100)}%` }}
                  title={`${m.label}: ${formatCurrency(m.income)}`}
                />
                <div
                  className="flex-1 rounded-t-sm bg-error/60 transition-[height]"
                  style={{ height: `${Math.max(2, (m.expense / maxValue) * 100)}%` }}
                  title={`${m.label}: −${formatCurrency(m.expense)}`}
                />
              </div>
              <span className="text-[10px] capitalize text-muted-foreground">{m.label}</span>
            </div>
          ))}
        </div>
        <div className="mt-3 flex items-center gap-4 text-xs text-muted-foreground">
          <span className="flex items-center gap-1">
            <span className="h-2 w-2 rounded-full bg-primary" /> {t('buchhaltung.stats.income')}
          </span>
          <span className="flex items-center gap-1">
            <span className="h-2 w-2 rounded-full bg-error/60" /> {t('buchhaltung.stats.expenses')}
          </span>
        </div>
      </div>

      {/* Ausgaben nach Kategorie */}
      <div className="rounded-xl border border-border bg-card p-6">
        <div className="mb-4 flex items-center gap-2">
          <PieChart className="h-5 w-5 text-primary" aria-hidden="true" />
          <h3 className="text-sm font-medium text-foreground">{t('buchhaltung.reports.expensesByCategory')}</h3>
        </div>
        <div className="space-y-3">
          {byCategory.entries.map(([name, amount], idx) => (
            <div key={name}>
              <div className="mb-1 flex items-center justify-between text-sm">
                <span className="text-foreground">
                  {t(`buchhaltung.categories.${name}`, { defaultValue: name })}
                </span>
                <span className="text-muted-foreground tabular-nums">{formatCurrency(amount)}</span>
              </div>
              <div className="h-2 rounded-full bg-secondary">
                <div
                  className={`h-full rounded-full ${categoryColors[idx % categoryColors.length]}`}
                  style={{ width: `${(amount / byCategory.total) * 100}%` }}
                />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
