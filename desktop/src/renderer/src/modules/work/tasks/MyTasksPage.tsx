/**
 * My Tasks page showing tasks assigned to the current user across all projects.
 *
 * Groups tasks by status and displays task title with project key prefix,
 * status/priority badges, assignee, and due date.
 */
import { useState, useEffect, useMemo } from 'react'
import { Search, CheckSquare, ChevronLeft, ChevronRight } from 'lucide-react'
import { useMyTasks } from '@/api/hooks/useTasks'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'

const PAGE_SIZE = 50

const PRIORITY_CONFIG: Record<string, { label: string; className: string }> = {
  urgent: { label: 'Dringend', className: 'bg-red-100 text-red-700 border-red-300' },
  high: { label: 'Hoch', className: 'bg-orange-100 text-orange-700 border-orange-300' },
  normal: { label: 'Normal', className: 'bg-blue-100 text-blue-700 border-blue-300' },
  low: { label: 'Niedrig', className: 'bg-gray-100 text-gray-500 border-gray-300' },
}

interface TaskGroup {
  statusName: string
  statusColor: string
  tasks: TaskItem[]
}

interface TaskItem {
  id?: string
  title?: string
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

export default function MyTasksPage() {
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)

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
    include_completed: false,
  })

  const tasks: TaskItem[] = data?.tasks ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // Group tasks by status
  const groups = useMemo(() => {
    const map = new Map<string, TaskGroup>()
    for (const task of tasks) {
      const key = task.status_name || 'Ohne Status'
      if (!map.has(key)) {
        map.set(key, {
          statusName: key,
          statusColor: task.status_color || '#6b7280',
          tasks: [],
        })
      }
      map.get(key)!.tasks.push(task)
    }
    return Array.from(map.values())
  }, [tasks])

  function formatDueDate(dueDate?: string): string | null {
    if (!dueDate) return null
    const date = new Date(dueDate)
    const now = new Date()
    const diffMs = date.getTime() - now.getTime()
    const diffDays = Math.ceil(diffMs / (1000 * 60 * 60 * 24))

    if (diffDays < 0) return `${Math.abs(diffDays)} Tage ueberfaellig`
    if (diffDays === 0) return 'Heute'
    if (diffDays === 1) return 'Morgen'
    return date.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' })
  }

  function isDueOverdue(dueDate?: string): boolean {
    if (!dueDate) return false
    return new Date(dueDate).getTime() < Date.now()
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
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Meine Aufgaben</h1>
      </div>

      {/* Search bar */}
      <div className="relative max-w-sm">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="Aufgaben suchen..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-9"
        />
      </div>

      {/* Task list */}
      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-14 w-full" />
          ))}
        </div>
      ) : tasks.length === 0 ? (
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
        </div>
      ) : (
        <>
          <div className="space-y-6">
            {groups.map((group) => (
              <div key={group.statusName}>
                {/* Group header */}
                <div className="flex items-center gap-2 mb-2">
                  <div
                    className="h-3 w-3 rounded-full"
                    style={{ backgroundColor: group.statusColor }}
                  />
                  <h2 className="text-sm font-semibold text-foreground">
                    {group.statusName}
                  </h2>
                  <span className="text-xs text-muted-foreground">
                    ({group.tasks.length})
                  </span>
                </div>

                {/* Tasks in this group */}
                <div className="space-y-1">
                  {group.tasks.map((task) => {
                    const priorityConfig = PRIORITY_CONFIG[task.priority ?? 'normal']
                    const dueLabel = formatDueDate(task.due_date)
                    const overdue = isDueOverdue(task.due_date)

                    return (
                      <div
                        key={task.id}
                        className="flex items-center gap-3 rounded-md border border-border px-3 py-2 hover:bg-accent/50 cursor-pointer transition-colors"
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
    </div>
  )
}
