/**
 * Sortable/filterable task list view with user-selectable grouping.
 *
 * Groups tasks by status, assignee, priority, due date, or flat view.
 * Each group is collapsible. Subtask nesting is reflected with
 * indentation and collapse/expand toggles at each level.
 */
import { useState, useMemo } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { useTasks } from '@/api/hooks/useTasks'
import {
  useProjectMembers,
  useProjectPreference,
  useSetPreference,
} from '@/api/hooks/useProjects'
import TaskListHeader, {
  type GroupBy,
  type SortBy,
} from './TaskListHeader'
import TaskRow from './TaskRow'
import type { TaskData, StatusOption, MemberOption } from './TaskRow'
import StatusBadge from '../components/StatusBadge'
import PriorityBadge from '../components/PriorityBadge'
import type { Priority } from '../components/PriorityBadge'

interface TaskListViewProps {
  projectId: string
  statuses: StatusOption[]
}

interface TaskGroup {
  key: string
  label: string
  color?: string
  isClosed?: boolean
  priority?: Priority
  tasks: TaskData[]
}

/** Build a parent-children map from flat task list. */
function buildTree(tasks: TaskData[]): Map<string, TaskData[]> {
  const children = new Map<string, TaskData[]>()
  for (const t of tasks) {
    const parentId = t.parent_task_id ?? '__root__'
    const list = children.get(parentId) ?? []
    list.push(t)
    children.set(parentId, list)
  }
  return children
}

/** Group tasks by the selected dimension. */
function groupTasks(
  rootTasks: TaskData[],
  groupBy: GroupBy,
  statuses: StatusOption[]
): TaskGroup[] {
  if (groupBy === 'none') {
    return [{ key: '__all__', label: 'Alle Aufgaben', tasks: rootTasks }]
  }

  const map = new Map<string, TaskGroup>()

  if (groupBy === 'status') {
    // Pre-populate with all statuses to maintain order
    for (const s of statuses) {
      if (s.id) {
        map.set(s.id, {
          key: s.id,
          label: s.name ?? 'Ohne Status',
          color: s.color,
          isClosed: s.is_closed,
          tasks: [],
        })
      }
    }
  }

  for (const task of rootTasks) {
    let key: string
    let label: string
    let color: string | undefined
    let isClosed: boolean | undefined
    let priority: Priority | undefined

    switch (groupBy) {
      case 'status':
        key = task.status_id ?? '__none__'
        label = task.status_name ?? 'Ohne Status'
        color = task.status_color
        isClosed = task.is_closed
        break
      case 'assignee':
        key = task.assignee_id ?? '__none__'
        label = task.assignee_name ?? 'Nicht zugewiesen'
        break
      case 'priority':
        key = task.priority ?? 'medium'
        label =
          task.priority === 'urgent'
            ? 'Dringend'
            : task.priority === 'high'
              ? 'Hoch'
              : task.priority === 'low'
                ? 'Niedrig'
                : 'Normal'
        priority = (task.priority as Priority) ?? 'medium'
        break
      case 'due_date': {
        if (!task.due_date) {
          key = '__no_date__'
          label = 'Kein Datum'
        } else {
          const d = new Date(task.due_date)
          const now = new Date()
          const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
          const taskDay = new Date(d.getFullYear(), d.getMonth(), d.getDate())
          const diffDays = Math.floor(
            (taskDay.getTime() - today.getTime()) / (1000 * 60 * 60 * 24)
          )

          if (diffDays < 0) {
            key = '__overdue__'
            label = 'Ueberfaellig'
            color = '#ef4444'
          } else if (diffDays === 0) {
            key = '__today__'
            label = 'Heute'
            color = '#f59e0b'
          } else if (diffDays <= 7) {
            key = '__this_week__'
            label = 'Diese Woche'
            color = '#3b82f6'
          } else {
            key = '__later__'
            label = 'Spaeter'
            color = '#6b7280'
          }
        }
        break
      }
      default:
        key = '__all__'
        label = 'Alle'
    }

    const existing = map.get(key)
    if (existing) {
      existing.tasks.push(task)
    } else {
      map.set(key, { key, label, color, isClosed, priority, tasks: [task] })
    }
  }

  // For due_date grouping, enforce a logical order
  if (groupBy === 'due_date') {
    const order = ['__overdue__', '__today__', '__this_week__', '__later__', '__no_date__']
    const sorted: TaskGroup[] = []
    for (const k of order) {
      const g = map.get(k)
      if (g && g.tasks.length > 0) sorted.push(g)
    }
    return sorted
  }

  // For priority grouping, enforce priority order
  if (groupBy === 'priority') {
    const order = ['urgent', 'high', 'medium', 'low']
    const sorted: TaskGroup[] = []
    for (const k of order) {
      const g = map.get(k)
      if (g && g.tasks.length > 0) sorted.push(g)
    }
    return sorted
  }

  return Array.from(map.values()).filter((g) => g.tasks.length > 0)
}

