/**
 * DnD Kanban board with @dnd-kit for drag-and-drop between status columns.
 *
 * Groups tasks by status_id into columns. Supports optimistic updates via
 * useMoveTask mutation with queryClient.setQueryData for instant feedback.
 * Uses closestCorners collision detection for multi-container scenarios.
 * Fractional ordering for sort_order between neighbors.
 */
import { useState, useMemo, useCallback } from 'react'
import {
  DndContext,
  DragOverlay,
  closestCorners,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  type DragStartEvent,
  type DragEndEvent,
  type DragOverEvent,
} from '@dnd-kit/core'
import { sortableKeyboardCoordinates } from '@dnd-kit/sortable'
import { useQueryClient } from '@tanstack/react-query'
import { useTasks, useMoveTask } from '@/api/hooks/useTasks'
import { useWorkStore } from '@/stores/work'
import { Skeleton } from '@/components/ui/skeleton'
import KanbanColumn from './KanbanColumn'
import KanbanCard from './KanbanCard'
import type { StatusInfo } from './KanbanColumn'
import type { TaskData } from '../list/TaskRow'

interface KanbanBoardProps {
  projectId: string
  statuses: StatusInfo[]
}

export default function KanbanBoard({
  projectId,
  statuses,
}: KanbanBoardProps) {
  const queryClient = useQueryClient()
  const moveTask = useMoveTask()
  const openTaskPanel = useWorkStore((s) => s.openTaskPanel)
  const [activeTask, setActiveTask] = useState<TaskData | null>(null)

  // Sensors for DnD
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 5 },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  )

  // Fetch all tasks for this project
  const { data: tasksData, isLoading } = useTasks({
    project_id: projectId,
    page_size: 500,
    include_completed: true,
  })

  const tasks: TaskData[] = tasksData?.tasks ?? []

  // Group tasks by status
  const columnData = useMemo(() => {
    const byStatus = new Map<string, TaskData[]>()
    for (const s of statuses) {
      byStatus.set(s.id, [])
    }
    for (const t of tasks) {
      const statusId = t.status_id ?? ''
      const list = byStatus.get(statusId)
      if (list) {
        list.push(t)
      }
    }
    return byStatus
  }, [tasks, statuses])

  // Build children map per column for subtask nesting
  const childrenMaps = useMemo(() => {
    const maps = new Map<string, Map<string, TaskData[]>>()
    for (const [statusId, columnTasks] of columnData.entries()) {
      const childrenMap = new Map<string, TaskData[]>()
      const columnTaskIds = new Set(columnTasks.map((t) => t.id ?? ''))
      for (const t of columnTasks) {
        if (t.parent_task_id && columnTaskIds.has(t.parent_task_id)) {
          const list = childrenMap.get(t.parent_task_id) ?? []
          list.push(t)
          childrenMap.set(t.parent_task_id, list)
        }
      }
      maps.set(statusId, childrenMap)
    }
    return maps
  }, [columnData])

  // Column task ID sets for parent checking
  const columnTaskIdSets = useMemo(() => {
    const sets = new Map<string, Set<string>>()
    for (const [statusId, columnTasks] of columnData.entries()) {
      sets.set(statusId, new Set(columnTasks.map((t) => t.id ?? '')))
    }
    return sets
  }, [columnData])

  // Find which column a task belongs to
  function findTaskColumn(taskId: string): string | null {
    for (const [statusId, columnTasks] of columnData.entries()) {
      if (columnTasks.some((t) => t.id === taskId)) {
        return statusId
      }
    }
    return null
  }

  // Resolve a droppable/sortable id to a status
  function resolveStatusId(overId: string): string | null {
    // Column drop zone IDs
    if (overId.startsWith('column-')) {
      return overId.replace('column-', '')
    }
    // It's a task ID - find its column
    return findTaskColumn(overId)
  }

  function handleDragStart(event: DragStartEvent) {
    const task = event.active.data.current?.task as TaskData | undefined
    setActiveTask(task ?? null)
  }

  function handleDragOver(event: DragOverEvent) {
    // We can handle visual feedback here if needed
    // For now, the isOver state on KanbanColumn handles visual cues
  }

  function handleDragEnd(event: DragEndEvent) {
    setActiveTask(null)

    const { active, over } = event
    if (!over) return

    const taskId = active.id as string
    const task = active.data.current?.task as TaskData | undefined
    if (!task) return

    const newStatusId = resolveStatusId(over.id as string)
    if (!newStatusId) return

    // Same column, no status change needed
    if (newStatusId === task.status_id) return

    // Calculate new sort_order: place at end of target column
    const targetTasks = columnData.get(newStatusId) ?? []
    let newSortOrder = 1.0
    if (targetTasks.length > 0) {
      // Sort by existing sort_order to find the maximum
      const maxOrder = Math.max(
        ...targetTasks.map((t) => (t as TaskData & { sort_order?: number }).sort_order ?? 0)
      )
      newSortOrder = maxOrder + 1.0
    }

    // Optimistic update: move card in cache immediately
    const queryKey = ['tasks', { project_id: projectId, page_size: 500, include_completed: true }]
    queryClient.setQueryData(queryKey, (old: typeof tasksData) => {
      if (!old?.tasks) return old
      return {
        ...old,
        tasks: old.tasks.map((t: TaskData) =>
          t.id === taskId
            ? { ...t, status_id: newStatusId }
            : t
        ),
      }
    })

    // Fire mutation
    moveTask.mutate(
      { id: taskId, status_id: newStatusId, sort_order: newSortOrder },
      {
        onError: () => {
          // Rollback: invalidate to refetch from server
          queryClient.invalidateQueries({ queryKey: ['tasks'] })
        },
      }
    )
  }

  function handleTaskClick(taskId: string) {
    openTaskPanel(taskId)
  }

  if (isLoading) {
    return (
      <div className="flex gap-4 p-4 overflow-x-auto">
        {statuses.map((s) => (
          <div key={s.id} className="w-[280px] shrink-0 space-y-2">
            <Skeleton className="h-10 w-full rounded-lg" />
            <Skeleton className="h-24 w-full rounded-lg" />
            <Skeleton className="h-24 w-full rounded-lg" />
            <Skeleton className="h-24 w-full rounded-lg" />
          </div>
        ))}
      </div>
    )
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCorners}
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragEnd={handleDragEnd}
    >
      <div className="flex gap-3 p-4 overflow-x-auto h-full">
        {statuses.map((status) => {
          const columnTasks = columnData.get(status.id) ?? []
          const childrenMap = childrenMaps.get(status.id) ?? new Map()
          const columnTaskIds = columnTaskIdSets.get(status.id) ?? new Set()

          return (
            <KanbanColumn
              key={status.id}
              status={status}
              tasks={columnTasks}
              childrenMap={childrenMap}
              columnTaskIds={columnTaskIds}
              onTaskClick={handleTaskClick}
            />
          )
        })}
      </div>

      {/* Drag overlay - renders the card being dragged */}
      <DragOverlay>
        {activeTask ? (
          <KanbanCard task={activeTask} isDragOverlay />
        ) : null}
      </DragOverlay>
    </DndContext>
  )
}
