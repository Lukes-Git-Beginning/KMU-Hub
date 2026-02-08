/**
 * Cross-project task search page with filters and highlighted results.
 *
 * Route: /work/search
 * Features:
 * - Auto-focused search input
 * - TaskFilterBar for multi-dimensional filtering
 * - Results show project key + task key, title with highlighted term,
 *   status, priority, assignee, and due date
 * - Pagination at bottom
 */
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  ChevronLeft,
  ChevronRight,
  Calendar,
  User,
} from 'lucide-react'
import { useSearchTasks } from '@/api/hooks/useTasks'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import StatusBadge from './StatusBadge'
import PriorityBadge from './PriorityBadge'
import type { Priority } from './PriorityBadge'
import TaskFilterBar, { EMPTY_FILTERS } from './TaskFilterBar'
import type { TaskFilters } from './TaskFilterBar'

const PAGE_SIZE = 20

/**
 * Highlight matching search terms in text by wrapping them in <mark> tags.
 */
function highlightText(text: string, query: string): React.ReactNode {
  if (!query || query.length < 2) return text
  const parts = text.split(new RegExp(`(${query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi'))
  return parts.map((part, i) =>
    part.toLowerCase() === query.toLowerCase() ? (
      <mark key={i} className="bg-yellow-200 text-foreground rounded-sm px-0.5">
        {part}
      </mark>
    ) : (
      part
    )
  )
}

export default function TaskSearchView() {
  const navigate = useNavigate()
  const [searchInput, setSearchInput] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')
  const [page, setPage] = useState(1)
  const [filters, setFilters] = useState<TaskFilters>(EMPTY_FILTERS)

  // Debounce search input
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(searchInput)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [searchInput])

  const { data, isLoading } = useSearchTasks({
    q: debouncedQuery,
    project_ids:
      filters.projectIds.length > 0 ? filters.projectIds : undefined,
    assignee_ids:
      filters.assigneeIds.length > 0 ? filters.assigneeIds : undefined,
    include_completed: filters.includeCompleted || undefined,
    page,
    page_size: PAGE_SIZE,
  })

  const tasks = data?.tasks ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div className="p-6 space-y-4">
      {/* Search input */}
      <div className="relative max-w-lg">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="Suche nach Aufgaben, Projekten und mehr..."
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          className="pl-9 text-base h-10"
          autoFocus
        />
      </div>

      {/* Filter bar */}
      <TaskFilterBar filters={filters} onFiltersChange={setFilters} />

      {/* Results */}
      {debouncedQuery.length < 2 ? (
        <div className="flex flex-col items-center justify-center py-16">
          <Search className="h-12 w-12 text-muted-foreground" />
          <p className="mt-4 text-lg font-medium text-foreground">
            Suche nach Aufgaben, Projekten und mehr
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            Mindestens 2 Zeichen eingeben, um die Suche zu starten.
          </p>
        </div>
      ) : isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-14 w-full" />
          ))}
        </div>
      ) : tasks.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16">
          <Search className="h-10 w-10 text-muted-foreground" />
          <p className="mt-4 text-base font-medium text-foreground">
            Keine Ergebnisse fuer &apos;{debouncedQuery}&apos;
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            Versuche andere Suchbegriffe oder weniger Filter.
          </p>
        </div>
      ) : (
        <>
          <p className="text-sm text-muted-foreground">
            {total} Ergebnis{total !== 1 ? 'se' : ''} fuer &quot;{debouncedQuery}&quot;
          </p>

          <div className="space-y-1">
            {tasks.map((task) => {
              const taskKey =
                task.project_key && task.task_number
                  ? `${task.project_key}-${task.task_number}`
                  : ''

              return (
                <div
                  key={task.id}
                  className="flex items-center gap-3 rounded-md border border-border px-4 py-2.5 hover:bg-accent/50 cursor-pointer transition-colors"
                  onClick={() => {
                    if (task.project_id && task.id) {
                      navigate(
                        `/work/projects/${task.project_id}/tasks/${task.id}`
                      )
                    }
                  }}
                >
                  {/* Task key */}
                  {taskKey && (
                    <span className="text-xs font-mono text-muted-foreground shrink-0 w-20">
                      {taskKey}
                    </span>
                  )}

                  {/* Title (highlighted) */}
                  <span className="text-sm flex-1 truncate">
                    {highlightText(task.title ?? '', debouncedQuery)}
                  </span>

                  {/* Status */}
                  {task.status_name && (
                    <StatusBadge
                      name={task.status_name}
                      color={task.status_color}
                      className="shrink-0"
                    />
                  )}

                  {/* Priority */}
                  <PriorityBadge
                    priority={(task.priority as Priority) ?? 'medium'}
                    compact
                  />

                  {/* Assignee */}
                  {task.assignee_name && (
                    <span className="text-xs text-muted-foreground flex items-center gap-1 shrink-0 max-w-[100px] truncate">
                      <User className="h-3 w-3 shrink-0" />
                      {task.assignee_name}
                    </span>
                  )}

                  {/* Due date */}
                  {task.due_date && (
                    <span className="text-xs text-muted-foreground flex items-center gap-1 shrink-0">
                      <Calendar className="h-3 w-3" />
                      {new Date(task.due_date).toLocaleDateString('de-DE', {
                        day: '2-digit',
                        month: '2-digit',
                      })}
                    </span>
                  )}
                </div>
              )
            })}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between pt-2">
              <p className="text-sm text-muted-foreground">
                Seite {page} von {totalPages}
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
