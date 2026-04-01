/**
 * Snooze popover with preset buttons and custom date/time picker.
 *
 * Presets: 1 Stunde, Morgen frueh (09:00), Nächste Woche (Mon 09:00).
 * Custom: Date + Time inputs with confirm button.
 */
import { useState } from 'react'
import { Clock } from 'lucide-react'
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from '@/components/ui/popover'

interface SnoozePopoverProps {
  onSnooze: (isoTimestamp: string) => void
  children: React.ReactNode
}

function addHours(date: Date, hours: number): Date {
  const result = new Date(date)
  result.setHours(result.getHours() + hours)
  return result
}

function nextDayAt9(): Date {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  d.setHours(9, 0, 0, 0)
  return d
}

function nextMondayAt9(): Date {
  const d = new Date()
  const day = d.getDay()
  // Days until next Monday: if Sun(0)->1, Mon(1)->7, Tue(2)->6, etc.
  const daysUntilMonday = day === 0 ? 1 : 8 - day
  d.setDate(d.getDate() + daysUntilMonday)
  d.setHours(9, 0, 0, 0)
  return d
}

function formatDateForInput(date: Date): string {
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

function formatSnoozeDisplay(date: Date): string {
  return date.toLocaleDateString('de-DE', {
    weekday: 'short',
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function SnoozePopover({ onSnooze, children }: SnoozePopoverProps) {
  const [open, setOpen] = useState(false)
  const [customDate, setCustomDate] = useState(
    formatDateForInput(nextDayAt9()),
  )
  const [customTime, setCustomTime] = useState('09:00')

  const handlePreset = (date: Date) => {
    onSnooze(date.toISOString())
    setOpen(false)
  }

  const handleCustom = () => {
    if (!customDate || !customTime) return
    const dt = new Date(`${customDate}T${customTime}:00`)
    if (isNaN(dt.getTime())) return
    onSnooze(dt.toISOString())
    setOpen(false)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent className="w-64 p-3" align="start">
        <div className="space-y-1">
          <p className="text-xs font-medium text-muted-foreground mb-2">
            Später erinnern
          </p>

          {/* Presets */}
          <button
            onClick={() => handlePreset(addHours(new Date(), 1))}
            className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-foreground hover:bg-secondary transition-colors"
          >
            <Clock className="h-3.5 w-3.5 text-muted-foreground" />
            1 Stunde
          </button>
          <button
            onClick={() => handlePreset(nextDayAt9())}
            className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-foreground hover:bg-secondary transition-colors"
          >
            <Clock className="h-3.5 w-3.5 text-muted-foreground" />
            Morgen frueh
            <span className="ml-auto text-xs text-muted-foreground">
              {formatSnoozeDisplay(nextDayAt9())}
            </span>
          </button>
          <button
            onClick={() => handlePreset(nextMondayAt9())}
            className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-foreground hover:bg-secondary transition-colors"
          >
            <Clock className="h-3.5 w-3.5 text-muted-foreground" />
            Nächste Woche
            <span className="ml-auto text-xs text-muted-foreground">
              {formatSnoozeDisplay(nextMondayAt9())}
            </span>
          </button>

          {/* Divider */}
          <div className="my-2 border-t border-border" />

          {/* Custom */}
          <p className="text-xs font-medium text-muted-foreground mb-1">
            Benutzerdefiniert
          </p>
          <div className="flex gap-2">
            <input
              type="date"
              value={customDate}
              onChange={(e) => setCustomDate(e.target.value)}
              className="flex-1 rounded-md border border-border bg-background px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring"
            />
            <input
              type="time"
              value={customTime}
              onChange={(e) => setCustomTime(e.target.value)}
              className="w-20 rounded-md border border-border bg-background px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring"
            />
          </div>
          <button
            onClick={handleCustom}
            className="mt-1 w-full rounded-md bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            Später erinnern
          </button>
        </div>
      </PopoverContent>
    </Popover>
  )
}
