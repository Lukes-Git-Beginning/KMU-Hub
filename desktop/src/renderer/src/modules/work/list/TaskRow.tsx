/**
 * Single task row in the list view with inline editing capabilities.
 *
 * Shows task key, title, status badge, priority badge, assignee, due date,
 * and subtask indicators. Supports indentation for nested subtasks with
 * collapse/expand toggles.
 */
import { useState } from 'react'
import {
  ChevronRight,
  ChevronDown,
  Lock,
  ListTree,
  Calendar,
  User,
} from 'lucide-react'
import { cn } from '@/lib'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'


import { useUpdateTask } from '@/api/hooks/useTasks'
import { useWorkStore } from '@/stores/work'
import StatusBadge from '../components/StatusBadge'
import PriorityBadge from '../components/PriorityBadge'
import type { Priority } from '../components/PriorityBadge'

interface TaskData {
  id?: string
  title?: string
  project_key?: string
  task_number?: number
  status_id?: string
  status_name?: string
  status_color?: string
  is_closed?: boolean
  priority?: string
  assignee_id?: string
  assignee_name?: string
  due_date?: string
  has_blocked_deps?: boolean
  subtask_count?: number
  completed_subtask_count?: number
  parent_task_id?: string
}

interface StatusOption {
  id?: string
  name?: string
  color?: string
  is_closed?: boolean
}

interface MemberOption {
  user_id?: string
  display_name?: string
  email?: string
}

interface TaskRowProps {
  task: TaskData
  depth: number
  isExpanded: boolean
  hasChildren: boolean
  onToggleExpand: () => void
  statuses: StatusOption[]
  members: MemberOption[]
}

