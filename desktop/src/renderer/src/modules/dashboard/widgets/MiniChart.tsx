/**
 * Mini Chart widget — monthly revenue bar chart with 12-month overview.
 */
import { memo, useState } from 'react'
import { BarChart3 } from 'lucide-react'
import { useFinanceDashboard } from '@/api/hooks/useFinance'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

function fmt(n: number) {
  return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(n)
}

function MiniChart(_props: WidgetProps) {
  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null)
  const { data: dashboard, isLoading } = useFinanceDashboard()

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div role="status" aria-label="Laden" className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    )
  }

  const MONTH_LABELS = ['Jan', 'Feb', 'Maer', 'Apr', 'Mai', 'Jun', 'Jul', 'Aug', 'Sep', 'Okt', 'Nov', 'Dez']
  const chartData = (dashboard?.monthly_revenue ?? []).map((d) => {
    const date = new Date(d.month)
    return { month: d.month, label: MONTH_LABELS[date.getMonth()], revenue: d.revenue }
  })
  const total = chartData.reduce((sum, d) => sum + d.revenue, 0)
  const max = chartData.length > 0 ? Math.max(...chartData.map((d) => d.revenue)) : 1

  return (
    <div className="flex h-full flex-col p-4">
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-500/10">
            <BarChart3 className="h-4 w-4 text-blue-600" />
          </div>
          <div>
            <p className="text-xs font-medium text-muted-foreground">Jahresumsatz</p>
            <p className="text-sm font-bold text-foreground">{fmt(total)}</p>
          </div>
        </div>
        {hoveredIdx !== null && chartData[hoveredIdx] && (
          <div className="text-right animate-fade-up">
            <p className="text-xs text-muted-foreground">{chartData[hoveredIdx].label}</p>
            <p className="text-sm font-bold text-foreground">{fmt(chartData[hoveredIdx].revenue)}</p>
          </div>
        )}
      </div>

      {/* Bar chart */}
      <div className="flex-1 flex items-end gap-1 min-h-0">
        {chartData.length > 0 ? (
          chartData.map((d, i) => {
            const h = (d.revenue / max) * 100
            const isLast = i === chartData.length - 1
            const isHovered = hoveredIdx === i
            return (
              <div
                key={d.month}
                className="flex flex-1 flex-col items-center gap-1 cursor-pointer"
                onMouseEnter={() => setHoveredIdx(i)}
                onMouseLeave={() => setHoveredIdx(null)}
              >
                <div className="w-full flex-1 flex items-end">
                  <div
                    className={`w-full rounded-t-sm transition-all duration-200 ${
                      isHovered
                        ? 'bg-blue-500'
                        : isLast
                          ? 'bg-blue-500'
                          : 'bg-blue-500/30'
                    }`}
                    style={{ height: `${h}%` }}
                  />
                </div>
                <span className={`text-[8px] leading-none ${
                  isHovered || isLast ? 'text-foreground font-medium' : 'text-muted-foreground'
                }`}>
                  {d.label}
                </span>
              </div>
            )
          })
        ) : (
          <div className="flex w-full items-center justify-center">
            <p className="text-xs text-muted-foreground">Keine Umsatzdaten vorhanden</p>
          </div>
        )}
      </div>
    </div>
  )
}

export default memo(MiniChart)
