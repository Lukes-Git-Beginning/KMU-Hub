/**
 * Controls bar above the task list providing grouping, sorting, and filtering.
 *
 * Group by: Status, Assignee, Priority, Due Date, None (flat).
 * Sort within groups: Created, Due Date, Priority, Title, Task Number + asc/desc.
 * Quick filter pills for priority levels, assignee, and status dropdowns.
 */
import { ArrowUpDown, Filter, ChevronDown, ChevronUp } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'

export type GroupBy = 'status' | 'assignee' | 'priority' | 'due_date' | 'none'
export type SortBy = 'created_at' | 'due_date' | 'priority' | 'title' | 'task_number'

interface TaskListHeaderProps {
  groupBy: GroupBy
  onGroupByChange: (value: GroupBy) => void
  sortBy: SortBy
  onSortByChange: (value: SortBy) => void
  sortDesc: boolean
  onSortDescChange: (value: boolean) => void
  filterPriority: string | null
  onFilterPriorityChange: (value: string | null) => void
  filterStatusId: string | null
  onFilterStatusIdChange: (value: string | null) => void
  statuses: Array<{ id?: string; name?: string; color?: string }>
}

const GROUP_OPTIONS: Array<{ value: GroupBy; label: string }> = [
  { value: 'status', label: 'Status' },
  { value: 'assignee', label: 'Zustaendig' },
  { value: 'priority', label: 'Prioritaet' },
  { value: 'due_date', label: 'Faelligkeitsdatum' },
  { value: 'none', label: 'Keine Gruppierung' },
]

const SORT_OPTIONS: Array<{ value: SortBy; label: string }> = [
  { value: 'created_at', label: 'Erstellt' },
  { value: 'due_date', label: 'Faellig am' },
  { value: 'priority', label: 'Prioritaet' },
  { value: 'title', label: 'Titel' },
  { value: 'task_number', label: 'Aufgabennummer' },
]

const PRIORITY_PILLS: Array<{ value: string; label: string; className: string }> = [
  { value: 'urgent', label: 'Dringend', className: 'bg-red-50 text-red-700 border-red-200 hover:bg-red-100' },
  { value: 'high', label: 'Hoch', className: 'bg-orange-50 text-orange-700 border-orange-200 hover:bg-orange-100' },
  { value: 'medium', label: 'Normal', className: 'bg-blue-50 text-blue-700 border-blue-200 hover:bg-blue-100' },
  { value: 'low', label: 'Niedrig', className: 'bg-gray-50 text-gray-500 border-gray-200 hover:bg-gray-100' },
]

export default function TaskListHeader({
  groupBy,
  onGroupByChange,
  sortBy,
  onSortByChange,
  sortDesc,
  onSortDescChange,
  filterPriority,
  onFilterPriorityChange,
  filterStatusId,
  onFilterStatusIdChange,
  statuses,
}: TaskListHeaderProps) {
  const hasActiveFilters = filterPriority !== null || filterStatusId !== null

  return (
    <div className="flex flex-wrap items-center gap-2 pb-3 border-b border-border">
      {/* Group by */}
      <div className="flex items-center gap-1.5">
        <span className="text-xs text-muted-foreground">Gruppieren:</span>
        <Select value={groupBy} onValueChange={(v) => onGroupByChange(v as GroupBy)}>
          <SelectTrigger className="h-7 w-[140px] text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {GROUP_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Sort by */}
      <div className="flex items-center gap-1.5">
        <span className="text-xs text-muted-foreground">Sortieren:</span>
        <Select value={sortBy} onValueChange={(v) => onSortByChange(v as SortBy)}>
          <SelectTrigger className="h-7 w-[130px] text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {SORT_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7"
          onClick={() => onSortDescChange(!sortDesc)}
          title={sortDesc ? 'Absteigend' : 'Aufsteigend'}
        >
          {sortDesc ? (
            <ChevronDown className="h-3.5 w-3.5" />
          ) : (
            <ChevronUp className="h-3.5 w-3.5" />
          )}
        </Button>
      </div>

      {/* Divider */}
      <div className="h-5 w-px bg-border" />

      {/* Filter popover */}
      <Popover>
        <PopoverTrigger asChild>
          <Button
            variant={hasActiveFilters ? 'secondary' : 'ghost'}
            size="sm"
            className="h-7 gap-1 text-xs"
          >
            <Filter className="h-3 w-3" />
            Filter
            {hasActiveFilters && (
              <span className="ml-1 rounded-full bg-primary px-1.5 py-0.5 text-[10px] text-primary-foreground">
                {[filterPriority, filterStatusId].filter(Boolean).length}
              </span>
            )}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-72 space-y-3" align="start">
          {/* Priority quick filters */}
          <div className="space-y-1.5">
            <span className="text-xs font-medium text-muted-foreground">Prioritaet</span>
            <div className="flex flex-wrap gap-1">
              {PRIORITY_PILLS.map((pill) => (
                <button
                  key={pill.value}
                  type="button"
                  className={`rounded-full border px-2.5 py-0.5 text-xs transition-colors ${
                    filterPriority === pill.value
                      ? pill.className + ' ring-1 ring-offset-1 ring-current'
                      : 'bg-background text-muted-foreground border-border hover:bg-accent'
                  }`}
                  onClick={() =>
                    onFilterPriorityChange(
                      filterPriority === pill.value ? null : pill.value
                    )
                  }
                >
                  {pill.label}
                </button>
              ))}
            </div>
          </div>

          {/* Status filter */}
          <div className="space-y-1.5">
            <span className="text-xs font-medium text-muted-foreground">Status</span>
            <Select
              value={filterStatusId ?? '__all__'}
              onValueChange={(v) =>
                onFilterStatusIdChange(v === '__all__' ? null : v)
              }
            >
              <SelectTrigger className="h-8 text-xs">
                <SelectValue placeholder="Alle Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">Alle Status</SelectItem>
                {statuses.filter((s) => s.id).map((s) => (
                  <SelectItem key={s.id} value={s.id!}>
                    <span className="flex items-center gap-2">
                      <span
                        className="h-2 w-2 rounded-full inline-block"
                        style={{ backgroundColor: s.color || '#6b7280' }}
                      />
                      {s.name}
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Clear filters */}
          {hasActiveFilters && (
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-full text-xs"
              onClick={() => {
                onFilterPriorityChange(null)
                onFilterStatusIdChange(null)
              }}
            >
              Filter zuruecksetzen
            </Button>
          )}
        </PopoverContent>
      </Popover>
    </div>
  )
}
