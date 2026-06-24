/**
 * Calendar view for a project's tasks, bucketed by due date.
 *
 * Renders a Monday-aligned month grid; each task with a due date appears as a
 * draggable chip on its day. Dragging a chip to another day updates the task's
 * due_date via useUpdateTask (optimistic). Tasks without a due date sit in an
 * "ohne Fälligkeit" tray and can be dragged onto a day to schedule them.
 * Clicking a chip opens the task detail slide-over.
 */
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  useDraggable,
  useDroppable,
  pointerWithin,
  type DragStartEvent,
  type DragEndEvent,
} from '@dnd-kit/core'
import { ChevronLeft, ChevronRight, Lock } from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'
import { cn } from '@/lib'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useTasks, useUpdateTask } from '@/api/hooks/useTasks'
import { useWorkStore } from '@/stores/work'
import type { TaskData } from '../list/TaskRow'
import type { Priority } from '../components/PriorityBadge'
import {
  getMonthGrid,
  getMonthLabel,
  getWeekdayLabels,
  dateToKey,
  dueDateToKey,
  isSameMonth,
  isWeekend,
} from './calendar-utils'

const TASKS_QUERY = { page_size: 500, include_completed: true } as const

const priorityDot: Record<Priority, string> = {
  urgent: 'bg-destructive',
  high: 'bg-orange-500',
  normal: 'bg-blue-500',
  low: 'bg-gray-400',
}

interface WorkCalendarViewProps {
  projectId: string
}

