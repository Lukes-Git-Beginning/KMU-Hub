import { Clock, X } from 'lucide-react'
import type { RecentSearch } from '@/stores/search'

interface RecentSearchesProps {
  searches: RecentSearch[]
  onSelect: (query: string) => void
  onRemove: (query: string) => void
  onClearAll: () => void
}

export function RecentSearches({
  searches,
  onSelect,
  onRemove,
  onClearAll,
}: RecentSearchesProps) {
  if (searches.length === 0) return null

  return (
    <div className="px-2 py-1.5">
      <div className="flex items-center justify-between px-2 pb-1">
        <span className="text-xs font-medium text-muted-foreground">
          Letzte Suchen
        </span>
        <button
          onClick={onClearAll}
          className="text-xs text-muted-foreground transition-colors hover:text-foreground"
        >
          Alle loeschen
        </button>
      </div>
      {searches.map((search) => (
        <div
          key={search.query}
          className="group flex items-center gap-2 rounded-md px-3 py-1.5 text-sm transition-colors hover:bg-accent"
        >
          <Clock className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
          <button
            onClick={() => onSelect(search.query)}
            className="min-w-0 flex-1 truncate text-left text-foreground/80"
          >
            {search.query}
          </button>
          <button
            onClick={() => onRemove(search.query)}
            className="shrink-0 rounded p-0.5 text-muted-foreground opacity-0 transition-opacity hover:text-foreground group-hover:opacity-100"
          >
            <X className="h-3 w-3" />
          </button>
        </div>
      ))}
    </div>
  )
}
