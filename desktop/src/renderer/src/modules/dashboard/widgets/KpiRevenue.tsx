/**
 * KPI Revenue widget — monthly revenue with comparison to previous month.
 */
import { memo } from 'react'
import { useTranslation } from 'react-i18next'
import { Euro, TrendingUp, TrendingDown } from 'lucide-react'
import { useFinanceDashboard } from '@/api/hooks/useFinance'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

function fmt(n: number) {
  return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(n)
}

function KpiRevenue(_props: WidgetProps) {
  const { t } = useTranslation()
  const { data: dashboard, isLoading } = useFinanceDashboard()

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div role="status" aria-label={t('common.loading')} className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    )
  }

  const current = dashboard?.revenue_this_month ?? 0
  const previous = dashboard?.revenue_last_month ?? 0
  const changePercent = previous > 0 ? ((current - previous) / previous) * 100 : 0
  const isUp = changePercent >= 0

  const MONTH_LABELS = ['Jan', 'Feb', 'Maer', 'Apr', 'Mai', 'Jun', 'Jul', 'Aug', 'Sep', 'Okt', 'Nov', 'Dez']
  const chartData = (dashboard?.monthly_revenue ?? []).map((d) => {
    const date = new Date(d.month)
    return { month: d.month, label: MONTH_LABELS[date.getMonth()], revenue: d.revenue }
  })
  const max = chartData.length > 0 ? Math.max(...chartData.map((d) => d.revenue)) : 1

  return (
    <div className="flex h-full flex-col justify-between p-4">
      {/* Top: value + change */}
      <div>
        <div className="flex items-center gap-2 mb-1">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-success/10">
            <Euro className="h-4 w-4 text-success" />
          </div>
          <span className="text-xs font-medium text-muted-foreground">{t('dashboard.kpiRevenue.monthlyRevenue')}</span>
        </div>
        <p className="text-2xl font-bold text-foreground">{fmt(current)}</p>
        <div className="mt-1 flex items-center gap-1">
          {isUp ? (
            <TrendingUp className="h-3.5 w-3.5 text-success" />
          ) : (
            <TrendingDown className="h-3.5 w-3.5 text-destructive" />
          )}
          <span className={`text-xs font-medium ${isUp ? 'text-success' : 'text-destructive'}`}>
            {isUp ? '+' : ''}{changePercent.toFixed(1)}%
          </span>
          <span className="text-xs text-muted-foreground">{t('dashboard.kpiRevenue.vsPreviousMonth', { amount: fmt(previous) })}</span>
        </div>
      </div>

      {/* Bottom: mini bar chart */}
      <div className="flex items-end gap-1 mt-3" style={{ height: 48 }}>
        {chartData.length > 0 ? (
          chartData.map((d, i) => (
            <div key={d.month} className="flex flex-1 flex-col items-center gap-1">
              <div
                className={`w-full rounded-sm transition-all ${
                  i === chartData.length - 1 ? 'bg-success' : 'bg-success/30'
                }`}
                style={{ height: `${(d.revenue / max) * 40}px` }}
              />
              <span className="text-[9px] text-muted-foreground">{d.label}</span>
            </div>
          ))
        ) : (
          <div className="flex w-full items-center justify-center">
            <p className="text-[10px] text-muted-foreground">{t('dashboard.kpiRevenue.noMonthlyData')}</p>
          </div>
        )}
      </div>
    </div>
  )
}

export default memo(KpiRevenue)
