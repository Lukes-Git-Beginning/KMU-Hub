import { Search, X } from 'lucide-react'
import { useWikiStore } from '@/stores/wiki'

export function WikiSearch() {
  const searchQuery = useWikiStore((s) => s.searchQuery)
  const setSearchQuery = useWikiStore((s) => s.setSearchQuery)

  return (
    <div className="px-2 py-2">
      <div className="relative">
        <Search className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Wiki durchsuchen..."
          className="h-8 w-full rounded-md border border-border bg-transparent pl-7 pr-7 text-xs outline-none transition-colors placeholder:text-muted-foreground focus:border-primary"
        />
        {searchQuery && (
          <button
            onClick={() => setSearchQuery('')}
            className="absolute right-1.5 top-1/2 -translate-y-1/2 rounded p-0.5 text-muted-foreground hover:text-foreground"
          >
            <X className="h-3 w-3" />
          </button>
        )}
      </div>
    </div>
  )
}
