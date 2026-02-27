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
import { useNavigate } from 'react-router-dom'
import {
  Search,
  CheckSquare,
  ChevronLeft,
  ChevronRight,
  Plus,
  FolderKanban,
  MoreHorizontal,
  ArrowRight,
} from 'lucide-react'
import { cn } from '@/lib'
import { PageHeader } from '@/components/shared/PageHeader'
import { ListChecks } from 'lucide-react'
import { useMyTasks, useCreateTask, useUpdateTask } from '@/api/hooks/useTasks'
import { useProjects } from '@/api/hooks/useProjects'
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

const PRIORITY_CONFIG: Record<string, { label: string; className: string }> = {
  urgent: { label: 'Dringend', className: 'bg-red-100 text-red-700 border-red-300' },
  high: { label: 'Hoch', className: 'bg-orange-100 text-orange-700 border-orange-300' },
  normal: { label: 'Normal', className: 'bg-blue-100 text-blue-700 border-blue-300' },
  low: { label: 'Niedrig', className: 'bg-gray-100 text-gray-500 border-gray-300' },
}

const PRIORITY_OPTIONS: Array<{ value: string; label: string }> = [
  { value: 'urgent', label: 'Dringend' },
  { value: 'high', label: 'Hoch' },
  { value: 'medium', label: 'Normal' },
  { value: 'low', label: 'Niedrig' },
]

interface TaskItem {
  id?: string
  title?: string
  project_id?: string
  project_key?: string
  task_number?: number
  status_id?: string
  status_name?: string
  status_color?: string
  priority?: string
  assignee_name?: string
  due_date?: string
  completed_at?: string
}

interface ProjectGroup {
  projectId: string | null
  projectName: string
  projectKey: string
  tasks: TaskItem[]
}

