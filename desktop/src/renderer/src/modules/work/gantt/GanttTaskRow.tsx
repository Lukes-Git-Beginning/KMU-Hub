/**
 * Individual task row with horizontal bar on the timeline.
 *
 * Left side (fixed): task info with key, title, assignee.
 * Right side (scrollable): colored bar at computed position with
 * tooltip, critical path highlight, and completed styling.
 */
import { useMemo } from 'react'
import { cn } from '@/lib'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { format } from 'date-fns'
import { de } from 'date-fns/locale'
import type { GanttTask, TimelineConfig } from './gantt-utils'
import {
  taskToBar,
  getBarColor,
  ROW_HEIGHT,
  BAR_HEIGHT,
  TASK_INFO_WIDTH,
} from './gantt-utils'

interface GanttTaskRowProps {
  task: GanttTask
  config: TimelineConfig
  rowIndex: number
  onTaskClick: (taskId: string) => void
  timelineWidth: number
}

export default function GanttTaskRow({
  task,
  config,
  rowIndex,
  onTaskClick,
  timelineWidth,
}: GanttTaskRowProps) {
  const bar = useMemo(() => taskToBar(task, config), [task, config])
  const barColor = useMemo(() => getBarColor(task), [task])
  const isCompleted = !!task.completed_at

  const taskKey =
    task.project_key && task.task_number
      ? `${task.project_key}-${task.task_number}`
      : task.task_number
        ? `#${task.task_number}`
        : ''

  const tooltipText = [
    task.title,
    task.start ? `Start: ${format(task.start, 'dd.MM.yyyy', { locale: de })}` : null,
    task.end ? `Fällig: ${format(task.end, 'dd.MM.yyyy', { locale: de })}` : null,
    task.assignee_name ? `Zugewiesen: ${task.assignee_name}` : null,
    task.status_name ? `Status: ${task.status_name}` : null,
  ]
    .filter(Boolean)
    .join('\n')

  return (
    <div
      className={cn(
        'flex border-b border-border hover:bg-muted/20',
        rowIndex % 2 === 0 ? 'bg-background' : 'bg-muted/5'
      )}
      style={{ height: ROW_HEIGHT }}
    >
      {/* Fixed left: task info */}
      <div
        className="flex-shrink-0 flex items-center gap-2 border-r border-border px-3 overflow-hidden cursor-pointer hover:bg-muted/30"
        style={{ width: TASK_INFO_WIDTH }}
        onClick={() => onTaskClick(task.id)}
      >
        <span className="text-[10px] font-mono text-muted-foreground flex-shrink-0">
          {taskKey}
        </span>
        <span
          className={cn(
            'text-sm truncate flex-1',
            isCompleted && 'line-through text-muted-foreground'
          )}
        >
          {task.title}
        </span>
        {task.assignee_name && (
          <span
            className="flex-shrink-0 flex items-center justify-center rounded-full bg-muted text-[10px] font-medium"
            style={{ width: 20, height: 20 }}
            title={task.assignee_name}
          >
            {task.assignee_name.charAt(0).toUpperCase()}
          </span>
        )}
      </div>

      {/* Scrollable right: bar area */}
      <div className="relative flex-1 overflow-hidden" style={{ minWidth: timelineWidth }}>
        {bar ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <div
                className={cn(
                  'absolute rounded cursor-pointer transition-shadow hover:shadow-md',
                  task.is_on_critical_path && 'ring-2 ring-red-500',
                  isCompleted && 'opacity-50'
                )}
                style={{
                  left: bar.x,
                  top: (ROW_HEIGHT - BAR_HEIGHT) / 2,
                  width: bar.width,
                  height: BAR_HEIGHT,
                  backgroundColor: barColor,
                }}
                onClick={() => onTaskClick(task.id)}
              >
                {/* Title inside bar if wide enough */}
                {bar.width > 60 && (
                  <span className="absolute inset-0 flex items-center px-2 text-[10px] text-white truncate font-medium">
                    {task.title}
                  </span>
                )}
                {isCompleted && (
                  <div
                    className="absolute top-1/2 left-0 right-0 border-t border-white/60"
                    style={{ transform: 'translateY(-50%)' }}
                  />
                )}
              </div>
            </TooltipTrigger>
            <TooltipContent side="top" className="max-w-xs whitespace-pre-line text-xs">
              {tooltipText}
            </TooltipContent>
          </Tooltip>
        ) : (
          <div
            className="absolute flex items-center text-xs text-muted-foreground italic px-2"
            style={{ top: (ROW_HEIGHT - BAR_HEIGHT) / 2, height: BAR_HEIGHT }}
          >
            Kein Fälligkeitsdatum
          </div>
        )}
      </div>
    </div>
  )
}
