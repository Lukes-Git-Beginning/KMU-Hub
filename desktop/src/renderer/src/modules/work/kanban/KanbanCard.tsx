/**
 * Draggable task card for the Kanban board.
 *
 * Uses useSortable from @dnd-kit/sortable for drag-and-drop functionality.
 * Shows task key, title, priority, assignee, due date, blocked indicator,
 * and subtask progress. Supports an overlay variant for the DragOverlay.
 */
import { useTranslation } from 'react-i18next'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { Lock, User, Calendar } from 'lucide-react'
import { cn } from '@/lib'
import type { TaskData } from '../list/TaskRow'
import PriorityBadge from '../components/PriorityBadge'
import type { Priority } from '../components/PriorityBadge'

interface KanbanCardProps {
  task: TaskData
  isDragOverlay?: boolean
  onClick?: () => void
}

function formatDueDate(date: string | undefined, t: (key: string, opts?: Record<string, unknown>) => string): string | null {
  if (!date) return null
  const d = new Date(date)
  const now = new Date()
  const diffMs = d.getTime() - now.getTime()
  const diffDays = Math.ceil(diffMs / (1000 * 60 * 60 * 24))
  if (diffDays < 0) return `${Math.abs(diffDays)}d`
  if (diffDays === 0) return t('work.time.today')
  if (diffDays === 1) return t('work.time.tomorrow')
  return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' })
}

function isDueOverdue(date?: string): boolean {
  if (!date) return false
  return new Date(date).getTime() < Date.now()
}

export default function KanbanCard({
  task,
  isDragOverlay = false,
  onClick,
}: KanbanCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: task.id ?? '',
    data: { task },
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  const { t } = useTranslation()
  const dueLabel = formatDueDate(task.due_date, t)
  const overdue = isDueOverdue(task.due_date)
  const subtaskCount = task.subtask_count ?? 0
  const completedSubtasks = task.completed_subtask_count ?? 0
  const subtaskProgress = subtaskCount > 0 ? (completedSubtasks / subtaskCount) * 100 : 0

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      className={cn(
        'rounded-lg border border-border bg-card p-3 shadow-sm transition-shadow cursor-grab active:cursor-grabbing',
        'hover:shadow-md hover:border-border/80',
        isDragging && 'opacity-30',
        isDragOverlay && 'shadow-lg border-primary/30 rotate-2 scale-105',
        task.is_closed && 'opacity-60'
      )}
      onClick={(e) => {
        // Only trigger click if not dragging
        if (!isDragging && onClick) {
          e.stopPropagation()
          onClick()
        }
      }}
    >
      {/* Top row: task key + priority */}
      <div className="flex items-center justify-between mb-1.5">
        {task.project_key && task.task_number ? (
          <span className="text-[10px] font-mono text-muted-foreground">
            {task.project_key}-{task.task_number}
          </span>
        ) : (
          <span />
        )}
        <div className="flex items-center gap-1">
          {task.has_blocked_deps && (
            <Lock className="h-3 w-3 text-warning-foreground" title={t('work.tasks.blocked')} />
          )}
          <PriorityBadge
            priority={(task.priority as Priority) ?? 'medium'}
            compact
          />
        </div>
      </div>

      {/* Title */}
      <p
        className={cn(
          'text-sm font-medium leading-snug line-clamp-2',
          task.is_closed && 'line-through'
        )}
      >
        {task.title}
      </p>

      {/* Bottom row: assignee, due date, subtask progress */}
      <div className="mt-2 flex items-center justify-between text-[11px] text-muted-foreground">
        <div className="flex items-center gap-2">
          {/* Assignee */}
          {task.assignee_name && (
            <span className="flex items-center gap-0.5 truncate max-w-[80px]" title={task.assignee_name}>
              <User className="h-3 w-3 shrink-0" />
              <span className="truncate">{task.assignee_name}</span>
            </span>
          )}

          {/* Due date */}
          {dueLabel && (
            <span
              className={cn(
                'flex items-center gap-0.5',
                overdue && 'text-destructive font-medium'
              )}
            >
              <Calendar className="h-3 w-3" />
              {dueLabel}
            </span>
          )}
        </div>

        {/* Subtask progress */}
        {subtaskCount > 0 && (
          <div className="flex items-center gap-1.5">
            <span className="text-[10px]">
              {completedSubtasks}/{subtaskCount}
            </span>
            <div className="h-1 w-10 rounded-full bg-muted overflow-hidden">
              <div
                className="h-full rounded-full bg-primary transition-all"
                style={{ width: `${subtaskProgress}%` }}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
