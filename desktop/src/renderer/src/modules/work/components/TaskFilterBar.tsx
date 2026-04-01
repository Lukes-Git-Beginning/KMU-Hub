/**
 * Reusable filter component for task search and list views.
 *
 * Provides multi-select filters for project, assignee, status, priority,
 * due date range, and a completed tasks toggle. Composes filter values
 * into query parameters for search/list hooks.
 */
import {
  FolderKanban,
  CircleDot,
  AlertTriangle,
  Calendar,
  RotateCcw,
} from 'lucide-react'
import { cn } from '@/lib'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Badge } from '@/components/ui/badge'
import { useProjects } from '@/api/hooks/useProjects'

export interface TaskFilters {
  projectIds: string[]
  assigneeIds: string[]
  statusIds: string[]
  priorities: string[]
  dueDateFrom?: string
  dueDateTo?: string
  includeCompleted: boolean
}

const EMPTY_FILTERS: TaskFilters = {
  projectIds: [],
  assigneeIds: [],
  statusIds: [],
  priorities: [],
  dueDateFrom: undefined,
  dueDateTo: undefined,
  includeCompleted: false,
}

const PRIORITY_OPTIONS = [
  { value: 'urgent', label: 'Dringend', className: 'text-destructive' },
  { value: 'high', label: 'Hoch', className: 'text-orange-600' },
  { value: 'medium', label: 'Normal', className: 'text-blue-600' },
  { value: 'low', label: 'Niedrig', className: 'text-gray-500' },
]

interface TaskFilterBarProps {
  filters: TaskFilters
  onFiltersChange: (filters: TaskFilters) => void
  /** Hide project filter (e.g. when inside a project detail page) */
  hideProjectFilter?: boolean
}

