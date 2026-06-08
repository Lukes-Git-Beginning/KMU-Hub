import { useMemo, useRef, useEffect } from 'react'
import { Clock } from 'lucide-react'
import { Popover, PopoverTrigger, PopoverContent } from '@/components/ui/popover'
import { cn } from '@/lib/cn'

/**
 * Cosmi time picker — a clickable field that opens a popover with two
 * scrollable hour/minute columns. Replaces the native <input type="time">
 * (whose spinner controls don't match the Cosmi look).
 *
 * Value format: "HH:MM" (24h). Reusable across modules.
 */
export interface TimePickerProps {
  value: string
  onChange: (value: string) => void
  /** Minute step in the picker list (default 5). */
  minuteStep?: number
  disabled?: boolean
  ariaLabel?: string
  className?: string
}

export function TimePicker({ value, onChange, minuteStep = 5, disabled, ariaLabel, className }: TimePickerProps) {
  const [hh, mm] = splitTime(value)

  const hours = useMemo(() => Array.from({ length: 24 }, (_, i) => i), [])
  const minutes = useMemo(
    () => Array.from({ length: Math.ceil(60 / minuteStep) }, (_, i) => i * minuteStep),
    [minuteStep],
  )

  const setHour = (h: number) => onChange(`${pad(h)}:${pad(mm)}`)
  const setMinute = (m: number) => onChange(`${pad(hh)}:${pad(m)}`)

  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          disabled={disabled}
          aria-label={ariaLabel}
          className={cn(
            'flex h-9 w-full items-center gap-2 rounded-md border border-border bg-input-background px-3 text-sm text-foreground transition-colors hover:bg-secondary/50 focus:outline-none focus:ring-2 focus:ring-focus-ring disabled:opacity-50',
            className,
          )}
        >
          <Clock className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
          <span className="tabular-nums">{pad(hh)}:{pad(mm)}</span>
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <div className="flex h-56">
          <TimeColumn values={hours} selected={hh} onSelect={setHour} />
          <div className="w-px bg-border" />
          <TimeColumn values={minutes} selected={mm} onSelect={setMinute} />
        </div>
      </PopoverContent>
    </Popover>
  )
}

function TimeColumn({
  values,
  selected,
  onSelect,
}: {
  values: number[]
  selected: number
  onSelect: (v: number) => void
}) {
  const activeRef = useRef<HTMLButtonElement>(null)

  // Scroll the selected value into view when the popover opens.
  useEffect(() => {
    activeRef.current?.scrollIntoView({ block: 'center' })
  }, [])

  return (
    <div className="w-16 overflow-y-auto py-1">
      {values.map((v) => {
        const isActive = v === selected
        return (
          <button
            key={v}
            ref={isActive ? activeRef : undefined}
            type="button"
            onClick={() => onSelect(v)}
            className={cn(
              'flex w-full items-center justify-center px-3 py-1.5 text-sm tabular-nums transition-colors',
              isActive
                ? 'bg-primary font-medium text-primary-foreground'
                : 'text-foreground hover:bg-secondary',
            )}
          >
            {pad(v)}
          </button>
        )
      })}
    </div>
  )
}

function splitTime(value: string): [number, number] {
  const m = /^(\d{1,2}):(\d{1,2})$/.exec(value ?? '')
  if (!m) return [0, 0]
  const h = Math.min(23, Math.max(0, Number(m[1])))
  const min = Math.min(59, Math.max(0, Number(m[2])))
  return [h, min]
}

function pad(n: number): string {
  return String(n).padStart(2, '0')
}