export default function WorkCalendarView({ projectId }: WorkCalendarViewProps) {
  const { t, i18n } = useTranslation()
  const queryClient = useQueryClient()
  const updateTask = useUpdateTask()
  const openTaskPanel = useWorkStore((s) => s.openTaskPanel)
  const [monthOffset, setMonthOffset] = useState(0)
  const [activeTask, setActiveTask] = useState<TaskData | null>(null)

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor)
  )

  const { data, isLoading } = useTasks({
    project_id: projectId,
    ...TASKS_QUERY,
  })
  const tasks: TaskData[] = useMemo(() => data?.tasks ?? [], [data?.tasks])

  // Reference "today" — stable for the component lifetime; offset drives nav.
  const today = useMemo(() => new Date(), [])
  const todayKey = dateToKey(today)
  const grid = useMemo(() => getMonthGrid(today, monthOffset), [today, monthOffset])
  const weekdays = useMemo(() => getWeekdayLabels(i18n.language), [i18n.language])
  const monthLabel = getMonthLabel(grid.monthIndex, grid.year, i18n.language)

  // Bucket tasks by local due-date key; collect undated ones separately.
  const { byDay, undated } = useMemo(() => {
    const map = new Map<string, TaskData[]>()
    const none: TaskData[] = []
    for (const task of tasks) {
      const key = dueDateToKey(task.due_date)
      if (!key) {
        none.push(task)
        continue
      }
      const list = map.get(key) ?? []
      list.push(task)
      map.set(key, list)
    }
    return { byDay: map, undated: none }
  }, [tasks])

  function handleDragStart(event: DragStartEvent) {
    setActiveTask((event.active.data.current?.task as TaskData) ?? null)
  }

  function handleDragEnd(event: DragEndEvent) {
    setActiveTask(null)
    const { active, over } = event
    if (!over) return
    const task = active.data.current?.task as TaskData | undefined
    if (!task?.id) return

    const newKey = String(over.id) // day cell id === date key
    const currentKey = dueDateToKey(task.due_date)
    if (newKey === currentKey) return

    // Optimistic: update the cached task's due_date for instant feedback.
    const queryKey = ['tasks', { project_id: projectId, ...TASKS_QUERY }]
    queryClient.setQueryData(queryKey, (old: typeof data) => {
      if (!old?.tasks) return old
      return {
        ...old,
        tasks: old.tasks.map((tk: TaskData) =>
          tk.id === task.id ? { ...tk, due_date: newKey } : tk
        ),
      }
    })

    updateTask.mutate(
      { id: task.id, due_date: newKey },
      {
        onError: () => {
          queryClient.invalidateQueries({ queryKey: ['tasks'] })
        },
      }
    )
  }

  if (isLoading) {
    return (
      <div className="p-6 space-y-3">
        <Skeleton className="h-8 w-48" />
        <div className="grid grid-cols-7 gap-2">
          {Array.from({ length: 35 }).map((_, i) => (
            <Skeleton key={i} className="h-24 w-full rounded-lg" />
          ))}
        </div>
      </div>
    )
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={pointerWithin}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
    >
      <div className="flex h-full flex-col">
        {/* Toolbar: month navigation */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <div className="flex items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              onClick={() => setMonthOffset((m) => m - 1)}
              title={t('work.calendar.prevMonth')}
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <Button
              size="sm"
              variant={monthOffset === 0 ? 'secondary' : 'outline'}
              onClick={() => setMonthOffset(0)}
            >
              {t('work.calendar.today')}
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => setMonthOffset((m) => m + 1)}
              title={t('work.calendar.nextMonth')}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
          <span className="text-sm font-semibold text-foreground capitalize">
            {monthLabel}
          </span>
        </div>

        {/* Scrollable calendar body */}
        <div className="flex-1 min-h-0 overflow-y-auto p-4 space-y-4">
          {/* Weekday header */}
          <div className="grid grid-cols-7 gap-2">
            {weekdays.map((w, i) => (
              <div
                key={w}
                className={cn(
                  'text-xs font-medium text-muted-foreground px-1',
                  (i === 5 || i === 6) && 'text-muted-foreground/60'
                )}
              >
                {w}
              </div>
            ))}
          </div>

          {/* Day grid */}
          <div className="grid grid-cols-7 gap-2">
            {grid.days.map((day) => {
              const key = dateToKey(day)
              const dayTasks = byDay.get(key) ?? []
              return (
                <DayCell
                  key={key}
                  dateKey={key}
                  day={day}
                  isToday={key === todayKey}
                  isOutside={!isSameMonth(day, grid.monthIndex)}
                  weekend={isWeekend(day)}
                  tasks={dayTasks}
                  todayKey={todayKey}
                  onTaskClick={(id) => openTaskPanel(id)}
                />
              )
            })}
          </div>

          {/* Undated tray */}
          <div className="rounded-lg border border-dashed border-border bg-muted/30 p-3">
            <p className="text-xs font-medium text-muted-foreground mb-2">
              {t('work.calendar.noDueDate')}{' '}
              <span className="text-muted-foreground/60">({undated.length})</span>
            </p>
            {undated.length === 0 ? (
              <p className="text-xs text-muted-foreground/60">
                {t('work.calendar.noUndatedTasks')}
              </p>
            ) : (
              <div className="flex flex-wrap gap-1.5">
                {undated.map((task) => (
                  <TaskChip
                    key={task.id}
                    task={task}
                    onClick={() => task.id && openTaskPanel(task.id)}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      <DragOverlay>
        {activeTask ? <TaskChip task={activeTask} isOverlay /> : null}
      </DragOverlay>
    </DndContext>
  )
}

// ---------------------------------------------------------------------------
// Day cell (droppable)
// ---------------------------------------------------------------------------

interface DayCellProps {
  dateKey: string
  day: Date
  isToday: boolean
  isOutside: boolean
  weekend: boolean
  tasks: TaskData[]
  todayKey: string
  onTaskClick: (id: string) => void
}

function DayCell({
  dateKey,
  day,
  isToday,
  isOutside,
  weekend,
  tasks,
  todayKey,
  onTaskClick,
}: DayCellProps) {
  const { setNodeRef, isOver } = useDroppable({ id: dateKey })
  const MAX_VISIBLE = 3
  const visible = tasks.slice(0, MAX_VISIBLE)
  const overflow = tasks.length - visible.length

  return (
    <div
      ref={setNodeRef}
      className={cn(
        'min-h-[112px] rounded-lg border p-1.5 flex flex-col gap-1 transition-colors',
        isOutside
          ? 'border-transparent bg-muted/20'
          : weekend
            ? 'border-border bg-muted/30'
            : 'border-border bg-card',
        isOver && 'border-primary ring-1 ring-primary/40 bg-primary/5'
      )}
    >
      <div className="flex items-center justify-between px-0.5">
        <span
          className={cn(
            'text-xs',
            isToday
              ? 'flex h-5 w-5 items-center justify-center rounded-full bg-primary text-primary-foreground font-semibold'
              : isOutside
                ? 'text-muted-foreground/40'
                : 'text-muted-foreground'
          )}
        >
          {day.getDate()}
        </span>
      </div>
      <div className="flex flex-col gap-1 overflow-hidden">
        {visible.map((task) => (
          <TaskChip
            key={task.id}
            task={task}
            todayKey={todayKey}
            onClick={() => task.id && onTaskClick(task.id)}
          />
        ))}
        {overflow > 0 && (
          <span className="px-1 text-[10px] text-muted-foreground">
            +{overflow}
          </span>
        )}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Task chip (draggable)
// ---------------------------------------------------------------------------

interface TaskChipProps {
  task: TaskData
  todayKey?: string
  isOverlay?: boolean
  onClick?: () => void
}

function TaskChip({ task, todayKey, isOverlay = false, onClick }: TaskChipProps) {
  const { t } = useTranslation()
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: task.id ?? '',
    data: { task },
  })

  const priority = (task.priority as Priority) ?? 'normal'
  const dueKey = dueDateToKey(task.due_date)
  const overdue =
    !task.is_closed && !!todayKey && !!dueKey && dueKey < todayKey

  return (
    <button
      ref={setNodeRef}
      type="button"
      {...attributes}
      {...listeners}
      onClick={(e) => {
        if (!isDragging && onClick) {
          e.stopPropagation()
          onClick()
        }
      }}
      title={task.title}
      className={cn(
        'group flex w-full items-center gap-1 rounded px-1.5 py-0.5 text-left text-[11px] leading-tight',
        'border border-border/60 bg-background hover:bg-accent transition-colors cursor-grab active:cursor-grabbing',
        task.is_closed && 'opacity-60 line-through',
        overdue && 'border-destructive/40 bg-error-light',
        isDragging && 'opacity-30',
        isOverlay && 'shadow-lg border-primary/40 max-w-[200px]'
      )}
    >
      <span
        className={cn('h-1.5 w-1.5 shrink-0 rounded-full', priorityDot[priority])}
      />
      {task.has_blocked_deps && (
        <span className="shrink-0" title={t('work.tasks.blocked')}>
          <Lock className="h-2.5 w-2.5 text-warning-foreground" />
        </span>
      )}
      <span className="truncate">{task.title}</span>
    </button>
  )
}