/** Sort tasks by the given sort key. */
function sortTasks(tasks: TaskData[], sortBy: SortBy, sortDesc: boolean): TaskData[] {
  const sorted = [...tasks].sort((a, b) => {
    let cmp = 0
    switch (sortBy) {
      case 'title':
        cmp = (a.title ?? '').localeCompare(b.title ?? '', 'de')
        break
      case 'due_date':
        cmp = (a.due_date ?? '9999').localeCompare(b.due_date ?? '9999')
        break
      case 'priority': {
        const order = { urgent: 0, high: 1, normal: 2, low: 3 }
        cmp =
          (order[a.priority as keyof typeof order] ?? 2) -
          (order[b.priority as keyof typeof order] ?? 2)
        break
      }
      case 'task_number':
        cmp = (a.task_number ?? 0) - (b.task_number ?? 0)
        break
      case 'created_at':
      default:
        // Tasks arrive in server order (created_at desc), keep original
        cmp = 0
        break
    }
    return sortDesc ? -cmp : cmp
  })
  return sorted
}

export default function TaskListView({
  projectId,
  statuses,
}: TaskListViewProps) {
  // Preferences
  const { data: prefData } = useProjectPreference(projectId)
  const setPreference = useSetPreference()

  const [groupBy, setGroupBy] = useState<GroupBy>(
    (prefData?.list_group_by as GroupBy) ?? 'status'
  )
  const [sortBy, setSortBy] = useState<SortBy>(
    (prefData?.list_sort_by as SortBy) ?? 'created_at'
  )
  const [sortDesc, setSortDesc] = useState(prefData?.list_sort_desc ?? false)
  const [filterPriority, setFilterPriority] = useState<string | null>(null)
  const [filterStatusId, setFilterStatusId] = useState<string | null>(null)

  // Track expanded groups and tasks
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(new Set())
  const [expandedTasks, setExpandedTasks] = useState<Set<string>>(new Set())

  // Fetch tasks
  const { data: tasksData, isLoading } = useTasks({
    project_id: projectId,
    priority: filterPriority as TaskData['priority'] | undefined,
    status_id: filterStatusId ?? undefined,
    sort_by: sortBy,
    sort_desc: sortDesc,
    page_size: 200,
  })

  // Fetch members for inline editing
  const { data: membersData } = useProjectMembers(projectId)
  const members: MemberOption[] = membersData?.members ?? []

  const tasks: TaskData[] = tasksData?.tasks ?? []

  // Build tree and group
  const childrenMap = useMemo(() => buildTree(tasks), [tasks])

  const rootTasks = useMemo(
    () => tasks.filter((t) => !t.parent_task_id),
    [tasks]
  )

  const groups = useMemo(
    () => groupTasks(rootTasks, groupBy, statuses),
    [rootTasks, groupBy, statuses]
  )

  // Persist preference changes
  function handleGroupByChange(value: GroupBy) {
    setGroupBy(value)
    setPreference.mutate({ projectId, list_group_by: value })
  }

  function handleSortByChange(value: SortBy) {
    setSortBy(value)
    setPreference.mutate({ projectId, list_sort_by: value })
  }

  function handleSortDescChange(value: boolean) {
    setSortDesc(value)
    setPreference.mutate({ projectId, list_sort_desc: value })
  }

  function toggleGroup(key: string) {
    setCollapsedGroups((prev) => {
      const next = new Set(prev)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      return next
    })
  }

  function toggleExpand(taskId: string) {
    setExpandedTasks((prev) => {
      const next = new Set(prev)
      if (next.has(taskId)) {
        next.delete(taskId)
      } else {
        next.add(taskId)
      }
      return next
    })
  }

  /** Recursively render task and its children. */
  function renderTaskTree(task: TaskData, depth: number): React.ReactNode {
    const taskId = task.id ?? ''
    const children = childrenMap.get(taskId) ?? []
    const hasChildren = children.length > 0
    const isExpanded = expandedTasks.has(taskId)

    return (
      <div key={taskId}>
        <TaskRow
          task={task}
          depth={depth}
          isExpanded={isExpanded}
          hasChildren={hasChildren}
          onToggleExpand={() => toggleExpand(taskId)}
          statuses={statuses}
          members={members}
        />
        {hasChildren && isExpanded && (
          <div>
            {sortTasks(children, sortBy, sortDesc).map((child) =>
              renderTaskTree(child, depth + 1)
            )}
          </div>
        )}
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-2 p-4">
        {Array.from({ length: 10 }).map((_, i) => (
          <Skeleton key={i} className="h-9 w-full rounded-md" />
        ))}
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="px-4 pt-3">
        <TaskListHeader
          groupBy={groupBy}
          onGroupByChange={handleGroupByChange}
          sortBy={sortBy}
          onSortByChange={handleSortByChange}
          sortDesc={sortDesc}
          onSortDescChange={handleSortDescChange}
          filterPriority={filterPriority}
          onFilterPriorityChange={setFilterPriority}
          filterStatusId={filterStatusId}
          onFilterStatusIdChange={setFilterStatusId}
          statuses={statuses}
        />
      </div>

      <div className="flex-1 overflow-y-auto px-4 py-2">
        {groups.length === 0 || rootTasks.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <p className="text-sm text-muted-foreground">
              Keine Aufgaben gefunden.
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {groups.map((group) => {
              const isCollapsed = collapsedGroups.has(group.key)
              const sortedTasks = sortTasks(group.tasks, sortBy, sortDesc)

              return (
                <div key={group.key}>
                  {/* Group header */}
                  {groupBy !== 'none' && (
                    <button
                      type="button"
                      className="flex w-full items-center gap-2 rounded px-2 py-1 text-sm font-medium hover:bg-accent/50 transition-colors"
                      onClick={() => toggleGroup(group.key)}
                    >
                      {isCollapsed ? (
                        <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
                      ) : (
                        <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
                      )}

                      {groupBy === 'status' && (
                        <StatusBadge
                          name={group.label}
                          color={group.color}
                          isClosed={group.isClosed}
                        />
                      )}
                      {groupBy === 'priority' && group.priority && (
                        <PriorityBadge priority={group.priority} />
                      )}
                      {groupBy !== 'status' && groupBy !== 'priority' && (
                        <>
                          {group.color && (
                            <span
                              className="h-2.5 w-2.5 rounded-full"
                              style={{ backgroundColor: group.color }}
                            />
                          )}
                          <span>{group.label}</span>
                        </>
                      )}

                      <span className="text-xs text-muted-foreground ml-1">
                        ({group.tasks.length})
                      </span>
                    </button>
                  )}

                  {/* Group tasks */}
                  {!isCollapsed && (
                    <div className={groupBy !== 'none' ? 'mt-1' : ''}>
                      {sortedTasks.map((task) => renderTaskTree(task, 0))}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