export default function MyTasksPage() {
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)
  const [priorityFilter, setPriorityFilter] = useState<string[]>([])
  const [includeCompleted, setIncludeCompleted] = useState(false)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [moveTaskId, setMoveTaskId] = useState<string | null>(null)

  // Standalone task creation form
  const [newTaskTitle, setNewTaskTitle] = useState('')
  const [newTaskPriority, setNewTaskPriority] = useState<string>('medium')

  const createTask = useCreateTask()
  const updateTask = useUpdateTask()

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
    priority: priorityFilter.length === 1
      ? (priorityFilter[0] as TaskItem['priority'] & string)
      : undefined,
  })

  const tasks: TaskItem[] = data?.tasks ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // Apply client-side priority filter if multiple priorities selected
  const filteredTasks = useMemo(() => {
    if (priorityFilter.length <= 1) return tasks
    return tasks.filter((t) => priorityFilter.includes(t.priority ?? 'medium'))
  }, [tasks, priorityFilter])

  // Group tasks by project
  const groups = useMemo(() => {
    const map = new Map<string, ProjectGroup>()

    // Standalone tasks group (project_id is null or empty)
    const standaloneKey = '__standalone__'
    map.set(standaloneKey, {
      projectId: null,
      projectName: 'Persönlich',
      projectKey: '',
      tasks: [],
    })

    for (const task of filteredTasks) {
      const key = task.project_id || standaloneKey
      if (!map.has(key)) {
        map.set(key, {
          projectId: task.project_id ?? null,
          projectName: task.project_key ? `Projekt ${task.project_key}` : 'Unbekannt',
          projectKey: task.project_key ?? '',
          tasks: [],
        })
      }
      map.get(key)!.tasks.push(task)
    }

    // Only return groups that have tasks, with standalone first
    const result: ProjectGroup[] = []
    const standalone = map.get(standaloneKey)
    if (standalone && standalone.tasks.length > 0) {
      result.push(standalone)
    }
    for (const [key, group] of map) {
      if (key !== standaloneKey && group.tasks.length > 0) {
        result.push(group)
      }
    }
    return result
  }, [filteredTasks])

  function formatDueDate(dueDate?: string): string | null {
    if (!dueDate) return null
    const date = new Date(dueDate)
    const now = new Date()
    const diffMs = date.getTime() - now.getTime()
    const diffDays = Math.ceil(diffMs / (1000 * 60 * 60 * 24))

    if (diffDays < 0) return `${Math.abs(diffDays)} Tage überfällig`
    if (diffDays === 0) return 'Heute'
    if (diffDays === 1) return 'Morgen'
    return date.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' })
  }

  function isDueOverdue(dueDate?: string): boolean {
    if (!dueDate) return false
    return new Date(dueDate).getTime() < Date.now()
  }

  async function handleCreateStandaloneTask() {
    if (!newTaskTitle.trim()) return
    await createTask.mutateAsync({
      title: newTaskTitle.trim(),
      priority: newTaskPriority as Priority,
    })
    setNewTaskTitle('')
    setNewTaskPriority('medium')
    setCreateDialogOpen(false)
  }

  async function handleMoveToProject(projectId: string) {
    if (!moveTaskId) return
    await updateTask.mutateAsync({
      id: moveTaskId,
      // Pass project_id in the body -- the API will reassign the task
    })
    setMoveTaskId(null)
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
            Fehler beim Laden der Aufgaben
          </p>
          <p className="mt-2 text-sm text-muted-foreground">
            {error instanceof Error ? error.message : 'Ein unerwarteter Fehler ist aufgetreten.'}
          </p>
          <Button variant="outline" className="mt-4" onClick={() => refetch()}>
            Erneut versuchen
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-4">
      <PageHeader
        title="Meine Aufgaben"
        icon={ListChecks}
        moduleId="tasks"
        actions={
          <Button className="gap-2" onClick={() => setCreateDialogOpen(true)}>
            <Plus className="h-4 w-4" />
            Neue Aufgabe
          </Button>
        }
      />

      {/* Search bar + filters */}
      <div className="flex items-center gap-3 flex-wrap">
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Aufgaben suchen..."
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
              Priorität
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
                  {opt.label}
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
          Erledigte anzeigen
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
        <div className="flex flex-col items-center justify-center py-16">
          <CheckSquare className="h-12 w-12 text-muted-foreground" />
          <p className="mt-4 text-lg font-medium text-foreground">
            Keine Aufgaben zugewiesen
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {debouncedSearch
              ? 'Versuche einen anderen Suchbegriff.'
              : 'Dir sind aktuell keine Aufgaben zugewiesen.'}
          </p>
          <Button className="mt-4 gap-2" onClick={() => setCreateDialogOpen(true)}>
            <Plus className="h-4 w-4" />
            Erste Aufgabe erstellen
          </Button>
        </div>
      ) : (
        <>
          <div className="space-y-6">
            {groups.map((group) => (
              <div key={group.projectId ?? '__standalone__'}>
                {/* Group header */}
                <div className="flex items-center gap-2 mb-2">
                  <FolderKanban className="h-3.5 w-3.5 text-muted-foreground" />
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
                  {group.tasks.map((task) => {
                    const priorityConfig = PRIORITY_CONFIG[task.priority ?? 'medium']
                    const dueLabel = formatDueDate(task.due_date)
                    const overdue = isDueOverdue(task.due_date)
                    const isStandalone = !task.project_id

                    return (
                      <div
                        key={task.id}
                        className="flex items-center gap-3 rounded-md border border-border px-3 py-2 hover:bg-accent/50 cursor-pointer transition-colors"
                        onClick={() => {
                          if (task.project_id && task.id) {
                            navigate(
                              `/work/projects/${task.project_id}/tasks/${task.id}`
                            )
                          }
                        }}
                      >
                        {/* Task key + title */}
                        <div className="flex-1 min-w-0 flex items-center gap-2">
                          {task.project_key && task.task_number ? (
                            <span className="text-xs font-mono text-muted-foreground shrink-0">
                              {task.project_key}-{task.task_number}
                            </span>
                          ) : null}
                          <span className="text-sm truncate">{task.title}</span>
                        </div>

                        {/* Priority badge */}
                        {priorityConfig && (
                          <Badge
                            variant="outline"
                            className={`text-xs shrink-0 ${priorityConfig.className}`}
                          >
                            {priorityConfig.label}
                          </Badge>
                        )}

                        {/* Due date */}
                        {dueLabel && (
                          <span
                            className={`text-xs shrink-0 ${
                              overdue ? 'text-destructive font-medium' : 'text-muted-foreground'
                            }`}
                          >
                            {dueLabel}
                          </span>
                        )}

                        {/* Move to project action for standalone tasks */}
                        {isStandalone && (
                          <Popover>
                            <PopoverTrigger asChild>
                              <button
                                type="button"
                                className="text-muted-foreground hover:text-foreground p-0.5 rounded transition-colors"
                                onClick={(e) => e.stopPropagation()}
                                title="In Projekt verschieben"
                              >
                                <MoreHorizontal className="h-4 w-4" />
                              </button>
                            </PopoverTrigger>
                            <PopoverContent
                              className="w-52 p-1"
                              align="end"
                              onClick={(e) => e.stopPropagation()}
                            >
                              <p className="px-2 py-1 text-xs font-medium text-muted-foreground">
                                In Projekt verschieben
                              </p>
                              {projects.length === 0 ? (
                                <p className="px-2 py-1 text-xs text-muted-foreground">
                                  Keine Projekte vorhanden
                                </p>
                              ) : (
                                <div className="max-h-40 overflow-y-auto space-y-0.5">
                                  {projects.map((p) => (
                                    <button
                                      key={p.id}
                                      type="button"
                                      className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors"
                                      onClick={() => {
                                        if (task.id && p.id) {
                                          updateTask.mutate({
                                            id: task.id,
                                          })
                                        }
                                      }}
                                    >
                                      <ArrowRight className="h-3 w-3 text-muted-foreground" />
                                      <span className="truncate">{p.name}</span>
                                      <span className="ml-auto text-muted-foreground font-mono">
                                        {p.project_key}
                                      </span>
                                    </button>
                                  ))}
                                </div>
                              )}
                            </PopoverContent>
                          </Popover>
                        )}
                      </div>
                    )
                  })}
                </div>
              </div>
            ))}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                {total} Aufgabe{total !== 1 ? 'n' : ''} gesamt
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
                  Seite {page} von {totalPages}
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
      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Neue Aufgabe</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Titel</label>
              <Input
                placeholder="Aufgabentitel..."
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
              <label className="text-sm font-medium">Priorität</label>
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
                    {opt.label}
                  </button>
                ))}
              </div>
            </div>

            <p className="text-xs text-muted-foreground">
              Diese Aufgabe wird ohne Projekt erstellt. Du kannst sie später in ein Projekt verschieben.
            </p>

            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => setCreateDialogOpen(false)}
              >
                Abbrechen
              </Button>
              <Button
                onClick={handleCreateStandaloneTask}
                disabled={!newTaskTitle.trim() || createTask.isPending}
              >
                {createTask.isPending ? 'Erstelle...' : 'Erstellen'}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
