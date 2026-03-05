/**
 * My Tasks widget — personal open tasks with deadlines.
 * Uses real API hook to fetch tasks assigned to the current user.
 */
import { memo, useMemo } from 'react'
import { CheckCircle2, Circle, AlertTriangle, Clock } from 'lucide-react'
import { useMyTasks } from '@/api/hooks/useTasks'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

const PRIORITY_STYLE: Record<string, string> = {
  urgent: 'text-red-500',
  high: 'text-red-500',
  medium: 'text-amber-500',
  low: 'text-muted-foreground',
}

function formatDueDate(dateStr: string | undefined): string {
  if (!dateStr) return ''
  const today = new Date()
  const todayStr = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`
  const tomorrow = new Date(today)
  tomorrow.setDate(tomorrow.getDate() + 1)
  const tomorrowStr = `${tomorrow.getFullYear()}-${String(tomorrow.getMonth() + 1).padStart(2, '0')}-${String(tomorrow.getDate()).padStart(2, '0')}`

  if (dateStr === todayStr) return 'Heute'
  if (dateStr === tomorrowStr) return 'Morgen'

  // Format as dd.MM.
  const parts = dateStr.split('-')
  if (parts.length === 3) {
    return `${parseInt(parts[2], 10)}.${parseInt(parts[1], 10)}.`
  }
  return dateStr
}

interface WidgetTask {
  id: string
  title: string
  project: string
  priority: string
  due: string
  done: boolean
}

function MyTasks(_props: WidgetProps) {
  const { data, isLoading } = useMyTasks({ page_size: 8, include_completed: true })

  const widgetTasks = useMemo<WidgetTask[]>(() => {
    const tasks = (data as { tasks?: Array<Record<string, unknown>> })?.tasks ?? []
    return tasks.map((t) => {
      const status = (t.status as string) ?? ''
      const statusId = (t.status_id as string) ?? ''
      const isDone = status === 'done' || status === 'completed' || statusId === 'done'
      const priority = (t.priority as string) ?? 'medium'

      return {
        id: (t.id as string) ?? '',
        title: (t.title as string) ?? '',
        project: ((t as Record<string, unknown>).project as { name?: string })?.name ?? '',
        priority: priority === 'urgent' ? 'high' : priority,
        due: formatDueDate(t.due_date as string | undefined),
        done: isDone,
      }
    })
  }, [data])

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    )
  }

  if (!widgetTasks.length) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <p className="text-sm text-muted-foreground">Keine Aufgaben</p>
      </div>
    )
  }

  const open = widgetTasks.filter((t) => !t.done)
  const done = widgetTasks.filter((t) => t.done)

  return (
    <div className="flex h-full flex-col">
      {/* Summary */}
      <div className="flex items-center justify-between px-4 pt-4 pb-2">
        <span className="text-xs text-muted-foreground">{open.length} offen · {done.length} erledigt</span>
        <span className="flex items-center gap-1 text-xs text-red-500">
          <AlertTriangle className="h-3 w-3" />
          {open.filter((t) => t.due === 'Heute').length} heute faellig
        </span>
      </div>

      {/* Task list */}
      <div className="flex-1 overflow-auto divide-y divide-border">
        {widgetTasks.map((task) => (
          <div
            key={task.id}
            className="flex items-start gap-3 px-4 py-2.5 hover:bg-accent/50 cursor-pointer transition-colors"
          >
            {task.done ? (
              <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-emerald-500" />
            ) : (
              <Circle className={`mt-0.5 h-4 w-4 shrink-0 ${PRIORITY_STYLE[task.priority] ?? 'text-muted-foreground'}`} />
            )}
            <div className="min-w-0 flex-1">
              <p className={`text-sm ${task.done ? 'line-through text-muted-foreground' : 'font-medium text-foreground'}`}>
                {task.title}
              </p>
              <div className="flex items-center gap-2 mt-0.5">
                {task.project && (
                  <span className="text-[10px] rounded-full bg-secondary px-1.5 py-0.5 text-muted-foreground">
                    {task.project}
                  </span>
                )}
                {task.due && (
                  <span className="flex items-center gap-0.5 text-[10px] text-muted-foreground">
                    <Clock className="h-2.5 w-2.5" />
                    {task.due}
                  </span>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

export default memo(MyTasks)