export default function TaskFilterBar({
  filters,
  onFiltersChange,
  hideProjectFilter = false,
}: TaskFilterBarProps) {
  const { data: projectsData } = useProjects({ page_size: 100 })
  const projects = projectsData?.projects ?? []

  const activeFilterCount = [
    filters.projectIds.length > 0,
    filters.assigneeIds.length > 0,
    filters.statusIds.length > 0,
    filters.priorities.length > 0,
    !!filters.dueDateFrom || !!filters.dueDateTo,
    filters.includeCompleted,
  ].filter(Boolean).length

  function toggleArrayValue(
    arr: string[],
    value: string
  ): string[] {
    return arr.includes(value)
      ? arr.filter((v) => v !== value)
      : [...arr, value]
  }

  function resetFilters() {
    onFiltersChange(EMPTY_FILTERS)
  }

  return (
    <div className="flex items-center gap-2 flex-wrap">
      {/* Project filter */}
      {!hideProjectFilter && (
        <Popover>
          <PopoverTrigger asChild>
            <Button
              variant="outline"
              size="sm"
              className={cn(
                'h-7 gap-1 text-xs',
                filters.projectIds.length > 0 && 'border-primary text-primary'
              )}
            >
              <FolderKanban className="h-3.5 w-3.5" />
              Projekt
              {filters.projectIds.length > 0 && (
                <Badge
                  variant="secondary"
                  className="h-4 min-w-4 px-1 text-xs rounded-full"
                >
                  {filters.projectIds.length}
                </Badge>
              )}
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-56 p-1" align="start">
            <div className="max-h-48 overflow-y-auto space-y-0.5">
              {projects.length === 0 ? (
                <p className="px-2 py-1.5 text-xs text-muted-foreground">
                  Keine Projekte
                </p>
              ) : (
                projects.map((p) => (
                  <button
                    key={p.id}
                    type="button"
                    className={cn(
                      'flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors',
                      p.id && filters.projectIds.includes(p.id) && 'bg-accent'
                    )}
                    onClick={() =>
                      p.id &&
                      onFiltersChange({
                        ...filters,
                        projectIds: toggleArrayValue(
                          filters.projectIds,
                          p.id
                        ),
                      })
                    }
                  >
                    <span
                      className={cn(
                        'h-3 w-3 rounded-sm border',
                        p.id && filters.projectIds.includes(p.id)
                          ? 'bg-primary border-primary'
                          : 'border-border'
                      )}
                    />
                    <span className="truncate">{p.name}</span>
                    <span className="ml-auto text-muted-foreground font-mono">
                      {p.project_key}
                    </span>
                  </button>
                ))
              )}
            </div>
          </PopoverContent>
        </Popover>
      )}

      {/* Priority filter */}
      <Popover>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            className={cn(
              'h-7 gap-1 text-xs',
              filters.priorities.length > 0 && 'border-primary text-primary'
            )}
          >
            <AlertTriangle className="h-3.5 w-3.5" />
            Priorität
            {filters.priorities.length > 0 && (
              <Badge
                variant="secondary"
                className="h-4 min-w-4 px-1 text-xs rounded-full"
              >
                {filters.priorities.length}
              </Badge>
            )}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-44 p-1" align="start">
          <div className="space-y-0.5">
            {PRIORITY_OPTIONS.map((opt) => (
              <button
                key={opt.value}
                type="button"
                className={cn(
                  'flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors',
                  filters.priorities.includes(opt.value) && 'bg-accent'
                )}
                onClick={() =>
                  onFiltersChange({
                    ...filters,
                    priorities: toggleArrayValue(
                      filters.priorities,
                      opt.value
                    ),
                  })
                }
              >
                <span
                  className={cn(
                    'h-3 w-3 rounded-sm border',
                    filters.priorities.includes(opt.value)
                      ? 'bg-primary border-primary'
                      : 'border-border'
                  )}
                />
                <span className={opt.className}>{opt.label}</span>
              </button>
            ))}
          </div>
        </PopoverContent>
      </Popover>

      {/* Due date filter */}
      <Popover>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            className={cn(
              'h-7 gap-1 text-xs',
              (filters.dueDateFrom || filters.dueDateTo) &&
                'border-primary text-primary'
            )}
          >
            <Calendar className="h-3.5 w-3.5" />
            Fällig
            {(filters.dueDateFrom || filters.dueDateTo) && (
              <Badge
                variant="secondary"
                className="h-4 px-1 text-xs rounded-full"
              >
                1
              </Badge>
            )}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-56 p-3 space-y-2" align="start">
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">Von</label>
            <Input
              type="date"
              className="h-7 text-xs"
              value={filters.dueDateFrom ?? ''}
              onChange={(e) =>
                onFiltersChange({
                  ...filters,
                  dueDateFrom: e.target.value || undefined,
                })
              }
            />
          </div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">Bis</label>
            <Input
              type="date"
              className="h-7 text-xs"
              value={filters.dueDateTo ?? ''}
              onChange={(e) =>
                onFiltersChange({
                  ...filters,
                  dueDateTo: e.target.value || undefined,
                })
              }
            />
          </div>
          {(filters.dueDateFrom || filters.dueDateTo) && (
            <Button
              variant="ghost"
              size="sm"
              className="w-full h-7 text-xs"
              onClick={() =>
                onFiltersChange({
                  ...filters,
                  dueDateFrom: undefined,
                  dueDateTo: undefined,
                })
              }
            >
              Datum zurücksetzen
            </Button>
          )}
        </PopoverContent>
      </Popover>

      {/* Include completed toggle */}
      <Button
        variant="outline"
        size="sm"
        className={cn(
          'h-7 gap-1 text-xs',
          filters.includeCompleted && 'border-primary text-primary'
        )}
        onClick={() =>
          onFiltersChange({
            ...filters,
            includeCompleted: !filters.includeCompleted,
          })
        }
      >
        <CircleDot className="h-3.5 w-3.5" />
        Erledigte
      </Button>

      {/* Reset all */}
      {activeFilterCount > 0 && (
        <Button
          variant="ghost"
          size="sm"
          className="h-7 gap-1 text-xs text-muted-foreground"
          onClick={resetFilters}
        >
          <RotateCcw className="h-3.5 w-3.5" />
          Zurücksetzen
        </Button>
      )}
    </div>
  )
}

 
export { EMPTY_FILTERS }