export default function TaskRow({
  task,
  depth,
  isExpanded,
  hasChildren,
  onToggleExpand,
  statuses,
  members,
}: TaskRowProps) {
  const updateTask = useUpdateTask()
  const openTaskPanel = useWorkStore((s) => s.openTaskPanel)
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleValue, setTitleValue] = useState(task.title ?? '')

  function handleTitleSave() {
    setEditingTitle(false)
    const trimmed = titleValue.trim()
    if (trimmed && trimmed !== task.title && task.id) {
      updateTask.mutate({ id: task.id, title: trimmed })
    } else {
      setTitleValue(task.title ?? '')
    }
  }

  function handleStatusChange(statusId: string) {
    if (task.id && statusId !== task.status_id) {
      updateTask.mutate({ id: task.id, status_id: statusId })
    }
  }

  function handlePriorityChange(priority: string) {
    if (task.id && priority !== task.priority) {
      updateTask.mutate({
        id: task.id,
        priority: priority as Priority,
      })
    }
  }

  function handleAssigneeChange(assigneeId: string) {
    if (task.id) {
      updateTask.mutate({
        id: task.id,
        assignee_id: assigneeId === '__none__' ? undefined : assigneeId,
      })
    }
  }

  function handleDueDateChange(dueDate: string) {
    if (task.id) {
      updateTask.mutate({
        id: task.id,
        due_date: dueDate || undefined,
      })
    }
  }

  function formatDueDate(date?: string): string | null {
    if (!date) return null
    const d = new Date(date)
    const now = new Date()
    const diffMs = d.getTime() - now.getTime()
    const diffDays = Math.ceil(diffMs / (1000 * 60 * 60 * 24))
    if (diffDays < 0) return `${Math.abs(diffDays)}d überfällig`
    if (diffDays === 0) return 'Heute'
    if (diffDays === 1) return 'Morgen'
    return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' })
  }

  function isDueOverdue(date?: string): boolean {
    if (!date) return false
    // eslint-disable-next-line react-hooks/purity -- Date.now() needed for overdue check
    return new Date(date).getTime() < Date.now()
  }

  function toDateInputValue(date: unknown): string {
    if (!date) return ''
    try {
      const d = date instanceof Date ? date : new Date(String(date))
      if (isNaN(d.getTime())) return ''
      return d.toISOString().split('T')[0]
    } catch {
      return ''
    }
  }

  const dueLabel = formatDueDate(task.due_date)
  const overdue = isDueOverdue(task.due_date)
  const indentPx = depth * 24

  return (
    <div
      className={cn(
        'group flex items-center gap-2 rounded-md border border-transparent px-2 py-1.5 transition-colors hover:bg-accent/50 hover:border-border',
        task.is_closed && 'opacity-60'
      )}
      style={{ paddingLeft: `${8 + indentPx}px` }}
    >
      {/* Expand/collapse toggle */}
      <div className="w-5 shrink-0 flex justify-center">
        {hasChildren ? (
          <button
            type="button"
            className="rounded p-0.5 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
            onClick={onToggleExpand}
          >
            {isExpanded ? (
              <ChevronDown className="h-3.5 w-3.5" />
            ) : (
              <ChevronRight className="h-3.5 w-3.5" />
            )}
          </button>
        ) : null}
      </div>

      {/* Task key */}
      {task.project_key && task.task_number ? (
        <span className="shrink-0 text-xs font-mono text-muted-foreground w-16 truncate">
          {task.project_key}-{task.task_number}
        </span>
      ) : (
        <span className="w-16 shrink-0" />
      )}

      {/* Title (inline-editable) */}
      <div className="flex-1 min-w-0">
        {editingTitle ? (
          <Input
            className="h-7 text-sm"
            value={titleValue}
            onChange={(e) => setTitleValue(e.target.value)}
            onBlur={handleTitleSave}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleTitleSave()
              if (e.key === 'Escape') {
                setTitleValue(task.title ?? '')
                setEditingTitle(false)
              }
            }}
            autoFocus
          />
        ) : (
          <button
            type="button"
            className="text-sm text-left truncate w-full hover:underline cursor-pointer"
            onClick={() => task.id && openTaskPanel(task.id)}
            onDoubleClick={() => setEditingTitle(true)}
            title="Klicken zum Oeffnen, Doppelklick zum Bearbeiten"
          >
            {task.title}
          </button>
        )}
      </div>

      {/* Blocked indicator */}
      {task.has_blocked_deps && (
        <Lock className="h-3.5 w-3.5 shrink-0 text-yellow-500" title="Blockiert" />
      )}

      {/* Subtask count */}
      {(task.subtask_count ?? 0) > 0 && (
        <span className="shrink-0 flex items-center gap-0.5 text-xs text-muted-foreground" title="Unteraufgaben">
          <ListTree className="h-3 w-3" />
          {task.completed_subtask_count ?? 0}/{task.subtask_count}
        </span>
      )}

      {/* Status (inline edit via popover) */}
      <Popover>
        <PopoverTrigger asChild>
          <button type="button" className="shrink-0 cursor-pointer">
            <StatusBadge
              name={task.status_name ?? 'Offen'}
              color={task.status_color}
              isClosed={task.is_closed}
            />
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-48 p-1" align="end">
          <div className="space-y-0.5">
            {statuses.map((s) => (
              <button
                key={s.id}
                type="button"
                className={cn(
                  'flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors',
                  s.id === task.status_id && 'bg-accent'
                )}
                onClick={() => s.id && handleStatusChange(s.id)}
              >
                <span
                  className="h-2.5 w-2.5 rounded-full"
                  style={{ backgroundColor: s.color || '#6b7280' }}
                />
                {s.name}
              </button>
            ))}
          </div>
        </PopoverContent>
      </Popover>

      {/* Priority (inline edit via popover) */}
      <Popover>
        <PopoverTrigger asChild>
          <button type="button" className="shrink-0 cursor-pointer">
            <PriorityBadge priority={(task.priority as Priority) ?? 'medium'} compact />
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-40 p-1" align="end">
          <div className="space-y-0.5">
            {(['urgent', 'high', 'medium', 'low'] as const).map((p) => (
              <button
                key={p}
                type="button"
                className={cn(
                  'flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors',
                  p === task.priority && 'bg-accent'
                )}
                onClick={() => handlePriorityChange(p)}
              >
                <PriorityBadge priority={p} compact />
                <span>
                  {p === 'urgent'
                    ? 'Dringend'
                    : p === 'high'
                      ? 'Hoch'
                      : p === 'medium'
                        ? 'Normal'
                        : 'Niedrig'}
                </span>
              </button>
            ))}
          </div>
        </PopoverContent>
      </Popover>

      {/* Assignee (inline edit) */}
      <Popover>
        <PopoverTrigger asChild>
          <button
            type="button"
            className="shrink-0 flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground cursor-pointer w-24 truncate"
            title={task.assignee_name || 'Nicht zugewiesen'}
          >
            <User className="h-3 w-3" />
            <span className="truncate">
              {task.assignee_name || '-'}
            </span>
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-48 p-1" align="end">
          <div className="space-y-0.5">
            <button
              type="button"
              className={cn(
                'flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors',
                !task.assignee_id && 'bg-accent'
              )}
              onClick={() => handleAssigneeChange('__none__')}
            >
              Nicht zugewiesen
            </button>
            {members.map((m) => (
              <button
                key={m.user_id}
                type="button"
                className={cn(
                  'flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors',
                  m.user_id === task.assignee_id && 'bg-accent'
                )}
                onClick={() => m.user_id && handleAssigneeChange(m.user_id)}
              >
                {m.display_name || m.email || 'Benutzer'}
              </button>
            ))}
          </div>
        </PopoverContent>
      </Popover>

      {/* Due date (inline edit) */}
      <Popover>
        <PopoverTrigger asChild>
          <button
            type="button"
            className={cn(
              'shrink-0 flex items-center gap-1 text-xs cursor-pointer w-24',
              overdue
                ? 'text-destructive font-medium'
                : dueLabel
                  ? 'text-muted-foreground'
                  : 'text-muted-foreground/50'
            )}
          >
            <Calendar className="h-3 w-3" />
            <span className="truncate">{dueLabel || '-'}</span>
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-3" align="end">
          <div className="space-y-2">
            <label className="text-xs font-medium text-muted-foreground">
              Fällig am
            </label>
            <Input
              type="date"
              className="h-8 text-xs"
              value={toDateInputValue(task.due_date)}
              onChange={(e) => handleDueDateChange(e.target.value)}
            />
            {task.due_date && (
              <Button
                variant="ghost"
                size="sm"
                className="h-7 w-full text-xs"
                onClick={() => handleDueDateChange('')}
              >
                Datum entfernen
              </Button>
            )}
          </div>
        </PopoverContent>
      </Popover>
    </div>
  )
}

export type { TaskData, StatusOption, MemberOption }
