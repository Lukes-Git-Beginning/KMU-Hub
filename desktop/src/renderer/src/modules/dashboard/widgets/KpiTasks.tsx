/**
 * KPI Tasks widget — task completion progress with breakdown.
 */
import { memo, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { CheckCircle2, Circle, AlertTriangle } from 'lucide-react'
import { useMyTasks } from '@/api/hooks/useTasks'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

function KpiTasks(_props: WidgetProps) {
  const { t } = useTranslation()
  const { data, isLoading } = useMyTasks({ include_completed: true })

  const stats = useMemo(() => {
    const tasks = (data as { tasks?: Array<Record<string, unknown>>; total?: number })?.tasks ?? []
    const total = (data as { total?: number })?.total ?? tasks.length
    const now = new Date()
    const todayStr = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`

    let done = 0
    let inProgress = 0
    let overdue = 0

    for (const t of tasks) {
      const status = (t.status as string) ?? ''
      const statusId = (t.status_id as string) ?? ''
      const dueDate = (t.due_date as string) ?? ''

      // Determine if task is done — check common status patterns
      const isDone = status === 'done' || status === 'completed' || statusId === 'done'

      if (isDone) {
        done++
      } else if (dueDate && dueDate < todayStr) {
        overdue++
      } else {
        inProgress++
      }
    }

    return { total, done, inProgress, overdue }
  }, [data])

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div role="status" aria-label={t('common.loading')} className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    )
  }

  const { total, done, inProgress, overdue } = stats
  const pct = total > 0 ? Math.round((done / total) * 100) : 0

  const segments = [
    { label: t('dashboard.kpiTasks.done'), count: done, color: 'bg-success', textColor: 'text-success', icon: CheckCircle2 },
    { label: t('dashboard.kpiTasks.inProgress'), count: inProgress, color: 'bg-blue-500', textColor: 'text-blue-600', icon: Circle },
    { label: t('dashboard.kpiTasks.overdue'), count: overdue, color: 'bg-destructive', textColor: 'text-destructive', icon: AlertTriangle },
  ]

  return (
    <div className="flex h-full flex-col justify-between p-4">
      {/* Progress header */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs font-medium text-muted-foreground">{t('dashboard.kpiTasks.thisWeek')}</span>
          <span className="text-xs font-bold text-foreground">{pct}%</span>
        </div>

        {/* Progress bar */}
        <div className="h-2.5 w-full rounded-full bg-secondary overflow-hidden flex">
          <div
            className="h-full bg-success transition-all"
            style={{ width: total > 0 ? `${(done / total) * 100}%` : '0%' }}
          />
          <div
            className="h-full bg-blue-500 transition-all"
            style={{ width: total > 0 ? `${(inProgress / total) * 100}%` : '0%' }}
          />
          <div
            className="h-full bg-destructive transition-all"
            style={{ width: total > 0 ? `${(overdue / total) * 100}%` : '0%' }}
          />
        </div>

        <p className="mt-2 text-lg font-bold text-foreground">
          {done} <span className="text-sm font-normal text-muted-foreground">{t('dashboard.kpiTasks.of', { total })}</span>
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
