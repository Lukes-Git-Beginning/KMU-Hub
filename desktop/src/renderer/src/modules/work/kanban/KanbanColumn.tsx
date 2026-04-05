/**
 * Single Kanban column representing one status.
 *
 * Uses useDroppable from @dnd-kit/core for drop target.
 * Groups tasks by parent hierarchy: top-level tasks render as KanbanCard,
 * subtasks whose parent is in the same column render in KanbanSubtaskGroup.
 */
import { useTranslation } from 'react-i18next'
import { useDroppable } from '@dnd-kit/core'
import {
  SortableContext,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { Plus } from 'lucide-react'
import { cn } from '@/lib'
import { ScrollArea } from '@/components/ui/scroll-area'
import type { TaskData } from '../list/TaskRow'
import KanbanCard from './KanbanCard'
import KanbanSubtaskGroup from './KanbanSubtaskGroup'

interface StatusInfo {
  id: string
  name: string
  color?: string
  is_closed?: boolean
}

interface KanbanColumnProps {
  status: StatusInfo
  /** All tasks in this column (flat, includes subtasks) */
  tasks: TaskData[]
  /** Map of taskId -> direct children that are in this same column */
  childrenMap: Map<string, TaskData[]>
  /** Set of all task IDs in this column (for checking if parent is here) */
  columnTaskIds: Set<string>
  onTaskClick: (taskId: string) => void
  onAddTask?: () => void
}

export default function KanbanColumn({
  status,
  tasks,
  childrenMap,
  columnTaskIds,
  onTaskClick,
  onAddTask,
}: KanbanColumnProps) {
  const { t } = useTranslation()
  const { setNodeRef, isOver } = useDroppable({
    id: `column-${status.id}`,
    data: { statusId: status.id },
  })

  // Identify displayable tasks: top-level (no parent) or parent is in a different column
  const displayTasks = tasks.filter((t) => {
    if (!t.parent_task_id) return true
    // Parent is not in this column -> show as standalone
    return !columnTaskIds.has(t.parent_task_id)
  })

  const taskIds = displayTasks.map((t) => t.id ?? '')

  return (
    <div
      className={cn(
        'flex flex-col rounded-lg bg-muted/30 border border-border/50 min-w-[280px] w-[280px] shrink-0',
        isOver && 'border-primary/50 bg-primary/5'
      )}
    >
      {/* Column header */}
      <div className="flex items-center justify-between px-3 py-2.5 border-b border-border/50">
        <div className="flex items-center gap-2">
          <span
            className="h-2.5 w-2.5 rounded-full shrink-0"
            style={{ backgroundColor: status.color || '#6b7280' }}
          />
          <span className="text-sm font-medium truncate">{status.name}</span>
          <span className="text-xs text-muted-foreground">
            {tasks.length}
          </span>
        </div>
        {onAddTask && (
          <button
            type="button"
            className="rounded p-0.5 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
            onClick={onAddTask}
            title={t('work.tasks.newTask')}
          >
            <Plus className="h-3.5 w-3.5" />
          </button>
        )}
      </div>

      {/* Column body */}
      <div ref={setNodeRef} className="flex-1 min-h-0">
        <ScrollArea className="h-full">
          <SortableContext
            items={taskIds}
            strategy={verticalListSortingStrategy}
          >
            <div className="space-y-2 p-2">
              {displayTasks.length === 0 ? (
                <div className="rounded-lg border-2 border-dashed border-border/60 p-4 text-center">
                  <p className="text-xs text-muted-foreground">
                    {t('work.kanban.emptyColumn')}
                  </p>
                </div>
              ) : (
                displayTasks.map((task) => {
                  const taskId = task.id ?? ''
                  const subtasksInColumn = childrenMap.get(taskId) ?? []

                  return (
                    <div key={taskId}>
                      <KanbanCard
                        task={task}
                        onClick={() => onTaskClick(taskId)}
                      />
                      {subtasksInColumn.length > 0 && (
                        <KanbanSubtaskGroup
                          subtasks={subtasksInColumn}
                          depth={1}
                          onTaskClick={onTaskClick}
                          childrenMap={childrenMap}
                        />
                      )}
                    </div>
                  )
                })
              )}
            </div>
          </SortableContext>
        </ScrollArea>
      </div>
    </div>
  )
}

export type { StatusInfo }
