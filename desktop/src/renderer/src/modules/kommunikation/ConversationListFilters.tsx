import { SlidersHorizontal } from 'lucide-react'
import type { ConversationStatus } from '@/types/communication'

// ---------------------------------------------------------------------------
// Status filter options
// ---------------------------------------------------------------------------

interface StatusOption {
  id: ConversationStatus | 'all'
  label: string
  color: string
}

const statusOptions: StatusOption[] = [
  { id: 'all', label: 'Alle', color: '' },
  { id: 'open', label: 'Offen', color: 'bg-success' },
  { id: 'pending', label: 'Wartend', color: 'bg-warning' },
  { id: 'resolved', label: 'Geloest', color: 'bg-blue-500' },
  { id: 'closed', label: 'Geschlossen', color: 'bg-muted-foreground' },
]

// ---------------------------------------------------------------------------
// Sort options
// ---------------------------------------------------------------------------

export type ConversationSort = 'newest' | 'oldest' | 'priority' | 'unread'

interface SortOption {
  id: ConversationSort
  label: string
}

const sortOptions: SortOption[] = [
  { id: 'newest', label: 'Neueste' },
  { id: 'oldest', label: 'Aelteste' },
  { id: 'priority', label: 'Prioritaet' },
  { id: 'unread', label: 'Ungelesen' },
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ConversationListFiltersProps {
  statusFilter: ConversationStatus | 'all'
  onStatusChange: (status: ConversationStatus | 'all') => void
  sort: ConversationSort
  onSortChange: (sort: ConversationSort) => void
  totalCount: number
}

export function ConversationListFilters({
  statusFilter,
  onStatusChange,
  sort,
  onSortChange,
  totalCount,
}: ConversationListFiltersProps) {
  return (
    <div className="flex items-center gap-2 px-3 py-2 border-b border-border">
      {/* Status chips */}
      <div className="flex items-center gap-1 flex-1 overflow-x-auto">
        {statusOptions.map((opt) => {
          const isActive = statusFilter === opt.id
          return (
            <button
              key={opt.id}
              onClick={() => onStatusChange(opt.id)}
              className={`flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-medium whitespace-nowrap transition-colors ${
                isActive
                  ? 'bg-foreground/10 text-foreground'
                  : 'text-muted-foreground hover:bg-accent'
              }`}
            >
              {opt.color && (
                <span className={`h-1.5 w-1.5 rounded-full ${opt.color}`} />
              )}
              {opt.label}
            </button>
          )
        })}
      </div>

      {/* Count */}
      <span className="text-[10px] text-muted-foreground shrink-0">
        {totalCount}
      </span>

      {/* Sort dropdown */}
      <div className="relative shrink-0">
        <select
          value={sort}
          onChange={(e) => onSortChange(e.target.value as ConversationSort)}
          className="appearance-none rounded-md border border-border bg-transparent pl-6 pr-2 py-1 text-[11px] text-muted-foreground outline-none cursor-pointer hover:bg-accent transition-colors"
        >
          {sortOptions.map((opt) => (
            <option key={opt.id} value={opt.id}>{opt.label}</option>
          ))}
        </select>
        <SlidersHorizontal className="absolute left-1.5 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground pointer-events-none" />
      </div>
    </div>
  )
}
