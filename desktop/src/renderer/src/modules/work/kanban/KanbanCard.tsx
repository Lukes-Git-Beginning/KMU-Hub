/**
 * Draggable task card for the Kanban board.
 *
 * Uses useSortable from @dnd-kit/sortable for drag-and-drop functionality.
 * Shows task key, title, priority, assignee, due date, blocked indicator,
 * and subtask progress. Supports an overlay variant for the DragOverlay.
 */
import { useState, type SyntheticEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import {
  Lock,
  User,
  Calendar,
  Circle,
  CheckCircle2,
  MoreHorizontal,
  UserPlus,
  CalendarDays,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import { cn } from '@/lib'
import { useUpdateTask, useDeleteTask } from '@/api/hooks/useTasks'
import { useAuthStore } from '@/stores/auth'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import type { TaskData } from '../list/TaskRow'
import type { Priority } from '../components/PriorityBadge'
import PriorityBadge from '../components/PriorityBadge'
import TaskLabelChips from '../components/TaskLabelChips'

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
  const updateTask = useUpdateTask()
  const deleteTask = useDeleteTask()
  const currentUserId = useAuthStore((s) => s.user?.id)
  const [confirmDelete, setConfirmDelete] = useState(false)

  const dueLabel = formatDueDate(task.due_date, t)
  const overdue = isDueOverdue(task.due_date)
  const subtaskCount = task.subtask_count ?? 0
  const completedSubtasks = task.completed_subtask_count ?? 0
  const subtaskProgress = subtaskCount > 0 ? (completedSubtasks / subtaskCount) * 100 : 0
  const done = !!task.is_closed

  function toggleComplete() {
    if (!task.id) return
    updateTask.mutate(
      { id: task.id, completed_at: done ? null : new Date().toISOString(), is_closed: !done },
      {
        onSuccess: () =>
          toast.success(
            done
              ? t('work.myTasks.reopened', { defaultValue: 'Aufgabe wieder geöffnet' })
              : t('work.myTasks.completedToast', { defaultValue: 'Aufgabe abgeschlossen' }),
          ),
      },
    )
  }

  function assignToMe() {
    if (!task.id || !currentUserId) return
    updateTask.mutate(
      { id: task.id, assignee_id: currentUserId },
      { onSuccess: () => toast.success(t('work.myTasks.assignedToMe', { defaultValue: 'Dir zugewiesen' })) },
    )
  }

  function setQuickDue(days: number | null) {
    if (!task.id) return
    let due: string | null = null
    if (days !== null) {
      const d = new Date()
      d.setDate(d.getDate() + days)
      d.setHours(12, 0, 0, 0)
      due = d.toISOString()
    }
    updateTask.mutate(
      { id: task.id, due_date: due },
      { onSuccess: () => toast.success(t('work.myTasks.dueUpdated', { defaultValue: 'Fälligkeit aktualisiert' })) },
    )
  }

  function doDelete() {
    if (!task.id) return
    deleteTask.mutate(task.id, {
      onSuccess: () => {
        toast.success(t('work.myTasks.deletedToast', { defaultValue: 'Aufgabe gelöscht' }))
        setConfirmDelete(false)
      },
    })
  }

  /** Stop dnd-kit from starting a drag (and the card-open click) when interacting
   *  with an inline control. */
  const stopDrag = (e: SyntheticEvent) => e.stopPropagation()

  return (
    <>
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      className={cn(
        'group rounded-lg border border-border bg-card p-3 shadow-sm transition-shadow cursor-grab active:cursor-grabbing',
        'hover:shadow-md hover:border-border/80',
        isDragging && 'opacity-30',
        isDragOverlay && 'shadow-lg border-primary/30 rotate-2 scale-105',
        task.is_closed && 'opacity-60'
      )}
      onClick={(e) => {
        // Don't open the detail panel when interacting with an inline control
        // (checkbox / action menu) or mid-drag. closest() is reliable even when
        // Radix's asChild trigger swallows our stopPropagation.
        if (isDragging || !onClick) return
        if ((e.target as HTMLElement).closest('[data-card-control]')) return
        e.stopPropagation()
        onClick()
      }}
    >
      {/* Top row: quick-complete + key on the left, priority + actions on the right */}
      <div className="flex items-center justify-between mb-1.5">
        <div className="flex items-center gap-1.5 min-w-0">
          {!isDragOverlay && (
            <button
              type="button"
              data-card-control
              onPointerDown={stopDrag}
              onClick={(e) => {
                stopDrag(e)
                toggleComplete()
              }}
              className="shrink-0 text-muted-foreground transition-colors hover:text-success"
              aria-label={
                done
                  ? t('work.myTasks.reopen', { defaultValue: 'Wieder öffnen' })
                  : t('work.myTasks.complete', { defaultValue: 'Abschließen' })
              }
              title={
                done
                  ? t('work.myTasks.reopen', { defaultValue: 'Wieder öffnen' })
                  : t('work.myTasks.complete', { defaultValue: 'Abschließen' })
              }
            >
              {done ? (
                <CheckCircle2 className="h-3.5 w-3.5 text-success" />
              ) : (
                <Circle className="h-3.5 w-3.5" />
              )}
            </button>
          )}
          {task.project_key && task.task_number ? (
            <span className="text-[10px] font-mono text-muted-foreground truncate">
              {task.project_key}-{task.task_number}
            </span>
          ) : null}
        </div>
        <div className="flex items-center gap-1">
          {task.has_blocked_deps && (
            <Lock className="h-3 w-3 text-warning-foreground" title={t('work.tasks.blocked')} />
          )}
          <PriorityBadge
            priority={(task.priority as Priority) ?? 'medium'}
            compact
          />
          {!isDragOverlay && (
            <Popover>
              <PopoverTrigger asChild>
                <button
                  type="button"
                  data-card-control
                  onPointerDown={stopDrag}
                  onClick={stopDrag}
                  className="shrink-0 rounded p-0.5 text-muted-foreground opacity-0 transition-all hover:bg-accent hover:text-foreground focus:opacity-100 group-hover:opacity-100"
                  aria-label={t('work.myTasks.actions', { defaultValue: 'Aktionen' })}
                  title={t('work.myTasks.actions', { defaultValue: 'Aktionen' })}
                >
                  <MoreHorizontal className="h-3.5 w-3.5" />
                </button>
              </PopoverTrigger>
              <PopoverContent
                className="w-52 p-1"
                align="end"
                onPointerDown={stopDrag}
                onClick={stopDrag}
              >
                {/* Complete / reopen */}
                <button
                  type="button"
                  className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors"
                  onClick={() => toggleComplete()}
                >
                  {done ? (
                    <Circle className="h-3.5 w-3.5 text-muted-foreground" />
                  ) : (
                    <CheckCircle2 className="h-3.5 w-3.5 text-success" />
                  )}
                  {done
                    ? t('work.myTasks.reopen', { defaultValue: 'Wieder öffnen' })
                    : t('work.myTasks.complete', { defaultValue: 'Abschließen' })}
                </button>
                {/* Assign to me */}
                <button
                  type="button"
                  className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors"
                  onClick={() => assignToMe()}
                >
                  <UserPlus className="h-3.5 w-3.5 text-muted-foreground" />
                  {t('work.myTasks.assignToMe', { defaultValue: 'Mir zuweisen' })}
                </button>
                {/* Due date quick-pick */}
                <div className="mt-1 border-t border-border pt-1">
                  <p className="flex items-center gap-1.5 px-2 py-1 text-[11px] font-medium text-muted-foreground">
                    <CalendarDays className="h-3 w-3" />
                    {t('work.list.dueDate', { defaultValue: 'Fälligkeit' })}
                  </p>
                  <div className="flex flex-wrap gap-1 px-1 pb-1">
                    {[
                      { d: 0, key: 'work.time.today', dv: 'Heute' },
                      { d: 1, key: 'work.time.tomorrow', dv: 'Morgen' },
                      { d: 7, key: 'work.myTasks.dueNextWeek', dv: 'Nächste Woche' },
                      { d: null, key: 'work.myTasks.dueClear', dv: 'Entfernen' },
                    ].map((opt) => (
                      <button
                        key={opt.dv}
                        type="button"
                        className="rounded border border-border px-2 py-1 text-[11px] text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
                        onClick={() => setQuickDue(opt.d)}
                      >
                        {t(opt.key, { defaultValue: opt.dv })}
                      </button>
                    ))}
                  </div>
                </div>
                {/* Delete */}
                <div className="mt-1 border-t border-border pt-1">
                  <button
                    type="button"
                    className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs text-destructive hover:bg-error-light transition-colors"
                    onClick={() => setConfirmDelete(true)}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                    {t('common.delete', { defaultValue: 'Löschen' })}
                  </button>
                </div>
              </PopoverContent>
            </Popover>
          )}
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

      {/* Labels */}
      {task.id && <TaskLabelChips taskId={task.id} max={3} size="xs" className="mt-2" />}

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

    {/* Delete confirmation */}
    <Dialog open={confirmDelete} onOpenChange={(v) => { if (!v) setConfirmDelete(false) }}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>{t('work.myTasks.deleteTitle', { defaultValue: 'Aufgabe löschen?' })}</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-muted-foreground">
          {t('work.myTasks.deleteBody', {
            defaultValue: '„{title}" wird dauerhaft gelöscht. Das kann nicht rückgängig gemacht werden.',
            title: task.title ?? '',
          })}
        </p>
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="outline" onClick={() => setConfirmDelete(false)}>
            {t('common.cancel')}
          </Button>
          <Button variant="destructive" onClick={doDelete} disabled={deleteTask.isPending}>
            {t('common.delete', { defaultValue: 'Löschen' })}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
    </>
  )
}
