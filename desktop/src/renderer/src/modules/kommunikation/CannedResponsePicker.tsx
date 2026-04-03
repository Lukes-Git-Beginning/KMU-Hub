// TODO: Wire to backend — no canned response API exists yet. Currently uses mock store data.
import { useState, useMemo } from 'react'
import { Zap, Search, Hash } from 'lucide-react'
import { useKommunikationStore } from '@/stores/kommunikation'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface CannedResponsePickerProps {
  onSelect: (content: string) => void
}

export function CannedResponsePicker({ onSelect }: CannedResponsePickerProps) {
  const cannedResponses = useKommunikationStore((s) => s.cannedResponses)
  const incrementUsage = useKommunikationStore((s) => s.incrementCannedResponseUsage)
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')

  const filtered = useMemo(() => {
    if (!search.trim()) return cannedResponses
    const q = search.toLowerCase()
    return cannedResponses.filter(
      (r) =>
        r.title.toLowerCase().includes(q) ||
        r.content.toLowerCase().includes(q) ||
        r.shortcut?.toLowerCase().includes(q) ||
        r.category.toLowerCase().includes(q),
    )
  }, [cannedResponses, search])

  // Group by category
  const grouped = useMemo(() => {
    const groups: Record<string, typeof filtered> = {}
    for (const r of filtered) {
      const cat = r.category || 'allgemein'
      if (!groups[cat]) groups[cat] = []
      groups[cat].push(r)
    }
    return groups
  }, [filtered])

  const handleSelect = (id: string, content: string) => {
    onSelect(content)
    incrementUsage(id)
    setOpen(false)
    setSearch('')
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          title="Schnellantwort einfügen"
        >
          <Zap className="h-4 w-4" />
        </button>
      </PopoverTrigger>
      <PopoverContent
        className="w-72 p-0"
        side="top"
        align="start"
        sideOffset={8}
      >
        {/* Search */}
        <div className="p-2 border-b border-border">
          <div className="relative">
            <Search className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Suchen oder /kürzel..."
              className="h-7 w-full rounded border border-border bg-transparent pl-7 pr-2 text-xs outline-none placeholder:text-muted-foreground focus:border-primary"
              autoFocus
            />
          </div>
        </div>

        {/* Results */}
        <div className="max-h-64 overflow-y-auto">
          {Object.keys(grouped).length === 0 ? (
            <div className="p-3 text-center text-xs text-muted-foreground">
              Keine Schnellantworten gefunden
            </div>
          ) : (
            Object.entries(grouped).map(([category, responses]) => (
              <div key={category}>
                <div className="flex items-center gap-1.5 px-3 py-1.5 bg-secondary/30">
                  <Hash className="h-3 w-3 text-muted-foreground" />
                  <span className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">
                    {category}
                  </span>
                </div>
                {responses.map((r) => (
                  <button
                    key={r.id}
                    onClick={() => handleSelect(r.id, r.content)}
                    className="flex w-full flex-col gap-0.5 px-3 py-2 text-left hover:bg-accent transition-colors"
                  >
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-medium text-foreground">{r.title}</span>
                      {r.shortcut && (
                        <code className="rounded bg-secondary px-1 py-0.5 text-[10px] text-muted-foreground font-mono">
                          {r.shortcut}
                        </code>
                      )}
                      <span className="ml-auto text-[10px] text-muted-foreground">
                        {r.usageCount}x
                      </span>
                    </div>
                    <p className="text-[11px] text-muted-foreground truncate">{r.content}</p>
                  </button>
                ))}
              </div>
            ))
          )}
        </div>
      </PopoverContent>
    </Popover>
  )
}
