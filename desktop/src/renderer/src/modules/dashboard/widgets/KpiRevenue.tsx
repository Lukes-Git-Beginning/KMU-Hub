/**
 * KPI Revenue widget — monthly revenue with comparison to previous month.
 */
import { memo } from 'react'
import { Euro, TrendingUp, TrendingDown } from 'lucide-react'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

const MOCK = {
  currentMonth: 84_750,
  previousMonth: 71_200,
  monthlyData: [52_300, 61_400, 58_900, 67_100, 71_200, 84_750],
  labels: ['Sep', 'Okt', 'Nov', 'Dez', 'Jan', 'Feb'],
}

function fmt(n: number) {
  return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(n)
}

function KpiRevenue(_props: WidgetProps) {
  const diff = MOCK.currentMonth - MOCK.previousMonth
  const pct = ((diff / MOCK.previousMonth) * 100).toFixed(1)
  const isUp = diff >= 0
  const max = Math.max(...MOCK.monthlyData)

  return (
    <div className="flex h-full flex-col justify-between p-4">
      {/* Top: value + change */}
      <div>
        <div className="flex items-center gap-2 mb-1">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-500/10">
            <Euro className="h-4 w-4 text-emerald-600" />
          </div>
          <span className="text-xs font-medium text-muted-foreground">Monatsumsatz</span>
        </div>
        <p className="text-2xl font-bold text-foreground">{fmt(MOCK.currentMonth)}</p>
        <div className="mt-1 flex items-center gap-1">
          {isUp ? (
            <TrendingUp className="h-3.5 w-3.5 text-emerald-500" />
          ) : (
            <TrendingDown className="h-3.5 w-3.5 text-red-500" />
          )}
          <span className={`text-xs font-medium ${isUp ? 'text-emerald-600' : 'text-red-600'}`}>
            {isUp ? '+' : ''}{pct}%
          </span>
          <span className="text-xs text-muted-foreground">vs. Vormonat</span>
        </div>
      </div>

      {/* Bottom: mini bar chart */}
      <div className="flex items-end gap-1 mt-3" style={{ height: 48 }}>
        {MOCK.monthlyData.map((val, i) => (
          <div key={i} className="flex flex-1 flex-col items-center gap-1">
            <div
              className={`w-full rounded-sm transition-all ${
                i === MOCK.monthlyData.length - 1 ? 'bg-emerald-500' : 'bg-emerald-500/30'
              }`}
              style={{ height: `${(val / max) * 40}px` }}
            />
            <span className="text-[9px] text-muted-foreground">{MOCK.labels[i]}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

export default memo(KpiRevenue)
