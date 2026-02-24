/**
 * KPI Tasks widget — task completion progress with breakdown.
 */
import { memo } from 'react'
import { CheckCircle2, Circle, AlertTriangle } from 'lucide-react'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

const MOCK = {
  total: 24,
  done: 17,
  inProgress: 4,
  overdue: 3,
}

function KpiTasks(_props: WidgetProps) {
  const pct = Math.round((MOCK.done / MOCK.total) * 100)

  const segments = [
    { label: 'Erledigt', count: MOCK.done, color: 'bg-emerald-500', textColor: 'text-emerald-600', icon: CheckCircle2 },
    { label: 'In Arbeit', count: MOCK.inProgress, color: 'bg-blue-500', textColor: 'text-blue-600', icon: Circle },
    { label: 'Ueberfaellig', count: MOCK.overdue, color: 'bg-red-500', textColor: 'text-red-600', icon: AlertTriangle },
  ]

  return (
    <div className="flex h-full flex-col justify-between p-4">
      {/* Progress header */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs font-medium text-muted-foreground">Aufgaben diese Woche</span>
          <span className="text-xs font-bold text-foreground">{pct}%</span>
        </div>

        {/* Progress bar */}
        <div className="h-2.5 w-full rounded-full bg-secondary overflow-hidden flex">
          <div
            className="h-full bg-emerald-500 transition-all"
            style={{ width: `${(MOCK.done / MOCK.total) * 100}%` }}
          />
          <div
            className="h-full bg-blue-500 transition-all"
            style={{ width: `${(MOCK.inProgress / MOCK.total) * 100}%` }}
          />
          <div
            className="h-full bg-red-500 transition-all"
            style={{ width: `${(MOCK.overdue / MOCK.total) * 100}%` }}
          />
        </div>

        <p className="mt-2 text-lg font-bold text-foreground">
          {MOCK.done} <span className="text-sm font-normal text-muted-foreground">von {MOCK.total}</span>
        </p>
      </div>

      {/* Breakdown */}
      <div className="mt-3 space-y-1.5">
        {segments.map((seg) => {
          const Icon = seg.icon
          return (
            <div key={seg.label} className="flex items-center gap-2">
              <Icon className={`h-3.5 w-3.5 ${seg.textColor}`} />
              <span className="flex-1 text-xs text-muted-foreground">{seg.label}</span>
              <span className={`text-xs font-semibold ${seg.textColor}`}>{seg.count}</span>
            </div>
          )
        })}
      </div>
    </div>
  )
}

export default memo(KpiTasks)
