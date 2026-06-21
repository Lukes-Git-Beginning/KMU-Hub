import { useTranslation } from 'react-i18next'
import type { OutcomeBreakdownEntry } from '@/api/dialer-client'

interface OutcomeDonutChartProps {
  data: OutcomeBreakdownEntry[]
  total: number
  size?: number
  strokeWidth?: number
}

export default function OutcomeDonutChart({
  data,
  total,
  size = 160,
  strokeWidth = 20,
}: OutcomeDonutChartProps) {
  const { t } = useTranslation()
  const radius = (size - strokeWidth) / 2
  const circumference = 2 * Math.PI * radius
  const center = size / 2

  // Pre-compute offsets to avoid mutable state during render
  const segments = data.reduce<Array<{ entry: OutcomeBreakdownEntry; dashLength: number; offset: number }>>(
    (acc, entry) => {
      const pct = total > 0 ? entry.count / total : 0
      const dashLength = pct * circumference
      const prevOffset = acc.length > 0 ? acc[acc.length - 1].offset + acc[acc.length - 1].dashLength : 0
      acc.push({ entry, dashLength, offset: prevOffset })
      return acc
    },
    [],
  )

  return (
    <div className="flex flex-col items-center gap-4">
      <div className="relative" style={{ width: size, height: size }}>
        <svg width={size} height={size} className="-rotate-90">
          {/* Background circle */}
          <circle
            cx={center}
            cy={center}
            r={radius}
            fill="none"
            stroke="var(--border)"
            strokeWidth={strokeWidth}
          />
          {/* Data segments */}
          {segments.map(({ entry, dashLength, offset }) => (
              <circle
                key={entry.outcome_id}
                cx={center}
                cy={center}
                r={radius}
                fill="none"
                stroke={entry.outcome_color}
                strokeWidth={strokeWidth}
                strokeDasharray={`${dashLength} ${circumference - dashLength}`}
                strokeDashoffset={-offset}
                strokeLinecap="round"
                className="transition-all duration-700"
              />
          ))}
        </svg>
        {/* Center label */}
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-2xl font-bold text-foreground tabular-nums">{total}</span>
          <span className="text-xs text-muted-foreground">{t('dialer.dashboard.callsLabel')}</span>
        </div>
      </div>

      {/* Legend */}
      <div className="flex flex-wrap justify-center gap-x-4 gap-y-1.5">
        {data.map((entry) => (
          <div key={entry.outcome_id} className="flex items-center gap-1.5">
            <span
              className="h-2.5 w-2.5 rounded-full shrink-0"
              style={{ backgroundColor: entry.outcome_color }}
            />
            <span className="text-xs text-muted-foreground">{entry.outcome_label}</span>
            <span className="text-xs font-medium tabular-nums">{entry.count}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
