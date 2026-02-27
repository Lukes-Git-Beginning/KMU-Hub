/**
 * KPI Revenue widget — monthly revenue with comparison to previous month.
 */
import { memo } from 'react'
import { Euro, TrendingUp, TrendingDown } from 'lucide-react'
import { KPI, MONTHLY_REVENUE } from '@/mocks/mock-db'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

function fmt(n: number) {
  return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(n)
}

const last6 = MONTHLY_REVENUE.slice(-6)

function KpiRevenue(_props: WidgetProps) {
  const { current, previous, changePercent } = KPI.revenue
  const isUp = changePercent >= 0
  const max = Math.max(...last6.map((d) => d.revenue))

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
        <p className="text-2xl font-bold text-foreground">{fmt(current)}</p>
        <div className="mt-1 flex items-center gap-1">
          {isUp ? (
            <TrendingUp className="h-3.5 w-3.5 text-emerald-500" />
          ) : (
            <TrendingDown className="h-3.5 w-3.5 text-red-500" />
          )}
          <span className={`text-xs font-medium ${isUp ? 'text-emerald-600' : 'text-red-600'}`}>
            {isUp ? '+' : ''}{changePercent.toFixed(1)}%
          </span>
          <span className="text-xs text-muted-foreground">vs. Vormonat ({fmt(previous)})</span>
        </div>
      </div>

      {/* Bottom: mini bar chart */}
      <div className="flex items-end gap-1 mt-3" style={{ height: 48 }}>
        {last6.map((d, i) => (
          <div key={d.month} className="flex flex-1 flex-col items-center gap-1">
            <div
              className={`w-full rounded-sm transition-all ${
                i === last6.length - 1 ? 'bg-emerald-500' : 'bg-emerald-500/30'
              }`}
              style={{ height: `${(d.revenue / max) * 40}px` }}
            />
            <span className="text-[9px] text-muted-foreground">{d.label}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

export default memo(KpiRevenue)
