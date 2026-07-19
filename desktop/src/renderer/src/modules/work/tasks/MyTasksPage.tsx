/**
 * My Tasks page showing tasks assigned to the current user across all projects.
 *
 * Features:
 * - Groups tasks by project (with standalone tasks in "Persönlich" section)
 * - Standalone task creation (no project required)
 * - Filter controls: priority, due date, show completed
 * - Move standalone tasks to a project
 */
import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  ChevronLeft,
  ChevronRight,
  Plus,
  FolderKanban,
  Flag,
  CircleDot,
  CalendarClock,
  MoreHorizontal,
  ArrowRight,
  Circle,
  CheckCircle2,
  Trash2,
  UserPlus,
  CalendarDays,
} from 'lucide-react'
import { toast } from 'sonner'
import { cn } from '@/lib'
import { PageHeader } from '@/components/shared/PageHeader'
import { EmptyState } from '@/components/shared/EmptyState'
import { EmptyTasks } from '@/components/shared/illustrations'
import { ListChecks } from 'lucide-react'
import { useMyTasks, useCreateTask, useUpdateTask, useDeleteTask } from '@/api/hooks/useTasks'
import { useProjects } from '@/api/hooks/useProjects'
import { useWorkPrefsStore, type MyTasksGroupBy } from '@/stores/workPrefs'
import { useWorkStore } from '@/stores/work'
import { useAuthStore } from '@/stores/auth'
import TaskDetailPanel from './TaskDetailPanel'
import { useHasCapability } from '@/hooks/useCapability'
import { useTaskCan } from '../useWorkPermissions'
import { RestrictedModeBadge } from '@/components/shared/rbac/RestrictedModeBadge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
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
import type { Priority } from '../components/PriorityBadge'

const PAGE_SIZE = 50

// ---------------------------------------------------------------------------
// Per-task row — extracted so useTaskCan can be called at the top level
// (React Rules of Hooks forbid calling hooks inside map callbacks).
// ---------------------------------------------------------------------------
interface MyTaskRowProps {
  task: TaskItem
  rowPad: string
  projects: { id?: string; name?: string; project_key?: string }[]
  onOpen: (task: TaskItem) => void
  onToggleComplete: (task: TaskItem) => void
  onAssignToMe: (task: TaskItem) => void
  onSetQuickDue: (task: TaskItem, days: number | null) => void
  onMoveToProject: (task: TaskItem, project: { id?: string; name?: string }) => void
  onDeleteTarget: (task: TaskItem) => void
  formatDueDate: (date?: string) => string | null
  isDueOverdue: (date?: string) => boolean
  t: (key: string, opts?: Record<string, unknown>) => string
}

