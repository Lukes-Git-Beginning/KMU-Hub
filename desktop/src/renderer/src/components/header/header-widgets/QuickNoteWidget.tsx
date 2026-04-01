/**
 * Header mini-widget — quick pinned note / reminder.
 */
import { useState } from 'react'
import { StickyNote, X } from 'lucide-react'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { Button } from '@/components/ui/button'

export function QuickNoteWidget() {
  const [note, setNote] = useState(() =>
    localStorage.getItem('kmuhub-quicknote') ?? ''
  )

  const handleSave = (value: string) => {
    setNote(value)
    if (value) {
      localStorage.setItem('kmuhub-quicknote', value)
    } else {
      localStorage.removeItem('kmuhub-quicknote')
    }
  }

  return (
    <Popover>
      <Tooltip>
        <TooltipTrigger asChild>
          <PopoverTrigger asChild>
            <button className="relative flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs hover:bg-accent transition-colors">
              <StickyNote className={`h-3.5 w-3.5 ${note ? 'text-warning' : 'text-muted-foreground'}`} />
              {note && (
                <span className="hidden md:inline text-muted-foreground max-w-[80px] truncate">
                  {note}
                </span>
              )}
              {note && (
                <span className="h-1.5 w-1.5 rounded-full bg-warning absolute top-1 right-1" />
              )}
            </button>
          </PopoverTrigger>
        </TooltipTrigger>
        <TooltipContent side="bottom">
          {note || 'Kurznotiz hinzufügen'}
        </TooltipContent>
      </Tooltip>

      <PopoverContent className="w-64 p-3" side="bottom" align="center">
        <div className="flex items-center justify-between mb-2">
          <p className="text-xs font-medium text-muted-foreground">Kurznotiz</p>
          {note && (
            <Button
              variant="ghost"
              size="icon"
              className="h-5 w-5"
              onClick={() => handleSave('')}
            >
              <X className="h-3 w-3" />
            </Button>
          )}
        </div>
        <textarea
          value={note}
          onChange={(e) => handleSave(e.target.value)}
          placeholder="Erinnerung, Notiz..."
          rows={3}
          maxLength={200}
          className="w-full rounded-md border border-border bg-input-background px-2.5 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
        />
        <p className="text-right text-[10px] text-muted-foreground mt-1">
          {note.length}/200
        </p>
      </PopoverContent>
    </Popover>
  )
}
