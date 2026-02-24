/**
 * Mini Chart widget — monthly revenue bar chart with 12-month overview.
 */
import { memo, useState } from 'react'
import { BarChart3 } from 'lucide-react'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

const MOCK_DATA = [
  { month: 'Mär', value: 42_100 },
  { month: 'Apr', value: 38_700 },
  { month: 'Mai', value: 51_200 },
  { month: 'Jun', value: 47_800 },
  { month: 'Jul', value: 55_300 },
  { month: 'Aug', value: 43_600 },
  { month: 'Sep', value: 52_300 },
  { month: 'Okt', value: 61_400 },
  { month: 'Nov', value: 58_900 },
  { month: 'Dez', value: 67_100 },
  { month: 'Jan', value: 71_200 },
  { month: 'Feb', value: 84_750 },
]

function fmt(n: number) {
  return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(n)
}

function MiniChart(_props: WidgetProps) {
  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null)
  const max = Math.max(...MOCK_DATA.map((d) => d.value))
  const total = MOCK_DATA.reduce((sum, d) => sum + d.value, 0)

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
        {hoveredIdx !== null && (
          <div className="text-right animate-fade-up">
            <p className="text-xs text-muted-foreground">{MOCK_DATA[hoveredIdx].month}</p>
            <p className="text-sm font-bold text-foreground">{fmt(MOCK_DATA[hoveredIdx].value)}</p>
          </div>
        )}
      </div>

      {/* Bar chart */}
      <div className="flex-1 flex items-end gap-1 min-h-0">
        {MOCK_DATA.map((d, i) => {
          const h = (d.value / max) * 100
          const isLast = i === MOCK_DATA.length - 1
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
                {d.month}
              </span>
            </div>
          )
        })}
      </div>
    </div>
  )
}

export default memo(MiniChart)
