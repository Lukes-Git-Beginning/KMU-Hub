/**
 * 5.6 — Canned Response Picker (inline dropdown)
 *
 * Small popover that appears in the ticket reply area for quick-picking
 * a canned response. Shows title + shortcut + category.
 *
 * Uses the Radix Popover (portal + collision-aware positioning) so it never
 * gets clipped by the surrounding modal's `overflow-hidden` and always stays
 * inside the viewport — it sits in the reply toolbar (bottom-right of the
 * ticket modal) and opens upward, right-aligned.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Zap, Search } from 'lucide-react'
import { useCannedResponses } from '@/api/hooks/useHelpdesk'
import { wireCannedToDisplay, type DisplayCannedResponse } from '@/api/helpdesk-adapters'
import { Popover, PopoverTrigger, PopoverContent } from '@/components/ui/popover'

interface CannedResponsePickerProps {
  onSelect: (content: string) => void
}

export function CannedResponsePicker({ onSelect }: CannedResponsePickerProps) {
  const { t } = useTranslation()
  const { data: wireCanned = [] } = useCannedResponses()
  const cannedResponses: DisplayCannedResponse[] = wireCanned.map(wireCannedToDisplay)
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')

  const filtered = cannedResponses.filter((r) => {
    if (!search) return true
    const q = search.toLowerCase()
    return (
      r.title.toLowerCase().includes(q) ||
      r.shortcut.toLowerCase().includes(q) ||
      r.category.toLowerCase().includes(q)
    )
  })

  const handlePick = (r: DisplayCannedResponse) => {
    onSelect(r.content.replace(/<[^>]*>/g, ''))
    setOpen(false)
    setSearch('')
  }

  return (
    <Popover open={open} onOpenChange={(v) => { setOpen(v); if (!v) setSearch('') }}>
      <PopoverTrigger asChild>
        <button
          className="flex items-center gap-1 rounded-lg px-2 py-1 text-xs text-muted-foreground hover:bg-secondary transition-colors"
          title={t('helpdesk.cannedResponses.insertTemplate')}
        >
          <Zap className="h-3.5 w-3.5" />
          {t('helpdesk.cannedResponses.templates')}
        </button>
      </PopoverTrigger>
      <PopoverContent side="top" align="end" sideOffset={6} className="w-72 p-0 overflow-hidden">
        {/* Search */}
        <div className="p-2 border-b border-border">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('common.search')}
              autoFocus
              className="w-full rounded-md border border-border bg-card pl-8 pr-2 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
            />
          </div>
        </div>

        {/* List */}
        <div className="max-h-48 overflow-y-auto py-1">
          {filtered.length === 0 && (
            <p className="px-3 py-2 text-xs text-muted-foreground text-center">{t('helpdesk.cannedResponses.noTemplatesFound')}</p>
          )}
          {filtered.map((r) => (
            <button
              key={r.id}
              onClick={() => handlePick(r)}
              className="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-secondary transition-colors"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-1.5">
                  <span className="text-xs font-medium text-foreground truncate">{r.title}</span>
                  <span className="rounded bg-secondary px-1 py-0.5 text-[9px] font-mono text-muted-foreground shrink-0">
                    {r.shortcut}
                  </span>
                </div>
                <span className="text-[10px] text-muted-foreground">{r.category}</span>
              </div>
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}
