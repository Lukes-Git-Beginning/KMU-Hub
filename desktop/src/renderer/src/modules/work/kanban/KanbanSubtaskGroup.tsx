/**
 * Collapsible subtask group within Kanban columns.
 *
 * Renders subtasks as visually smaller, indented cards beneath their parent
 * card within the same column. Uses collapse/expand toggle on the parent.
 * Maximum visual nesting depth: 3 levels on Kanban (deeper levels show
 * "N more nested" link). Keeps the board clean per design principle.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronRight, ChevronDown } from 'lucide-react'
import { cn } from '@/lib'
import type { TaskData } from '../list/TaskRow'
import PriorityBadge from '../components/PriorityBadge'
import type { Priority } from '../components/PriorityBadge'

interface KanbanSubtaskGroupProps {
  subtasks: TaskData[]
  depth: number
  onTaskClick: (taskId: string) => void
  /** Recursive subtask map: taskId -> direct children in this column */
  childrenMap: Map<string, TaskData[]>
}

const MAX_KANBAN_DEPTH = 3

export default function KanbanSubtaskGroup({
  subtasks,
  depth,
  onTaskClick,
  childrenMap,
}: KanbanSubtaskGroupProps) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(depth < 2)

  if (subtasks.length === 0) return null

  // Beyond max depth, show a summary link
  if (depth >= MAX_KANBAN_DEPTH) {
    return (
      <div className="ml-3 border-l-2 border-muted pl-2 py-0.5">
        <button
          type="button"
          className="text-xs text-muted-foreground hover:text-foreground transition-colors"
          onClick={() => {
            if (subtasks[0]?.id) onTaskClick(subtasks[0].id)
          }}
        >
          {t('work.kanban.moreNested', { count: subtasks.length })}
        </button>
      </div>
    )
  }

  return (
    <div className="ml-3 border-l-2 border-muted">
      {/* Collapse/expand toggle */}
      <button
        type="button"
        className="flex items-center gap-1 px-2 py-0.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        {expanded ? (
          <ChevronDown className="h-3 w-3" />
        ) : (
          <ChevronRight className="h-3 w-3" />
        )}
        {t('work.tasks.subtaskCount', { count: subtasks.length })}
      </button>

      {expanded && (
        <div className="space-y-1 pl-2 pb-1">
          {subtasks.map((task) => {
            const taskId = task.id ?? ''
            const nestedChildren = childrenMap.get(taskId) ?? []

            return (
              <div key={taskId}>
                {/* Subtask mini-card */}
                <button
                  type="button"
                  className={cn(
                    'w-full text-left rounded border border-border bg-background/80 px-2 py-1 text-xs transition-colors hover:bg-accent/50',
                    task.is_closed && 'opacity-60 line-through'
                  )}
                  onClick={() => onTaskClick(taskId)}
                >
                  <div className="flex items-center gap-1.5">
                    {task.project_key && task.task_number ? (
                      <span className="font-mono text-[10px] text-muted-foreground shrink-0">
                        {task.project_key}-{task.task_number}
                      </span>
                    ) : null}
                    <span className="truncate flex-1">{task.title}</span>
                    <PriorityBadge
                      priority={(task.priority as Priority) ?? 'normal'}
                      compact
                      className="scale-75"
                    />
                  </div>
                </button>

                {/* Recursive nested children */}
                {nestedChildren.length > 0 && (
                  <KanbanSubtaskGroup
                    subtasks={nestedChildren}
                    depth={depth + 1}
                    onTaskClick={onTaskClick}
                    childrenMap={childrenMap}
                  />
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