function MyTaskRow({
  task,
  rowPad,
  projects,
  onOpen,
  onToggleComplete,
  onAssignToMe,
  onSetQuickDue,
  onMoveToProject,
  onDeleteTarget,
  formatDueDate,
  isDueOverdue,
  t,
}: MyTaskRowProps) {
  const can = useTaskCan(task)
  const priorityConfig = PRIORITY_CONFIG[task.priority ?? 'normal']
  const dueLabel = formatDueDate(task.due_date)
  const overdue = isDueOverdue(task.due_date)
  const moveTargets = projects.filter((p) => p.id !== task.project_id)
  const hasMenu = can.complete || can.assignSelf || can.edit || (moveTargets.length > 0 && can.edit) || can.delete

  return (
    <div
      key={task.id}
      className={cn(
        'flex items-center gap-3 rounded-md border border-border px-3 hover:bg-accent/50 transition-colors',
        rowPad,
      )}
    >
      {/* Quick-complete checkbox */}
      {can.complete ? (
        <button
          type="button"
          onClick={() => onToggleComplete(task)}
          className={cn(
            'group/check shrink-0 rounded-full p-1.5 transition-colors',
            task.completed_at
              ? 'text-success hover:bg-success/10'
              : 'text-muted-foreground hover:bg-success/10 hover:text-success',
          )}
          aria-label={
            task.completed_at
              ? t('work.myTasks.reopen', { defaultValue: 'Wieder öffnen' })
              : t('work.myTasks.complete', { defaultValue: 'Abschließen' })
          }
          title={
            task.completed_at
              ? t('work.myTasks.reopen', { defaultValue: 'Wieder öffnen' })
              : t('work.myTasks.complete', { defaultValue: 'Abschließen' })
          }
        >
          {task.completed_at ? (
            <CheckCircle2 className="h-6 w-6" />
          ) : (
            <>
              <Circle className="h-6 w-6 group-hover/check:hidden" />
              <CheckCircle2 className="hidden h-6 w-6 group-hover/check:block" />
            </>
          )}
        </button>
      ) : (
        <span className="shrink-0 p-1.5 text-muted-foreground/30">
          {task.completed_at ? (
            <CheckCircle2 className="h-6 w-6 text-success/50" />
          ) : (
            <Circle className="h-6 w-6" />
          )}
        </span>
      )}

      {/* Open task — title + meta */}
      <button
        type="button"
        onClick={() => onOpen(task)}
        className="flex flex-1 min-w-0 items-center gap-3 rounded text-left transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
      >
        <span className="flex-1 min-w-0 flex items-center gap-2">
          {task.project_key && task.task_number ? (
            <span className="text-xs font-mono text-muted-foreground shrink-0">
              {task.project_key}-{task.task_number}
            </span>
          ) : null}
          <span
            className={cn(
              'text-sm truncate',
              task.completed_at && 'line-through text-muted-foreground',
            )}
          >
            {task.title}
          </span>
        </span>

        {priorityConfig && (
          <span
            className={cn(
              'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold shrink-0',
              priorityConfig.className,
            )}
          >
            {t(priorityConfig.labelKey)}
          </span>
        )}

        {dueLabel && (
          <span
            className={cn(
              'text-xs shrink-0',
              overdue ? 'text-destructive font-medium' : 'text-muted-foreground',
            )}
          >
            {dueLabel}
          </span>
        )}
      </button>

      {/* Quick actions menu — only if at least one action is available */}
      {hasMenu && (
        <Popover>
          <PopoverTrigger asChild>
            <button
              type="button"
              className="text-muted-foreground hover:text-foreground p-0.5 rounded transition-colors shrink-0"
              title={t('work.myTasks.actions', { defaultValue: 'Aktionen' })}
              aria-label={t('work.myTasks.actions', { defaultValue: 'Aktionen' })}
            >
              <MoreHorizontal className="h-4 w-4" />
            </button>
          </PopoverTrigger>
          <PopoverContent className="w-56 p-1" align="end">
            {can.complete && (
              <button
                type="button"
                className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors"
                onClick={() => onToggleComplete(task)}
              >
                {task.completed_at ? (
                  <Circle className="h-3.5 w-3.5 text-muted-foreground" />
                ) : (
                  <CheckCircle2 className="h-3.5 w-3.5 text-success" />
                )}
                {task.completed_at
                  ? t('work.myTasks.reopen', { defaultValue: 'Wieder öffnen' })
                  : t('work.myTasks.complete', { defaultValue: 'Abschließen' })}
              </button>
            )}
            {can.assignSelf && (
              <button
                type="button"
                className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors"
                onClick={() => onAssignToMe(task)}
              >
                <UserPlus className="h-3.5 w-3.5 text-muted-foreground" />
                {t('work.myTasks.assignToMe', { defaultValue: 'Mir zuweisen' })}
              </button>
            )}

            {can.edit && (
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
                      onClick={() => onSetQuickDue(task, opt.d)}
                    >
                      {t(opt.key, { defaultValue: opt.dv })}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {can.edit && moveTargets.length > 0 && (
              <div className="mt-1 border-t border-border pt-1">
                <p className="px-2 py-1 text-[11px] font-medium text-muted-foreground">
                  {t('work.myTasks.moveToProject')}
                </p>
                <div className="max-h-32 overflow-y-auto space-y-0.5">
                  {moveTargets.map((p) => (
                    <button
                      key={p.id}
                      type="button"
                      className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors"
                      onClick={() => onMoveToProject(task, p)}
                    >
                      <ArrowRight className="h-3 w-3 text-muted-foreground" />
                      <span className="truncate">{p.name}</span>
                      <span className="ml-auto font-mono text-muted-foreground">
                        {p.project_key}
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            )}

            {can.delete && (
              <div className="mt-1 border-t border-border pt-1">
                <button
                  type="button"
                  className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs text-destructive hover:bg-error-light transition-colors"
                  onClick={() => onDeleteTarget(task)}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                  {t('common.delete', { defaultValue: 'Löschen' })}
                </button>
              </div>
            )}
          </PopoverContent>
        </Popover>
      )}
    </div>
  )
}

const PRIORITY_CONFIG: Record<string, { labelKey: string; className: string }> = {
  urgent: { labelKey: 'work.priority.urgent', className: 'bg-error-light text-destructive border-destructive/30' },
  high: { labelKey: 'work.priority.high', className: 'bg-orange-100 text-orange-700 border-orange-300' },
  normal: { labelKey: 'work.priority.normal', className: 'bg-blue-100 text-blue-700 border-blue-300' },
  low: { labelKey: 'work.priority.low', className: 'bg-gray-100 text-gray-500 border-gray-300' },
}

const PRIORITY_OPTIONS: Array<{ value: string; labelKey: string }> = [
  { value: 'urgent', labelKey: 'work.priority.urgent' },
  { value: 'high', labelKey: 'work.priority.high' },
  { value: 'normal', labelKey: 'work.priority.normal' },
  { value: 'low', labelKey: 'work.priority.low' },
]

interface TaskItem {
  id?: string
  title?: string
  project_id?: string
  project_key?: string
  project_name?: string
  task_number?: number
  status_id?: string
  status_name?: string
  status_color?: string
  priority?: string
  assignee_name?: string
  assignee_id?: string
  created_by?: string
  due_date?: string
  completed_at?: string
}

interface ProjectGroup {
  /** Unique render key — distinguishes groups across all grouping modes. */
  id: string
  projectId: string | null
  projectName: string
  projectKey: string
  tasks: TaskItem[]
}

const PRIORITY_ORDER = ['urgent', 'high', 'normal', 'low']

/** Bucket a task into a group for the given grouping mode. */
function groupForTask(
  task: TaskItem,
  groupBy: MyTasksGroupBy,
  t: (k: string, opts?: Record<string, unknown>) => string,
): { key: string; order: number; label: string } {
  if (groupBy === 'priority') {
    const p = task.priority ?? 'normal'
    const order = PRIORITY_ORDER.indexOf(p)
    return { key: `prio-${p}`, order: order < 0 ? 99 : order, label: t(`work.priority.${p}`) }
  }
  if (groupBy === 'status') {
    const name = task.status_name || t('work.list.noStatus')
    return { key: `status-${name}`, order: 0, label: name }
  }
  if (groupBy === 'dueDate') {
    if (!task.due_date) return { key: 'due-none', order: 4, label: t('work.myTasks.dueNone') }
    const diff = Math.ceil((new Date(task.due_date).getTime() - Date.now()) / 86_400_000)
    if (diff < 0) return { key: 'due-overdue', order: 0, label: t('work.myTasks.dueOverdue') }
    if (diff === 0) return { key: 'due-today', order: 1, label: t('work.time.today') }
    if (diff <= 7) return { key: 'due-week', order: 2, label: t('work.myTasks.dueThisWeek') }
    return { key: 'due-later', order: 3, label: t('work.myTasks.dueLater') }
  }
  // project (default) handled separately
  return { key: '__project__', order: 0, label: '' }
}

export default function MyTasksPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)
  const [priorityFilter, setPriorityFilter] = useState<string[]>([])
  const [includeCompleted, setIncludeCompleted] = useState(false)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)

  // Standalone task creation form
  const [newTaskTitle, setNewTaskTitle] = useState('')
  const [newTaskPriority, setNewTaskPriority] = useState<string>('normal')

  const createTask = useCreateTask()
  const updateTask = useUpdateTask()
  const deleteTask = useDeleteTask()
  const openTaskPanel = useWorkStore((s) => s.openTaskPanel)
  const currentUserId = useAuthStore((s) => s.user?.id)
  const [deleteTarget, setDeleteTarget] = useState<TaskItem | null>(null)

  const groupBy = useWorkPrefsStore((s) => s.myTasksGroupBy)
  const density = useWorkPrefsStore((s) => s.density)
  const canCreateTask = useHasCapability('work:task:create')

  const { data: projectsData } = useProjects({ page_size: 100 })
  const projects = projectsData?.projects ?? []

   
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  const { data, isLoading, error, refetch } = useMyTasks({
    page,
    page_size: PAGE_SIZE,
    search: debouncedSearch || undefined,
    include_completed: includeCompleted || undefined,
    priority: (priorityFilter.length === 1 ? priorityFilter[0] : undefined) as
      | 'urgent'
      | 'high'
      | 'normal'
      | 'low'
      | undefined,
  })

  const tasks: TaskItem[] = useMemo(() => data?.tasks ?? [], [data?.tasks])
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // Apply client-side priority filter if multiple priorities selected
  const filteredTasks = useMemo(() => {
    if (priorityFilter.length <= 1) return tasks
    return tasks.filter((t) => priorityFilter.includes(t.priority ?? 'normal'))
  }, [tasks, priorityFilter])

  // Group tasks by the user's chosen dimension (work settings).
  const groups = useMemo(() => {
    if (groupBy === 'project') {
      const map = new Map<string, ProjectGroup>()
      const standaloneKey = '__standalone__'
      map.set(standaloneKey, {
        id: standaloneKey,
        projectId: null,
        projectName: t('work.myTasks.personal'),
        projectKey: '',
        tasks: [],
      })
      for (const task of filteredTasks) {
        const key = task.project_id || standaloneKey
        if (!map.has(key)) {
          map.set(key, {
            id: key,
            projectId: task.project_id ?? null,
            projectName: task.project_name || (task.project_key ? `${t('work.settings.project')} ${task.project_key}` : t('work.myTasks.unknown')),
            projectKey: task.project_key ?? '',
            tasks: [],
          })
        }
        map.get(key)!.tasks.push(task)
      }
      const result: ProjectGroup[] = []
      const standalone = map.get(standaloneKey)
      if (standalone && standalone.tasks.length > 0) result.push(standalone)
      for (const [key, group] of map) {
        if (key !== standaloneKey && group.tasks.length > 0) result.push(group)
      }
      return result
    }

    // priority / status / dueDate
    const map = new Map<string, ProjectGroup & { order: number }>()
    for (const task of filteredTasks) {
      const { key, order, label } = groupForTask(task, groupBy, t)
      if (!map.has(key)) {
        map.set(key, { id: key, projectId: null, projectName: label, projectKey: '', tasks: [], order })
      }
      map.get(key)!.tasks.push(task)
    }
    return [...map.values()].sort((a, b) => a.order - b.order)
  }, [filteredTasks, groupBy, t])

  const rowPad = density === 'compact' ? 'py-1' : 'py-2'
  const GroupIcon =
    groupBy === 'priority' ? Flag : groupBy === 'status' ? CircleDot : groupBy === 'dueDate' ? CalendarClock : FolderKanban

  function formatDueDate(dueDate?: string): string | null {
    if (!dueDate) return null
    const date = new Date(dueDate)
    const now = new Date()
    const diffMs = date.getTime() - now.getTime()
    const diffDays = Math.ceil(diffMs / (1000 * 60 * 60 * 24))

    if (diffDays < 0) return t('work.list.daysOverdue', { count: Math.abs(diffDays) })
    if (diffDays === 0) return t('work.time.today')
    if (diffDays === 1) return t('work.time.tomorrow')
    return date.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' })
  }

  function isDueOverdue(dueDate?: string): boolean {
    if (!dueDate) return false
    // eslint-disable-next-line react-hooks/purity -- Date.now() needed for overdue check
    return new Date(dueDate).getTime() < Date.now()
  }

  async function handleCreateStandaloneTask() {
    if (!newTaskTitle.trim()) return
    await createTask.mutateAsync({
      title: newTaskTitle.trim(),
      priority: newTaskPriority as Priority,
      // Assign to the current user so the standalone task shows up in "My Tasks".
      assignee_id: currentUserId,
    })
    setNewTaskTitle('')
    setNewTaskPriority('normal')
    setCreateDialogOpen(false)
  }

  /** Open a task: project tasks go to the full detail page, standalone tasks
   *  open the centered detail modal (the full page route needs a project). */
  function openTask(task: TaskItem) {
    if (!task.id) return
    if (task.project_id) {
      navigate(`/work/projects/${task.project_id}/tasks/${task.id}`)
    } else {
      openTaskPanel(task.id)
    }
  }

  function handleMoveToProject(task: TaskItem, project: { id?: string; name?: string }) {
    if (!task.id || !project.id) return
    updateTask.mutate(
      { id: task.id, project_id: project.id },
      {
        onSuccess: () =>
          toast.success(
            t('work.myTasks.movedToProject', { project: project.name ?? '' })
          ),
      }
    )
  }

  function toggleComplete(task: TaskItem) {
    if (!task.id) return
    const done = !!task.completed_at
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

  function setQuickDue(task: TaskItem, days: number | null) {
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

  function assignToMe(task: TaskItem) {
    if (!task.id || !currentUserId) return
    updateTask.mutate(
      { id: task.id, assignee_id: currentUserId },
      { onSuccess: () => toast.success(t('work.myTasks.assignedToMe', { defaultValue: 'Dir zugewiesen' })) },
    )
  }

  function confirmDelete() {
    if (!deleteTarget?.id) return
    deleteTask.mutate(deleteTarget.id, {
      onSuccess: () => {
        toast.success(t('work.myTasks.deletedToast', { defaultValue: 'Aufgabe gelöscht' }))
        setDeleteTarget(null)
      },
    })
  }

  function togglePriorityFilter(value: string) {
    setPriorityFilter((prev) =>
      prev.includes(value) ? prev.filter((v) => v !== value) : [...prev, value]
    )
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            {t('work.myTasks.loadError')}
          </p>
          <p className="mt-2 text-sm text-muted-foreground">
            {error instanceof Error ? error.message : t('work.common.unexpectedError')}
          </p>
          <Button variant="outline" className="mt-4" onClick={() => refetch()}>
            {t('common.retry')}
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-4">
      <PageHeader
        title={t('work.myTasks.title')}
        icon={ListChecks}
        moduleId="tasks"
        actions={
          <div className="flex items-center gap-2">
            <RestrictedModeBadge module="work" />
            {canCreateTask && (
              <Button className="gap-2" onClick={() => setCreateDialogOpen(true)}>
                <Plus className="h-4 w-4" />
                {t('work.tasks.newTask')}
              </Button>
            )}
          </div>
        }
      />

      {/* Search bar + filters */}
      <div className="flex items-center gap-3 flex-wrap">
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder={t('work.myTasks.searchPlaceholder')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>

        {/* Priority filter */}
        <Popover>
          <PopoverTrigger asChild>
            <Button
              variant="outline"
              size="sm"
              className={cn(
                'h-8 gap-1 text-xs',
                priorityFilter.length > 0 && 'border-primary text-primary'
              )}
            >
              {t('work.tasks.priority')}
              {priorityFilter.length > 0 && (
                <Badge
                  variant="secondary"
                  className="h-4 min-w-4 px-1 text-xs rounded-full"
                >
                  {priorityFilter.length}
                </Badge>
              )}
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-40 p-1" align="start">
            <div className="space-y-0.5">
              {PRIORITY_OPTIONS.map((opt) => (
                <button
                  key={opt.value}
                  type="button"
                  className={cn(
                    'flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors',
                    priorityFilter.includes(opt.value) && 'bg-accent'
                  )}
                  onClick={() => togglePriorityFilter(opt.value)}
                >
                  <span
                    className={cn(
                      'h-3 w-3 rounded-sm border',
                      priorityFilter.includes(opt.value)
                        ? 'bg-primary border-primary'
                        : 'border-border'
                    )}
                  />
                  {t(opt.labelKey)}
                </button>
              ))}
            </div>
          </PopoverContent>
        </Popover>

        {/* Include completed toggle */}
        <Button
          variant="outline"
          size="sm"
          className={cn(
            'h-8 text-xs',
            includeCompleted && 'border-primary text-primary'
          )}
          onClick={() => setIncludeCompleted(!includeCompleted)}
        >
          {t('work.myTasks.showCompleted')}
        </Button>
      </div>

      {/* Task list grouped by project */}
      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-14 w-full" />
          ))}
        </div>
      ) : filteredTasks.length === 0 ? (
        <EmptyState
          illustration={<EmptyTasks />}
          title={t('work.myTasks.noTasksAssigned')}
          description={
            debouncedSearch
              ? t('work.projects.tryOtherSearch')
              : t('work.myTasks.noTasksHint')
          }
          action={canCreateTask ? { label: t('work.myTasks.createFirst'), onClick: () => setCreateDialogOpen(true) } : undefined}
        />
      ) : (
        <>
          <div className="space-y-6">
            {groups.map((group) => (
              <div key={group.id}>
                {/* Group header */}
                <div className="flex items-center gap-2 mb-2">
                  <GroupIcon className="h-3.5 w-3.5 text-muted-foreground" />
                  <h2 className="text-sm font-semibold text-foreground">
                    {group.projectName}
                  </h2>
                  {group.projectKey && (
                    <Badge variant="outline" className="text-xs font-mono">
                      {group.projectKey}
                    </Badge>
                  )}
                  <span className="text-xs text-muted-foreground">
                    ({group.tasks.length})
                  </span>
                </div>

                {/* Tasks in this group */}
                <div className="space-y-1">
                  {group.tasks.map((task) => (
                    <MyTaskRow
                      key={task.id}
                      task={task}
                      rowPad={rowPad}
                      projects={projects}
                      onOpen={openTask}
                      onToggleComplete={toggleComplete}
                      onAssignToMe={assignToMe}
                      onSetQuickDue={setQuickDue}
                      onMoveToProject={handleMoveToProject}
                      onDeleteTarget={setDeleteTarget}
                      formatDueDate={formatDueDate}
                      isDueOverdue={isDueOverdue}
                      t={t}
                    />
                  ))}
                </div>
              </div>
            ))}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                {t('work.myTasks.totalTasks', { count: total })}
              </p>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
                <span className="text-sm text-muted-foreground">
                  {t('work.pagination.page', { page, totalPages })}
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}
        </>
      )}

      {/* Standalone task creation dialog */}
      <Dialog open={canCreateTask && createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t('work.tasks.newTask')}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t('work.list.title')}</label>
              <Input
                placeholder={t('work.tasks.titlePlaceholder')}
                value={newTaskTitle}
                onChange={(e) => setNewTaskTitle(e.target.value)}
                autoFocus
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && newTaskTitle.trim()) {
                    handleCreateStandaloneTask()
                  }
                }}
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t('work.tasks.priority')}</label>
              <div className="flex gap-2">
                {PRIORITY_OPTIONS.map((opt) => (
                  <button
                    key={opt.value}
                    type="button"
                    className={cn(
                      'rounded-md border px-3 py-1.5 text-xs font-medium transition-colors',
                      newTaskPriority === opt.value
                        ? 'border-primary bg-primary/10 text-primary'
                        : 'border-border text-muted-foreground hover:bg-accent'
                    )}
                    onClick={() => setNewTaskPriority(opt.value)}
                  >
                    {t(opt.labelKey)}
                  </button>
                ))}
              </div>
            </div>

            <p className="text-xs text-muted-foreground">
              {t('work.myTasks.standaloneHint')}
            </p>

            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => setCreateDialogOpen(false)}
              >
                {t('common.cancel')}
              </Button>
              <Button
                onClick={handleCreateStandaloneTask}
                disabled={!newTaskTitle.trim() || createTask.isPending}
              >
                {createTask.isPending ? t('work.tasks.creating') : t('common.create')}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Delete confirmation — only reachable via can.delete rows, guard nonetheless */}
      <Dialog open={!!deleteTarget} onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>{t('work.myTasks.deleteTitle', { defaultValue: 'Aufgabe löschen?' })}</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            {t('work.myTasks.deleteBody', {
              defaultValue: '„{title}" wird dauerhaft gelöscht. Das kann nicht rückgängig gemacht werden.',
              title: deleteTarget?.title ?? '',
            })}
          </p>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              {t('common.cancel')}
            </Button>
            <Button variant="destructive" onClick={confirmDelete} disabled={deleteTask.isPending}>
              {t('common.delete', { defaultValue: 'Löschen' })}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Detail modal for standalone tasks (project tasks navigate to full page) */}
      <TaskDetailPanel />
    </div>
  )
}
